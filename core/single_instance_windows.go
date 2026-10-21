//go:build windows

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32DLL      = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW = kernel32DLL.NewProc("CreateMutexW")
)

const (
	UI_MUTEX_NAME = "Local\\RsyncControllerUIMutex"
	UI_PIPE_NAME  = `\\.\pipe\rsync-ui-ipc`
)

type LaunchPayload struct {
	Mode      string   `json:"mode,omitempty"`   // "copy", "pull", "move_to", "move_from", "open"
	Server    string   `json:"server,omitempty"` // e.g. "alpha"
	Src       string   `json:"src,omitempty"`
	Dest      string   `json:"dest,omitempty"`
	Paths     []string `json:"paths,omitempty"`
	AutoStage bool     `json:"autoStage,omitempty"`
}

type SingleInstanceCoordinator struct {
	mutexHandle     windows.Handle
	IsMaster        bool
	pipeLn          net.Listener
	OnPathAdded     func(path string)
	OnLaunchPayload func(payload LaunchPayload)
}

// CheckSingleInstance determines if this process is the master instance or a secondary forwarder.
func CheckSingleInstance() (*SingleInstanceCoordinator, error) {
	namePtr, err := windows.UTF16PtrFromString(UI_MUTEX_NAME)
	if err != nil {
		return nil, err
	}

	// Create or open the named mutex directly to capture ERROR_ALREADY_EXISTS reliably
	r0, _, callErr := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	h := windows.Handle(r0)
	if h == 0 || h == windows.InvalidHandle {
		return nil, fmt.Errorf("failed to create named mutex: %w", callErr)
	}

	if callErr == windows.ERROR_ALREADY_EXISTS {
		// Secondary instance
		return &SingleInstanceCoordinator{
			mutexHandle: h,
			IsMaster:    false,
		}, nil
	}

	// Master instance
	return &SingleInstanceCoordinator{
		mutexHandle: h,
		IsMaster:    true,
	}, nil
}

// ForwardPayloadDirect connects to the running pipe server and sends the payload.
func ForwardPayloadDirect(payload LaunchPayload) error {
	conn, err := DialPipe(UI_PIPE_NAME, 3*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to master UI pipe: %w", err)
	}
	defer conn.Close()

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(conn, "%s\n", string(data))
	return err
}

// ForwardPayload sends a structured launch payload to the running master instance.
func (c *SingleInstanceCoordinator) ForwardPayload(payload LaunchPayload) error {
	if c.mutexHandle != 0 && c.mutexHandle != windows.InvalidHandle {
		defer windows.CloseHandle(c.mutexHandle)
	}
	return ForwardPayloadDirect(payload)
}

// ForwardToMaster sends file paths to the running master instance via local named pipe.
func (c *SingleInstanceCoordinator) ForwardToMaster(paths []string) error {
	return c.ForwardPayload(LaunchPayload{
		Mode:      "copy",
		Paths:     paths,
		AutoStage: true,
	})
}

// StartPipeServer starts listening for payloads forwarded by secondary instances.
func (c *SingleInstanceCoordinator) StartPipeServer() error {
	ln, err := OpenPipeListener(UI_PIPE_NAME)
	if err != nil {
		return err
	}
	c.pipeLn = ln

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(cn net.Conn) {
				defer cn.Close()
				scanner := bufio.NewScanner(cn)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}

					if strings.HasPrefix(line, "{") {
						var payload LaunchPayload
						if err := json.Unmarshal([]byte(line), &payload); err == nil {
							if c.OnLaunchPayload != nil {
								c.OnLaunchPayload(payload)
								continue
							}
							if payload.Src != "" && c.OnPathAdded != nil {
								c.OnPathAdded(payload.Src)
								continue
							}
						}
					}

					// Plain path fallback
					if c.OnLaunchPayload != nil {
						c.OnLaunchPayload(LaunchPayload{
							Mode:      "copy",
							Src:       line,
							Paths:     []string{line},
							AutoStage: true,
						})
					} else if c.OnPathAdded != nil {
						c.OnPathAdded(line)
					}
				}
			}(conn)
		}
	}()
	return nil
}

// Close releases the mutex and pipe listener.
func (c *SingleInstanceCoordinator) Close() {
	if c.pipeLn != nil {
		_ = c.pipeLn.Close()
	}
	if c.mutexHandle != 0 && c.mutexHandle != windows.InvalidHandle {
		_ = windows.ReleaseMutex(c.mutexHandle)
		_ = windows.CloseHandle(c.mutexHandle)
	}
}
