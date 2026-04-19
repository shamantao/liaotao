# PRD — 聊濤 liáo tāo (liaotao)

| Field     | Value                                              |
|-----------|----------------------------------------------------|
| Date      | 2026-04-10                                         |
| Status    | Working draft (aligned with implemented v0.1.7)    |
| Author    | shamantao                                               |
| Stack     | Go (Wails v3) + Svelte frontend (compiled via Vite) |
| Version   | 0.1.7                                              |

---

## 1. Problem Statement

The AI landscape is fragmented across dozens of providers and protocols: local models via Ollama or LM Studio, cloud APIs (OpenAI, Groq, Mistral, OpenRouter), and tool-calling ecosystems via MCP (Model Context Protocol). Users who work with multiple AI sources daily must juggle separate interfaces, deal with CORS restrictions in browser-only apps, and accept heavyweight Electron/Docker solutions (Open WebUI, LibreChat, LobeChat) that consume hundreds of megabytes.

There is **no ultra-lightweight desktop application** that can connect to all these sources **simultaneously** from a single, native-speed interface — with full MCP tool-calling support and zero browser sandbox restrictions.

### Specific pain points

| Pain | Root Cause |
|------|------------|
| Browser apps can't call cloud APIs | CORS restrictions block cross-origin requests |
| Browser apps can't spawn MCP stdio processes | Browser sandbox prevents `child_process` |
| Heavyweight alternatives require Docker or Electron | 100-500 MB install, 200+ MB RAM idle |
| Single-provider lock-in | Each tool supports only one API format |
| API keys exposed in browser localStorage | No secure storage in pure web apps |

### Origin and lineage

liaotao is the evolution of **smOllama**, a lightweight vanilla JS chatbot for Ollama (~3 KB app code). The smOllama v2 analysis (see `smOllama/docs/PRD-smOllama-v2.md`) identified that a pure browser approach cannot solve CORS, MCP stdio, and secure key storage. The decision was made to wrap the frontend in a **Wails v3 (Go) shell**. The frontend was initially built as vanilla HTML/CSS/JS, then migrated to **Svelte** for component isolation, scoped CSS, and reactive state management — while remaining compiled to standard JS with zero runtime overhead.

---

## 2. Product Vision

**聊濤 liáo tāo** is an ultra-lightweight desktop AI chat application (~8-12 MB binary) that connects to **any AI source** — local models, cloud APIs, and MCP tool servers — simultaneously from a single window. Built with Go (backend) and Svelte (frontend, compiled to vanilla JS via Vite), it starts in under 500 ms, requires zero configuration for basic use, and lets power users add providers, tools, and plugins without touching code. It is the Swiss Army knife for anyone who works with multiple AI sources daily.

**Name meaning**: 聊 (liáo) = to chat; 濤 (tāo) = waves/flow — "chatting waves", a conversational flow across AI sources.

---

## 3. Goals

- **G1**: Connect to 4+ provider types simultaneously (Ollama, OpenAI-compatible, MCP servers, Internet AI APIs).
- **G2**: Ship as a single native binary (~8-12 MB) for macOS, Windows, and Linux.
- **G3**: Start to interactive in under 500 ms.
- **G4**: Support MCP tool-calling (both stdio and HTTP/SSE transport) — no separate proxy needed.
- **G5**: Keep the frontend under 150 KB of compiled JS (excluding vendored KaTeX/Prism).
- **G6**: Allow adding a new provider adapter in under 80 lines of Go code.
- **G7**: Provide a migration path from smOllama (import existing conversations from IndexedDB export).
- **G8**: Route requests across multiple providers automatically, respecting free-tier quotas and user-defined priority order — maximizing free access before falling back to paid accounts.

---

## 4. Non-Goals (Scope Boundaries)

- **NG1**: No cloud sync, no multi-user, no authentication.
- **NG2**: No model training, fine-tuning, or model download management.
- **NG3**: No replacement for full platforms (Open WebUI, LibreChat) — liaotao stays minimal.
- **NG4**: No mobile app — desktop only (macOS, Windows, Linux).
- **NG5**: No plugin marketplace or auto-update system in v1.
- **NG6**: No bundled AI model — users bring their own providers.
- **NG7**: No Electron, no React/Vue/Angular. Frontend uses Svelte (compiled, not a runtime framework) with a minimal Vite build step.

---

## 5. Stakeholders & Personas

| Persona              | Description                                                   | Primary Need                              |
|----------------------|---------------------------------------------------------------|-------------------------------------------|
| **Local AI Tinkerer**| Runs Ollama + LM Studio, tests many models daily              | Switch between local providers instantly   |
| **Cloud API User**   | Uses OpenAI/Groq/OpenRouter, wants a clean minimal UI         | Connect to cloud API with key, no SDK      |
| **Developer**        | Integrates AI into workflows, relies on MCP tools             | Tool-calling + function execution from chat|
| **Minimalist**       | Values portability, hates bloated apps                        | Single binary, <1s startup, tiny footprint |
| **smOllama Migrant** | Existing smOllama user, wants more providers                  | Seamless transition, import conversations  |

