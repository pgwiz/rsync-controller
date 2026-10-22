# Architectural Decisions & Invariants (`dev_md_guides/memory.md`)
_Append-only with explicit supersession. Never summarize away without explicit user consent._

## System Invariants

1. **Zero Web / No Webview Runtime**:
   The desktop application is built purely on native Win32 controls (`SysListView32`, `SysTabControl32`, `SysTreeView32`, `msctls_statusbar32`, `BUTTON`, `EDIT`, `COMBOBOX`). Absolutely zero browser engines, Electron, webviews, or headless webdrivers are permitted.
2. **Zero Square Bracket Policy in UI Text**:
   All UI button labels, tabs, status bar texts, operation banners, transfer logs, service notifications, and row types must strictly avoid square brackets `[` and `]` (e.g. use `READY`, `File`, `Folder`, `Up`, `ACTIVE: LOCAL COPY: ...`, `EXECUTE RSYNC`, `SYNC RUN`, `OK:`, `FAIL:`, `INFO:`, `WARN:`).
3. **Non-Blocking Destination Validation**:
   When Destination is empty or identical to Source, the UI must NEVER spawn blocking modal popup dialogs (`MessageBox`). The UI shifts focus to `hwndTxtDest`, selects all text (`EM_SETSEL`), flashes a warning on the status bar, and logs the event to the Transfer Log.
4. **Zero Hardcoded Working Paths as Fallbacks**:
   `stagedDest` must never silently default to the current working directory (`E:\Backup\pgwiz\rsync`) or hardcoded test targets (`C:\Backup\Target`). Staging a new copy or move resets `stagedDest = ""` unless explicitly provided.
5. **Explorer Shell Argument Backslash Padding**:
   All shell commands registered in the Windows Registry must format directory and file tokens with trailing space padding (`--src="%1 "` and `--bg="%V "`) to eliminate Windows `\"` escaping corruption on paths ending in backslashes.
6. **Isolated Explorer Submenu Pools**:
   Cascading Explorer context menus must strictly bifurcate item targets (`Directory\ContextMenus\RsyncItem`) and background targets (`Directory\ContextMenus\RsyncBg`). Background targets must pass only `%V` (strictly zero `%1` references).

---

## Architectural Decision Records (ADRs)

### ADR-001: Pure Native Win32 UI Architecture
- **Date**: 2026-10-20
- **Status**: Accepted
- **Context**: Needed an ultra-responsive, zero-dependency desktop UI that launches in <50ms, consumes <15MB RAM, and runs seamlessly on Windows 10/11 without requiring Edge/WebView2 runtimes.
- **Decision**: Implemented a pure Win32 GDI user interface using `github.com/gonutz/w32/v2` and direct `syscall` bindings.
- **Consequences**: Exceptional performance, zero installer runtime footprint, full compatibility across fresh Windows installs. Requires manual owner-drawing for custom themes and button bevels.

### ADR-002: Dual-Engine Rsync Integration (GNU rsync & Rust rsync-win)
- **Date**: 2026-10-21
- **Status**: Accepted
- **Context**: Windows environments may supply either classical Cygwin/MSYS2 GNU rsync (`rsync -av --progress <src> <dest>`) or the modern Rust-based `rsync-win` port (`rsync -a -v --progress --src <src> --dest <dest>`). The Rust parser rejects GNU-specific flags (`-z`, `--info=progress2`).
- **Decision**: Implemented dynamic runtime detection (`core.IsRsyncWin`) via cached `rsync --version` inspection. The execution engine dynamically shapes CLI parameters, converts backslashes to forward slashes (`filepath.ToSlash`), and executes source file removal (`os.RemoveAll`) post-transfer for atomic moves on `rsync-win`.
- **Consequences**: Universal compatibility with whatever `rsync` executable is present in `%PATH%`.

### ADR-003: Explorer Cascaded Submenus via ExtendedSubCommandsKey Pools
- **Date**: 2026-10-21
- **Status**: Accepted
- **Context**: Registering standard cascading context menus using inline `SubCommands` or static `shell` subkeys caused erratic behavior, ghost menus, and verb drops when right-clicking directory backgrounds.
- **Decision**: Adopted the Windows 7/10/11 `ExtendedSubCommandsKey` pool pattern used by modern shell extensions (e.g. PowerShell 7). Created separate registry pools: `Directory\ContextMenus\RsyncItem` for files and folders, and `Directory\ContextMenus\RsyncBg` for empty background space.
- **Consequences**: Completely stable cascading submenus with custom icons, accelerators, and separators across both items and empty folder backgrounds.

### ADR-004: Anti-Recursion Single Instance Protection (FindWindowW + Named Pipe)
- **Date**: 2026-10-21
- **Status**: Accepted
- **Context**: Selecting 50 files in Windows Explorer and clicking context menu verb spawned 50 separate process instances, causing high CPU spikes and window recursion.
- **Decision**: Implemented desktop-level `FindWindowW("RsyncControllerWin32Class")` detection combined with named pipe IPC (`\\.\pipe\rsync-ui-ipc`). Secondary processes forward structured JSON `LaunchPayload` to the master instance and exit (`os.Exit(0)`). Added pipe delivery confirmation so dead window handles safely fall through.
- **Consequences**: Single-instance guarantee: subsequent selections immediately aggregate into the active window's staging drawer without spawning new windows.

### ADR-005: Dedicated Dual Source & Destination Address Bars with SWAP
- **Date**: 2026-10-22
- **Status**: Accepted
- **Context**: Tab 0 previously displayed only the Source address bar, with Destination hidden in state or prompted via modal dialogs, leading to accidental transfers into default working directories.
- **Decision**: Placed a dedicated Destination address bar (`hwndTxtDest`), `BROWSE...` button, and `SWAP ⇄` button on Row 2 (Y: 40) directly below the Source address bar. Clicking `SWAP ⇄` reverses paths, updates text boxes, and reloads the file list in the new source directory.
- **Consequences**: Both transfer endpoints are always explicitly visible. Direction can be flipped instantly with zero modal interruptions.
