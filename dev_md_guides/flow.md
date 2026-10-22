# Procedural Execution Flows & Operational Recipes (`dev_md_guides/flow.md`)

_Standard execution workflows, tool-chain recipes, and step-by-step verification procedures._  
_Maintained by the dev-md-compactor Conversational Reflection Protocol._

---

## 1. Local Build & Quality Gate Execution Flow

```powershell
# Step 1: Run comprehensive core test suite
go test -v ./core

# Step 2: Build GUI desktop executable without console window
go build -ldflags "-H windowsgui" -o rsync-ui.exe ./cmd/rsync_ui

# Step 3: Build CLI orchestrator and health monitor
go build -o rs_copy.exe ./cmd/rs_copy

# Step 4: Verify zero compiler warnings and clean exit codes
```

---

## 2. Explorer Shell Registration & Refresh Flow

```powershell
# Step 1: Register rsyn_copy cascading submenus in HKCU and HKLM (if elevated)
.\rsync-ui.exe --register-shell

# Step 2: Verify registered keys in registry
Get-ItemProperty "HKCU:\Software\Classes\Directory\shell\Rsync"
Get-ChildItem "HKCU:\Software\Classes\Directory\ContextMenus\RsyncItem\shell"
Get-ChildItem "HKCU:\Software\Classes\Directory\ContextMenus\RsyncBg\shell"

# Step 3: To unregister shell integration cleanly
.\rsync-ui.exe --unregister-shell
```

---

## 3. GitHub Actions CI & Release Deployment Pipeline Flow

```powershell
# Step 1: Verify all unit tests pass locally
go test -v ./...

# Step 2: Stage and commit updated files
git add .
git commit -m "feat(ui): release summary"
git push origin main

# Step 3: Monitor GitHub Actions workflow run
gh run list --limit 3
gh run watch <run-id>

# Step 4: Package release archive
Compress-Archive -Path rsync-ui.exe, rs_copy.exe, assets, config.json, README.md, changelog.md -DestinationPath rsyn_copy-v3.X.X-windows-amd64.zip -Force

# Step 5: Publish GitHub Release
gh release create v3.X.X rsyn_copy-v3.X.X-windows-amd64.zip --title "rsyn_copy v3.X.X - Description" --notes "Release notes"
```

---

## 4. Context Compaction & Living Documentation Sync Flow

```bash
# Step 1: Preview ground truth state and git diffs
python dev-md-compactor/scripts/run_compactor.py --report-only

# Step 2: Synchronize dev_md_guides/
python dev-md-compactor/scripts/run_compactor.py

# Step 3: Verify secrets are not tracked
git status dev_md_guides/credentials.md
```