---

## 6. Functional Requirements

### 6.1 Multi-Provider Architecture (Go Backend)

- Description: Go-side provider registry with runtime-swappable adapters. Each adapter normalizes a different AI API protocol into a unified internal message format. The frontend communicates with providers exclusively through Wails bindings (Go functions exposed to JS).
- Acceptance criteria:
  - [ ] Provider adapter interface in Go: `ListModels()`, `StreamChat()`, `TestConnection()`, `SupportsTools()`.
  - [ ] Minimum 4 built-in adapters: Ollama, OpenAI-compatible, MCP, Generic HTTP.
  - [ ] Provider registry allows adding/removing providers at runtime.
  - [ ] All HTTP calls happen Go-side (no CORS restriction).
  - [ ] Streaming responses pushed to frontend via Wails events.
  - [ ] Adding a new adapter = one Go file implementing the interface.

### 6.2 Ollama Provider

- Description: Native Ollama API adapter. Preserves all smOllama v1 functionality.
- Acceptance criteria:
  - [ ] `GET /api/tags` for model listing.
  - [ ] `POST /api/chat` with NDJSON streaming.
  - [ ] Custom parameters passthrough (temperature, context length, `num_ctx`, etc.).
  - [ ] Robust stream parsing (buffered line reader).
  - [ ] Auto-detect local Ollama on `localhost:11434` at startup.

### 6.3 OpenAI-Compatible Provider

- Description: Adapter for any server exposing the OpenAI API format. Covers LM Studio, LocalAI, Jan, vLLM, llama.cpp, OpenRouter, Groq, Together AI, Mistral, Cohere, and OpenAI itself.
- Acceptance criteria:
  - [ ] `GET /v1/models` for model listing.
  - [ ] `POST /v1/chat/completions` with SSE streaming.
  - [ ] API key stored securely in Go-side config (not in browser).
  - [ ] Correct SSE parsing (`data: {...}\n\n`, handling `[DONE]`).
  - [ ] Support for `tool_choice` and function calling for models that support it.
  - [ ] Non-streaming fallback for servers that don't support SSE.

### 6.4 MCP (Model Context Protocol) Support

- Description: Full MCP client in Go — no proxy needed. Supports both MCP transport protocols.
- Acceptance criteria:
  - **stdio transport**:
    - [ ] Go spawns MCP server process (`os/exec.Command`).
    - [ ] JSON-RPC communication over stdin/stdout.
    - [ ] Process lifecycle managed (start on demand, graceful shutdown).
  - **HTTP/SSE transport**:
    - [ ] Connect to remote MCP servers via HTTP.
    - [ ] SSE event stream for server notifications.
  - **Tool loop**:
    - [ ] Discover tools from MCP server (`tools/list`).
    - [ ] Inject available tools into chat request (for models that support function calling).
    - [ ] Execute tool calls returned by the model.
    - [ ] Re-inject tool results and continue generation.
    - [ ] Display tool execution status in the UI (pending → running → done/error).
  - **Built-in tools** (no external MCP server needed):
    - [ ] `current_datetime` — returns current date/time.
    - [ ] `calculator` — evaluates mathematical expressions.
    - [ ] `read_file` — reads a local file (with path validation via `allowed_roots`).
    - [ ] `web_fetch` — fetches a URL and returns text content.

### 6.5 Internet AI API Profiles

- Description: Pre-configured connection profiles for popular cloud AI services.
- Acceptance criteria:
  - [ ] Pre-configured profiles: OpenRouter, Groq, Together AI, Mistral, Cohere.
  - [ ] Each profile sets the correct base URL, auth header format, and known quirks.
  - [ ] Rate limit handling (429 → exponential backoff with jitter).
  - [ ] Display remaining quota if API header exposes it.
  - [ ] Brief description + link to get API key for each service.

### 6.6 Unified Model Selector (Frontend)

- Description: Grouped, searchable model dropdown spanning all configured providers.
- Acceptance criteria:
  - [ ] Models grouped by provider with visual distinction (color badges).
  - [ ] Search/filter input across all providers.
  - [ ] "Last used" models pinned at top.
  - [ ] Provider status indicator (connected ✓ / disconnected ✗).
  - [ ] Lazy fetch: model lists fetched on dropdown open, cached per session.

### 6.7 Chat Interface (Frontend — smOllama Heritage)

- Description: Carry forward the proven smOllama UI with enhancements.
- Acceptance criteria:
  - [ ] Real-time streaming display (word-by-word).
  - [ ] Markdown rendering (bold, italic, links, headings, lists, tables, blockquotes).
  - [ ] LaTeX rendering via KaTeX (display and inline).
  - [ ] Code block syntax highlighting via Prism.
  - [ ] `<think>` tag support (collapsible reasoning blocks).
  - [ ] Message actions: copy, edit+regenerate, delete, view raw.
  - [ ] Token statistics per message (tokens in/out, tokens/sec, duration).
  - [ ] System prompt configuration per provider or per conversation.
  - [ ] Temperature and context length controls.
  - [ ] Stop generation button (cancels Go-side HTTP request).

