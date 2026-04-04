# Adapter: wails-go

## What this adapter provides
- `internal/config/config.go` — layered TOML config with validation
- `internal/paths/paths.go` — safe path manager with allowed_roots guard
- `internal/logger/logger.go` — structured logging (JSON file + console)
- `main.go` — Wails v3 entry point wiring the above
- `frontend/` — minimal HTML/CSS/JS frontend (no framework)

## Requirements
- Go >= 1.22 (`brew install go` or https://go.dev/dl/)
- Wails CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
- Node.js >= 20 (for frontend assets, optional if no npm deps)
- Platform prerequisites: https://wails.io/docs/gettingstarted/installation

## Getting started
```bash
wails3 dev
```

## Running tests
```bash
go test ./...
```

## Config layers
1. `config/default.toml` (bundled with the app)
2. `~/.config/<app>/user.toml` (optional user overrides)
3. `config/project.toml` (optional local project overrides, not committed)
4. Environment variables with prefix `APP__` (for example `APP__APP__MODE=normal`)

## Notes
- Always start in `mode = "debug"` until workflow is validated.
- Logs are written to `logs/app.log` in JSON format with rotation.
- All write operations are restricted to `path_manager.allowed_roots`.
