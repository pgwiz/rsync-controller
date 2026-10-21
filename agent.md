# Agent Operational Guide

## Build & Test Instructions
- Run unit tests:
  ```powershell
  go test ./...
  ```
- Build Native UI:
  ```powershell
  go build -ldflags="-H windowsgui" -o rsync-ui.exe ./cmd/rsync_ui
  ```
- Build CLI & Health Tool:
  ```powershell
  go build -o rs_copy.exe ./cmd/rs_copy
  ```
- Build Service Daemon:
  ```powershell
  go build -o rsync-service.exe ./cmd/rsync_service
  ```

## Context Menu Maintenance
- If context menus fail to display cascaded flyouts, run `fix_menus.bat` as Administrator or invoke `rs_copy.exe --repair` from an elevated prompt.
- Never set a `(Default)` string value or empty `SubCommands` value on parent verb keys when using `ExtendedSubCommandsKey`.