### 6.8 Conversation Management

- Description: SQLite-based conversation storage (replacing IndexedDB, since we now have a Go backend).
- Acceptance criteria:
  - [ ] SQLite database in `data/liaotao.db`.
  - [ ] Conversations table: id, title, provider_id, model, created_at, updated_at.
  - [ ] Messages table: id, conversation_id, role, content, tool_calls, token_stats, created_at.
  - [ ] Tags table: id, name, color.
  - [ ] Conversation-tags junction table.
  - [ ] Sidebar grouped by date (Today, Yesterday, This Week, etc.).
  - [ ] Sidebar includes a persistent search input at the top (above Projects and Conversations) to find conversations quickly.
  - [ ] Search filters by conversation title, message content (FTS), and project name.
  - [ ] `Escape` clears the search query and restores the full list.
  - [ ] Sidebar keeps two primary collapsible sections in fixed order: Projects, then Conversations.
  - [ ] Item actions use a unified contextual menu pattern (`...`) instead of always-visible button rows.
  - [ ] Compact density is the default sidebar display mode, with a user toggle for a comfortable mode.
  - [ ] Full-text search across message content (SQLite FTS5).
  - [ ] Export as Markdown or JSON.
  - [ ] Import from JSON (including smOllama IndexedDB export format).

### 6.9 Settings & Configuration

- Description: Layered TOML config (tao-init pattern) + GUI settings panel.
- Acceptance criteria:
  - [ ] Config layers: `config/default.toml` → `~/.config/liaotao/user.toml` → env vars.
  - [ ] Settings UI: General | Providers | Tools (MCP) | About.
  - [ ] Add/edit/remove providers from the UI.
  - [ ] Per-provider enable/disable toggle.
  - [ ] "Test Connection" button per provider.
  - [ ] Import/Export full configuration as TOML file.
  - [ ] API keys stored in Go-side config file, never exposed to frontend JS.

### 6.10 Keyboard Shortcuts

- Description: Productivity shortcuts for power users.
- Acceptance criteria:
  - [ ] `Ctrl/Cmd+K` — New chat.
  - [ ] `Ctrl/Cmd+/` — Focus input.
  - [ ] `Ctrl/Cmd+B` — Toggle sidebar.
  - [ ] `Escape` — Close settings / stop generation.
  - [ ] `Ctrl/Cmd+Shift+C` — Copy last assistant message.

### 6.11 Smart Router (Quota-Aware, Priority-Based Routing)

- Description: An intelligent routing layer that dispatches chat requests across configured providers according to user-defined priority and free-tier quota limits. The goal: maximize free AI access automatically, without the user having to think about it. The user sets an ordered list of providers ("try Groq first, then OpenRouter, then my paid OpenAI account") and a token budget per provider per period. The router handles the rest silently.
- Acceptance criteria:
  - [ ] Provider registry with user-configurable priority order (drag-to-reorder in Settings).
  - [ ] Per-provider quota configuration: daily or monthly token limit + reset schedule (midnight local time, or 1st of month).
  - [ ] Token usage tracked locally in SQLite (`provider_quota_usage` table: provider_id, period_start, tokens_used).
  - [ ] On each request: router selects the highest-priority provider with quota remaining.
  - [ ] Automatic fallback: when a provider's quota is reached, the next provider in priority order is used transparently.
  - [ ] User notified when all configured quotas are exhausted — clear message with a link to Settings.
  - [ ] Quota status visible in Settings (remaining tokens per provider for the current period).
  - [ ] Manual override: user can pin a specific provider/model for the current conversation, bypassing the router.
  - [ ] Quota counters reset automatically based on each provider's configured schedule.

---

## 6.12 UX/UI Design System & Theming

- Description: liaotao follows a modern, clean dark mode design language derived from the **Aurica Circular** brand palette. The UI must feel contemporary (2025+ standards), with flat opaque surfaces, consistent spacing, and no decorative gradients or blur effects. The design system is architected as a **pluggable theme layer** so the community can create and distribute custom themes without modifying the core application.

### 6.12.1 Brand Palette — Aurica Circular

The core palette is derived from the Aurica Circular brand. These are the **seed colors**; the design system extends them into a full token set.

| Role | Hex | Name | Usage |
|------|-----|------|-------|
| Primary | `#1E3D59` | Midnight Blue | Headers, active elements, primary actions |
| Accent | `#3E8E7E` | Jade Green | Interactive highlights, links, success states |
| Background | `#1B1B1B` | Ink Black | App background, deepest layer |
| Text | `#E9E3D5` | Warm Beige | Primary text color on dark surfaces |

Extended palette (derived):

| Role | Hex | Derivation | Usage |
|------|-----|------------|-------|
| Surface | `#222222` | Lighten Ink Black +4% | Cards, sidebar, panels |
| Surface Elevated | `#2A2A2A` | Lighten Ink Black +7% | Menus, dropdowns, modals |
| Border | `#333333` | Lighten Ink Black +10% | Subtle separators |
| Text Secondary | `#A8A29E` | Desaturate Warm Beige | Labels, metadata, hints |
| Danger | `#EF4444` | Standard red | Delete, errors, destructive actions |
| Warning | `#F59E0B` | Standard amber | Caution states |
| Accent Hover | `#4DA897` | Lighten Jade +10% | Hover state for accent elements |
| Primary Hover | `#275177` | Lighten Midnight +12% | Hover state for primary elements |

