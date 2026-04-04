# CHANGELOG

## [0.1.1] - 2026-04-04

### Added
- Full Go/Wails v3 foundation: app bootstrap, config loader, path manager, SQLite db, logger
- Chat bindings skeleton (`internal/bindings/chat.go`) with `SendMessage` / `CancelGeneration`
- Frontend Chat UI: sidebar (collapse + drag-resize), model selector in toolbar, streaming hook, markdown renderer, Prism code highlight, KaTeX math
- Settings tab with card grid layout
- macOS Finder launcher (`Lancer-liaotao.command`) using `go run .`
- `build/config.yml` versioned (was missing, broke startup for all users)
- `scripts/healthcheck.sh` — 16/16 checks green

### Changed
- Frontend CSS split into `base.css` / `chat.css` / `settings.css` (was `smollama.css`)
- `.gitignore`: exclude `/logs/`, `/data/`, keep `build/config.yml`

---

## [Unreleased]


