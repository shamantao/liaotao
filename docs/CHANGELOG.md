# CHANGELOG

## [0.1.2] - 2026-04-10

### Fixed (DEBT)
- **DEBT-01**: `conversations.provider_id` migrated from `TEXT` to `INTEGER FK` — conversations no longer lose their provider link on rename/delete. Three regression tests added (`TestConversation_*`).
- **DEBT-03**: Implemented missing built-in MCP tools `read_file` (with `allowed_roots` sandbox) and `web_fetch` (with SSRF guard blocking localhost and RFC-1918 ranges). 12 unit tests added (`TestBuiltin*`, `TestDispatch*`).
- **DEBT-04**: Aligned documentation — PRD and CHANGELOG now use the actual DB filename `liaotao.db` (was `conversations.db`).
- **DEBT-05**: Updated PRD Delivery Plan phases status (Phases 0–3 ✅ Complete); marked SET-01, SET-03, SET-04 as done in backlog.
- **DEBT-06**: Removed hard-coded absolute paths from `config/default.toml`. All paths now use `$HOME/.config/liaotao/...` with `$HOME` expansion at runtime. Added `config/project.toml.example` for developer local overrides.

### Changed
- `config/default.toml`: portable `$HOME`-relative paths for all directories and DB.
- `internal/config/config.go`: added `expandHomeDirPaths()` post-load step.
- `internal/bindings/service.go`: `NewService()` accepts variadic `allowedRoots` (non-breaking); `CreateConversationPayload.ProviderID` is now `int64`.
- `internal/db/db.go`: added `ApplySchemaForTest()` helper; `migrateConversationsProviderID()` runs on startup.
- `.gitignore`: added `config/project.toml` exclusion.

---

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


