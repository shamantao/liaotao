# Backlog — 聊濤 liaotao

> Last updated: 2026-04-04
> Status: active

---

## MVP v1 — Universal AI Chat

> Goal: a simple, multilingual chat interface with history, connecting to OpenAI-compatible APIs and MCP servers.

### v1.0 — Foundation

- [ ] **ARCH-01** Wails v3 project bootstrap (Go backend + HTML/CSS/JS frontend)
- [ ] **ARCH-02** Go ↔ JS binding layer: define all Wails bindings (chat, providers, settings, conversations)
- [ ] **ARCH-03** Layered TOML config system (`config/default.toml` → `~/.config/liaotao/user.toml` → env)
- [ ] **ARCH-04** Structured logging (slog, JSON file + console)
- [ ] **ARCH-05** Safe path manager with `allowed_roots` guard
- [ ] **ARCH-06** SQLite database init + schema migrations (pure Go, no CGO)

### v1.1 — Chat UI

- [ ] **UI-01** Two-tab layout: Chat | Settings
- [ ] **UI-02** Chat view: message list with user/assistant bubbles
- [ ] **UI-03** Text input area with Send button + Enter key
- [ ] **UI-04** Streaming display (word-by-word via Wails events)
- [ ] **UI-05** Stop generation button (cancel Go-side HTTP request)
- [ ] **UI-06** Markdown rendering (bold, italic, links, headings, lists, tables, blockquotes)
- [ ] **UI-07** Code block syntax highlighting (Prism.js, vendored)
- [ ] **UI-08** LaTeX rendering (KaTeX, vendored)
- [ ] **UI-09** `<think>` tag support (collapsible reasoning blocks)
- [ ] **UI-10** Message actions: copy, edit + regenerate, delete
- [ ] **UI-11** Responsive / clean dark theme (ported from smOllama CSS)

### v1.2 — i18n (Internationalization)

- [ ] **I18N-01** i18n system: JSON translation files (`frontend/i18n/en.json`, `fr.json`, etc.)
- [ ] **I18N-02** Language selector in Settings (persisted in config)
- [ ] **I18N-03** English as default language, complete translation
- [ ] **I18N-04** French translation
- [ ] **I18N-05** All UI strings externalized (zero hardcoded text in HTML/JS)

### v1.3 — OpenAI-Compatible Provider

- [ ] **PROV-01** Go adapter interface: `ListModels()`, `StreamChat()`, `TestConnection()`, `SupportsTools()`
- [ ] **PROV-02** OpenAI-compatible adapter Go-side (`/v1/models`, `/v1/chat/completions`)
- [ ] **PROV-03** SSE streaming parser (`data: {...}\n\n`, `[DONE]` handling)
- [ ] **PROV-04** API key storage in Go config (never exposed to frontend)
- [ ] **PROV-05** "Test Connection" button with latency display
- [ ] **PROV-06** Error mapping: HTTP status → user-friendly message (404, 429, 500, 503)
- [ ] **PROV-07** Rate limit handling (429 → exponential backoff with jitter)
- [ ] **PROV-08** Pre-configured profiles: OpenRouter, Groq, Together AI, Mistral, OpenAI
- [ ] **PROV-09** Custom provider: user enters base URL + API key

### v1.4 — MCP Support

- [ ] **MCP-01** MCP client in Go: JSON-RPC protocol implementation
- [ ] **MCP-02** stdio transport: spawn MCP server process, communicate via stdin/stdout
- [ ] **MCP-03** HTTP/SSE transport: connect to remote MCP servers
- [ ] **MCP-04** Tool discovery (`tools/list`) on MCP server connect
- [ ] **MCP-05** Tool-call loop: detect model tool calls → execute → re-inject result → continue
- [ ] **MCP-06** Built-in tools (no external server needed): `current_datetime`, `calculator`
- [ ] **MCP-07** MCP server management in Settings: add/remove/enable/disable servers
- [ ] **MCP-08** Tool execution status in UI (pending → running → done/error)
- [ ] **MCP-09** Tool call result display in chat (collapsible, formatted)

### v1.5 — Model & Provider Management

- [ ] **MOD-01** Unified model selector: grouped by provider, with search/filter
- [ ] **MOD-02** Model selection per conversation (saved in DB)
- [ ] **MOD-03** Provider status indicator (connected ✓ / disconnected ✗)
- [ ] **MOD-04** "Last used" models pinned at top
- [ ] **MOD-05** Lazy model fetch (on dropdown open, cached per session)
- [ ] **MOD-06** Temperature + max tokens controls per conversation
- [ ] **MOD-07** System prompt per conversation

### v1.6 — Conversation History

- [ ] **CONV-01** SQLite schema: `conversations` (id, title, provider_id, model, created_at, updated_at)
- [ ] **CONV-02** SQLite schema: `messages` (id, conversation_id, role, content, tool_calls, token_stats, created_at)
- [ ] **CONV-03** Sidebar conversation list, grouped by date (Today, Yesterday, This Week, Older)
- [ ] **CONV-04** New conversation button
- [ ] **CONV-05** Rename conversation (auto-generated title from first message, editable)
- [ ] **CONV-06** Delete conversation (with confirmation)
- [ ] **CONV-07** Search conversations by title
- [ ] **CONV-08** Token stats per message (tokens in/out, speed, duration)

### v1.7 — Settings Tab

- [ ] **SET-01** Settings tab layout: General | Providers | MCP Servers
- [ ] **SET-02** General: language, theme (dark only for v1), default system prompt
- [ ] **SET-03** Providers: list, add, edit, remove, enable/disable, test, reorder
- [ ] **SET-04** MCP Servers: list, add, edit, remove, enable/disable, show available tools
- [ ] **SET-05** Import/Export configuration as TOML
- [ ] **SET-06** About section: version, links, credits

