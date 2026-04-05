# 聊濤 liaotao

> Desktop AI chat — Wails v3 · Go · SQLite · Vanilla JS

**liaotao** is a native desktop application for chatting with AI models. It connects to
any OpenAI-compatible API (OpenRouter, Groq, Mistral, Together AI, OpenAI, custom) and
to local [Ollama](https://ollama.com) instances. No cloud dependency, no telemetry —
everything runs locally except the model calls you explicitly configure.

---

## Features (v1.2)

### Chat
- Real-time **streaming** responses (word-by-word display)
- **Markdown rendering**: bold, italic, headings, lists, tables, blockquotes, links
- **Code blocks** with syntax highlighting (Prism.js, vendored, offline)
- **LaTeX** math rendering (KaTeX, vendored, offline)
- **`<think>` tag** support: collapsible reasoning blocks (DeepSeek R1, QwQ, etc.)
- **Message actions**: copy · edit + regenerate · delete
- **Stop generation** button (cancels the in-flight HTTP request server-side)

### Providers
- **OpenAI-compatible** adapter: `/v1/models` + `/v1/chat/completions` SSE streaming
- **Ollama** adapter: auto-detected on `localhost:11434`, NDJSON streaming
- **Pre-configured profiles**: OpenRouter, Groq, Together AI, Mistral, OpenAI
- **Custom provider**: any base URL + optional API key
- **Test Connection** button with latency display
- **CRUD** persisted in SQLite: add, edit, delete, enable/disable providers
- Rate-limit handling: 429 → exponential backoff with jitter (3 attempts)
- Error mapping: HTTP 404 / 429 / 500 / 503 → user-friendly message

### Chat UI
- Two-tab layout: **Chat** | **Settings**
- Dark theme (Blue `#1E3D59` · Green `#3E8E7E` · Black `#1B1B1B` · Beige `#E9E3D5`)
- Conversation persisted in SQLite across sessions

---

## Requirements

| Dependency | Version  | Install                                          |
|------------|----------|--------------------------------------------------|
| Go         | ≥ 1.22   | `brew install go` or https://go.dev/dl/          |
| Wails CLI  | v3 alpha | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |
| macOS      | ≥ 13     | WebKit 2 required                                |

For Windows / Linux prerequisites: https://wails.io/docs/gettingstarted/installation

---

## Getting Started

```bash
# Clone and run in dev mode
git clone <repo>
cd liaotao
wails3 dev
```

On first run, the app creates:
- `data/liaotao.db` — local SQLite database
- `logs/` — structured JSON logs
- `~/.config/liaotao/user.toml` — user config overrides (optional)

---

## Configuration

Edit `config/default.toml` for defaults.  
User overrides go in `~/.config/liaotao/user.toml` (never committed).

Key settings:

```toml
[app]
mode = "debug"   # debug | normal

[database]
path = "/path/to/data/liaotao.db"

[logger]
level = "debug"  # trace | debug | info | warn | error
```

---

## Running Tests

```bash
go test ./...
```

All 14 unit tests must pass before committing.

---

## Project Layout

```
liaotao/
├── main.go                    # Wails v3 entry point
├── config/default.toml        # bundled default config
├── data/                      # SQLite DB (not committed)
├── docs/                      # technical documentation
├── frontend/
│   ├── index.html
│   ├── css/                   # base.css · chat.css · settings.css
│   ├── js/app.js              # all frontend logic (ES module)
│   ├── katex/                 # vendored KaTeX
│   └── prism/                 # vendored Prism.js
├── internal/
│   ├── bindings/              # Wails-exposed Go services
│   ├── config/                # layered TOML config
│   ├── db/                    # SQLite init + migrations
│   ├── logger/                # slog JSON + console
│   └── paths/                 # safe path manager
└── scripts/                   # QA baseline scripts
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical architecture.

---

## Versioning

Follows SemVer. See [docs/CHANGELOG.md](docs/CHANGELOG.md) for release history.
