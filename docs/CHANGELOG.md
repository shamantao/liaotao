# CHANGELOG

## [0.3.1] - 2026-04-13

### Fixed
- Windows portable startup no longer fails when `config/default.toml` is missing next to `liaotao.exe`.
- Config loader now falls back to built-in defaults and continues normal initialization.

### Tests
- Added regression test for missing default config fallback in `internal/config/config_test.go`.

## [0.3.0] - 2026-04-13

### Added
- **UPD-01**: Startup update check against GitHub Releases API.
- **UPD-02**: Non-blocking update banner in UI (dismissible, localized).
- **UPD-03**: One-click "Download & Install" flow for platform-specific binaries (macOS/Linux/Windows).
- **UPD-04**: SHA256 checksum verification during update install, with failed-binary cleanup on mismatch.
- **UPD-05**: Manual "Check for updates" button in Settings > About.
- New setting `autoCheckUpdates` (enabled by default) in Settings > General.

### Changed
- Update UI strings expanded for EN / FR / zh-TW to cover download/install/check states.
- `docs/backlog-liaotao.md`: v2.7 Auto-Update stories marked complete (UPD-01..UPD-05).

### Tests
- Existing update/version tests remain green in `internal/bindings` test suite.

### PRD Conformity Note
- **Divergence from current PRD draft**: PRD non-goal **NG5** states "No ... auto-update system in v1", while 0.3.0 introduces auto-update capabilities.
- Product implication: this release implements an intentional scope expansion beyond the current PRD draft and should be reflected in the next PRD revision.

## [0.2.3] - 2026-04-12

### Added
- **CORCT-01**: Adjustable chat font size (5 levels: XS / S / M / L / XL) in Settings > General. The current size (L = 1 rem) is kept as default. Action button icons are proportionally resized (from 36 px down to 26 px at L, scaling with font choice).

### Changed
- CI: Added `.gitlab-ci.yml` — 3-stage pipeline (test / build / publish:release) triggered on semver tags.
- CI: `release-version.sh` now reads `CI_COMMIT_TAG` (GitLab) in addition to `GITHUB_REF_NAME` (GitHub Actions).

## [0.2.2] - 2026-04-11

### Added
- **BUILD-04**: New GitHub Actions workflow for cross-platform build and packaging on macOS (Intel + Apple Silicon + universal attempt), Windows x64, and Linux x64.
- **BUILD-05**: New release version resolver script (`ci/release-version.sh`) using git tag (`vX.Y.Z`) with fallback to `VERSION`.
- New CI build orchestration script (`ci/build-release.sh`) to run Wails build and package portable artifacts.

### Changed
- Release pipeline now collects native installer artifacts when available (`.dmg`, `.pkg`, `.msi`, `.exe`, `.AppImage`, `.deb`, `.rpm`) and uploads them as CI artifacts.

## [0.2.1] - 2026-04-11

### Added
- **KEY-01**: Global shortcut `Ctrl/Cmd+K` to create a new conversation.
- **KEY-02**: Global shortcut `Ctrl/Cmd+/` to focus the chat input.
- **KEY-03**: Global shortcut `Ctrl/Cmd+B` to toggle the sidebar.
- **KEY-04**: `Escape` now stops generation when streaming, otherwise returns from Settings to Chat.
- **KEY-05**: Keyboard shortcuts cheat sheet added to Settings > About.

### Changed
- About shortcuts display now adapts modifier labels by platform (`⌘` on macOS, `Ctrl` on other platforms).
- Localized labels added for keyboard shortcuts (EN/FR/zh-TW).

## [0.2.0] - 2026-04-11

### Added
- **UX-01**: Mode Expert/Simple — toggle dans Settings > Général. En mode Simple, la toolbar n'affiche que Provider, Modèle, Refresh et Style de réponse.
- **UX-01**: Sélecteur de style de réponse (Précise / Équilibrée / Créative) dans la toolbar, toujours visible.
- **UX-01**: Mapping automatique de température : Précise → 0.2, Équilibrée → 0.7, Créative → 1.0 (mode Simple uniquement).
- **UX-01**: `expert_mode` et `response_style` persistés en SQLite et restaurés au démarrage.
- Traductions complètes des nouveaux clés EN / FR / zh-TW.

### Removed
- Barre "derniers modèles utilisés" (chips sous la toolbar) — le modèle courant est déjà visible dans le sélecteur et dans chaque réponse.

## [0.1.9] - 2026-04-11

