//go:build windows

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	advapi32      = windows.NewLazySystemDLL("advapi32.dll")
	regDeleteTree = advapi32.NewProc("RegDeleteTreeW")
)

// deleteSubKeyTree recursively deletes a registry subkey and all of its descendants.
func deleteSubKeyTree(root registry.Key, subKey string) error {
	pSub, err := windows.UTF16PtrFromString(subKey)
	if err != nil {
		return err
	}
	r, _, _ := regDeleteTree.Call(uintptr(root), uintptr(unsafe.Pointer(pSub)))
	if r != 0 && r != uintptr(windows.ERROR_FILE_NOT_FOUND) {
		return windows.Errno(r)
	}
	return nil
}

// resolveIconPath returns the absolute path to icon.ico if present,
// or falls back to custom icon or exe icon.
func resolveIconPath(exePath string, cfg *AppConfig) string {
	if !cfg.Shell.ShowIcons {
		return ""
	}
	if cfg.Shell.CustomIconPath != "" {
		if _, err := os.Stat(cfg.Shell.CustomIconPath); err == nil {
			return cfg.Shell.CustomIconPath
		}
	}
	baseDir := filepath.Dir(exePath)
	candidate := filepath.Join(baseDir, "assets", "icon.ico")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return fmt.Sprintf(`"%s",0`, exePath)
}

