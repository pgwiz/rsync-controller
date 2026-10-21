# Changelog

All notable changes to the Rsync Controller utility are documented in this file.

## (3.0.0) - Rsync-Win Dual Engine, Background Menu Separation, Theme Customizer & Shadow Controls

### Added
- **Rsync-Win Dual Engine Support**:
  - Automatically detects whether the installed `rsync` binary is GNU rsync or Rust-based `rsync-win` (via cached `rsync --version` inspection).
  - Handles `rsync-win` specific CLI flags: `-a -v --progress --src <src> --dest <dest>` with forward slashes (`filepath.ToSlash`).
  - Supports multiple staged files by sequential invocation with error aggregation.
  - Implemented local move cleanup via `os.RemoveAll(src)` after successful copy confirmation.
  - Real-time line-by-line stdout and stderr streaming to the Transfer Log via `WM_APP_RSYNC_LOG`.
- **Interactive Theme Palette & Shadow Customizer**:
  - Native Windows color dialog (`ChooseColorW` from `comdlg32.dll`) to customize:
    - Canvas Background Color
    - Window / Surface Color
    - Button Background Color
    - Button Text Color
    - Accent / Highlight Color
    - Window Border & Frame Color
  - Shadow Depth Controller:
    - `0px (Flat - No Shadow)`
    - `1px (Subtle Depth)`
    - `2px (Standard Tactile)`
    - `4px (Bold 3D Retro)`
    - `6px (Deep Comic Shadow)`
  - Tactile owner-drawn button shadow rendering: background shadow offset by `theme.ShadowDepth` in `BorderColor`, with active button body shifting down and right on press.
  - Top header `THEME: ... ▾` menu includes a direct "Customize Palette & Shadows..." action accessible from any tab.
- **Dedicated Background Explorer Context Menu Pool**:
  - Separated item targets (`Directory`, `Folder`, `Drive`, `*`) and background targets (`Directory\Background`, `Drive\Background`, `DesktopBackground`).
  - Background targets use `Directory\ContextMenus\RsyncBg` passing only `--bg="%V "` (no `%1`), preventing Windows Explorer from dropping background commands.

### Fixed
- **Resolved Rsync Exit Status 2 Failure**:
  - Eliminated unexpected arguments (`-z`, `--info=progress2`) rejected by `rsync-win` CLI parser.
