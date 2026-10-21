//go:build windows

package core

import (
	"fmt"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	DaemonServiceName = "RsyncControllerDaemon"
	DaemonServiceDesc = "Elevated background disk indexer and rsync daemon for Rsync Controller."
)

// InstallAndStartService installs or updates the daemon service and starts it.
func InstallAndStartService(svcExePath string) error {
	absExe, err := filepath.Abs(svcExePath)
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Service Control Manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(DaemonServiceName)
	if err != nil {
		// Service does not exist, create it
		s, err = m.CreateService(DaemonServiceName, absExe, mgr.Config{
			StartType:   mgr.StartAutomatic,
			DisplayName: "Rsync Controller Engine",
			Description: DaemonServiceDesc,
		})
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	} else {
		// Update binary path
		cfg, err := s.Config()
		if err == nil {
			cfg.BinaryPathName = absExe
			cfg.Description = DaemonServiceDesc
			_ = s.UpdateConfig(cfg)
		}
	}
	defer s.Close()

	// Check status and start if not running
	status, err := s.Query()
	if err == nil && status.State != svc.Running && status.State != svc.StartPending {
		if err := s.Start(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	}

	return nil
}

// UninstallService stops and deletes the daemon service.
func UninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(DaemonServiceName)
	if err != nil {
		return nil // already not installed
	}
	defer s.Close()

	// Attempt stop
	_, _ = s.Control(svc.Stop)
	time.Sleep(300 * time.Millisecond)

	return s.Delete()
}

// IsDaemonServiceRunning checks if RsyncControllerDaemon is currently in Running state.
func IsDaemonServiceRunning() bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	s, err := m.OpenService(DaemonServiceName)
	if err != nil {
		return false
	}
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return false
	}
	return st.State == svc.Running
}