// RegisterShellMenus registers context menus across all Explorer contexts using
// the Microsoft ExtendedSubCommandsKey pattern (verified from PowerShell 7).
func RegisterShellMenus(exePath string, cfg *AppConfig) error {
	absExe, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}

	iconTarget := resolveIconPath(absExe, cfg)

	itemTargets := []string{
		`Directory\shell\Rsync`,
		`Folder\shell\Rsync`,
		`Drive\shell\Rsync`,
		`*\shell\Rsync`,
	}
	bgTargets := []string{
		`Directory\Background\shell\Rsync`,
		`Drive\Background\shell\Rsync`,
		`DesktopBackground\Shell\Rsync`,
	}

	// Determine registry roots: always HKCU, plus HKLM if elevated
	roots := []registry.Key{registry.CURRENT_USER}
	if IsSelfElevated() {
		roots = append(roots, registry.LOCAL_MACHINE)
	}

	for _, root := range roots {
		// 1. Purge old keys
		for _, t := range itemTargets {
			_ = deleteSubKeyTree(root, `Software\Classes\`+t)
		}
		for _, t := range bgTargets {
			_ = deleteSubKeyTree(root, `Software\Classes\`+t)
		}
		_ = deleteSubKeyTree(root, `Software\Classes\Directory\ContextMenus\Rsync`)
		_ = deleteSubKeyTree(root, `Software\Classes\Directory\ContextMenus\RsyncItem`)
		_ = deleteSubKeyTree(root, `Software\Classes\Directory\ContextMenus\RsyncBg`)

		// 2. Create the item ContextMenus pool (RsyncItem)
		poolItemPath := `Software\Classes\Directory\ContextMenus\RsyncItem\shell`
		poolItemKey, _, err := registry.CreateKey(root, poolItemPath, registry.ALL_ACCESS)
		if err == nil {
			registerSubmenuPoolItems(poolItemKey, absExe, iconTarget, cfg, false)
			poolItemKey.Close()
		}

		// 3. Create the background ContextMenus pool (RsyncBg, no %1 parameters)
		poolBgPath := `Software\Classes\Directory\ContextMenus\RsyncBg\shell`
		poolBgKey, _, err := registry.CreateKey(root, poolBgPath, registry.ALL_ACCESS)
		if err == nil {
			registerSubmenuPoolItems(poolBgKey, absExe, iconTarget, cfg, true)
			poolBgKey.Close()
		}

		menuTitle := cfg.Shell.MenuTitle
		if menuTitle == "" {
			menuTitle = "rsyn_copy"
		}

		// 4. Register item parent trigger verbs pointing to RsyncItem pool
		for _, t := range itemTargets {
			parentKey, _, err := registry.CreateKey(root, `Software\Classes\`+t, registry.ALL_ACCESS)
			if err != nil {
				continue
			}

			_ = parentKey.SetStringValue("MUIVerb", menuTitle)
			_ = parentKey.SetStringValue("ExtendedSubCommandsKey", `Directory\ContextMenus\RsyncItem`)
			if iconTarget != "" {
				_ = parentKey.SetStringValue("Icon", iconTarget)
			} else {
				_ = parentKey.DeleteValue("Icon")
			}

			switch strings.ToLower(cfg.Shell.Position) {
			case "top":
				_ = parentKey.SetStringValue("Position", "Top")
			case "bottom":
				_ = parentKey.SetStringValue("Position", "Bottom")
			default:
				_ = parentKey.DeleteValue("Position")
			}

			_ = parentKey.DeleteValue("")
			_ = parentKey.DeleteValue("SubCommands")
			parentKey.Close()
		}

		// 5. Register background parent trigger verbs pointing to RsyncBg pool
		for _, t := range bgTargets {
			parentKey, _, err := registry.CreateKey(root, `Software\Classes\`+t, registry.ALL_ACCESS)
			if err != nil {
				continue
			}

			_ = parentKey.SetStringValue("MUIVerb", menuTitle)
			_ = parentKey.SetStringValue("ExtendedSubCommandsKey", `Directory\ContextMenus\RsyncBg`)
			if iconTarget != "" {
				_ = parentKey.SetStringValue("Icon", iconTarget)
			} else {
				_ = parentKey.DeleteValue("Icon")
			}

			switch strings.ToLower(cfg.Shell.Position) {
			case "top":
				_ = parentKey.SetStringValue("Position", "Top")
			case "bottom":
				_ = parentKey.SetStringValue("Position", "Bottom")
			default:
				_ = parentKey.DeleteValue("Position")
			}

			_ = parentKey.DeleteValue("")
			_ = parentKey.DeleteValue("SubCommands")
			parentKey.Close()
		}
	}

	notifyShell()
	return nil
}

// registerSubmenuPoolItems writes the action commands into the ContextMenus pool.
// When isBg is true, only %V is passed (no %1) so Windows Explorer executes background verbs properly.
func registerSubmenuPoolItems(poolKey registry.Key, exePath, iconTarget string, cfg *AppConfig, isBg bool) {
	accel := cfg.Shell.UseAccelerators
	seps := cfg.Shell.UseSeparators

	var cmdCopy, cmdPull, cmdMoveTo, cmdMoveFrom, cmdOpen string
	if isBg {
		cmdCopy = fmt.Sprintf(`"%s" --mode=copy --bg="%%V "`, exePath)
		cmdPull = fmt.Sprintf(`"%s" --mode=pull --bg="%%V "`, exePath)
		cmdMoveTo = fmt.Sprintf(`"%s" --mode=move_to --bg="%%V "`, exePath)
		cmdMoveFrom = fmt.Sprintf(`"%s" --mode=move_from --bg="%%V "`, exePath)
		cmdOpen = fmt.Sprintf(`"%s" --mode=open --bg="%%V "`, exePath)
	} else {
		cmdCopy = fmt.Sprintf(`"%s" --mode=copy --src="%%1 "`, exePath)
		cmdPull = fmt.Sprintf(`"%s" --mode=pull --dest="%%1 "`, exePath)
		cmdMoveTo = fmt.Sprintf(`"%s" --mode=move_to --src="%%1 "`, exePath)
		cmdMoveFrom = fmt.Sprintf(`"%s" --mode=move_from --dest="%%1 "`, exePath)
		cmdOpen = fmt.Sprintf(`"%s" --mode=open --src="%%1 "`, exePath)
	}

	// 01: Copy to...
	copyLabel := "Copy to..."
	if accel {
		copyLabel = "&Copy to..."
	}
	addPoolItem(poolKey, "01_copy", copyLabel, cmdCopy, iconTarget, false)

	// 02: Move to...
	moveToLabel := "Move to..."
	if accel {
		moveToLabel = "&Move to..."
	}
	addPoolItem(poolKey, "02_move_to", moveToLabel, cmdMoveTo, iconTarget, false)

	// 03: Pull from...
	pullLabel := "Pull from..."
	if accel {
		pullLabel = "&Pull from..."
	}
	addPoolItem(poolKey, "03_pull", pullLabel, cmdPull, iconTarget, false)

	// 04: Move from...
	moveFromLabel := "Move from..."
	if accel {
		moveFromLabel = "Move &from..."
	}
	addPoolItem(poolKey, "04_move_from", moveFromLabel, cmdMoveFrom, iconTarget, false)

	// Servers
	idx := 5
	for i, srv := range cfg.Servers {
		hasSep := (i == 0 && seps)
		cName := fmt.Sprintf("%02d_srv_%s_push", idx, srv.ID)
		cLabel := fmt.Sprintf("Push to %s...", srv.Name)
		if accel {
			cLabel = fmt.Sprintf("P&ush to %s...", srv.Name)
		}
		var srvCopyCmd, srvPullCmd string
		if isBg {
			srvCopyCmd = fmt.Sprintf(`"%s" --mode=copy --server="%s" --bg="%%V "`, exePath, srv.ID)
			srvPullCmd = fmt.Sprintf(`"%s" --mode=pull --server="%s" --bg="%%V "`, exePath, srv.ID)
		} else {
			srvCopyCmd = fmt.Sprintf(`"%s" --mode=copy --server="%s" --src="%%1 "`, exePath, srv.ID)
			srvPullCmd = fmt.Sprintf(`"%s" --mode=pull --server="%s" --dest="%%1 "`, exePath, srv.ID)
		}
		addPoolItem(poolKey, cName, cLabel, srvCopyCmd, iconTarget, hasSep)
		idx++

		pName := fmt.Sprintf("%02d_srv_%s_pull", idx, srv.ID)
		pLabel := fmt.Sprintf("Pull from %s...", srv.Name)
		if accel {
			pLabel = fmt.Sprintf("Pul&l from %s...", srv.Name)
		}
		addPoolItem(poolKey, pName, pLabel, srvPullCmd, iconTarget, false)
		idx++
	}

	// Open Controller
	openLabel := "Open rsyn_copy..."
	if accel {
		openLabel = "&Open rsyn_copy..."
	}
	addPoolItem(poolKey, "99_open_ui", openLabel, cmdOpen, iconTarget, seps)
}

func addPoolItem(poolKey registry.Key, keyName, label, commandStr, iconTarget string, sepBefore bool) {
	itemKey, _, err := registry.CreateKey(poolKey, keyName, registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer itemKey.Close()

	_ = itemKey.SetStringValue("MUIVerb", label)
	if iconTarget != "" {
		_ = itemKey.SetStringValue("Icon", iconTarget)
	} else {
		_ = itemKey.DeleteValue("Icon")
	}
	if sepBefore {
		_ = itemKey.SetStringValue("SeparatorBefore", "")
		_ = itemKey.SetDWordValue("CommandFlags", 0x00000020)
	} else {
		_ = itemKey.DeleteValue("SeparatorBefore")
		_ = itemKey.DeleteValue("CommandFlags")
	}

	cmdKey, _, err := registry.CreateKey(itemKey, "command", registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer cmdKey.Close()

	_ = cmdKey.SetStringValue("", commandStr)
}

// UnregisterShellMenus removes all Rsync context menu keys from HKCU and HKLM.
func UnregisterShellMenus() error {
	targets := []string{
		`Directory\shell\Rsync`,
		`Directory\Background\shell\Rsync`,
		`Folder\shell\Rsync`,
		`Drive\shell\Rsync`,
		`Drive\Background\shell\Rsync`,
		`DesktopBackground\Shell\Rsync`,
		`*\shell\Rsync`,
		`Directory\ContextMenus\Rsync`,
		`Directory\ContextMenus\RsyncItem`,
		`Directory\ContextMenus\RsyncBg`,
	}

	for _, t := range targets {
		_ = deleteSubKeyTree(registry.CURRENT_USER, `Software\Classes\`+t)
		if IsSelfElevated() {
			_ = deleteSubKeyTree(registry.LOCAL_MACHINE, `Software\Classes\`+t)
		}
	}

	notifyShell()
	return nil
}

// notifyShell broadcasts changes to Windows Shell
func notifyShell() {
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	shChangeNotify := shell32.NewProc("SHChangeNotify")
	shChangeNotify.Call(0x08000000, 0x1000, 0, 0)
}

// IsSelfElevated checks whether the current process has administrative privileges.
func IsSelfElevated() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}