### 6.12.2 Design Principles

1. **Flat & Opaque**: No decorative gradients on surfaces. No `backdrop-filter: blur()`. Backgrounds are solid colors. Depth is expressed through subtle border + elevation (shadow), not transparency.
2. **Consistent Spacing Scale**: All spacing uses a **4px base grid** (`0.25rem` increments): `4, 8, 12, 16, 24, 32, 48, 64`. No arbitrary values like `0.42rem` or `0.55rem`.
3. **Typography**: System font stack (`-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', sans-serif`). Monospace: `'SF Mono', 'Fira Code', 'Cascadia Code', monospace`.
4. **Icon System**: SVG icons only (preferably from a single icon set: Lucide, Phosphor, or custom). No Unicode characters as button icons — they render inconsistently across platforms and are not accessible.
5. **Energy Efficiency**: Dark mode is mandatory. True black (`#1B1B1B`) as base background for OLED energy savings. Accent colors used sparingly.
6. **Minimal Chrome**: The interface fades away to let the conversation be the focus. Controls appear on hover or in contextual menus, not as permanent clutter.
7. **Responsive**: Single breakpoint at 900px. Mobile layout stacks sidebar as full-screen overlay.

### 6.12.3 Design Token Architecture

CSS custom properties are organized in **three layers** to enable community theming:

| Layer | Prefix | Example | Who defines it |
|-------|--------|---------|----------------|
| **Primitives** | `--color-*`, `--space-*`, `--radius-*` | `--color-jade-500: #3E8E7E` | Theme author |
| **Semantic** | `--surface-*`, `--text-*`, `--border-*`, `--accent-*` | `--surface-primary: var(--color-gray-900)` | Theme author |
| **Component** | `--bubble-*`, `--sidebar-*`, `--composer-*` | `--bubble-bg-user: var(--surface-elevated)` | Core app (stable API) |

**Rule**: Svelte component `<style>` blocks must ONLY reference **semantic** and **component** tokens via `var()` — never raw color values. Theme files define primitives and map them to semantic tokens. This guarantees that swapping a theme file recolors the entire app. Svelte's scoped CSS eliminates class-name collisions across components.

### 6.12.4 Theme Plugin Contract

A theme plugin is a standard liaotao plugin (see §6.12.5) that registers a CSS file overriding semantic tokens.

```js
// Example: plugins/themes/solarized-dark.js
window.liaotaoPlugins.registerTheme({
  id: "solarized-dark",
  name: "Solarized Dark",
  author: "Community",
  version: "1.0.0",
  css: "./themes/solarized-dark.css",   // relative to plugin dir
});
```

The theme CSS file MUST only override `:root` custom properties (primitives + semantic tokens). It MUST NOT redefine component layout, grid structure, or class selectors. This ensures forward compatibility when the core app evolves.

```css
/* themes/solarized-dark.css — VALID theme file */
:root {
  --color-base-bg: #002b36;
  --color-base-text: #839496;
  --surface-primary: var(--color-base-bg);
  --text-primary: var(--color-base-text);
  /* ... semantic overrides only ... */
}
```

### 6.12.5 Theme Registration & Loading

- Themes are discovered via the existing plugin system (`plugins/` directory).
- A theme plugin calls `window.liaotaoPlugins.registerTheme()` (new API surface).
- `theme-loader.js` injects the theme CSS as a `<link>` element with `id="liaotao-theme"`.
- Switching themes replaces the `href` of this link element (no page reload).
- The active theme ID is persisted in settings (`settings.theme`).
- The built-in default dark theme is always available as fallback.
- The Settings > General > Theme dropdown lists all registered themes.

### 6.12.6 Acceptance Criteria

- [ ] All CSS colors are expressed as `var()` references — zero hardcoded hex values in Svelte component `<style>` blocks.
- [ ] The default dark theme renders identically to the current design intent (Aurica Circular palette).
- [ ] A community theme can recolor the entire app by providing a single CSS file of ≤ 100 lines.
- [ ] The Settings > Theme selector lists all registered themes and applies them without reload.
- [ ] Contrast ratio ≥ 4.5:1 (WCAG AA) for all text on its background in the default theme.
- [ ] No Unicode characters used as button icons — all icons are SVG.
- [ ] No `backdrop-filter`, no decorative gradients on surfaces.
- [ ] All spacing values align to the 4px grid.

---

## 7. Non-Functional Requirements

