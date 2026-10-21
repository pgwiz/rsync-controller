# Project Memory & Architecture Notes

## Overview
Rsync Controller is an ultra-fast, single-instance file synchronization orchestrator for Windows (with Linux cross-compilation support). It integrates directly into the Windows Shell (File Explorer) and features a zero-dependency, pure native Win32 user interface.

## Core Rules & Constraints
1. **Zero Web / No Webdriver**: Absolutely no browser, headless browser, webdriver, or webview runtime. The UI is 100% native Win32 controls (`SysListView32`, `SysTabControl32`, `msctls_statusbar32`, `BUTTON`, `EDIT`, `COMBOBOX`).
2. **Zero Square Bracket Policy in UI**: Per user requirement, all UI buttons, tabs, status messages, operation banners, console logs, service daemons, offline queue statuses, and row types must strictly avoid square brackets (e.g., `READY`, `File`, `Folder`, `Up`, `ACTIVE: LOCAL COPY: ...`, `EXECUTE RSYNC`, `SYNC RUN`, `OK:`, `FAIL:`, `INFO:`, `WARN:`, `QUEUED`, `FAILED`).
3. **Markdown File Restriction**: Restrict project documentation strictly to `README.md`, `changelog.md`, `memory.md`, and `agent.md`.
4. **Explorer Submenu Protocol**: Windows Explorer cascaded flyout submenus use separate, isolated item and background pools to guarantee clean menus and prevent HKCR merging of stale HKLM verbs:
   - Item Menu Pool: `Directory\ContextMenus\RsyncItem\shell` (`01_copy`, `02_move_to`, `03_pull`, `04_move_from`, `05_srv_alpha_push`, `06_srv_alpha_pull`, `99_open_ui`). Trigger verbs point to `ExtendedSubCommandsKey = "Directory\ContextMenus\RsyncItem"` with `--src="%1 "` or `--dest="%1 "`.
   - Background Menu Pool: `Directory\ContextMenus\RsyncBg\shell`. Trigger verbs point to `ExtendedSubCommandsKey = "Directory\ContextMenus\RsyncBg"` with `--bg="%V "` (strictly zero `%1` references).
   - Trailing space padding (`"%V "`, `"%1 "`) prevents Windows Explorer from corrupting trailing backslashes with escaped quotes.
   - Crucial: `(Default)` and `SubCommands` value entries MUST NOT exist on the parent verb.
   - When options are toggled off in settings, keys/values (`Position`, `Icon`, `SeparatorBefore`, `CommandFlags`) must be explicitly deleted from registry.
   - Single-instance detection uses direct `CreateMutexW` syscall checking for `ERROR_ALREADY_EXISTS` (183) combined with `FindWindowW("RsyncControllerWin32Class")`, forwarding JSON `LaunchPayload` over `\\.\pipe\rsync-ui-ipc`.
   - Normal launch guard: UI startup only schedules modal context action triggers when `--mode=...`, item clicks, or background clicks are explicitly requested, keeping standard application launches pristine and unprompted.
   - Thread COM apartment is initialized once on startup via `OleInitialize`.
5. **Corner Radius & Theming**:
   - `CornerRadius` setting (0, 4, 8, 14 px) controls both GDI button rounding via `RoundRect` and window borders via Windows 11 `DWMWA_WINDOW_CORNER_PREFERENCE` (1=sharp, 3=small, 2=round).
   - Full 18-theme suite natively implemented in `theme_engine.go` matching mockups in `Themes examples/` with instant switching and live preview.
   - Active theme and radius persist to `config.json`.
   - TreeView (`WC_TREEVIEW`) is active by default side-by-side with ListView for immediate folder navigation.
