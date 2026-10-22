# Session Operational Changelog
_Append-only. Newest first. Never edit past entries._

## [2026-10-22 12:57:52 UTC] — Branch `main` (HEAD: `937f2ae`)
- **Event**: Automated Context Compaction
- **Operational Scope**: Synchronized repository ground-truth into `dev_md_guides/`

### AST & Code Modifications
- *No AST code modifications detected in active diff.*

---
## [2026-10-22 12:55:00 UTC] — Branch `main` (HEAD: `937f2ae`)
- **Event**: Release v3.1.0 Deployment & Context Compaction
- **Operational Scope**: Dual Source & Destination bars on Tab 0, non-blocking transfer validation, in-app ListView background menu, dynamic drive detection, and custom theme shadow colors.

### AST & Code Modifications
- `cmd/rsync_ui/main.go`:
  - Added `hwndLblDest`, `hwndTxtDest`, `hwndBtnBrowseDest`, `hwndBtnSwap` (Row 2, Y: 40).
  - Subclassed `hwndTxtDest` in `editSubclassProc` with Enter key handling.
  - Added `showListViewBackgroundContextMenu` for empty list right-clicks.
  - Implemented non-blocking validation in `runRsyncTransfer` focusing `hwndTxtDest` on empty/identical paths.
  - Implemented dynamic drive population in `hwndCmbUnivDrive` using `getAllSystemDrives()`.
  - Added `Button Shadow Color...` customizer menu item with `chooseColorDialog`.
  - Fixed single-instance check to verify named-pipe delivery before exiting.
- `cmd/rsync_ui/treeview.go`:
  - Formatted drive labels by letter (`C: (Local Disk)`, `D: (ChromeOS)`, `E: (Data)`).
- `cmd/rsync_ui/theme_engine.go`:
  - Added `ShadowColor` parsing in `initTheme` under `case "custom"`.
  - Added initial `applyThemeToTreeView` call in `onCreate`.
- `core/config.go`:
  - Added `ShadowColor string` to `CustomThemeConfig`.

### Build & Test Outcomes
- `go test -v ./core`: All 7 tests passed (0 failures).
- Executables compiled: `rsync-ui.exe` (Win32 GUI) and `rs_copy.exe` (CLI).
- GitHub Actions CI workflow `39759103567`: Passed (`✓ build in 1m3s`).
- GitHub Release `v3.1.0` tagged and published with distribution zip.

---

## [2026-10-21 15:20:00 UTC] — Branch `main` (HEAD: `a6623c5`)
- **Event**: Release v3.0.0 Deployment & Public GitHub Repository Genesis
- **Operational Scope**: `rsyn_copy` Explorer branding, Rsync-Win dual-engine support, theme palette customizer, and background context menu separation.

---