| Category       | Requirement                                                     | Metric / Target                           |
|----------------|-----------------------------------------------------------------|-------------------------------------------|
| Performance    | App startup to interactive                                      | < 500 ms cold start                       |
| Performance    | Time to first streamed token                                    | < 100 ms Go overhead (provider latency excluded) |
| Performance    | Conversation with 5,000+ messages                               | No perceptible scroll/render lag           |
| Security       | API keys never sent to frontend                                 | Keys exist only in Go memory + config file |
| Security       | MCP tool execution sandboxed                                    | File tools respect `allowed_roots`, no arbitrary exec |
| Security       | XSS prevention                                                  | All user input sanitized before DOM insert |
| Usability      | New provider configured and tested                              | < 2 minutes                               |
| Usability      | Works with zero config                                          | Auto-detect localhost Ollama on first launch |
| Compatibility  | macOS (Intel + Apple Silicon)                                   | Native binary via `GOOS=darwin`            |
| Compatibility  | Windows 10/11 (x64)                                             | Native binary via `GOOS=windows`           |
| Compatibility  | Linux (x64, AppImage)                                           | Native binary via `GOOS=linux`             |
| Size           | Binary size                                                     | < 15 MB                                   |
| Size           | RAM usage idle                                                  | < 50 MB                                   |
| Size           | Frontend JS (compiled Svelte output)                            | < 150 KB                                  |

---

## 8. Engineering Rules

### 8.1 Code Organization

- Naming must be explicit and consistent (files, modules, commands).
- Keep source files below 400 lines when feasible.
- Modules have single responsibilities.
- Comments and technical documentation in code must be in English.
- Go code follows standard `gofmt` formatting and [Effective Go](https://go.dev/doc/effective_go) conventions.
- Frontend: Svelte components compiled to vanilla JS via Vite. Each component owns its markup, logic, and scoped styles. No runtime framework shipped to the browser.

### 8.2 Maintainability

- Semantic Versioning: `MAJOR.MINOR.PATCH`.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`).
- Reference issue numbers in commits when applicable.
- `CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/) format.

### 8.3 Testing Strategy

- **Mandatory baseline** (tao-init scripts):
  - `scripts/test-integrity.sh` — structural validation.
  - `scripts/test-dependencies.sh` — dependency checks.
  - `scripts/check-secrets.sh` — anti-secrets scan.
  - `scripts/healthcheck.sh` — environment validation.
- **Unit tests**: Go provider adapters tested with `httptest` mock servers. `go test ./...`.
- **Integration tests**: End-to-end chat flow with a mock Ollama server.
- **Frontend tests**: Manual smoke test matrix documented in `tests/MANUAL-TESTS.md`.
- Store mock responses in `tests/fixtures/`. Never commit real API keys.

### 8.4 Architecture Constraints

- Config: `config/default.toml` → `~/.config/liaotao/user.toml` → env (`APP__*`).
- Database: `data/liaotao.db` (SQLite via `modernc.org/sqlite` — pure Go, no CGO).
- Log directory: `logs/`.
- Temp directory: `.tmp/`.
- Reports directory: `reports/`.
- All paths validated against `allowed_roots` before write.
- `debug` mode is the startup default.
- Frontend communicates with Go exclusively via Wails bindings — no direct HTTP calls.
- **Cloud-sync exclusion**: if the project lives on a cloud-synced volume, the root MUST contain a sync-exclusion file (`.syncignore`, `.stignore`, or equivalent for your provider) that excludes `.git/`, `data/*.db-shm`, `data/*.db-wal`, and `data/*.db-journal`. Cloud-sync tools that replicate these files while the app is running will corrupt the SQLite WAL and the Git index, causing silent startup failures. The `.gitignore` must also exclude sync-conflict files (`*_conflict_*`).

### 8.5 Security & Dependency Hygiene

- Never commit credentials, keys, tokens, or private keys.
- API keys stored in a dedicated `~/.config/liaotao/secrets.toml` file with restrictive file permissions (0600), never in the SQLite database or user.toml. (Implementation: DEBT-02 — pending migration from current SQLite storage.)
- Run `scripts/check-secrets.sh` before opening a pull request.
- MCP process spawning: only whitelisted commands, never raw shell execution.
- All `os/exec.Command` calls use explicit argv (no shell interpretation).
- Go dependencies: `go mod tidy` + audit with `govulncheck`.

### 8.6 Quality Automation

- `scripts/test-all.sh` runs all checks (integrity + deps + secrets + `go test ./...`).
- Go tests run without live Ollama/API instance (httptest mocks).
- Cross-compilation tested in CI for all three platforms.

### 8.7 Human-Friendly Error Pattern

All settings forms that call Go bindings must provide clear, field-level error feedback. The pattern is centralized in `frontend/js/errors.js` and **must be used** in every new form.

**Why it exists**: Wails v3 surfaces Go errors as a JSON envelope (`{"message":"...","cause":{},"kind":"RuntimeError"}`). The raw Go error message (e.g. `UNIQUE constraint failed: mcp_servers.name (2067)`) is not acceptable UI copy.

**The contract**:

| Function | Responsibility |
|---|---|
| `parseWailsError(err)` | Parse Wails JSON envelope → `{ message: string, field: string\|null }`. Maps known SQLite/Go patterns to human copy via `ERROR_PATTERNS` array. |
| `applyFieldError(inputEl, statusEl, message)` | Adds `.field-error` CSS class (red border + glow) to `inputEl`; writes `message` to `statusEl`. |
| `clearFieldError(inputEl, statusEl)` | Removes `.field-error`; clears `statusEl`. |

