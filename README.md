# Rsync Controller & File Transfer Utility

An ultra-fast, cross-platform file transfer utility and Windows Shell orchestrator powered by `rsync`.

---

## Key Features

- **Ultra-Fast Disk Indexing**:
  - **Engine 1: Windows USN Journal (`FSCTL_ENUM_USN_DATA`)**: Queries NTFS change journal directly (~2-4s for 1,000,000 files).
  - **Engine 2: Raw MFT Sector Reader**: Direct low-level physical disk structure reader.
  - **Engine 3: Voidtools Everything SDK**: Instant 0ms queries via `Everything64.dll` IPC.
  - **Engine 4: Direct Filesystem Crawler**: Non-elevated fallback crawling.
- **Deep Windows Explorer Integration & Transfer Modes**:
  - Right-click cascaded context menu on directories, files, and background space (including drive roots via `Drive\Background`).
  - Streamlined native menu layout with direct keyboard access keys:
    - `Copy to...` (`Alt+C`)
    - `Move to...` (`Alt+M`)
    - `Pull from...` (`Alt+P`)
    - `Move from...` (`Alt+F`)
    - `Push to Alpha Server...` (`Alt+U`)
    - `Pull from Alpha Server...` (`Alt+L`)
    - `Open Rsync Controller...` (`Alt+O`)
  - Dedicated background menu pool (`Directory\ContextMenus\RsyncBg`) passing `--bg="%V "` with trailing space padding and strictly zero `%1` references, ensuring Explorer never drops or cancels background commands.
  - Smart Background Click Context Handling:
    - `Move to...`: Background folder is source directory (`stagedPath`), with destination prompted.
    - `Copy to...`: Background folder is destination directory (`stagedDest`), with button `CHOOSE SOURCE & COPY`.
    - `Move from...` & `Pull from...`: Background folder is destination directory (`stagedDest`), with button `CHOOSE SOURCE & MOVE` / `CHOOSE SOURCE & PULL`.
  - Active Item Focus: When opened with a selected item, it is actively selected (`LVIS_SELECTED`), focused (`LVIS_FOCUSED`), and scrolled into view (`LVM_ENSUREVISIBLE`).
  - Single-Instance Named Mutex & Anti-Recursion: Multi-file selections in Explorer aggregate into the single master window with zero duplicate window recursion.
- **Atomic Move & Transfer Operations**:
  - `Copy to...`: Selected items copied to target folder via Rsync.
  - `Move to...`: Selected items moved to target folder with automatic cleanup of source files via `--remove-source-files`.
  - `Pull from...`: Destination folder locked to active directory; syncs from selected source folder.
  - `Move from...`: Destination folder locked to active directory; moves from source folder with automatic cleanup of source files via `--remove-source-files`.
  - Tab 0 Mode Selector (`MODE: ... ▾`) and in-app right-click list view menus for instant operation switching.
