package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateKdeServiceMenu generates a Dolphin .desktop action file for KDE Plasma.
func GenerateKdeServiceMenu(exePath string, servers []*ServerConfig) (string, error) {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Service\n")
	b.WriteString("MimeType=all/allfiles;inode/directory;\n")

	actions := []string{"rs_local_copy", "rs_local_pull"}
	for _, srv := range servers {
		actions = append(actions, fmt.Sprintf("rs_%s_copy", srv.ID), fmt.Sprintf("rs_%s_pull", srv.ID))
	}
	b.WriteString(fmt.Sprintf("Actions=%s;\n", strings.Join(actions, ";")))
	b.WriteString("X-KDE-Submenu=Rsync\n\n")

	// Local Actions
	b.WriteString("[Desktop Action rs_local_copy]\n")
	b.WriteString("Name=rs local -> Copy to...\n")
	b.WriteString(fmt.Sprintf("Exec=%s --mode=copy --src=%%F\n\n", exePath))

	b.WriteString("[Desktop Action rs_local_pull]\n")
	b.WriteString("Name=rs local -> Pull from...\n")
	b.WriteString(fmt.Sprintf("Exec=%s --mode=pull --dest=%%f\n\n", exePath))

	// Server Actions
	for _, srv := range servers {
		b.WriteString(fmt.Sprintf("[Desktop Action rs_%s_copy]\n", srv.ID))
		b.WriteString(fmt.Sprintf("Name=rs server %s -> Copy to...\n", srv.Name))
		b.WriteString(fmt.Sprintf("Exec=%s --mode=copy --server=%s --src=%%F\n\n", exePath, srv.ID))

		b.WriteString(fmt.Sprintf("[Desktop Action rs_%s_pull]\n", srv.ID))
		b.WriteString(fmt.Sprintf("Name=rs server %s -> Pull from...\n", srv.Name))
		b.WriteString(fmt.Sprintf("Exec=%s --mode=pull --server=%s --dest=%%f\n\n", exePath, srv.ID))
	}

	return b.String(), nil
}

// InstallLinuxShellIntegration installs service menus for KDE and scripts for Nautilus.
func InstallLinuxShellIntegration(exePath string, servers []*ServerConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 1. KDE Dolphin ServiceMenu
	kdeDir := filepath.Join(home, ".local", "share", "kio", "servicemenus")
	_ = os.MkdirAll(kdeDir, 0755)
	kdeContent, _ := GenerateKdeServiceMenu(exePath, servers)
	_ = os.WriteFile(filepath.Join(kdeDir, "rsync-controller.desktop"), []byte(kdeContent), 0644)

	// 2. Nautilus Scripts
	nautilusDir := filepath.Join(home, ".local", "share", "nautilus", "scripts")
	_ = os.MkdirAll(nautilusDir, 0755)
	scriptContent := fmt.Sprintf("#!/bin/bash\n%s --mode=copy --src=\"$NAUTILUS_SCRIPT_SELECTED_FILE_PATHS\"\n", exePath)
	_ = os.WriteFile(filepath.Join(nautilusDir, "Rsync Copy To"), []byte(scriptContent), 0755)

	return nil
}
