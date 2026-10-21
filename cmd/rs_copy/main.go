// rs_copy.exe - Master launcher, setup installer, and watchdog for Rsync Controller.
// Runs with Administrator privileges (via embedded manifest) to guarantee service installation
// and system-wide Explorer shell integration.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rsync-controller/core"
)

const (
	tagOK    = "OK:"
	tagFail  = "FAIL:"
	tagInfo  = "INFO:"
	tagWarn  = "WARN:"
	tagFix   = "FIX:"
	tagReady = "READY:"
)

func main() {
	args := os.Args[1:]

	selfExe, err := os.Executable()
	if err != nil {
		fatalf("%s Cannot determine own executable path: %v", tagFail, err)
	}
	baseDir := filepath.Dir(selfExe)

	uiExe := filepath.Join(baseDir, "rsync-ui.exe")
	svcExe := filepath.Join(baseDir, "rsync-service.exe")
	iconIco := filepath.Join(baseDir, "assets", "icon.ico")

	repairMode := false
	statusMode := false
	setupOnly := false
	var passthroughArgs []string

	for _, a := range args {
		switch {
		case a == "--repair":
			repairMode = true
		case a == "--status":
			statusMode = true
		case a == "--setup-only":
			setupOnly = true
		default:
			passthroughArgs = append(passthroughArgs, a)
		}
	}

	fmt.Printf("%s Rsync Controller - Master Setup & Launcher\n", tagInfo)
	fmt.Printf("%s Base directory: %s\n", tagInfo, baseDir)

	// Check elevation
	isElevated := core.IsSelfElevated()
	if isElevated {
		fmt.Printf("%s Running with elevated Administrator privileges.\n", tagOK)
	} else if !statusMode {
		fmt.Printf("%s Running as standard user. Attempting self-elevation for setup...\n", tagWarn)
		if selfElevate(selfExe, args) {
			os.Exit(0)
		}
	}

	// 1. Verify binaries
	if _, err := os.Stat(uiExe); err != nil {
		fatalf("%s rsync-ui.exe not found at %s", tagFail, uiExe)
	}
	fmt.Printf("%s rsync-ui.exe      found\n", tagOK)

	if _, err := os.Stat(svcExe); err != nil {
		fmt.Printf("%s rsync-service.exe not found at %s\n", tagWarn, svcExe)
	} else {
		fmt.Printf("%s rsync-service.exe found\n", tagOK)
	}

	if _, err := os.Stat(iconIco); err == nil {
		fmt.Printf("%s Icon asset found: %s\n", tagOK, iconIco)
	}

	// 2. Load or create config
	cfgPath := filepath.Join(baseDir, "config.json")
	cfg, err := core.LoadConfig(cfgPath)
	if err != nil {
		cfg = core.DefaultConfig()
	}

	// 3. Register Shell Menus
	menuOK := checkMenuRegistered()
	if !menuOK || repairMode || isElevated {
		fmt.Printf("%s Synchronizing Windows Explorer context menus...\n", tagFix)
		if err := core.RegisterShellMenus(uiExe, cfg); err != nil {
			fmt.Printf("%s Context menu registration error: %v\n", tagWarn, err)
		} else {
			fmt.Printf("%s Context menus synchronized successfully across all contexts.\n", tagOK)
		}
	} else {
		fmt.Printf("%s Explorer context menus are active.\n", tagOK)
	}

	// 4. Windows Service Setup & Health Check
	svcRunning := checkServiceRunning()
	if (!svcRunning || repairMode) && !statusMode {
		fmt.Printf("%s Configuring Windows Service (RsyncControllerDaemon)...\n", tagFix)
		setupService(svcExe, repairMode)
	} else if svcRunning {
		fmt.Printf("%s Windows Service (RsyncControllerDaemon) is running.\n", tagOK)
	}

	// Status-only mode
	if statusMode {
		fmt.Printf("\n--- STATUS SUMMARY ---\n")
		fmt.Printf("  Menus registered : %v\n", checkMenuRegistered())
		fmt.Printf("  Service running  : %v\n", checkServiceRunning())
		fmt.Printf("  Elevated         : %v\n", core.IsSelfElevated())
		fmt.Printf("  UI binary        : %s\n", uiExe)
		fmt.Printf("  Service binary   : %s\n", svcExe)
		os.Exit(0)
	}

	if setupOnly {
		fmt.Printf("%s Initial setup completed successfully.\n", tagOK)
		os.Exit(0)
	}

	// 5. Launch UI Window
	fmt.Printf("%s Launching Rsync Controller UI...\n", tagReady)
	time.Sleep(100 * time.Millisecond)

	uiCmd := exec.Command(uiExe, passthroughArgs...)
	uiCmd.Stdin = os.Stdin
	uiCmd.Stdout = os.Stdout
	uiCmd.Stderr = os.Stderr

	if err := uiCmd.Run(); err != nil {
		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		os.Exit(exitCode)
	}
}

// selfElevate attempts to restart this process via UAC runas verb
func selfElevate(exePath string, args []string) bool {
	var escapedArgs []string
	for _, a := range args {
		escapedArgs = append(escapedArgs, fmt.Sprintf("'%s'", strings.ReplaceAll(a, "'", "''")))
	}
	argList := strings.Join(escapedArgs, ", ")

	psCmd := fmt.Sprintf(`Start-Process -FilePath '%s' -ArgumentList @(%s) -Verb RunAs`,
		strings.ReplaceAll(exePath, "'", "''"), argList)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	err := cmd.Start()
	return err == nil
}

func checkMenuRegistered() bool {
	out, err := exec.Command("reg", "query", `HKCU\Software\Classes\Directory\ContextMenus\RsyncItem\shell`).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), `ContextMenus\RsyncItem\shell`)
}

func checkServiceRunning() bool {
	return core.IsDaemonServiceRunning()
}

func setupService(svcExe string, forceReinstall bool) {
	if _, err := os.Stat(svcExe); err != nil {
		fmt.Printf("%s Service executable not found at %s, skipping service setup.\n", tagWarn, svcExe)
		return
	}

	if forceReinstall {
		_ = core.UninstallService()
		time.Sleep(300 * time.Millisecond)
	}

	if err := core.InstallAndStartService(svcExe); err != nil {
		fmt.Printf("%s Service setup error: %v\n", tagWarn, err)
		return
	}

	if core.IsDaemonServiceRunning() {
		fmt.Printf("%s Windows Service (%s) is running.\n", tagOK, core.DaemonServiceName)
	} else {
		fmt.Printf("%s Windows Service (%s) installed successfully.\n", tagOK, core.DaemonServiceName)
	}
}

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