**Adding a new error mapping**: append an entry to `ERROR_PATTERNS` in `errors.js`:
```js
{
  test: (m) => /unique constraint failed: my_table\.my_col/i.test(m),
  message: "Human-readable message.",
  field: "html-input-id",  // null if no single field is responsible
},
```

**Usage in a form**:
```js
import { parseWailsError, applyFieldError, clearFieldError } from "./errors.js";

// On form open / field focus → clear previous error
clearFieldError(nameInput, statusEl);

// In catch block
} catch (err) {
  const { message, field } = parseWailsError(err);
  const fieldEl = field ? document.getElementById(field) : null;
  applyFieldError(fieldEl, statusEl, message);
}
```

**CSS classes** (defined in `settings.css`):
- `.form-status-msg` — error text below form actions, right-aligned, danger color.
- `.field-error` — red border + 2 px glow on the failing input.

---

## 9. Documentation Baseline

Each release must ship:

- `README.md` — Install, usage, quick start, provider setup examples.
- `docs/ARCHITECTURE.md` — Module map, data flow, provider adapter contract.
- `docs/CHANGELOG.md` — Keep a Changelog format.
- `docs/CONTRIBUTING.md` — Code style, commit rules, how to add a provider.
- `docs/SECURITY.md` — Vulnerability reporting, API key handling policy.
- `docs/PROVIDERS.md` — Setup guide for each supported provider.
- `LICENSE` — Explicit license.

---

## 10. Assumptions & Constraints

### Assumptions

- Users have at least one AI provider accessible (local Ollama or cloud API key).
- Go 1.22+ is available on the build machine (not required on end-user machine — compiled binary).
- Wails v3 is stable enough for production desktop apps.
- OpenAI-compatible API format remains the de facto standard for cloud AI.
- MCP protocol will stabilize and gain wider adoption.
- SQLite is sufficient for conversation storage (no concurrent multi-user).

### Constraints

- **No CGO if possible**: Prefer pure-Go SQLite (`modernc.org/sqlite`) to simplify cross-compilation.
- **Wails WebView**: Uses the system WebView (WebKit on macOS, WebView2 on Windows, WebKitGTK on Linux). UI must be tested across all three.
- **Minimal build step**: Frontend uses Svelte + Vite. `npm run build` compiles `.svelte` components to vanilla JS served from `frontend/dist/`. KaTeX and Prism remain vendored (not npm-managed).
- **Single user**: No concurrency on the database — one user, one window.
- **File size**: Vendored KaTeX (~500 KB) and Prism (~100 KB) are the largest frontend assets.

---

## 11. Dependencies

| Dependency                  | Type            | Notes                                                            |
|-----------------------------|-----------------|------------------------------------------------------------------|
| Go 1.22+                    | Build tool      | Compile-time only. Not needed on end-user machine.               |
| Wails v3                    | Framework       | Desktop shell. Provides WebView + Go↔JS bindings.               |
| `github.com/BurntSushi/toml`| Go library     | TOML config parsing. Zero dependency itself.                     |
| `modernc.org/sqlite`        | Go library     | Pure-Go SQLite. No CGO needed.                                   |
| KaTeX                       | Vendored JS     | LaTeX rendering. ~500 KB. Loaded locally.                        |
| Prism.js                    | Vendored JS     | Code syntax highlighting. ~100 KB. Loaded locally.               |
| System WebView              | Runtime         | WebKit (macOS), WebView2 (Windows), WebKitGTK (Linux).          |
| Svelte 5                    | Frontend        | Component framework. Compiled away at build time — zero runtime overhead. |
| Vite 6                      | Build tool      | Dev server with HMR + production bundler. Replaces manual `<script>` loading. |
| Sync-exclusion file         | Config file     | **Required** when the project lives on a cloud-synced volume (`.syncignore`, `.stignore`, or provider equivalent). Must exclude `.git/`, `data/*.db-shm`, `data/*.db-wal`, `data/*.db-journal` to prevent SQLite WAL and Git index corruption during live sync. |

---

## 12. Risk Analysis

| Risk                                         | Likelihood | Impact | Mitigation                                                         |
|----------------------------------------------|------------|--------|--------------------------------------------------------------------|
| Wails v3 instability / breaking changes      | Medium     | High   | Pin Wails version. Monitor release notes. Fallback: Wails v2.     |
| WebView rendering differences across OS      | Medium     | Medium | Test on all 3 platforms. Use standard CSS only. No bleeding-edge APIs. |
| MCP protocol spec changes                    | Medium     | Medium | Isolate MCP code in `internal/mcp/`. Abstract tool-call format.   |
| Pure-Go SQLite performance for large DBs     | Low        | Low    | FTS5 indexing. Conversation archival for >10k messages.            |
| Cross-compilation gotchas (CGO-free)         | Low        | Medium | Use `modernc.org/sqlite` (pure Go). Test CI builds for all OS.    |
| Feature creep beyond minimalism              | Medium     | High   | Strict non-goals. Plugin system for opt-in features. Core < 150KB.|
| Cloud provider API key leakage               | Low        | High   | Keys encrypted at rest (AES-GCM), DB file 0600 when possible, never sent to frontend. |
| Ollama API breaking changes                  | Low        | Medium | Adapter pattern isolates changes. Version-pin tested API format.   |
| Svelte / Vite breaking changes               | Low        | Low    | Pin versions in `package.json`. Svelte compiles to vanilla JS — fallback is to freeze compiled output. |

