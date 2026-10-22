# Codebase Architecture & Directory Dependency Graph
_Last regenerated: 2026-10-22 12:57:52 UTC by dev-md-compactor_

## Project Manifests & Build Tools
- `go.mod`

## Module Directory Topography

### `Themes examples/`
- **Files (15)**: `rsync_controller_multi_theme_edition.html, theme-handdrawn-retro-pc-windows.jpg, theme-monochrome-classic-80s-90s-gui.jpg, theme-nostalgic-browser-media-popup-ui.jpg, theme-oldschool-browser-assets-loading.jpg, theme-pink-yellow-y2k-window-elements.jpg, theme-retro-os-folder-icons-media-player.jpg, theme-ui-reference-window.png` (+7 more)

### `assets/`
- **Files (3)**: `generate_icon.py, icon.ico, icon.png`
- **Discovered Module Dependencies**: `PIL, os`

### `cmd/rs_copy/`
- **Files (3)**: `app.manifest, main.go, rsrc_windows_amd64.syso`

### `cmd/rsync_service/`
- **Files (2)**: `main.go, rsrc_windows_amd64.syso`

### `cmd/rsync_ui/`
- **Files (6)**: `app.manifest, main.go, rsrc_windows_amd64.syso, theme_engine.go, treeview.go, win32_controls.go`

### `cmd/usn_bench/`
- **Files (1)**: `main.go`

### `core/`
- **Files (14)**: `config.go, core_test.go, engine_interface.go, fallback_walker.go, offline_queue.go, pipe_helper_windows.go, rsync_runner.go, search.go` (+6 more)

### `root/`
- **Files (28)**: `LICENSE, README.md, Solar Opposites (2020) Season 6 Episode 10- What Is The Miss.mp4, Solar Opposites (2020) Season 6 Episode 8- The Last Flight O.mp4, Solar Opposites (2020) Season 6 Episode 9- The Goocleus And.mp4, Solar_Opposites_480P_S06_E01.mp4, Solar_Opposites_480P_S06_E02.mp4, Solar_Opposites_480P_S06_E03.mp4` (+20 more)

### `scripts/`
- **Files (1)**: `fix_shell_menus.ps1`

### `themes/`
- **Files (4)**: `monochrome-mac.json, vaporwave-cyber.json, win95-classic.json, y2k-candy.json`

### `ui/`
- **Files (1)**: `index.html`

### `ui/assets/`
- **Files (2)**: `icon-192.png, icon.png`

### `ui/js/`
- **Files (1)**: `app.js`

### `ui/styles/`
- **Files (1)**: `tactile.css`

## Environment & Directory Catalog Reference
- Central Catalog: `dev_md_guides/directory.md` (sample committed as `dev_md_guides/directory.md.sample`).
- All workspace paths, servers, backend links, frontend links, and external endpoints are centralized in this catalog.
- Invariant: Never hardcode local filesystem paths or network URLs directly across project markdown files.

## Credentials & Secrets Reference
- Central Secrets Schema: `dev_md_guides/credentials.md` (sample committed as `dev_md_guides/credentials.md.sample`).
- Real secrets and sensitive tokens are kept strictly local in `credentials.md` and MUST NEVER be committed to GitHub.
- Invariant: Never commit credentials to GitHub; if the user ever specifies committing credentials, the agent must first explicitly warn the user about critical security risks.

## Architectural Invariants & Boundary Rules
- Internal modules should adhere to defined dependency boundaries without cyclic imports.
- Configuration, secrets, and environment overrides must not be hardcoded in application logic.
- Zero Hardcoded Endpoints: Do not hardcode machine directories, server IPs, backend links, or frontend links across markdown docs; resolve and reference them via directory.md (only directory.md.sample is committed to version control).
- Zero Credential Exposure: Never commit credentials.md or real secrets to git/GitHub. Only credentials.md.sample with sanitized placeholders is tracked. If the user explicitly asks to commit credentials, issue a critical security warning and require confirmation before proceeding.
