# rsyn_copy Controller & File Transfer Utility (v3.1.0)

[![Build & Test](https://github.com/pgwiz/rsync-controller/actions/workflows/ci.yml/badge.svg)](https://github.com/pgwiz/rsync-controller/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/pgwiz/rsync-controller?color=blue)](https://github.com/pgwiz/rsync-controller/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**rsyn_copy** is a high-performance, native Win32 file synchronization orchestrator and Windows Explorer shell extension powered by `rsync`. Built with pure native Win32 GDI controls (zero Electron, zero webview, zero browser runtimes), it delivers instantaneous startup (<50ms), a minimal memory footprint (<15MB RAM), and comprehensive local and remote file transfer workflows.

---

## Key Features

### 1. Dedicated Dual Source & Destination Bars with Instant Swap (New in v3.1.0)
- **Visible Two-Row Navigation**:
  - **Row 1 (Source)**: Path edit box (`hwndTxtTarget`), `GO`, `BROWSE...`, `JUMP ▾`, and real-time substring filter (`hwndTxtFilter`).
  - **Row 2 (Destination)**: Explicit Target Destination edit box (`hwndTxtDest`), `BROWSE...` button (`ID_BTN_BROWSE_DEST`), interactive `SWAP ⇄` button (`ID_BTN_SWAP`), and operation status banner (`hwndLblOpBanner`).
- **Interactive `SWAP ⇄` Mechanics**: Reverses Source and Destination paths with a single click, instantly navigating to the new Source directory, reloading the tree view and file list, and updating the operation banner.
- **Non-Blocking Destination Validation Guard**:
  - If Destination is empty or identical to Source when clicking `EXECUTE RSYNC`, **no modal popup dialog is spawned**.
  - Keyboard focus automatically shifts to `hwndTxtDest` with all text selected (`EM_SETSEL`), while a high-visibility warning flashes on the status bar and logs to the Transfer Log.
- **Elimination of Destination Locking**: Staging a new copy or move explicitly resets destination state, preventing accidental transfers into previously used folders or working directories.

### 2. Dual-Engine Rsync Compatibility (GNU rsync & Rust rsync-win)
- **Automatic Runtime Engine Detection**: Automatically detects whether the installed `rsync` executable in `%PATH%` is classical GNU rsync or the Rust-based `rsync-win` port (via cached `rsync --version` inspection).
- **Dynamic Parameter Shaping**:
  - Translates backslashes to forward slashes (`filepath.ToSlash`) to comply with Unix-style path standards.
  - Automatically shapes arguments for Rust `rsync-win` (`-a -v --progress --src <src> --dest <dest>`) while avoiding unsupported flags (`-z`, `--info=progress2`).
  - Sequential multi-item invocation with post-transfer source cleanup (`os.RemoveAll`) for atomic moves on `rsync-win`.
- **Live Output Streaming**: Real-time stdout and stderr output streamed line-by-line into the Transfer Log tab.

### 3. Deep Windows Explorer Shell Integration (`rsyn_copy`)
- **Cascading Flyout Submenus**: Integrated directly into File Explorer across directories, individual files, drive roots, and folder backgrounds via the official `ExtendedSubCommandsKey` pool pattern.
- **Dedicated Background Context Pool**:
  - Separates item targets (`Directory\ContextMenus\RsyncItem`) and empty folder background targets (`Directory\ContextMenus\RsyncBg`).
  - Background targets pass only `--bg="%V "` (strictly zero `%1` references), eliminating dropped commands.
  - Trailing space argument padding (`"%V "`, `"%1 "`) prevents Windows backslash-escaping bugs (`\"`) on drive roots and trailing slashes.
- **Smart Background Context Handling**:
  - `Copy to...`: Sets the active folder as Destination (`stagedDest`).
  - `Move to...`: Sets the active folder as Source (`stagedPath`).
  - `Pull from...` & `Move from...`: Sets the active folder as Destination (`stagedDest`).
- **In-App ListView Background Context Menu**: Right-clicking empty space inside the app's file list presents a context menu for `Copy to...`, `Move to...`, `Pull from...`, `Move from...`, `Check All`, `Uncheck All`, `Refresh (F5)`, `Reveal in Explorer`, `Copy Path`, and `Swap ⇄`.

### 4. Dynamic System Drive Discovery
- **Native Dual-Pane TreeView (`SysTreeView32`)**: Dynamically enumerates all mounted system drives (`C:\`, `D:\`, `E:\`, USB, optical, network) using Win32 `GetLogicalDrives`, `GetDriveTypeW`, and `GetVolumeInformationW`.
- **Standardized Drive Root Labels**: Drives are formatted and aligned by letter: `C: (Local Disk)`, `D: (ChromeOS)`, `E: (Data)`.
- **Universal Search Dynamic Drive Combo**: Dropdown automatically discovers all connected volumes in real time rather than using static lists.

### 5. 18 Retro GDI Themes & Live Palette Customizer
- **Authentic Retro & Modern Aesthetics**: Windows 95 Classic, Dark Obsidian, Y2K Cyber Candy, Cyber Vaporwave, Monochrome 1-Bit, Y2K Aqua, Warm Handdrawn Retro, Pastel Retro Grid, Nostalgic Media, Pink & Yellow Y2K Pop, Retrowave Vintage PC, Classic Win95/98 Teal, Y2K Blue Folders, Vintage Pixel PC, Y2K Music Chat, Desktop Music Player & Clock, System Dialogs Green/Blue, System Dialogs Sunset/Pink.
- **Interactive Theme Customizer**:
  - Native Windows color dialog (`ChooseColorW`) for Canvas, Surface, Button Face, Button Text, Accent, and Window Border colors.
  - **Shadow Color & Depth Customizer**: Select from 0px (Flat), 1px (Subtle), 2px (Standard), 4px (Bold 3D), and 6px (Deep Comic), with customizable shadow tint (`ShadowColor`).
  - Windows 11 window corner synchronization via `DWMWA_WINDOW_CORNER_PREFERENCE`.

### 6. Anti-Recursion Single-Instance IPC
- Desktop-level window detection (`FindWindowW("RsyncControllerWin32Class")`) combined with local named pipe IPC (`\\.\pipe\rsync-ui-ipc`).
- Multi-file selections in Windows Explorer immediately aggregate into the active instance's staging drawer without spawning duplicate windows.
- Delivery confirmation handshake ensures that if a window handle is stale or closed, the process cleanly falls through and takes over as the master instance.

### 7. Zero Square Bracket Policy
- Clean, unbracketed text formatting across all visible UI buttons, tabs, status bar notifications, transfer logs, and daemon messages (`READY`, `File`, `Folder`, `Up`, `ACTIVE: LOCAL COPY`, `EXECUTE RSYNC`, `SYNC RUN`, `OK:`, `FAIL:`, `INFO:`, `WARN:`).

---

## Repository Structure

```text
E:\Backup\pgwiz\rsync\
├── cmd\
│   ├── rsync_ui\          # Main Win32 desktop application
│   │   ├── main.go        # Window procedure, controls, layout & event loops
│   │   ├── theme_engine.go# 18 GDI themes, owner-drawn buttons, palette customizer
│   │   ├── treeview.go    # Dual-pane SysTreeView32 drive & folder explorer
│   │   └── win32_controls.go # Pure Win32 helper controls, clipboard, cue banners
│   ├── rs_copy\           # CLI orchestrator, elevation helper & watchdog
│   │   └── main.go
│   ├── rsync_service\     # Elevated background Windows Service daemon
│   └── usn_bench\         # Standalone USN Journal benchmark CLI tool
├── core\                  # Core transfer engine & OS integrations
│   ├── config.go          # Configuration, theme serialization & server profiles
│   ├── rsync.go           # Rsync execution runner, dual-engine detector & streaming
│   ├── shell_integration_windows.go # Explorer context menu installer (RsyncItem & RsyncBg)
│   ├── single_instance_windows.go   # Named mutex & IPC pipe aggregator
│   ├── pipe_helper_windows.go       # Win32 Named Pipe listener & dialer
│   ├── usn_windows.go     # FSCTL_ENUM_USN_DATA NTFS volume scanner
│   ├── fallback_walker.go # Non-elevated directory crawler
│   └── core_test.go       # Automated unit test suite
├── dev_md_guides\         # Persistent living documentation suite
│   ├── agent.md           # Master documentation index & file inventory
│   ├── branch.md          # Active git branch & worktree status
│   ├── changelog.md       # Chronological session operational audit log
│   ├── features.md        # Spec-Driven Development (SDD) feature matrix
│   ├── memory.md          # Architectural invariants & ADRs
│   ├── gotchas.md         # Categorized runtime bugs, root causes & fixes
│   ├── flow.md            # Execution workflows & test recipes
│   ├── playbook.md        # Operational runbooks & setup playbooks
│   ├── solution.md        # Empirical ground truths & parameter evolution ledgers
│   ├── directory.md.sample# Committed reference environment catalog
│   └── credentials.md.sample # Committed reference secrets schema
├── assets\
│   └── icon.ico           # Official application icon
├── config.json            # Active user configuration
├── dev_com_agent.md       # Universal root entry point for autonomous agents
├── README.md              # Project overview & documentation
├── changelog.md           # Public release changelog
└── memory.md              # Architectural decisions & conventions
```

---

## Quick Start & Usage

### 1. Build from Source
Prerequisites: Go 1.22+ on Windows x86_64.
```powershell
# Run unit tests
go test -v ./core

# Compile the native Win32 GUI executable (no console window)
go build -ldflags "-H windowsgui" -o rsync-ui.exe ./cmd/rsync_ui

# Compile the CLI orchestrator
go build -o rs_copy.exe ./cmd/rs_copy
```

### 2. Register Windows Explorer Context Menus
Run from any standard user terminal (Administrator privileges not required):
```powershell
.\rsync-ui.exe --register-shell
```
To remove context menu entries:
```powershell
.\rsync-ui.exe --unregister-shell
```

### 3. Launching the Application
```powershell
# Open interactive UI
.\rsync-ui.exe

# Open with specific source directory staged
.\rsync-ui.exe --mode=copy --src="C:\Projects\Data"
```

### 4. Global Keyboard Accelerators
- `F5`: Refresh active directory scan
- `Backspace`: Navigate up to parent folder
- `Alt + Left / Right`: Navigate backward / forward in history
- `Ctrl + A`: Check / stage all files in current view
- `Ctrl + F`: Shift keyboard focus to Search filter
- `Ctrl + D`: Trigger Dry Run test
- `Ctrl + E`: Execute Rsync transfer
- `Space`: Toggle checkbox staging for selected item
- `Delete`: Remove selected item from staging drawer

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
