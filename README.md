# liaotao

Lightweight local-first desktop AI chat app built with Go + Wails v3 and a vanilla HTML/CSS/JS frontend.

## Core features
- Multi-provider chat: OpenAI-compatible APIs, Ollama, and MCP tool servers.
- Smart routing with provider priority + quota-aware fallback.
- Conversation history with search, rename, delete, and per-message token stats.
- Per-conversation generation settings: model, temperature, max tokens, system prompt.
- Settings tab with General, Providers, MCP Servers, and About sections.
- Configuration import/export in TOML format.
- Provider API keys encrypted at rest in SQLite (application-level AES-GCM).

## Requirements
- Go >= 1.22 (`brew install go` or https://go.dev/dl/)
- Wails CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
- Node.js >= 20 (for frontend assets, optional if no npm deps)
- Platform prerequisites: https://wails.io/docs/gettingstarted/installation

## Development
```bash
wails3 dev
```

## Tests
```bash
go test ./internal/... -v -timeout 60s
```

## Configuration layers
1. `config/default.toml` (bundled with the app)
2. `~/.config/liaotao/user.toml` (optional user overrides)
3. `config/project.toml` (optional local project overrides, not committed)
4. Environment variables with prefix `APP__` (example: `APP__APP__MODE=normal`)

## Security notes
- SQLite database file is restricted to owner permissions (`0600`) when possible.
- Provider API keys are stored encrypted at rest.
- A local master key is used for encryption and can be overridden via:
	- `LIAOTAO_MASTER_KEY`
	- `LIAOTAO_MASTER_KEY_FILE`
