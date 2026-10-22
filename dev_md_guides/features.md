# Feature Roadmap & SDD Progress Matrix
_Last updated: 2026-10-22_

## Done
- [x] **Pure Win32 Desktop UI Architecture**: 100% native Win32 controls (`SysListView32`, `SysTabControl32`, `SysTreeView32`, `msctls_statusbar32`, `BUTTON`, `EDIT`, `COMBOBOX`) with zero Webview/Electron/browser runtime dependency (`cmd/rsync_ui/main.go`, `cmd/rsync_ui/win32_controls.go`).
- [x] **Rsync Dual-Engine Support**: Dynamic auto-detection of GNU rsync vs Rust `rsync-win` parser quirks, with path translation (`filepath.ToSlash`), argument shaping (`-a -v --progress --src ... --dest ...`), and atomic source cleanup for moves (`core/rsync.go`).
- [x] **Dedicated Dual Source & Destination Address Bars**: Tab 0 Row 1 Source bar (`hwndTxtTarget`) and Row 2 Destination bar (`hwndTxtDest`), with `BROWSE...` (`ID_BTN_BROWSE_DEST`) and `SWAP ⇄` (`ID_BTN_SWAP`) for bidirectional transfer switching (`cmd/rsync_ui/main.go`).
- [x] **Non-Blocking Destination Validation Guard**: Zero modal popups on transfer validation. Direct focus switching to `hwndTxtDest` with full selection (`EM_SETSEL`), high-visibility status bar alert, and log recording on empty or identical destination paths (`cmd/rsync_ui/main.go#runRsyncTransfer`).
- [x] **Windows Explorer Shell Cascaded Submenus**: Branded `rsyn_copy` flyout integration in File Explorer with separated item and background pools (`Directory\ContextMenus\RsyncItem` and `Directory\ContextMenus\RsyncBg`) and trailing space argument padding (`core/shell_integration_windows.go`).
- [x] **In-App ListView Background Context Menu**: Native `TrackPopupMenu` on empty list space offering `Copy to...`, `Move to...`, `Pull from...`, `Move from...`, `Check All`, `Uncheck All`, `Refresh (F5)`, `Reveal in Explorer`, `Copy Path`, and `Swap ⇄` (`cmd/rsync_ui/main.go#showListViewBackgroundContextMenu`).
- [x] **Dynamic System Drive Enumeration**: Dynamic logical drive detection (`A:`-`Z:`) displaying drive letters and volume labels (`C: (Local Disk)`, `D: (ChromeOS)`, `E: (Data)`) in both the TreeView and Universal Search combo box (`cmd/rsync_ui/treeview.go`, `cmd/rsync_ui/main.go`).
- [x] **18-Theme Retro GDI Engine & Palette Customizer**: Complete suite of themes with tactile owner-drawn buttons, pill bevels, drop shadows (0px, 1px, 2px, 4px, 6px), corner radius configuration, and Win32 `ChooseColorW` palette customizer with custom shadow color selection (`cmd/rsync_ui/theme_engine.go`).
- [x] **Universal Drive Search via NTFS USN Journal**: Ultra-fast MFT enumeration (`FSCTL_ENUM_USN_DATA`) with non-elevated crawler fallback (`core/usn_windows.go`, `core/fallback_walker.go`).
- [x] **Remote Server Sync & Health Probing**: SSH push/pull profiles with live ping latency and SSH handshake verification (`cmd/rsync_ui/main.go`, `core/config.go`).
- [x] **Single-Instance Anti-Recursion IPC**: Win32 window class detection (`FindWindowW`) combined with named pipe forwarding (`\\.\pipe\rsync-ui-ipc`) to eliminate duplicate window recursion on multi-file Explorer selections (`cmd/rsync_ui/main.go`, `core/single_instance_windows.go`).
- [x] **Zero Square Bracket Policy Enforcement**: Global elimination of square brackets across all visible UI strings, operation banners, status bars, and logging output.

## In Progress
- [/] **Automated Background Sync Tasks**: Scheduled cron/interval syncing for configured server profiles and local mirroring targets.
- [/] **Remote Profile Import / Export**: JSON/YAML serialization and migration for server profiles and credentials.

## Planned
- [ ] **Real-Time Bandwidth Throttling**: Live `--bwlimit` speed controller slider on Tab 0 bottom bar.
- [ ] **Rsync Daemon Direct Sockets**: Support for `rsync://` network protocol daemon targets without requiring SSH wrapper.

## Removed
- [-] **Modal Destination Popups**: Replaced with non-blocking edit box focus, text selection, and status alerts.
- [-] **Generic Shell Branding (`Rsync`)**: Replaced by official brand name `rsyn_copy`.