### v1.8 — Keyboard Shortcuts

- [ ] **KEY-01** `Ctrl/Cmd+K` → New chat
- [ ] **KEY-02** `Ctrl/Cmd+/` → Focus input
- [ ] **KEY-03** `Ctrl/Cmd+B` → Toggle sidebar
- [ ] **KEY-04** `Escape` → Close settings / stop generation
- [ ] **KEY-05** Shortcut cheat sheet in About

### v1.9 — Build & Distribution

- [ ] **BUILD-01** macOS build (Apple Silicon + Intel universal)
- [ ] **BUILD-02** Windows build (x64 installer + portable)
- [ ] **BUILD-03** Linux build (x64, AppImage)
- [ ] **BUILD-04** Cross-platform CI (GitHub Actions)
- [ ] **BUILD-05** Auto-version from git tag

---

## MVP v2 — Projects, Plugins & Attachments

> Goal: organize conversations by project, attach files, define directives, extend via plugins.

### v2.1 — Plugin System

- [ ] **PLUG-01** Plugin hook architecture (frontend-side event bus)
- [ ] **PLUG-02** Hook points: `beforeSend`, `afterReceive`, `onFileUpload`, `renderTool`, `onSaveConv`
- [ ] **PLUG-03** Plugin manager in Settings: list installed, enable/disable
- [ ] **PLUG-04** Plugin loading: JS files from `plugins/` directory
- [ ] **PLUG-05** Plugin template + documentation for third-party authors
- [ ] **PLUG-06** Built-in plugin: TTS (Web Speech API)
- [ ] **PLUG-07** Built-in plugin: prompt library (saved prompt templates)
- [ ] **PLUG-08** Built-in plugin: export conversation to Markdown/PDF

### v2.2 — File Attachments

- [ ] **FILE-01** Drag & drop file upload per conversation
- [ ] **FILE-02** File storage: `data/attachments/<conversation_id>/`
- [ ] **FILE-03** Supported formats: `.txt`, `.md`, `.pdf`, `.json`, `.csv`, `.py`, `.go`, `.js`, `.ts`
- [ ] **FILE-04** File preview in chat (inline for text, icon for binary)
- [ ] **FILE-05** File content injected into context (chunked if large)
- [ ] **FILE-06** Attachment list per conversation in sidebar
- [ ] **FILE-07** Delete attachment (move to trash, not hard delete)

### v2.3 — RAG (Retrieval-Augmented Generation)

- [ ] **RAG-01** Embedding pipeline: chunk files → embed via provider `/v1/embeddings` or Ollama `/api/embed`
- [ ] **RAG-02** Vector storage in SQLite (float32 BLOBs + cosine similarity in Go)
- [ ] **RAG-03** On query: embed query → top-K similar chunks → inject into system prompt
- [ ] **RAG-04** RAG scope: per-conversation attachments or per-project knowledge base
- [ ] **RAG-05** Embedding model selector (separate from chat model)
- [ ] **RAG-06** RAG status indicator in UI (indexing → ready, chunk count)

### v2.4 — Project Management

- [ ] **PROJ-01** Project entity in DB: id, name, description, created_at
- [ ] **PROJ-02** Conversations belong to a project (or "Unsorted" default project)
- [ ] **PROJ-03** Project selector in sidebar (filter conversations by project)
- [ ] **PROJ-04** Create / rename / archive project
- [ ] **PROJ-05** Project-scoped knowledge base (shared attachments across conversations)
- [ ] **PROJ-06** Project dashboard: conversation count, total tokens, file count

### v2.5 — Directives (System Prompts)

- [ ] **DIR-01** Global directives: default system prompt applied to all new conversations
- [ ] **DIR-02** Per-project directives: system prompt prepended for all conversations in a project
- [ ] **DIR-03** Per-conversation directives: override or append to project/global
- [ ] **DIR-04** Directive editor with preview (Markdown rendered)
- [ ] **DIR-05** Directive merge order: global → project → conversation (concatenated)
- [ ] **DIR-06** Directive library: save/load reusable directive templates
- [ ] **DIR-07** Directive variables: `{{project_name}}`, `{{date}}`, `{{language}}` auto-replaced

### v2.6 — Conversation Enhancements

- [ ] **CONV2-01** Full-text search across message content (SQLite FTS5)
- [ ] **CONV2-02** Conversation tags (user-defined, filter in sidebar)
- [ ] **CONV2-03** Export conversation as JSON / Markdown
- [ ] **CONV2-04** Import conversations from JSON (+ smOllama IndexedDB format)
- [ ] **CONV2-05** Conversation size indicator (approx. token count)
- [ ] **CONV2-06** Pin / favorite conversations

### v2.7 — Auto-Update

- [ ] **UPD-01** Version check on startup (fetch latest release from GitHub API)
- [ ] **UPD-02** Update notification in UI (non-blocking banner, dismissible)
- [ ] **UPD-03** One-click download of new binary (platform-aware)
- [ ] **UPD-04** Release signature verification (checksum validation)
- [ ] **UPD-05** "Check for updates" button in Settings > About

---

## Future (Post v2 — Ideas)

- [ ] Ollama-native provider adapter (NDJSON streaming, `/api/tags`)
- [ ] Voice input (Web Speech Recognition API)
- [ ] Multi-window: detach conversation into separate window
- [ ] Conversation branching (fork at any message)
- [ ] Image generation support (DALL-E, Stable Diffusion APIs)
- [ ] Agent mode: multi-step autonomous task execution
- [ ] Local model download manager (Ollama pull integration)
- [ ] Themes (light theme, custom CSS)

---

*End of backlog — 聊濤 liaotao*
