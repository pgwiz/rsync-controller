# Operational Playbooks & Framework Runbooks (`dev_md_guides/playbook.md`)

_Battle-tested operational execution recipes, multi-step runbooks, and framework verification workflows._  
_Maintained and updated in-place by the dev-md-compactor Conversational Reflection Protocol._

---

## 1. Active Operational Playbooks

### PLAY-001: Win32 Clean Build and Release Packaging
- **Category**: `Build & Packaging`
- **Prerequisites**: Go 1.22+ on Windows x86_64, `gh` CLI authenticated to GitHub.
- **Execution Runbook**:
  1. *Run Unit Tests*:
     ```powershell
     go test -v ./core
     ```
  2. *Compile Executables*:
     ```powershell
     go build -ldflags "-H windowsgui" -o rsync-ui.exe ./cmd/rsync_ui
     go build -o rs_copy.exe ./cmd/rs_copy
     ```
  3. *Verify Application Version & Assets*:
     Ensure `assets/icon.ico` and `config.json` exist alongside the newly generated `.exe` files.
  4. *Zip Distribution Package*:
     ```powershell
     Compress-Archive -Path rsync-ui.exe, rs_copy.exe, assets, config.json, README.md, changelog.md -DestinationPath rsyn_copy-v3.1.0-windows-amd64.zip -Force
     ```
  5. *Publish GitHub Release*:
     ```powershell
     gh release create v3.1.0 rsyn_copy-v3.1.0-windows-amd64.zip --title "rsyn_copy v3.1.0" --notes "Release notes"
     ```
- **Expected Verification Output**: `https://github.com/pgwiz/rsync-controller/releases/tag/v3.1.0` published successfully.
- **Failure Recovery**: If build fails with `access denied`, terminate any running `rsync-ui.exe` processes via Task Manager or `Stop-Process -Name rsync-ui -Force`.
- **Last Verified**: 2026-10-22 (Windows 11)

---

### PLAY-002: Shell Cascading Submenu Diagnostic & Registry Repair
- **Category**: `Environment Setup`
- **Prerequisites**: Windows PowerShell, user administrative access optional (HKCU works without elevation).
- **Execution Runbook**:
  1. *Run Automated Re-registration*:
     ```powershell
     .\rsync-ui.exe --register-shell
     ```
  2. *Verify HKCU Cascaded Submenu Keys*:
     ```powershell
     Get-ItemProperty "HKCU:\Software\Classes\Directory\shell\Rsync"
     Get-ChildItem "HKCU:\Software\Classes\Directory\ContextMenus\RsyncItem\shell"
     Get-ChildItem "HKCU:\Software\Classes\Directory\ContextMenus\RsyncBg\shell"
     ```
  3. *Verify Menu Title & Icon*:
     Ensure `MUIVerb` is set to `rsyn_copy` and `Icon` points to the valid absolute path of `assets\icon.ico`.
  4. *Test Shell Execution*:
     Right-click a folder in Windows Explorer -> `rsyn_copy` -> `Copy to...`. Verify `rsync-ui.exe` opens with the target folder staged.
- **Expected Verification Output**: Explorer right-click displays `rsyn_copy` submenu with icons and working action verbs.
- **Failure Recovery**: Run `.\rsync-ui.exe --unregister-shell` followed by `.\rsync-ui.exe --register-shell`.
- **Last Verified**: 2026-10-22 (Windows 11)

---

### PLAY-003: Dual-Engine Rsync Compatibility Verification Runbook
- **Category**: `Testing & QA`
- **Prerequisites**: `rsync` executable available in `%PATH%`.
- **Execution Runbook**:
  1. *Inspect Detected Rsync Version*:
     ```powershell
     rsync --version
     ```
  2. *Execute Live Rsync Test*:
     ```powershell
     go test -v -run TestRsyncWinDetection ./core
     go test -v -run TestRsyncExecutionLive ./core
     ```
  3. *Verify Execution Engine Output*:
     Ensure log output confirms `rsync-win detected: true/false` and transfers succeed with exit code 0.
- **Expected Verification Output**: `PASS: TestRsyncExecutionLive` with clean file transfer output.
- **Last Verified**: 2026-10-22 (Windows 11)