---

## 13. Delivery Plan

| Phase   | Scope                                                                                | Status       |
|---------|--------------------------------------------------------------------------------------|--------------|
| Phase 0 | **Foundation**: Wails v3 project scaffolding. Go↔JS binding layer. Frontend shell with smOllama UI ported. SQLite conversation storage. Config system. | ✅ Complete (v1.0 + v1.1) |
| Phase 1 | **Ollama Provider**: Refactor smOllama Ollama logic into Go adapter. Streaming via Wails events. Model selector. Settings panel. Auto-detect localhost Ollama. | ✅ Complete (v1.2 OLL-*) |
| Phase 2 | **OpenAI-Compatible + Smart Router**: OpenAI adapter in Go. SSE parsing. API key management. Pre-configured profiles. Quota-aware priority routing with auto-fallback. | ✅ Complete (v1.2 PROV-* + v1.3 ROUTER-*) |
| Phase 3 | **MCP Integration**: MCP client in Go (stdio + HTTP/SSE transport). Tool discovery. Tool execution loop. Built-in tools (datetime, calculator, read_file, web_fetch). | ✅ Complete (v1.4 MCP-* + DEBT-03) |
| Phase 4 | **Conversations+**: Full-text search (FTS5). Tags. Export/Import (Markdown, JSON). smOllama import. Conversation size indicator. | 🟡 In progress (v1.6 core delivered: schema, CRUD, search, rename, token stats) |
| Phase 5 | **Polish**: Model management. Keyboard shortcuts. Token counter. Settings General tab. i18n. Cross-platform build & test. Plugin hook architecture (frontend-side). Optional RAG plugin. | 🟡 In progress (v1.5 + v1.7 delivered; v1.8+ pending) |

---

## 14. Success KPIs

| KPI                                                    | Target                                    |
|--------------------------------------------------------|-------------------------------------------|
| Number of supported provider types                     | ≥ 4 (Ollama, OpenAI, MCP, Cloud APIs)    |
| Binary size (single platform)                          | < 15 MB                                   |
| Cold startup to interactive                            | < 500 ms                                  |
| RAM usage idle                                         | < 50 MB                                   |
| Time to add a new provider adapter                     | < 2 hours (< 80 lines Go)                 |
| Frontend JS size (app code)                            | < 100 KB                                  |
| Cross-platform builds passing                          | macOS + Windows + Linux CI green           |
| MCP tool execution round-trip                          | < 500 ms for built-in tools               |
| Conversation search latency (10k messages)             | < 200 ms (FTS5)                            |

---

## Appendix A — Provider Adapter Interface (Go)

```go
// providers/provider.go

// Provider defines the interface for all AI provider adapters.
type Provider interface {
    // ID returns the unique provider identifier.
    ID() string

    // Name returns the human-readable provider name.
    Name() string

    // Type returns the provider category: "ollama", "openai", "mcp", "custom".
    Type() string

    // ListModels fetches available models from the provider.
    ListModels(ctx context.Context) ([]Model, error)

    // StreamChat sends a chat request and streams response chunks via callback.
    StreamChat(ctx context.Context, req ChatRequest, onChunk func(Chunk)) error

    // TestConnection verifies connectivity and returns latency.
    TestConnection(ctx context.Context) (ConnectionResult, error)

    // SupportsTools reports whether this provider supports function/tool calling.
    SupportsTools() bool
}

// Model represents a single model available from a provider.
type Model struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    Provider string            `json:"provider"`
    Meta     map[string]string `json:"meta,omitempty"`
}

// ChatRequest is the normalized chat request sent to any provider.
type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature,omitempty"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Tools       []Tool    `json:"tools,omitempty"`
    Stream      bool      `json:"stream"`
}

// Chunk is a normalized streaming response chunk.
type Chunk struct {
    Content   string     `json:"content"`
    Done      bool       `json:"done"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    Stats     *Stats     `json:"stats,omitempty"`
}