6. **Navigation & Productivity Power Tools**:
   - Quick Jump Menu (`ID_BTN_JUMP = 110`): Instant jumping to Home, Desktop, Downloads, Documents, and mounted system drives.
   - Live Drive Space Gauge in Status Bar (Part 0): Continuous NTFS disk capacity monitoring via `GetDiskFreeSpaceExW`.
   - Global Keyboard Accelerators: Native message pump interception for `F5` (refresh), `Backspace` (parent folder), `Alt+Left/Right` (history), `Delete` (unstage), `Space` (toggle), `Ctrl+A` (stage all), `Ctrl+F` (search), `Ctrl+D` (dry run), `Ctrl+E` (execute).
   - In-App Right-Click Context Menus: Native `TrackPopupMenu` menus on both Local Explorer and Staged Drawer list views with Explorer reveal, path copying, and instant `Copy to...`, `Move to...`, `Pull from...`, `Move from...` execution.
7. **Core Transfer Operations & Move Mechanics**:
   - Four standard local transfer verbs supported across Explorer context menus, named pipe IPC, and Win32 UI:
     - `Copy to...`: Copies staged items to destination.
     - `Move to...`: Moves staged items to destination and removes source files upon confirmed transfer via `--remove-source-files`.
     - `Pull from...`: Pulls files from selected source into destination folder.
     - `Move from...`: Pulls files from selected source into destination folder and removes source files upon confirmed transfer via `--remove-source-files`.
   - Rsync atomic move support via `RsyncJob.IsMove` appending `--remove-source-files`.
   - Single-instance anti-recursion: Window-level `FindWindowW("RsyncControllerWin32Class")` detection combined with mutex checks ensures secondary instances unconditionally forward payloads, attach thread input (`AttachThreadInput`), bring the existing window to the foreground, and exit immediately (`os.Exit(0)`), completely eliminating duplicate window spawning during multi-file Explorer selections.
   - Tab 0 Mode Selector button (`MODE: ... ▾` - `ID_BTN_MODE = 139`) with live Win32 popup menu for instant mode switching.