- **Comprehensive TreeView Drive Enumeration**:
  - Native `SysTreeView32` displays all available system drives (`C:\`, `D:\`, `E:\`, etc.) using Win32 `GetLogicalDrives` and volume labels.
  - Drive roots and directories expand on demand, with full refresh on `RESCAN`.
- **Instant On-Type Search & Persistent Staging**:
  - Microsecond in-memory search across cached records with zero disk re-scanning.
  - Selection persistence: Typing in the search bar or changing tabs never clears staged items.
  - Dedicated Staged Selection summary drawer and review drawer.
  - Bottom summary bar calculating selected items count and total byte size.
- **Full 18-Theme Suite & Live Radius Controller**:
  - 18 authentic themes matching retro and modern OS aesthetics (Windows 95 Classic, Dark Obsidian, Y2K Cyber Candy, Cyber Vaporwave, Monochrome 1-Bit, Y2K Aqua, Warm Handdrawn Retro, Pastel Retro Grid, Nostalgic Media, Pink & Yellow Y2K Pop, Retrowave Vintage PC, Classic Win95/98 Teal, Y2K Blue Folders, Vintage Pixel PC, Y2K Music Chat, Desktop Music Player & Clock, System Dialogs Green/Blue, System Dialogs Sunset/Pink).
  - Live Corner Radius Controller (0px sharp retro, 4px modern subtle, 8px rounded Y2K, 14px full pill aqua) adjusting button shapes and Windows 11 DWM window corners.
  - Interactive Theme dropdown menu (`THEME: ... ▾`) on the top bar and Settings tab for instant live preview.
- **Top Quick Jump Menu (`JUMP ▾`) & Drive Space Storage Gauge**:
  - One-click navigation to User Home, Desktop, Downloads, Documents, and all mounted system drives (`C:\`, `D:\`, `E:\`, etc.).
  - 5-part responsive status bar with real-time drive free space capacity monitoring (`DRIVE E: 142.3 GB free / 476.9 GB (70% free)`).
- **Global Keyboard Accelerators & In-App Context Menus**:
  - Power hotkeys: `F5` (refresh), `Backspace` (parent directory), `Alt+Left/Right` (history), `Delete` (unstage), `Space` (toggle), `Ctrl+A` (stage all), `Ctrl+F` (search), `Ctrl+D` (dry run), `Ctrl+E` (execute).
  - Right-click context menus on both list views with `Reveal in Windows Explorer`, `Copy Full Path`, `Copy All Staged Paths`, and `Jump to Folder`.
- **Zero Square Bracket Policy in UI**:
  - Clean, unbracketed presentation across all buttons, tabs, banners, status bar indicators, and list items (e.g. `EXECUTE RSYNC`, `Local Explorer`, `ACTIVE: LOCAL COPY`, `READY`, `File`, `Folder`, `Up`).

---

## Directory Structure

```text
E:\Backup\pgwiz\rsync\
├── cmd\
│   ├── rsync_ui\          # Main interactive UI launcher and server
│   ├── rsync_service\     # Elevated background Windows Service daemon
│   └── usn_bench\         # Standalone USN Journal benchmark CLI tool
├── core\
│   ├── types.go           # FileRecord and VolumeIndex structures
│   ├── usn_windows.go     # FSCTL_ENUM_USN_DATA Win32 volume scanner
│   ├── search.go          # Microsecond in-memory search and filters
│   ├── fallback_walker.go # Non-elevated directory crawler
│   ├── engine_interface.go# Strategy engine switcher (USN / MFT / Everything / Crawler)
│   ├── config.go          # Configuration and server profiles
│   ├── single_instance_windows.go # Mutex & Named Pipe multi-file aggregator
│   ├── pipe_helper_windows.go    # Win32 Named Pipe listener
│   ├── shell_integration_windows.go # HKCU context menu installer
│   ├── shell_linux_gen.go # KDE Dolphin and Nautilus script generator
│   ├── rsync_runner.go    # Rsync execution with live progress parser
│   └── offline_queue.go   # Persistent queue for offline server transfers
├── themes\                # Ready-to-use themes
│   ├── y2k-candy.json
│   ├── win95-classic.json
│   ├── monochrome-mac.json
│   └── vaporwave-cyber.json
├── ui\                    # Frontend webview / browser interface
│   ├── index.html
│   ├── styles\tactile.css
│   └── js\app.js
├── Themes examples\       # 16 organized reference screenshots
├── config.json            # Active app configuration
├── rsync-ui.exe           # Compiled single-instance UI executable
├── rsync-service.exe      # Compiled background Windows Service daemon
└── README.md
```

---

## Quick Start & Usage

### 1. Register Right-Click Windows Explorer Menus
Run from standard user terminal (no Administrator rights needed):
```powershell
.\rsync-ui.exe --register-shell
```
To remove context menu entries:
```powershell
.\rsync-ui.exe --unregister-shell
```

### 2. Launching the UI Manually
```powershell
.\rsync-ui.exe
```
Or open directly with staged files from terminal:
```powershell
.\rsync-ui.exe --mode=copy --src="E:\Backup\pgwiz\rsync"
```

### 3. Optional: Install the Elevated Background Service
*(Required only if you want volume-level USN disk scans without running the UI as Administrator)*:
Open an elevated (Administrator) terminal and run:
```powershell
.\rsync-service.exe --install
```
To uninstall:
```powershell
.\rsync-service.exe --uninstall
```

### 4. Run Benchmark Test
```powershell
go run .\cmd\usn_bench\main.go E:
```

---

## Theme Customization & Community Packs

1. Open the UI and navigate to the **[Theme Studio]** tab.
2. Adjust sliders for **Window Radius**, **Button Radius**, **Drop Shadow**, **Gloss Specular Sheen**, and **Colors**.
3. Click **`[Export Theme (.zip / JSON)]`** to download your custom theme.
4. Share custom themes by submitting a Pull Request to the public theme directory.

---

## Testing

Run automated unit and integration tests:
```powershell
go test -v ./core/...
```