// ConnectionResult reports the outcome of a connectivity test.
type ConnectionResult struct {
    OK        bool   `json:"ok"`
    LatencyMs int64  `json:"latency_ms"`
    Error     string `json:"error,omitempty"`
}
```

---

## Appendix B — Target Architecture

```
liaotao/
├── main.go                         # Wails app entry point
├── go.mod / go.sum
├── package.json                    # Svelte + Vite deps (dev only)
├── vite.config.js                  # Vite build config
├── config/
│   ├── default.toml                # Shipped defaults
│   └── schema.md                   # Config documentation
├── internal/                       # Go backend (unchanged)
│   ├── config/
│   ├── logger/
│   ├── paths/
│   ├── providers/
│   ├── mcp/
│   ├── db/
│   └── bindings/
├── frontend/
│   ├── index.html                  # Vite entry point
│   ├── src/
│   │   ├── App.svelte              # Root component (shell layout)
│   │   ├── main.js                 # Svelte mount + Wails event wiring
│   │   ├── stores/                 # Svelte stores (replace appState)
│   │   │   ├── app.js              # Global app state (sidebar, theme)
│   │   │   ├── chat.js             # Active conversation, messages
│   │   │   ├── providers.js        # Provider list, models, router
│   │   │   └── settings.js         # User preferences
│   │   ├── lib/                    # Shared utilities
│   │   │   ├── bridge.js           # Wails bindings wrapper
│   │   │   ├── markdown.js         # Markdown + KaTeX + Prism
│   │   │   ├── icons.js            # Lucide SVG icon helper
│   │   │   └── errors.js           # Wails error parser
│   │   ├── components/             # Reusable UI components
│   │   │   ├── Sidebar.svelte
│   │   │   ├── ConversationList.svelte
│   │   │   ├── GroupList.svelte
│   │   │   ├── ChatView.svelte
│   │   │   ├── MessageBubble.svelte
│   │   │   ├── Composer.svelte
│   │   │   ├── ModelSelector.svelte
│   │   │   ├── Topbar.svelte
│   │   │   └── ContextMenu.svelte
│   │   ├── pages/                  # Top-level views
│   │   │   ├── Chat.svelte
│   │   │   └── Settings.svelte
│   │   └── themes/                 # Design tokens
│   │       ├── tokens.css          # Primitives (colors, spacing, radii)
│   │       └── semantic.css        # Semantic tokens (surface, text, border)
│   ├── public/                     # Static assets (copied as-is)
│   │   ├── katex/                  # Vendored
│   │   └── prism/                  # Vendored
│   └── dist/                       # Vite build output (git-ignored)
├── data/
├── docs/
├── scripts/
├── tests/
├── logs/
├── plugins/
├── reports/
└── .tmp/
```

---

## Appendix C — Go ↔ JS Communication Pattern

```
┌──────────────────────┐              ┌──────────────────────┐
│     Frontend (JS)     │              │     Backend (Go)      │
├──────────────────────┤              ├──────────────────────┤
│                      │  Wails Call  │                      │
│  wails.Call(         │─────────────▶│  func (b *ChatBinding)│
│    'ChatBinding',    │              │    SendMessage(req)   │
│    'SendMessage',    │              │      → provider.      │
│    {model, messages} │              │        StreamChat()   │
│  )                   │              │                      │
│                      │  Wails Event │                      │
│  wails.Events.On(    │◀─────────────│  wails.Events.Emit(  │
│    'chat:chunk',     │              │    'chat:chunk',      │
│    (chunk) => {      │              │    chunk)             │
│      appendToDOM()   │              │                      │
│    }                 │              │                      │
│  )                   │              │                      │
└──────────────────────┘              └──────────────────────┘
```

All HTTP calls to AI providers happen **Go-side** — no CORS restrictions.
All API keys stay **Go-side** — never exposed to the WebView.
Streaming uses Wails events for real-time push to the frontend.

---

## Appendix D — Streaming Format Comparison (Go Adapters)

| Provider      | Endpoint                    | Stream Format              | Go Parsing Strategy                       |
|---------------|-----------------------------|----------------------------|-------------------------------------------|
| Ollama        | `POST /api/chat`            | NDJSON (newline-delimited) | `bufio.Scanner` + `json.Unmarshal` per line |
| OpenAI        | `POST /v1/chat/completions` | SSE (`data: {...}\n\n`)    | `bufio.Scanner`, strip `data: ` prefix     |
| Groq          | `POST /v1/chat/completions` | SSE                        | Same as OpenAI                             |
| OpenRouter    | `POST /v1/chat/completions` | SSE                        | Same as OpenAI                             |
| Together AI   | `POST /v1/chat/completions` | SSE                        | Same as OpenAI                             |
| Mistral       | `POST /v1/chat/completions` | SSE                        | Same as OpenAI                             |
| Cohere        | `POST /v2/chat`             | SSE (custom)               | Custom parser for Cohere event types       |

> The `Provider` interface ensures each format is parsed in isolation. The rest of the app only sees normalized `Chunk{Content, Done, ToolCalls, Stats}`.

---

## Appendix E — Migration from smOllama

| smOllama Data     | liaotao Target        | Migration Strategy                       |
|-------------------|-----------------------|------------------------------------------|
| IndexedDB conversations | SQLite `conversations` + `messages` | Export from browser as JSON → `liaotao import` CLI command |
| localStorage settings    | `config/default.toml`               | Manual: copy server URL, system prompt, temperature |
| localStorage token stats | SQLite `message.token_stats`         | Not migrated (re-accumulated)            |

---

*End of PRD — 聊濤 liáo tāo v0.1*