### Added
- **SET-05**: New backend export endpoint `ExportConfigurationToFile` writes TOML directly to disk (`~/Downloads` fallback `~`) for reliable desktop exports.
- **CONV-08**: New backend message deletion endpoint `DeleteMessage` with persistent SQLite deletion and conversation timestamp refresh.
- Regression tests:
	- `TestSettings_ExportConfigurationToFile`
	- `TestConversation_DeleteMessagePersists`
	- `TestSettings_LanguageSupportsZhTW`

### Fixed
- **I18N-02**: `zh-TW` is now fully accepted and persisted by backend settings sanitization (no fallback to French).
- **I18N-05**: Language selector options now re-render correctly on language switch while preserving selected value.
- **CONV-03**: Conversation sidebar date/time now refreshes when language changes and uses locale-specific deterministic format.
- **UI-10**: Deleting a message now persists in DB and no longer reappears after app restart.
- **SET-05**: Export button now succeeds in desktop context via backend file write, with status path feedback in UI.

## [0.1.8] - 2026-04-11

### Added
- **I18N-01**: Lightweight i18n engine (`frontend/js/i18n.js`) — hierarchical JSON keys, `t(key, vars)` with `{{var}}` interpolation, automatic EN fallback.
- **I18N-01**: Translation bundles for English (`en.json`), French (`fr.json`), and Traditional Chinese (`zh-TW.json`).
- **I18N-02**: Language selector in General Settings now supports EN / FR / zh-TW (persisted in SQLite and applied instantly without reload).
- **I18N-05**: All static HTML strings annotated with `data-i18n` / `data-i18n-placeholder` / `data-i18n-title`; all dynamic JS strings migrated to `t()` in providers, conversations, chat, and MCP modules.

### Changed
- **I18N-03**: English is now the default language (was French).
- **I18N-04**: French translation complete (≈100 % UI coverage).

## [0.1.7] - 2026-04-10

### Added
- **SET-05**: Settings can now be exported and imported as TOML (general, providers, MCP servers).
- **SET-06**: New About section in Settings with runtime version, links and credits.

### Changed
- **SET-02**: General settings now persist language/theme and a default system prompt used for new conversations.
- **SET-07**: Provider API keys are now encrypted at rest in SQLite with transparent runtime decryption.

## [0.1.6] - 2026-04-10

### Added
- **CONV-03**: Conversation sidebar now groups items by recency (`Today`, `Yesterday`, `This Week`, `Older`) and shows a compact date/time line below each title.
- **CONV-07**: Conversation search added in the sidebar with backend search across both conversation titles and message content.
- **CONV-08**: Each message now exposes token stats in the chat history footer, including input/output token counts, generation duration, and throughput when available.
- **MOD-01**: Chat model selector now supports provider-grouped listing in Automat mode with a live filter input.
- **MOD-04**: "Last used" model chips were added below the prompt area for one-click reuse.

### Changed
- **CONV-05**: Conversations can now be renamed inline from the sidebar.
- New conversations automatically adopt a preview title from the first user message while preserving later manual renames.
- Conversation sidebar interactions now keep the active conversation synchronized after search and rename flows.
- Added regression/unit coverage for conversation rename, search, and auto-title behaviors.
- Message persistence now stores per-message token telemetry in the existing `messages.token_stats` JSON column, with estimated values used when providers do not return full metrics.
- **MOD-02**: Conversation records now persist model/provider runtime preferences so each chat restores its own generation setup.
- **MOD-03**: Provider entries now surface a connected/disconnected/unknown status indicator based on last-known checks.
- **MOD-06**: Temperature and max tokens are now configurable per conversation and propagated to generation requests.
- **MOD-07**: System prompt is now stored per conversation and injected into provider requests.

## [0.1.5] - 2026-04-10

### Added
- **CORCT-04**: Application branding updated with Liaotao logo in the top bar and frontend favicon (`frontend/assets/logo-liaotao.svg`).
- Native icon assets prepared for packaging:
	- `build/appicon.png` (shared source)
	- `build/darwin/icons.icns` (macOS Dock/app bundle)
	- `build/windows/icon.ico` (Windows executable/taskbar)
	- `build/linux/appicon.png` (Linux launcher/package)

### Changed
- **CORCT-05**: Chat layout refactor — provider/model selectors moved below the prompt area with a compact refresh control.
- **CORCT-06**: Main action buttons migrated from text labels to icon-only controls with tooltips/ARIA labels (chat composer, message actions, conversation delete, provider/MCP form actions).
- **MOD-05**: Model listing is now lazy-loaded when needed and cached per provider for the current session; manual refresh forces reload.
- **CONV-06**: Conversation deletion UX improved with inline confirmation and authoritative sidebar refresh from DB after deletion.
- `.gitignore`: allow versioning native icon assets under `build/`.

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