- **TreeView Drive Root Path Normalization**:
  - Normalized drive roots (`C:\`, `D:\`, `E:\`) to prevent `filepath.Clean` from stripping trailing slashes to relative drive identifiers (`C:`).
  - Added `SetErrorMode(SEM_FAILCRITICALERRORS | SEM_NOOPENFILEERRORBOX)` to suppress modal popups when querying unready or optical drives.
- **ListView Active Item Focus**:
  - Set `LVIF_STATE` (0x0008) in `LVITEMW.Mask` so `selectAndFocusListViewItem` actively selects (`LVIS_SELECTED`) and focuses (`LVIS_FOCUSED`) target items.
- **Zero Square Bracket Policy Compliance**:
  - Stripped all square brackets from `rs_copy.exe`, `rsync-ui.exe`, `rsync-service.exe`, `usn_bench`, and `core/offline_queue.go` outputs, replacing them with standard tags (`OK:`, `FAIL:`, `INFO:`, `WARN:`, `READY:`, `QUEUED`, `FAILED`).
- **Post-Transfer Auto-Refresh & Staging Reset**:
  - Implemented `WM_APP_TRANSFER_COMPLETE` dispatch upon successful transfer. Automatically clears the staged registry, resets the mini drawer, and refreshes the active folder listing, guaranteeing that moved files immediately disappear from the view.
- **Normal Startup Action Isolation**:
  - Guarded action trigger scheduling to only fire when `--mode=...`, an item click, or a background click is explicitly passed. Normal application launches (double-clicking the executable or launching via shortcut) now open cleanly without popping up destination picker dialogs.

### Added
- **Local Move To and Move From Operations**:
  - Local file operations now support complete symmetry between `Copy to...`, `Move to...`, `Pull from...`, and `Move from...`.
  - When `Move to...` is executed with items staged, Rsync transfers them to the destination and removes source files on completion (`--remove-source-files`).
  - When `Move from...` is executed, the active folder is locked as the destination, and files from the selected source folder are moved in and removed from source upon transfer.
- **Smart Background Click Context Handling**:
  - When right-clicking inside an empty folder or background (no file selected):
    - `Move to...`: Automatically designates the current folder as the **source directory** (`stagedPath`), with destination prompted.
    - `Copy to...`: Automatically designates the current folder as the **destination directory** (`stagedDest`), with button `CHOOSE SOURCE & COPY`.
    - `Move from...`: Automatically designates the current folder as the **destination directory** (`stagedDest`), with button `CHOOSE SOURCE & MOVE`.
    - `Pull from...`: Automatically designates the current folder as the **destination directory** (`stagedDest`), with button `CHOOSE SOURCE & PULL`.
    - `Open Rsync Controller...`: Opens directly into the active folder.
- **Active ListView Item Selection & Focus**:
  - When opened with an item selected in Windows Explorer, the target file/folder is actively selected (`LVIS_SELECTED`), focused (`LVIS_FOCUSED`), scrolled into view (`LVM_ENSUREVISIBLE`), and keyboard focus is set to the ListView (`w32.SetFocus`).
- **Comprehensive System Drives TreeView**:
  - TreeView (`SysTreeView32`) discovers and populates all system drives (`C:\`, `D:\`, `E:\`, etc.) using Win32 `GetLogicalDrives`, `GetDriveTypeW`, and `GetVolumeInformationW`.
  - Drive roots display authentic volume labels (`Local Disk (C:)`, `Backup (E:)`, etc.) with child subfolders expanding on click.
  - Added full TreeView drive and folder refresh when pressing `RESCAN` (`ID_BTN_RESCAN`).

### Fixed
- **Explorer Shell Background Command Execution**:
  - Registered `Drive\Background\shell\Rsync` in Windows Registry so drive roots support background right-click alongside `Directory\Background`.
  - Added trailing space padding to shell command arguments (`--src="%1 "` and `--bg="%V "`) to eliminate Windows `\"` backslash-quote escaping issues on paths ending in backslashes.
- **Anti-Recursion & Window Management**:
  - Added desktop-level `FindWindowW("RsyncControllerWin32Class")` detection prior to mutex creation, guaranteeing secondary processes immediately forward payloads and exit (`os.Exit(0)`).
  - Used `AttachThreadInput` in `bringWindowToForeground` to bypass Windows foreground locks.
  - Eliminated premature modal popup triggers (`scheduleActionTrigger`) on startup so background right-clicks open smoothly without blocking modal dialogs.

## [2.8.0] - Core Transfer Actions (Copy/Pull/Move To/From), Anti-Recursion Single Instance, & Mode Selectors

### Added
- **Core Local Transfer Actions Suite**:
  - Full support for 4 core transfer operations across Explorer context menus, named pipe IPC, and the Win32 UI:
    1. `Copy to...`: Stages selected files/folders and copies them via Rsync to a chosen target folder.
    2. `Move to...`: Stages selected files/folders, executes Rsync to target folder, and atomically removes source files upon confirmed transfer (`--remove-source-files`).
    3. `Pull from...`: Locks the active folder as destination, prompts for source directory, and syncs contents into the active folder.
    4. `Move from...`: Locks the active folder as destination, syncs contents from chosen source directory, and atomically removes source files upon confirmed transfer (`--remove-source-files`).
- **Windows Explorer Context Menu Verbs**:
  - Registered 4 distinct verbs in `Directory\ContextMenus\Rsync`:
    - `01_local_copy`: `rs local -> Copy to...` (`--mode=copy --src="%1" --bg="%V"`)
    - `02_local_pull`: `rs local -> Pull from...` (`--mode=pull --dest="%1" --bg="%V"`)
    - `03_local_move_to`: `rs local -> Move to...` (`--mode=move_to --src="%1" --bg="%V"`)
    - `04_local_move_from`: `rs local -> Move from...` (`--mode=move_from --dest="%1" --bg="%V"`)
    - Followed by remote server push/pull verbs and `Open Rsync Controller...`.
- **Tab 0 Mode Selector Button (`MODE: ... ▾` - `ID_BTN_MODE = 139`)**:
  - Added an interactive tactile button in the Tab 0 bottom bar alongside `EXECUTE RSYNC`.
  - Clicking displays a native Win32 popup menu to switch between `Copy to...`, `Move to...`, `Pull from...`, and `Move from...` with checkmark on active mode.
- **In-App Explorer Right-Click Context Menu Verbs**:
  - Right-clicking any item in Local Explorer listview provides instant actions:
    - `Stage Item for Sync` / `Unstage Item`
    - `Copy to... (Stage & Copy)`
    - `Move to... (Stage & Move)`
    - `Pull from... (Pull into this folder)`
    - `Move from... (Move into this folder)`
    - `Reveal in Windows Explorer`
    - `Copy Full Path`
- **Rsync Atomic Move Support**:
  - Added `IsMove bool` to `core.RsyncJob`. When true, `RunRsync` passes `--remove-source-files`.
  - Added CLI generator support for `--remove-source-files`.

### Fixed
- **Multi-File Context Menu Window Recursion**:
  - Resolved issue where selecting multiple items in File Explorer opened duplicate windows.
  - Secondary instances now connect via Named Pipe with a 4-second resilient retry loop.
  - Secondary instances unconditionally forward their payload, bring the primary window to the foreground via Win32 `FindWindowW` and `SetForegroundWindow`, and exit immediately (`os.Exit(0)`), completely preventing duplicate window creation.
  - Primary instance window initialization drains any queued payloads that arrived while the window was opening (`WM_APP_PAYLOAD`).

## [2.7.0] - 18-Theme Suite, Top Quick Jump Menu, Drive Space Gauge, Keyboard Accelerators & In-App Context Menus

### Added
- **Full 18-Theme Authentic Aesthetic Suite**:
  - Brought all 15 authentic themes directly from `Themes examples/rsync_controller_multi_theme_edition.html` and visual mockups into native Win32 GDI:
    1. `win95-classic`: Windows 95 Classic (Retro Bevel)
    2. `dark-obsidian`: Dark Obsidian (Cyber Night)
    3. `y2k-candy`: Y2K Cyber Candy (Retro Gel)
    4. `vaporwave-cyber`: Cyber Vaporwave (Neon 80s)
    5. `monochrome-mac`: Monochrome 1-Bit (Retro Classic)
    6. `aqua-win32`: Y2K Aqua (Win32 Teal & Cerulean)
    7. `warm-handdrawn`: Warm Handdrawn Retro PC (Parchment & Amber)
    8. `pastel-grid`: Pastel Retro Grid (Pink & Lavender)
    9. `nostalgic-media`: Nostalgic Media Popup UI (Cobalt & Ivory)
    10. `y2k-pink`: Pink & Yellow Y2K Pop (Raspberry & Lemon)
    11. `retrowave-pc`: Retrowave Vintage PC (Hot Magenta & Neon)
    12. `win95-figma`: Classic Win95/98 Teal Desktop
    13. `y2k-blue`: Y2K Classic Desktop Blue Folders
    14. `pixel-pc`: Vintage Pixel PC Screens (Ruby Crimson)
    15. `music-chat`: Y2K Music Chat Desktop (Coral Peach)
    16. `player-clock`: Y2K Desktop Music Player & Clock (Emerald)
    17. `dialog-green`: Y2K System Dialogs (Sky Blue & Soft Mint)
    18. `dialog-sunset`: Y2K System Dialogs (Sunset Magenta & Rose)
  - Every theme features tailored canvas/surface palettes, custom fonts (`Tahoma`, `Segoe UI`, `Consolas`, `Courier New`), tactile button styles (beveled retro, glowing retrowave, 3D drop-shadow pill shapes), and Windows 11 DWM titlebar color matching.
  - Interactive top bar dropdown menu (`THEME: ... ▾`) and Tab 5 Settings combobox now list all 18 themes with instant live preview.
- **Top Navigation Quick Jump Menu (`JUMP ▾` - `ID_BTN_JUMP = 110`)**:
  - Added a dedicated 65px Jump button right next to `BROWSE...` on the top address bar.
  - Clicking displays a native Win32 popup menu with instant one-click jumps to:
    - User Home directory
    - Desktop
    - Downloads
    - Documents
    - All detected mounted local drives (`C:\`, `D:\`, `E:\`, etc.)
- **Live Drive Free Space Storage Gauge in Status Bar**:
  - Implemented 5-part responsive Win32 status bar:
    - Part 0 (240px): Live disk storage metrics (`DRIVE E: 142.3 GB free / 476.9 GB (70% free)`) queried via `GetDiskFreeSpaceExW`.
    - Part 1 (150px): Active Indexing Engine (`ENGINE: USN SCANNER` / `ENGINE: CRAWLER`).
    - Part 2 (170px): Background Daemon state (`SERVICE: RUNNING` / `SERVICE: STOPPED`).
    - Part 3 (180px): Persistent Staged Summary (`Staged: 3 items (66.1 KB)`).
    - Part 4 (remainder): Live operational feedback (`READY`, `SCANNING DISK...`, `TRANSFER RUNNING`, `SETTINGS SAVED`).
- **Global Keyboard Accelerators & Hotkeys**:
  - Intercepts key events directly in the message pump before dispatch:
    - `F5`: Instant directory scan & TreeView refresh (`onRescanClicked`).
    - `Backspace`: Navigate up to parent folder (when focus is not inside an edit control).
    - `Alt+Left` / `Alt+Right`: History back & forward.
    - `Delete`: Unstage highlighted items from the drawer or list view.
    - `Space`: Toggle staging on the highlighted list view row.
    - `Ctrl+A`: Stage all files in the current folder.
    - `Ctrl+F`: Focus the search / filter input.
    - `Ctrl+D`: Instant Dry Run test.
    - `Ctrl+E`: Trigger Rsync sync execution.
- **In-App Right-Click Context Menus**:
  - Added native Win32 popup context menus for both list views on right-click (`NM_RCLICK`):
    - **Local Explorer List**: `Stage / Unstage Item`, `Reveal in Windows Explorer`, `Copy Full Path`.
    - **Staged Drawer List**: `Unstage <Item>`, `Jump to Folder in Explorer`, `Reveal in Windows Explorer`, `Copy Full Path`, `Copy All Staged Paths`, `Clear All Staged Items`.
- **Audio Feedback on Sync Completion**:
  - Triggers standard success chime (`procMessageBeep` with `MB_ICONASTERISK`) upon completion of Rsync transfer jobs.

### Added
- **Corner Radius Controller (`ID_CMB_RADIUS = 414`)**:
  - Added dedicated Corner Radius selection control to Tab 5 (Settings) supporting:
    - `0px (Sharp Retro 80s/90s)`: Razor-sharp corners with `DWMWCP_DONOTROUND` (1).
    - `4px (Subtle Modern)`: Modern subtle rounded corners with `DWMWCP_ROUNDSMALL` (3).
    - `8px (Rounded Y2K)`: Authentic Y2K rounded aesthetic with `DWMWCP_ROUND` (2).
    - `14px (Full Pill Aqua)`: Full pill-shaped rounding with `DWMWCP_ROUND` (2).
  - Corner radius dynamically updates button drawing via GDI `RoundRect` and window borders via Windows 11 `DWMWA_WINDOW_CORNER_PREFERENCE`, persisting directly to `config.json`.
  - Added interactive Corner Radius slider and preset buttons (`0px`, `4px`, `8px`, `14px`) to `Themes examples/rsync_controller_multi_theme_edition.html` with real-time CSS variable bindings.
- **Enabled TreeView by Default on Tab 0**:
  - TreeView (`WC_TREEVIEW`) is now active and visible alongside the ListView by default (`showTreeView = true`), providing continuous drive and folder navigation without needing to toggle options.

### Fixed
- **Explorer Shell Context Menu Action Routing & Single-Instance IPC**:
  - Fixed named mutex single-instance check in `core/single_instance_windows.go`: replaced standard Go library wrapper with direct `CreateMutexW` syscall to reliably capture `ERROR_ALREADY_EXISTS` (183).
  - Ensured secondary instances forward structured `LaunchPayload` JSON over named pipe (`\\.\pipe\rsync-ui-ipc`) and fallback to master if pipe is uninitialized.
  - Initialized OLE at application startup via `OleInitialize`, eliminating COM apartment threading conflicts (`RPC_E_CHANGED_MODE`) in shell folder dialogs, clipboard, and drag-and-drop.
  - Added automatic synchronization between HKCU and system-wide HKLM context menus via `rs_copy.exe --repair` fallback when running unelevated.
  - Sanitized unexpanded shell tokens (`%1`, `%V`, `%v`, `%w`) in `cleanArgParam`, preventing literal variable names from poisoning file paths.
- **Universal Square Bracket Removal**:
  - Completely stripped all square brackets (`[` and `]`) across:
    - Button labels: `BROWSE`, `SEARCH`, `THEME: WIN95 ▾`, `RESCAN`, `OPTIONS`, `HIDE DRAWER`, `STAGED (%d)`, `EXECUTE RSYNC`, `DRY RUN TEST`, `CHECK ALL`, `UNCHECK ALL`, `UNSTAGE`, `CLEAR`, `SYNC`, `TREE: ON`, `FOLDER VIEW`, `APPLY & SAVE SETTINGS`, `START SERVICE`, `COPY LOG`, `COPY RSYNC COMMAND`, `STAGE CHECKED`, `OPEN IN EXPLORER`, `PUSH TO SERVER`, `PULL FROM SERVER`, `PING SERVERS`.
    - Tab headers: `Local Explorer`, `Universal Search`, `Remote Sync`, `Server Health`, `Transfer Log`, `Settings`.
    - Operation banner: `ACTIVE: LOCAL COPY: ...`, `ACTIVE: LOCAL PULL: ...`, `ACTIVE: SERVER PUSH -> ...`, `MODE: LOCAL FILE MANAGER: ...`.
    - Status bar text: `ENGINE: USN SCANNER`, `SERVICE: RUNNING`, `READY`, `SETTINGS SAVED`, `TRANSFER RUNNING`, `TRANSFER COMPLETE`.
    - List view row indicators: `File`, `Folder`, `Up`, `Ready`, and `.. (Parent Directory)`.
    - HTML reference templates in `Themes examples/`.

## [2.5.0] - Automated Context Menu Routing, Window Corner Styles & Interactive Theme Picker

### Fixed
- **Explorer Context Menu Action Execution (`Copy to...` / `Pull from...`)**:
  - Resolved issue where `Copy to...` and `Pull from...` shell context menu commands only opened/focused the UI and sat in silence without executing work.
  - Implemented debounced action dispatcher (`scheduleActionTrigger` via custom Win32 message `WM_APP_TRIGGER_ACTION`):
    - **Local Copy to...**: Auto-stages right-clicked item(s), immediately prompts destination folder picker (`BrowseFolderDialog`), requests execution confirmation, and launches the Rsync transfer with real-time progress logging.
    - **Local Pull from...**: Locks the target destination to the right-clicked directory, immediately prompts source folder picker (`BrowseFolderDialog`), requests execution confirmation, and triggers the Rsync pull transfer.
    - **Remote Server Push / Pull**: Automatically navigates to Tab 2 (Remote Sync), selects the designated server in the profile combobox, and prompts for immediate transfer execution.
  - Added 350ms debounce window coalescing multi-file Explorer selections forwarded over `\\.\pipe\rsync-ui-ipc` before presenting the transfer action prompt.
- **Named Pipe Client/Server Handle Separation (`winPipeConn.isServer`)**:
  - Differentiated client vs. server handles in `core/pipe_helper_windows.go`: restricted `FlushFileBuffers` and `DisconnectNamedPipe` strictly to server connections, eliminating secondary-instance process blocking during IPC message dispatch.
- **Windows Command Line Quoting & Trailing Backslash Guard**:
  - Enhanced `cleanArgParam` to parse trailing-escaped quotes (e.g. `\"`) and preserve drive root backslashes (e.g. restoring `C:\` from `C:`), ensuring Explorer context menu arguments are never mangled.
- **"Open Rsync Controller..." Shell Command Routing**:
  - Updated context menu registry pool command for `05_open_ui` to pass `--mode=open --src="%1" --bg="%V"`, opening directly into the right-clicked directory without prompting for file transfers.

### Added
- **Interactive Theme Picker Dropdown Menu (`TrackPopupMenuEx`)**:
  - Replaced cyclical top navigation theme button with an interactive 120px dropdown button (`[ THEME: <NAME> ▾ ]`).
  - Clicking displays a sleek native Win32 popup menu listing all 5 themes with checkmark indication on the active selection.
  - Selecting any theme updates window styling immediately, refreshes the button caption, notifies the status bar (`[THEME: ... APPLIED]`), and persists the choice to `config.json`.
- **Native Windows 11 Window Corner Customization (`DWMWA_WINDOW_CORNER_PREFERENCE = 33`)**:
  - Dynamically configures window rounding based on theme aesthetic:
    - **Y2K Cyber Candy** & **Cyber Vaporwave**: `DWMWCP_ROUND` (2) for authentic Y2K rounded window corners.
    - **Dark Obsidian**: `DWMWCP_ROUNDSMALL` (3) for modern rounded corners.
    - **Windows 95 Classic** & **Monochrome 1-Bit**: `DWMWCP_DONOTROUND` (1) for razor-sharp retro rectangular borders.
- **Native Titlebar Caption Coloring (`DWMWA_CAPTION_COLOR = 35`, `DWMWA_TEXT_COLOR = 36`)**:
  - Customizes native titlebar color on Windows 11 (bubblegum pink for Y2K Candy, cyber purple for Vaporwave, deep obsidian for Dark Obsidian).
- **Y2K Candy Rounded Pill Buttons (`gdi32.dll` `RoundRect`)**:
  - Owner-drawn buttons in Y2K Candy theme rendered as rounded pills with solid 2px drop-shadows matching the design references in `Themes examples/`.

### Fixed
- **ListView & TreeView Selection Highlight Scope**:
  - Replaced unreliable `CDIS_SELECTED` custom draw state flag with authoritative `LVM_GETITEMSTATE` (`LVIS_SELECTED`) query and `TVM_GETNEXTITEM` (`TVGN_CARET`).
  - Resolved bug where unselected items had their backgrounds painted with the theme highlight color (`HighlightBg`), ensuring only the single individually selected item receives the accent highlight while all other items render on their clean `SurfaceBg`.
- **Full Owner-Drawn Tactile Buttons (`BS_OWNERDRAW` via `WM_DRAWITEM`)**:
  - **Windows 95 Classic**: Authentic 3D beveled edges (`EDGE_RAISED` / `EDGE_SUNKEN`), 1px tactile click displacement (`rc.Left++`, `rc.Top++`), inset dotted focus rectangle (`DrawFocusRect`), primary action solid black perimeter borders, and classic 3D embossed disabled text.
  - **Y2K Cyber Candy**: Neo-brutalist solid 2px drop-shadows with 2px click displacement into shadow, 2px midnight navy `#1A1A2E` borders, electric candy cyan `#00F0FF` selection highlight, candy raspberry `#D6006E` accents, and bubblegum pink `#FFEEF5` canvas.
  - **Cyber Vaporwave**: Dual-layer glowing neon borders (hot pink `#FF007F` and electric cyan `#00F0FF`), glowing text, and +2px click displacement.
  - **Monochrome 1-Bit**: Inverted solid jet black fill on click/selection with pure white text, high-contrast black borders, and classic 1-bit typography.
  - **Dark Obsidian**: Sleek charcoal `#2A2A3E` button faces, luminous blue `#0078D7` primary accents, and subtle inset depth.
- **Custom-Drawn SysTabControl32 (`TCS_OWNERDRAWFIXED`) & Subclassing**:
  - Owner-drawn tab headers with active indicators, elevated tabs (+2px height), and palette-matched typography.
  - Subclassed tab control window procedure (`subclassTabControl`) intercepting `WM_ERASEBKGND` with `currentTheme.CanvasBrush` to eliminate default grey background bleed.
- **UXTheme Integration (`uxtheme.dll` `SetWindowTheme`)**:
  - Applies `DarkMode_Explorer` for Dark Obsidian on Windows 10/11, enabling native dark scrollbars and dark list headers.
  - Reverts Aero skins on retro themes (`win95-classic`, `monochrome-mac`, `y2k-candy`, `vaporwave-cyber`) to expose authentic 3D beveled headers, grid lines, and custom colors.
- **Unified List & Tree Custom Draw Engine (`NM_CUSTOMDRAW`)**:
  - Intercepts `NM_CUSTOMDRAW` (`CDDS_PREPAINT` / `CDDS_ITEMPREPAINT`) across `hwndListView`, `hwndListViewUniv`, `hwndListViewStaged`, and `hwndTreeView`.
  - Enforces accurate theme text and highlight colors (`HighlightBg`, `HighlightText`) for selected rows across all Windows versions.
- **Header Control (`SysHeader32`) Styling**:
  - Directly themes ListView column headers via `LVM_GETHEADER`, applying `theme.FontBold` and UX theme mode.
- **Settings Theme Live-Preview**:
  - Added `CBN_SELCHANGE` handler on `hwndCmbTheme` (`ID_CMB_THEME`) in Tab 5 so selecting any theme immediately live-previews and changes the entire window.
- **Architectural Section Dividers**:
  - Renders 3D etched grooves (`EDGE_ETCHED`) for Windows 95, neon laser rules for Vaporwave, midnight navy rules for Y2K, and sleek charcoal rules for Obsidian separating navigation, banner, content, and action bar areas.

## [2.3.0] - Dual-Pane TreeView, Navigation Event Guards & Native Named Pipe IPC

### Added
- **Native Dual-Pane Directory TreeView (`SysTreeView32`)**:
  - Embedded Windows Explorer hierarchical tree navigation alongside the main file listview (`WC_TREEVIEW`, `TVS_HASBUTTONS`, `TVS_HASLINES`, `TVS_LINESATROOT`, `TVS_SHOWSELALWAYS`).
  - Auto-populates all active system drives (`A:` through `Z:`) at root.
  - Dynamic on-demand child folder expansion via `TVN_ITEMEXPANDINGW` (only queries subdirectories when nodes are expanded, eliminating initial startup delay).
  - Two-way directory synchronization: clicking tree folders instantly navigates the file listview (`TVN_SELCHANGEDW`), and navigating the listview or address bar automatically expands ancestor nodes and highlights the corresponding tree item (`syncTreeSelection`).
  - Integrated `[ TREE: ON ]` / `[ TREE: OFF ]` toggle button with responsive layout resizing.
- **Native Named Pipe IPC Client (`DialPipe`)**:
  - Implemented Win32 `CreateFile` named-pipe client dialer (`DialPipe`) in `core/pipe_helper_windows.go`, replacing Go standard library's unsupported `net.Dial("pipe", ...)` and resolving secondary-instance IPC forwarding.
  - Added unit test `TestPipeCommunication` verifying end-to-end named-pipe listener and client message transmission.
- **Refined Win32 GDI Theming & Control Accents**:
  - Status bar background styling via `SB_SETBKCOLOR` (`0x2001`) matching active theme canvas.
  - High-contrast operation banner typography using `currentTheme.AccentColor` in `WM_CTLCOLORSTATIC`.
  - Transfer log edit control styled with `SurfaceBg` and `TextColor` for consistent contrast in dark and retro themes.
  - Combobox dropdowns styled via `WM_CTLCOLORLISTBOX`.
  - TreeView background, text, and connector lines dynamically themed via `TVM_SETBKCOLOR`, `TVM_SETTEXTCOLOR`, and `TVM_SETLINECOLOR`.

### Fixed
- **Navigation Lock & Event Cascade on Item Selection**:
  - Resolved issue where checking or unchecking an item froze or blocked navigation buttons (`<`, `>`, `^`) and directory double-clicking.
  - Added reentrant `isPopulatingListView` guards in `clearListView`, `setListViewItemChecked`, `syncCurrentListViewChecks`, and `onListViewItemChanged` to eliminate recursive `LVN_ITEMCHANGED` cascade loops.
  - Updated `navBack()` to seamlessly fall back to `navUp()` when at the root of navigation history, keeping `<` and `^` enabled whenever a parent folder exists.
- **Context Menu Routing & Macro Sanitization**:
  - Context menu shell pool commands properly quote and differentiate background clicks (`--bg="%V"`) from item clicks (`--src="%1"` / `--dest="%1"`).
  - Fixed empty macro string handling so literal unexpanded `"%1"` and `"%V"` strings passed by Explorer do not trigger filesystem stat errors or lockups.
  - Startup payload auto-staging is strictly restricted to explicit CLI/shell commands (`AutoStage: explicitPathGiven && (stagedMode == "copy")`), preventing automatic staging of entire directory trees on normal application launch.

## [2.2.0] - Context Menu Workflows, Persistent Staging Drawer & Native Theme Engine

### Added
- **Structured IPC & Dedicated Explorer Action Workflows**:
  - Context menu actions (`Copy to...`, `Pull from...`, `Push to server...`, `Pull from server...`) now trigger dedicated interactive workflows instead of opening the UI generically.
  - Parameter sanitization (`cleanArgParam`) strips double quotes and handles Windows CRT trailing quote escapes (`\"`).
  - Added `--bg="%V"` support in `core/shell_integration_windows.go` and `scripts/fix_shell_menus.ps1` to seamlessly handle right-clicking empty space in Explorer directory backgrounds.
  - Thread-safe `WM_APP_PAYLOAD` Win32 message dispatching: incoming named-pipe payloads are processed directly on the UI thread, activating, restoring (`SW_RESTORE`), and foregrounding (`SetForegroundWindow`) the master window.
  - Mode duality with top Operation Banner:
    - **Local Copy**: Auto-stages right-clicked file/folder, opens Staged Drawer, and presents `[ CHOOSE DEST & COPY ]` with native folder picker.
    - **Local Pull**: Locks right-clicked directory as local target, presents `[ CHOOSE SOURCE & PULL ]`, and executes transfer.
    - **Server Push**: Auto-stages items, selects the target server in combobox, navigates to Tab 2 (Server Sync), and presents `[ PUSH TO <SERVER> ]`.
    - **Server Pull**: Locks target directory, selects target server, navigates to Tab 2, and presents `[ PULL FROM <SERVER> ]`.
- **Persistent Multi-Folder Staging & Mini Right Drawer**:
  - Selections are retained across folder changes, universal search queries, and navigation history via `stagedRegistry`.
  - Added collapsible Mini Right Summary Drawer (`hwndGrpDrawer`, `hwndListViewStaged`) displaying total staged items and size with `[ > HIDE ]` / `[ < STAGED (X) ]` toggle.
  - Deterministic sorted indexing (`stagedListPaths`) allows individual items to be un-staged via double-click or the `[ UNSTAGE ]` button without index mismatches.
- **Direct Drag-and-Drop Staging (`WM_DROPFILES`)**:
  - Drag files or directories from anywhere in Windows Explorer directly into the application window to immediately auto-stage them into the Mini Right Drawer with visual feedback and retro audio chimes.
- **Real-Time Transfer Speedometer & Audio Feedback**:
  - Status bar tracks live transfer metrics during rsync execution (`[45%] 12.5 MB/s | ETA: 0:00:14`).
  - Plays tactile retro system audio feedback chime (`MessageBeep`) upon transfer completion and drag-and-drop staging.
- **One-Click Rsync CLI Command Generator & Clipboard Copy**:
  - Added `[ COPY RSYNC CLI COMMAND ]` and `[ COPY LOG TO CLIPBOARD ]` buttons powered by pure Win32 clipboard APIs (`OpenClipboard`, `SetClipboardData`). Users can instantly copy the full CLI invocation to run headless or save to scripts.
- **Pure Win32 Native Theme Engine**:
  - Added 5 distinct themes matching the reference designs in `themes/` and `Themes examples/`:
    1. **Windows 95 Classic**: Vintage 3D beveled silver `#C0C0C0` with navy `#000080` accents and Tahoma typography.
    2. **Dark Obsidian**: Midnight cyber `#181825` with Windows 10/11 DWM immersive dark title bar and vibrant blue `#0078D7`.
    3. **Y2K Cyber Candy**: Pastel aqua/pink `#FFEEF5` with high-contrast text and clean Segoe UI styling.
    4. **Cyber Vaporwave**: Neon 80s retrowave `#180928` with hot pink `#FF007F`, cyan `#00F0FF`, and Consolas monospace.
    5. **Monochrome 1-Bit**: Vintage Macintosh / classic monochrome PC `#FFFFFF` with pure black `#000000` accents and Courier New typography.
  - GDI brush caching, `WM_ERASEBKGND` canvas rendering, `WM_CTLCOLORSTATIC` / `WM_CTLCOLOREDIT` / `WM_CTLCOLORBTN` styling, and `EnumChildWindows` dynamic font updates across all controls.
  - Live cycle button on top toolbar (`[ WIN 95 ]`, `[ OBSIDIAN ]`, `[ Y2K AQUA ]`, `[ VAPORWAVE ]`, `[ 1-BIT RETRO ]`) and combobox in Tab 5 (Settings).

## [2.1.2] - Native Win32 SCM Service Management

### Fixed
- **Direct Service Control Manager API**: Replaced brittle `cmd.exe /C sc.exe` subprocess execution with direct native Win32 SCM calls via `golang.org/x/sys/windows/svc/mgr` (`OpenSCManager`, `CreateService`, `StartService`). This eliminates cmd quote stripping errors and race conditions (`ERROR 1060 / 1072`) during service repair.
- **Service Verification**: `rs_copy.exe --repair` and `rsync-service.exe --install` now seamlessly install and start the background daemon directly with zero intermediate shell invocations.

## [2.1.1] - Top Universal Search Bar & Options System Fix

### Added
- **Top Header Universal Search Bar**: Added an always-visible Universal Search bar directly on the top header next to the Address bar and Browse buttons. Includes placeholder cue banner (`"Search entire drive..."`) and Enter key shortcut to immediately search the entire drive and activate results.
- **Edit Control Keyboard Subclassing**: Added subclassing across all edit controls (`hwndTxtTarget`, `hwndTxtTopUnivSearch`, `hwndTxtUnivSearch`, `hwndTxtFilter`) so pressing Enter immediately triggers directory navigation or search.
- **Settings Saved Feedback Dialog**: Visual confirmation dialog upon clicking `[ APPLY & SAVE ALL SETTINGS ]`.

### Fixed
- **Options & Tab Selection Click Handling**: Corrected the tab notification check to use `TCN_SELCHANGE` (`0xFFFFFDD9` / `-551`) instead of the double-click constant (`0xFFFFFFFD`), restoring mouse tab switching across all tabs.
- **Duplicate Tab Prevention**: Enforced `TCM_DELETEALLITEMS` before tab population and escaped literal ampersands (`&&`) to prevent mnemonic underlines.
- **Tab 5 Controls Layout**: Updated `onResize` to explicitly reposition all child checkboxes, labels, and comboboxes inside their respective groupboxes, and added full window invalidation on tab switches for crisp GDI rendering.

## [2.0.0] - Native Win32 Architecture & Complete Rewrite

### Added
- **Pure Win32 Desktop Engine**: Replaced webview / browser runtime with lightweight native Windows C-API controls using `github.com/gonutz/w32/v2`. Binary size reduced to ~5.2 MB; cold-start execution <2ms; memory footprint ~7 MB.
- **Hierarchical Directory Navigation**:
  - History buttons: `[ < ]` (Back), `[ > ]` (Forward), and `[ ^ ]` (Up to parent directory).
  - Editable breadcrumb / target address bar with `[ GO ]` and `[ BROWSE... ]` dialog support.
  - Double-click folder navigation in Explorer listview with `..` parent item traversal.
- **Item Selection Checkboxes**: Native `LVS_EX_CHECKBOXES` for individual file inclusion/exclusion, `[ SELECT ALL ]`, and `[ CLEAR ALL ]` actions.
- **Universal Drive-Wide Search**:
  - Independent drive-level index querying across any mounted volume (`C:`, `E:`, etc.).
  - Search results list with batch staging (`[ STAGE SELECTED FOR SYNC ]`) and Explorer reveal (`[ OPEN IN EXPLORER ]`).
- **Tactile RP Badge Icon**: Custom multi-resolution icon (`assets/icon.ico`) with crisp lowercase `rp` badge embedded across all executables (`rsync-ui.exe`, `rs_copy.exe`, `rsync-service.exe`).
- **One-Click Elevated Shell Repair**: `fix_menus.bat` and `scripts/fix_shell_menus.ps1` to configure Explorer context menus using Microsoft's `ExtendedSubCommandsKey` pattern.

### Changed
- **Zero Collision Layout**:
  - Moved the collapsible settings drawer into a dedicated, spacious tab (`[6] Options & Settings`).
  - Fixed top navigation bar at Y: 10 with responsive width calculation.
  - Tab controls anchored at Y: 46 with clean visibility toggling between tabs.
- **Enhanced Status Bar**: Real-time 4-part status bar tracking active indexer engine, background service state, selected file count, and sync progress.
