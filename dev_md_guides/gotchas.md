# Project Gotchas & Failure Mode Analysis (`dev_md_guides/gotchas.md`)

_Categorized operational failures, non-obvious runtime bugs, and battle-tested fixes._  
_Maintained by the dev-md-compactor Conversational Reflection Protocol._

---

## 1. Major Blockers (Fatal Crashes, Leaks & Data Loss Risks)

### GOTCHA-001: Silent Destination Locking and Working Directory Dumping
- **Severity**: Major Blocker
- **Trigger / Symptom**: When running Rsync transfers, files were unexpectedly copied into `E:\Backup\pgwiz\rsync` (e.g. `RUN: rsync -a -v --progress --src E:/entertainment/... --dest E:/Backup/pgwiz/rsync`) instead of the user's intended target directory.
- **Root Cause**:
  1. `stagedDest` retained its value from previous operations or background clicks and was never reset when new files were selected.
  2. `runRsyncTransfer` had a silent fallback (`if dest == "" { dest = "C:\Backup\Target" }`) and used the stale `stagedDest`.
  3. Destination was not displayed on Tab 0, making the target folder completely invisible to the user prior to execution.
- **Exact Fix**:
  1. Added dedicated Destination address bar (`hwndTxtDest`), `BROWSE...` button, and `SWAP ⇄` button on Tab 0 Row 2 (`cmd/rsync_ui/main.go`).
  2. In `handleLaunchPayload` and item click actions, if `p.Dest == ""`, explicitly reset `stagedDest = ""` and set `hwndTxtDest` to empty string.
  3. Added non-blocking validation guard in `runRsyncTransfer`: if destination is empty or matches source, shift focus to `hwndTxtDest`, select all text, flash a status bar warning, log to Transfer Log, and abort transfer.
- **Mechanics**: Guarantees that target destination is always visible, editable, and never defaults to stale or hardcoded directories.

---

### GOTCHA-002: Rsync Exit Status 2 on Rust `rsync-win` Executables
- **Severity**: Major Blocker
- **Trigger / Symptom**: Running transfers failed immediately with `FAIL: Transfer finished with error: exit status 2`.
- **Root Cause**: The installed `rsync` binary was the Rust-based `rsync-win` implementation. The execution engine passed standard GNU flags (`-z`, `--info=progress2`) which the Rust CLI parser strictly rejected with exit code 2.
- **Exact Fix**:
  Implemented `core.IsRsyncWin` in `core/rsync.go` to inspect cached `rsync --version` output. When `rsync-win` is detected, command arguments are transformed to `-a -v --progress --src <src> --dest <dest>` using forward slashes (`filepath.ToSlash`), omitting all unsupported flags.
- **Mechanics**: Dynamically bridges the divergence between GNU rsync and Rust rsync-win CLI specifications.

---

### GOTCHA-003: Multiple Window Recursion on Multi-File Explorer Selections
- **Severity**: Major Blocker
- **Trigger / Symptom**: Selecting 20+ files in Windows Explorer and clicking context menu verb spawned 20+ separate UI windows, causing high CPU spikes and UI locking.
- **Root Cause**: Windows Explorer launches the registered executable separately for every selected item when verbs are invoked across multiple items without single-instance interception.
- **Exact Fix**:
  Added desktop-level Win32 window class check (`FindWindowW("RsyncControllerWin32Class")`) in `main()`. If an existing window is detected, secondary processes forward their `LaunchPayload` to the master process via named pipe `\\.\pipe\rsync-ui-ipc`, bring the existing window to the foreground (`AttachThreadInput` + `SetForegroundWindow`), and immediately terminate (`os.Exit(0)`).
- **Mechanics**: Completely prevents multiple window spawning; all selected files aggregate cleanly into the active window's staging drawer.

---

## 2. Minor Quirks (Silent Defaults, Formatting & Syntax Edge Cases)

### GOTCHA-004: Windows Shell Trailing Backslash Quote Escaping (`\"`)
- **Severity**: Minor Quirk
- **Trigger / Symptom**: Shell context menus failed when invoked from drive roots (`C:\`) or folders ending with a backslash.
- **Root Cause**: Windows command-line tokenization treats `\"` as an escaped quotation mark rather than a backslash followed by a closing quote. Thus, `"--src=\"C:\\\""` caused the rest of the arguments to be swallowed into a single broken parameter.
- **Exact Fix**: Added a trailing space before the closing quote in all registry verb registrations: `"--src=\"%1 \""` and `"--bg=\"%V \""`. Handled the space in `cleanArgParam()`.
- **Mechanics**: The space separates the trailing backslash from the closing quote, ensuring clean parameter extraction.

---

### GOTCHA-005: ListView Background Right-Click Inaction
- **Severity**: Minor Quirk
- **Trigger / Symptom**: Right-clicking empty space in the file list did nothing.
- **Root Cause**: In `showListViewContextMenu()`, `sel := int(SendMessage(hwndLV, LVM_GETNEXTITEM, ...))` returned `-1` when no item was selected, and the function immediately executed `return`.
- **Exact Fix**: Added `if sel < 0 { showListViewBackgroundContextMenu(); return }` presenting a full background context menu (`Copy to...`, `Move to...`, `Pull from...`, `Move from...`, `Select All`, `Refresh`, `Reveal in Explorer`, `Copy Path`, `Swap`).
- **Mechanics**: Context menu triggers reliably whether clicking on a specific row or the empty list canvas.

---

### GOTCHA-006: Dead Window Handle Silently Swallowing Process Launches
- **Severity**: Minor Quirk
- **Trigger / Symptom**: Context menu options occasionally appeared to do nothing after a previous instance had closed.
- **Root Cause**: If `FindWindowW` detected a window handle in the process of closing or a zombie window, `core.ForwardPayloadDirect` failed, but `main()` unconditionally executed `os.Exit(0)`.
- **Exact Fix**: Updated single-instance check so that `os.Exit(0)` is ONLY called if `ForwardPayloadDirect` succeeds. If named pipe delivery fails, the process falls through and initializes as the new master instance.
- **Mechanics**: Eliminates silent process termination on stale or orphaned window handles.

---

### GOTCHA-007: Drive Root Truncation by `filepath.Clean`
- **Severity**: Minor Quirk
- **Trigger / Symptom**: Navigating to a drive root like `C:\` converted the path to relative drive identifier `C:`, failing directory scans.
- **Root Cause**: Go's `filepath.Clean("C:\\")` strips trailing slashes, leaving `C:`, which references the current working directory of drive C rather than the filesystem root.
- **Exact Fix**: Implemented `normalizeDriveRoot()` in `treeview.go` ensuring any 2-character drive string `X:` is normalized with a trailing backslash `X:\`.
- **Mechanics**: Restores absolute root path semantics across all drive operations.