8. **Smart Background Context Menus & Drive Enumeration**:
   - **Background right-click behavior (no file selected)**:
     - `Move to...`: Current folder is the **source directory** (`stagedPath`), with destination prompted.
     - `Copy to...`: Current folder is the **destination directory** (`stagedDest`), with button `CHOOSE SOURCE & COPY`.
     - `Move from...` & `Pull from...`: Current folder is the **destination directory** (`stagedDest`), with button `CHOOSE SOURCE & MOVE` / `CHOOSE SOURCE & PULL`.
     - `Open Rsync Controller...`: Opens directly into current folder without staging items.
   - **Active ListView Item Selection & Focus**:
     - When opened with a file selected in Explorer, the item is actively selected (`LVIS_SELECTED`), focused (`LVIS_FOCUSED`), scrolled into view (`LVM_ENSUREVISIBLE`), and keyboard focus is explicitly set to the ListView (`w32.SetFocus`).
   - **Shell Registry Padding**:
     - Windows shell arguments are formatted with trailing space padding (`--src="%1 "` and `--bg="%V "`) to eliminate Windows `\"` backslash-quote escaping issues on directory paths and drive roots.
     - Registered `Drive\Background\shell\Rsync` so drive roots (`C:\`, `E:\`, etc.) natively support background right-clicks.
   - **Comprehensive Drive Discovery**:
     - TreeView (`SysTreeView32`) dynamically populates all system drives (`C:\`, `D:\`, `E:\`, etc.) using Win32 `GetLogicalDrives`, `GetDriveTypeW`, and `GetVolumeInformationW`.
     - `RESCAN` button (`ID_BTN_RESCAN`) refreshes both drive roots and folder structures.

## Component Architecture
- `cmd/rsync_ui`: Native Win32 desktop application.
  - TreeView (`treeview.go`): Native `SysTreeView32` dual-pane directory tree with drive discovery (`A:`-`Z:`), on-demand expansion (`TVN_ITEMEXPANDINGW`), and two-way synchronization (`syncTreeSelection`, `TVN_SELCHANGEDW`).
  - Controls Guard (`win32_controls.go`): Reentrant `isPopulatingListView` guards on `setListViewItemChecked`, `clearListView`, and `addListViewRow` to eliminate infinite event recursion. Pure Win32 clipboard API, audio chimes, GDI `RoundRect`, and native popup menu dispatch (`TrackPopupMenuEx`).
  - Theme Engine (`theme_engine.go`): Full suite of 18 native GDI themes (Windows 95 Classic, Dark Obsidian, Y2K Cyber Candy, Cyber Vaporwave, Monochrome 1-Bit, Y2K Aqua, Warm Handdrawn Retro, Pastel Retro Grid, Nostalgic Media, Pink & Yellow Y2K Pop, Retrowave Vintage PC, Classic Win95/98 Teal, Y2K Blue Folders, Vintage Pixel PC, Y2K Music Chat, Desktop Music Player & Clock, System Dialogs Green/Blue, System Dialogs Sunset/Pink). Features full owner-drawn buttons (`BS_OWNERDRAW` with 3D bevels, tactile click displacement, Y2K pill-shaped buttons with `RoundRect` drop-shadows, and monochrome 1-bit inverted click states), owner-drawn tabs (`TCS_OWNERDRAWFIXED` with `subclassTabControl` background erasing), list & tree custom draw (`NM_CUSTOMDRAW`), header styling (`SysHeader32`), section dividers (`EDGE_ETCHED` / neon rules), UXTheme mode switching (`DarkMode_Explorer`), status bar styling (`SB_SETBKCOLOR`), interactive top-bar theme dropdown popup menu (`showThemePopupMenu`), Windows 11 window corner styles (`DWMWA_WINDOW_CORNER_PREFERENCE = 33`), and titlebar caption tinting (`DWMWA_CAPTION_COLOR = 35`, `DWMWA_TEXT_COLOR = 36`).
  - Staging Engine: Persistent multi-folder item registry (`stagedRegistry`, `stagedListPaths`), drag-and-drop ingestion (`WM_DROPFILES`), and collapsible Mini Right Drawer (`hwndListViewStaged`).
  - Operation Banner & Context Dispatcher: Mode-sensitive status banner with dynamic actions (`executeContextAction`), 350ms multi-selection debouncing (`WM_APP_TRIGGER_ACTION`), and automated destination/source directory pickers on shell verb invocations.
  - Live Transfer Monitor: Real-time status bar speedometer/ETA parsing and retro audio chimes.
  - Thread-Safe IPC: `WM_APP_PAYLOAD` queue message dispatching to activate and foreground running instances on shell verb invocations.
  - Tab 0: Local Folder Explorer (Dual-pane TreeView, Folder vs Recursive view, Checkboxes, Action buttons).
  - Tab 1: Universal Search (Drive-wide queries, Staging, Open directory).
  - Tab 2: Remote Server Sync (SSH server target selection, push/pull).
  - Tab 3: Server Health (Ping and SSH handshake verification).
  - Tab 4: Transfer Log (Streaming Rsync stdout/stderr).
  - Tab 5: Options & Settings (Shell integration toggles, Engine selector, Theme selector, Service daemon controls).
- `cmd/rs_copy`: CLI orchestrator, elevation helper, and health watchdog.
- `cmd/rsync_service`: Elevated Windows Service daemon for USN Journal reading without user elevation.
- `core/`:
  - `pipe_helper_windows.go`: Native Win32 named pipe server listener and client dialer (`DialPipe` via `windows.CreateFile`).
  - `usn_windows.go`: `FSCTL_ENUM_USN_DATA` NTFS indexer.
  - `fallback_walker.go`: Non-elevated directory crawler.
  - `single_instance_windows.go`: Named mutex and IPC pipe for multi-item selection aggregation.
  - `shell_integration_windows.go`: Automated registry configuration for Windows Explorer.
