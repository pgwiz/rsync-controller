# Agent Operational Guide (`agent.md`)

This document provides immediate, actionable technical context and operational procedures for autonomous coding agents and human developers maintaining the `rsync-controller` (`rsyn_copy`) codebase.

---

## 1. Quick Navigation & Context Hierarchy

1. **Root Router**: [`dev_com_agent.md`](dev_com_agent.md) — Universal entry point establishing ground rules and pointing to `dev_md_guides/`.
2. **Master Living Guides**: [`dev_md_guides/agent.md`](dev_md_guides/agent.md) — Master summary table, file inventories, and target budgets.
3. **Architectural Decisions (ADRs)**: [`dev_md_guides/memory.md`](dev_md_guides/memory.md) & [`memory.md`](memory.md).
4. **Known Gotchas & Fixes**: [`dev_md_guides/gotchas.md`](dev_md_guides/gotchas.md).
5. **Operational Runbooks**: [`dev_md_guides/playbook.md`](dev_md_guides/playbook.md).
6. **Validated Empirical Truths**: [`dev_md_guides/solution.md`](dev_md_guides/solution.md).

---

## 2. Inviolable Core Constraints

1. **Pure Win32 GDI Architecture**:
   - Zero Webview, Electron, or browser runtimes. All UI components are native Win32 controls (`SysListView32`, `SysTabControl32`, `SysTreeView32`, `msctls_statusbar32`, `BUTTON`, `EDIT`, `COMBOBOX`).
2. **Zero Square Bracket Policy in UI**:
   - Never use square brackets (`[` or `]`) in any visible UI strings, button labels, tab titles, status indicators, operation banners, or logging messages. Use standard tags (`OK:`, `FAIL:`, `INFO:`, `WARN:`, `READY`, `File`, `Folder`, `Up`).
3. **Non-Blocking Destination Validation**:
   - When Destination is missing or matches Source, never spawn modal popup dialogs (`MessageBox`). Highlight/focus the Destination bar (`w32.SetFocus(hwndTxtDest)`), select all text (`EM_SETSEL`), flash a status bar alert, log to Transfer Log, and abort.
4. **Zero Fallback Destination Locking**:
   - Staging a new copy or move resets `stagedDest = ""`. Never default or silently lock Destination to the application directory (`E:\Backup\pgwiz\rsync`) or hardcoded test paths (`C:\Backup\Target`).
5. **Explorer Shell Argument Padding**:
   - All registered shell commands must include trailing space padding inside quotes (`"--src=\"%1 \""` and `"--bg=\"%V \""`) to eliminate Windows `\"` backslash-escaping bugs.
6. **Documentation Limits**:
   - Only maintain the core markdown files at root (`README.md`, `changelog.md`, `memory.md`, `agent.md`, `dev_com_agent.md`) and the living documentation suite in `dev_md_guides/`.

---

## 3. Build & Test Procedures

### Run Test Suite
```powershell
go test -v ./core
```
*Note: In environments without native `rsync` installed in `%PATH%`, `core_test.go` automatically skips live transfer assertions and verifies argument shaping safely.*

### Compile Native Win32 GUI Executable (Production)
```powershell
go build -ldflags "-H windowsgui" -o rsync-ui.exe ./cmd/rsync_ui
```

### Compile CLI Orchestrator & Elevation Helper
```powershell
go build -o rs_copy.exe ./cmd/rs_copy
```

### Compile Elevated USN Service Daemon
```powershell
go build -o rsync-service.exe ./cmd/rsync_service
```

---

## 4. Shell Integration Management

### Register Explorer Menus
```powershell
.\rsync-ui.exe --register-shell
```
- Installs cascaded `rsyn_copy` menu into HKCU (and HKLM if elevated).
- Creates separated pools: `Directory\ContextMenus\RsyncItem` (items) and `Directory\ContextMenus\RsyncBg` (empty background).
- Trailing space argument padding prevents escaping corruption.

### Unregister Explorer Menus
```powershell
.\rsync-ui.exe --unregister-shell
```

---

## 5. UI Layout & Control ID Reference

| Control ID | Constant Name | Type | Description |
|---|---|---|---|
| `101` | `ID_BTN_BACK` | `BUTTON` | Navigate backward in folder history |
| `102` | `ID_BTN_FWD` | `BUTTON` | Navigate forward in folder history |
| `103` | `ID_BTN_UP` | `BUTTON` | Navigate up to parent directory |
| `104` | `ID_TXT_TARGET` | `EDIT` | Row 1: Source folder address bar |
| `105` | `ID_BTN_GO` | `BUTTON` | Navigate to path in Source bar |
| `106` | `ID_BTN_BROWSE` | `BUTTON` | Open folder browser for Source |
| `110` | `ID_BTN_JUMP` | `BUTTON` | Quick Jump dropdown menu |
| `124` | `ID_TXT_DEST` | `EDIT` | Row 2: Target Destination address bar |
| `125` | `ID_BTN_BROWSE_DEST`| `BUTTON` | Open folder browser for Destination |
| `126` | `ID_BTN_SWAP` | `BUTTON` | Swap Source and Destination paths (⇄) |
| `127` | `ID_LBL_OP_BANNER` | `STATIC` | Operation mode and transfer target banner |
| `122` | `ID_TAB_CONTROL` | `SysTabControl32` | Main navigation tabs (0–5) |
| `130` | `ID_TXT_FILTER` | `EDIT` | Quick search filter on Tab 0 |
| `132` | `ID_LISTVIEW` | `SysListView32` | Main file & folder list view |
| `137` | `ID_TREEVIEW` | `SysTreeView32` | Dual-pane logical drives directory tree |
| `133` | `ID_BTN_EXECUTE` | `BUTTON` | Execute Rsync transfer |
| `134` | `ID_BTN_DRYRUN` | `BUTTON` | Dry run test transfer |
| `139` | `ID_BTN_MODE` | `BUTTON` | Transfer mode selector (`MODE: ... ▾`) |

---

## 6. Single-Instance IPC Architecture

- **Window Class**: `RsyncControllerWin32Class`
- **Named Pipe**: `\\.\pipe\rsync-ui-ipc`
- **IPC Protocol**: Structured JSON serialization of `core.LaunchPayload`:
  ```go
  type LaunchPayload struct {
      Mode      string   `json:"mode"`
      Server    string   `json:"server,omitempty"`
      Src       string   `json:"src"`
      Dest      string   `json:"dest,omitempty"`
      Paths     []string `json:"paths"`
      AutoStage bool     `json:"auto_stage"`
  }
  ```
- **Handshake Recovery**: `main()` checks `FindWindowW`. If a window exists, it dials `\\.\pipe\rsync-ui-ipc` via `core.ForwardPayloadDirect()`. If delivery succeeds, it foregrounds the master window and exits. If delivery fails (due to a dead or closing window), it falls through and launches cleanly as master.

---

## 7. Global Keyboard Shortcuts

- `F5`: Rescan active folder
- `Backspace`: Move up to parent folder (when edit control is not focused)
- `Alt + Left / Right`: History back / forward
- `Ctrl + A`: Check / stage all files in current view
- `Ctrl + F`: Jump focus to Filter bar
- `Ctrl + D`: Dry run test
- `Ctrl + E`: Execute transfer
- `Space`: Toggle checkbox staging of focused item
- `Delete`: Remove selected item from staging drawer
