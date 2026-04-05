# Architecture — 聊濤 liaotao

> Last updated: 2026-04-05  
> Version: 0.1.x (v1.2)  
> Stack: Wails v3 · Go 1.25 · SQLite · Vanilla JS (ES Modules)

---

## 1. Overview

**liaotao** is a desktop AI chat application built with Wails v3. It connects to any
OpenAI-compatible API provider (OpenRouter, Groq, Mistral, Together AI, custom) and to
local Ollama instances. The entire UI runs in a native WebView (no Electron, no Node
runtime) with a Go backend exposed via Wails bindings.

Core objectives:
- Zero cloud dependency for the app itself (providers are user-configured)
- Full streaming (SSE for OpenAI-compat, NDJSON for Ollama)
- Persistent history in local SQLite
- Files under 400 lines; no runtime frameworks in the frontend

---

## 2. Technology Stack

| Layer        | Technology                          | Notes                            |
|--------------|-------------------------------------|----------------------------------|
| Desktop shell| Wails v3.0.0-alpha.74               | macOS WebKit, Windows WebView2   |
| Backend      | Go 1.25                             | darwin/arm64 primary target      |
| Database     | SQLite via `modernc.org/sqlite`     | pure Go, no CGO                  |
| Config       | TOML via `BurntSushi/toml`          | layered: default → user → env    |
| Frontend     | Vanilla HTML/CSS/JS (ES Modules)    | no npm, no bundler               |
| Markdown     | Custom JS renderer                  | vendored in `frontend/js/`       |
| Syntax HL    | Prism.js (vendored)                 | `frontend/prism/`                |
| LaTeX        | KaTeX (vendored)                    | `frontend/katex/`                |

---

## 3. Project Structure

```
liaotao/
├── main.go                    # Wails v3 app entry point
├── config/
│   └── default.toml           # default config (bundled)
├── data/
│   └── liaotao.db             # SQLite database (not committed)
├── docs/                      # technical documentation
├── frontend/
│   ├── index.html             # single-page shell
│   ├── css/
│   │   ├── base.css           # global variables, reset, layout
│   │   ├── chat.css           # chat bubbles, message actions
│   │   └── settings.css       # settings panel, provider cards
│   ├── js/
│   │   └── app.js             # all frontend logic (ES module)
│   ├── katex/                 # KaTeX vendored assets
│   └── prism/                 # Prism.js vendored assets
├── internal/
│   ├── bindings/
│   │   ├── service.go         # Wails service struct (bindings root)
│   │   ├── chat.go            # ListModels, SendMessage, CancelGeneration, streaming
│   │   ├── providers.go       # provider CRUD (SQLite)
│   │   ├── provider_openai.go # OpenAI-compatible adapter + Ollama adapter
│   │   └── connection.go      # TestConnection, latency measurement
│   ├── config/
│   │   └── config.go          # layered TOML config loader
│   ├── db/
│   │   └── db.go              # SQLite init, schema migrations
│   ├── logger/
│   │   └── logger.go          # slog-based JSON + console logger
│   └── paths/
│       └── paths.go           # safe path manager with allowed_roots
└── scripts/                   # baseline QA scripts (integrity, secrets, healthcheck)
```

---

## 4. Wails v3 Binding Architecture

### Go → JS binding rules

All public methods on `*bindings.Service` are automatically exposed to the frontend by
Wails v3. The FQN format for `ByName` calls is:

```
"liaotao/internal/bindings.Service.MethodName"
```

### JS-side call convention (`app.js` → `bridge.callService`)

```js
// Wails v3 runtime.js is an ES Module — MUST be loaded with type="module"
// <script type="module" src="/wails/runtime.js"></script>

// Call a Go binding:
// - pass payload only when the Go method has a non-context parameter
window.wails.Call.ByName(fqn)               // no payload (ctx-only method)
window.wails.Call.ByName(fqn, payload)       // with payload (ctx + one arg method)
```

**Critical**: methods that only take `context.Context` (e.g. `ListProviderProfiles`,
`ListModels`) must be called **without** a payload argument. Passing `{}` causes a
runtime error (`expects 1 arguments, got 2`).

### Event flow for streaming

```
Go: app.Get().EmitEvent("chat:token", token)
JS: window.wails.Events.On("chat:token", handler)
```

---

## 5. Database Schema (v1.x)

```sql
-- Provider table
CREATE TABLE providers (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL UNIQUE,
  type        TEXT NOT NULL,       -- "openai_compat" | "ollama"
  base_url    TEXT NOT NULL,
  api_key     TEXT,                -- stored locally, never sent to frontend
  description TEXT,
  use_in_rag  INTEGER DEFAULT 0,
  active      INTEGER DEFAULT 1,
  created_at  TEXT,
  updated_at  TEXT
);

-- Conversations and messages (v1.6+)
-- conversations: id, title, provider_id, model, created_at, updated_at
-- messages: id, conversation_id, role, content, tool_calls, token_stats, created_at
```

---

## 6. Config Layers (highest wins)

1. `config/default.toml` — bundled defaults
2. `~/.config/liaotao/user.toml` — user overrides (not committed)
3. Environment variables prefixed `LIAOTAO__` (not yet implemented)

---

## 7. Development Flow

```bash
# Run in dev mode (hot-reload frontend)
wails3 dev

# Run all tests
go test ./...

# Build for current platform
wails3 build
```

- Keep source files under **400 lines** of code (excluding comments/blanks).
- All code comments and docstrings must be in **English**.
- Only commit `config/default.toml`, never `user.toml` or `data/*.db`.

---

## 8. Testing Strategy

- Unit tests in `internal/bindings/` cover: provider CRUD, OpenAI adapter, connection test.
- No E2E tests in v1.x (deferred — Wails v3 webdriver support is alpha).
- Run `go test ./...` before any commit; all tests must pass.

---

## 9. Security Notes

- API keys are stored in SQLite on disk; never exposed to the JS frontend.
- `paths.go` enforces `allowed_roots` — no write outside declared roots.
- See `docs/SECURITY.md` for vulnerability reporting.
