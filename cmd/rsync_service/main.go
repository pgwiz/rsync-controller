//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"rsync-controller/core"
)

const (
	SERVICE_NAME = "RsyncControllerDaemon"
	SERVICE_DESC = "Elevated background disk indexer and rsync daemon for Rsync Controller."
	PIPE_NAME    = `\\.\pipe\rsync-daemon`
)

type DaemonRequest struct {
	Action string `json:"action"`
	Target string `json:"target"`
	Query  string `json:"query"`
}

type DaemonResponse struct {
	Status  string      `json:"status"`
	Engine  string      `json:"engine"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type rsyncWindowsService struct{}

func (m *rsyncWindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Start daemon loop in background goroutine
	stopChan := make(chan struct{})
	go runDaemonLoop(stopChan)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			close(stopChan)
			return false, 0
		}
	}
}

func main() {
	isService, err := svc.IsWindowsService()
	if err == nil && isService {
		_ = svc.Run(SERVICE_NAME, &rsyncWindowsService{})
		return
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--install":
			installService()
			return
		case "--uninstall":
			uninstallService()
			return
		case "--elevate-install":
			elevateInstall()
			return
		case "--run-standalone":
			stopChan := make(chan struct{})
			runDaemonLoop(stopChan)
			return
		}
	}

	fmt.Println("Usage: rsync-service.exe (--install | --uninstall | --elevate-install | --run-standalone)")
}

// elevateInstall triggers Windows UAC prompt to elevate and install
func elevateInstall() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("ERROR: Cannot determine executable path: %v\n", err)
		return
	}

	psCmd := fmt.Sprintf(`Start-Process -FilePath "%s" -ArgumentList "--install" -Verb RunAs`, exePath)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	if err := cmd.Run(); err != nil {
		fmt.Printf("ERROR: Elevation request failed: %v\n", err)
		return
	}
	fmt.Println("INFO: Requested Administrator elevation to install service.")
}

func installService() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("ERROR: Failed to locate executable: %v\n", err)
		os.Exit(1)
	}

	if err := core.InstallAndStartService(exePath); err != nil {
		fmt.Printf("ERROR: Service installation failed: %v\n", err)
		fmt.Println("NOTE: Please run terminal as Administrator to install system services.")
		os.Exit(1)
	}
	fmt.Printf("SUCCESS: Service %s installed and started successfully.\n", core.DaemonServiceName)
}

func uninstallService() {
	if err := core.UninstallService(); err != nil {
		fmt.Printf("ERROR: Service deletion failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SUCCESS: Service %s uninstalled successfully.\n", core.DaemonServiceName)
}

func runDaemonLoop(stopChan chan struct{}) {
	fmt.Printf("INFO: %s starting on pipe: %s\n", SERVICE_NAME, PIPE_NAME)

	engineMgr := core.NewEngineManager(core.EngineUSN)
	ln, err := core.OpenPipeListener(PIPE_NAME)
	if err != nil {
		fmt.Printf("ERROR: Failed to bind named pipe %s: %v\n", PIPE_NAME, err)
		return
	}
	defer ln.Close()

	fmt.Println("READY: Daemon listening for incoming requests from UI sessions...")

	var cachedIndex *core.VolumeIndex

	for {
		select {
		case <-stopChan:
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			var req DaemonRequest
			if err := json.NewDecoder(c).Decode(&req); err != nil {
				return
			}

			var resp DaemonResponse
			switch strings.ToLower(req.Action) {
			case "scan":
				idx, rootID, engineName, err := engineMgr.Scan(req.Target)
				if err != nil {
					resp = DaemonResponse{Status: "error", Message: err.Error()}
				} else {
					cachedIndex = idx
					children := idx.GetChildren(rootID)
					resp = DaemonResponse{
						Status: "ok",
						Engine: engineName,
						Data: map[string]interface{}{
							"totalRecords": len(idx.Records),
							"children":     children,
						},
					}
				}
			case "search":
				if cachedIndex == nil {
					resp = DaemonResponse{Status: "error", Message: "no index loaded"}
				} else {
					matches := cachedIndex.SearchAll(req.Query, 50)
					resp = DaemonResponse{
						Status: "ok",
						Engine: "in-memory",
						Data:   matches,
					}
				}
			default:
				resp = DaemonResponse{Status: "error", Message: "unknown action"}
			}

			_ = json.NewEncoder(c).Encode(resp)
		}(conn)
	}
}
