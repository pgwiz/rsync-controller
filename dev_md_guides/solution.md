# Validated Solutions & Empirical Truths (`dev_md_guides/solution.md`)

_The living, mutable catalog of battle-tested problem resolutions, working implementations, and empirical ground truths._  
_Maintained in-place by the dev-md-compactor Conversational Reflection Protocol._

---

## 1. Active Solutions Directory

### SOL-001: Robust Windows Explorer Cascaded Submenus via ExtendedSubCommandsKey Pools
- **Status**: `Verified`
- **Target Context**: Windows 10/11 Shell Registry Integration (Go / Win32)
- **Problem & Goal**: Provide a branded cascading context menu (`rsyn_copy`) in File Explorer for both individual files/folders and empty directory backgrounds without crashing, dropping verbs, or leaving stale registry keys.
- **Evolution Ledger**:
  1. *Baseline*: Registered direct `shell` subkeys under `Directory\shell\Rsync\shell`.
     - *Result*: Menus appeared intermittently, background right-click dropped `%V`, and Windows Explorer sometimes merged stale HKLM verbs.
  2. *Refinements (Adding X)*: Switched to `ExtendedSubCommandsKey` pointing to a unified pool in `Directory\ContextMenus\Rsync`.
     - *Result*: Improved stability, but background clicks still broke when Explorer attempted to evaluate `%1` on empty canvas clicks.
  3. *Latest Proven Truth (Adding X + Y)*:
     - Separated into two distinct pools:
       - Item Pool: `Directory\ContextMenus\RsyncItem\shell` with `--src="%1 "` or `--dest="%1 "`
       - Background Pool: `Directory\ContextMenus\RsyncBg\shell` with `--bg="%V "` (strictly zero `%1` references)
     - Added trailing space padding inside quotes (`"%V "`, `"%1 "`) to prevent Windows `\"` backslash escaping bugs.
     - Deleted `(Default)` and `SubCommands` value entries on parent trigger verbs.
- **Why It Works (Mechanics)**: The `ExtendedSubCommandsKey` pool pattern is the official Microsoft standard for modern cascading menus (used by PowerShell 7). Separating background targets avoids invalid `%1` parameter resolution.
- **Verification Evidence**: Verified live in File Explorer across drive roots (`C:\`, `E:\`), directories, individual files, and desktop backgrounds.

---

### SOL-002: Dual-Engine Rsync Flag & Path Translation (GNU rsync & Rust rsync-win)
- **Status**: `Verified`
- **Target Context**: Windows CLI Execution / File Transfer Engine
- **Problem & Goal**: Execute transfers reliably whether the system has classical MSYS2/Cygwin GNU rsync or the Rust `rsync-win` binary installed.
- **Evolution Ledger**:
  1. *Baseline*: Executed `rsync -avz --progress <src> <dest>`.
     - *Result*: Worked with GNU rsync, but failed immediately with exit status 2 on `rsync-win` because flags `-z` and standard positional arguments were rejected.
  2. *Refinements (Adding X)*: Inspected `rsync --version` at startup and cached the engine type (`core.IsRsyncWin`).
  3. *Latest Proven Truth (Adding X + Y)*:
     - If `rsync-win`:
       - Paths are converted to forward slashes: `filepath.ToSlash(path)`.
       - Arguments formatted as: `-a -v --progress --src <cleanSrc> --dest <cleanDest>`.
       - For `move_to` / `move_from`: sequential execution with post-transfer `os.RemoveAll(src)` cleanup upon confirmed transfer.
     - If GNU rsync:
       - Arguments formatted as standard positional: `rsync -av --progress <sources...> <dest>`.
       - Moves use native `--remove-source-files`.
- **Why It Works (Mechanics)**: Dynamically conforms to the exact CLI interface supported by each engine without runtime crashes.
- **Verification Evidence**: Automated test `TestRsyncWinDetection` and `TestRsyncExecutionLive` passing in `core/core_test.go`.

---

### SOL-003: Single-Instance Anti-Recursion IPC via Named Pipes
- **Status**: `Verified`
- **Target Context**: Win32 GUI Process Management
- **Problem & Goal**: Prevent multiple window spawning when multi-selecting files in Explorer and clicking a context menu verb.
- **Evolution Ledger**:
  1. *Baseline*: Used a named mutex (`CreateMutexW`).
     - *Result*: Secondary instances exited, but could not deliver their selected file paths to the master window.
  2. *Refinements (Adding X)*: Added a local named pipe (`\\.\pipe\rsync-ui-ipc`) for secondary instances to forward file paths.
     - *Result*: File paths forwarded, but orphaned or closing windows could cause new launches to silently disappear.
  3. *Latest Proven Truth (Adding X + Y)*:
     - Combined `FindWindowW("RsyncControllerWin32Class")` with named pipe delivery check.
     - Secondary instance attempts `ForwardPayloadDirect()`. If delivery succeeds, it brings the existing window to the foreground via `AttachThreadInput` + `SetForegroundWindow` and exits (`os.Exit(0)`).
     - If delivery fails (stale window handle), the process falls through and initializes as the active master instance.
- **Why It Works (Mechanics)**: Handshake confirmation ensures payloads are only discarded when guaranteed received by a live master instance.
- **Verification Evidence**: Selecting 10+ files simultaneously in Windows Explorer populates the existing drawer without spawning duplicate windows.

---

### SOL-004: Non-Blocking Destination Validation with Visual Cueing
- **Status**: `Verified`
- **Target Context**: Native Win32 UI / UX
- **Problem & Goal**: Prevent transfers from executing when Destination is empty or matches Source, without frustrating the user with modal alert popups.
- **Evolution Ledger**:
  1. *Baseline*: Popped up `w32.MessageBox(hwnd, "Destination folder required", ...)` modal dialog.
     - *Result*: Annoying modal interruptions that required extra mouse clicks to dismiss.
  2. *Latest Proven Truth*:
     - Highlight and focus Destination edit box: `w32.SetFocus(hwndTxtDest)`.
     - Select all text in Destination edit box: `w32.SendMessage(hwndTxtDest, 0x00B1 /* EM_SETSEL */, 0, ^uintptr(0))`.
     - Display a high-visibility alert on status bar part 4.
     - Write warning to Transfer Log and cleanly abort transfer execution.
- **Why It Works (Mechanics)**: Guides the user's attention directly to the input field requiring action without interrupting the interface flow.
- **Verification Evidence**: Verified live in UI with both empty and identical Source/Destination paths.
