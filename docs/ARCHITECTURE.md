# Architecture - Liaotao

Date: 2026-05-07
Status: Approved baseline
Desktop scope: macOS, Windows, Linux
Primary stack: Kotlin + Compose Desktop (JVM)

## 1. Architecture Decision

Liaotao will not continue on Wails.

The desktop application will use Kotlin + Compose Desktop as the primary runtime and UI stack.

This choice is driven by four constraints:
1. Desktop-only product for the current phase.
2. Need for a stable multi-OS desktop runtime.
3. Need for a rich local UI with low friction for settings, projects, conversations, and connectors.
4. Need to reuse proven product patterns observed in Kai without inheriting mobile/web complexity.

## 2. Product-to-Architecture Mapping

PRD requirements map to the following technical responsibilities:
1. Multi-source chat: connector layer + routing engine.
2. Projects and conversations: local persistence + indexed retrieval.
3. MCP support: dedicated MCP manager over HTTP/SSE.
4. Export/import portability: versioned JSON serializer and import validator.
5. Visible resilience: fallback, retry, timeout, and status reporting in UI.

## 3. System Overview

Liaotao is a local-first desktop application.

The UI, orchestration, persistence, and connector logic all run locally on the user machine. External systems are contacted only for AI inference or remote tools.

High-level flow:
1. User sends a message from the desktop UI.
2. Conversation service loads project context and active connector configuration.
3. Routing engine builds the execution plan.
4. Selected connector(s) execute requests.
5. Optional MCP tool calls are executed through the MCP manager.
6. Results are streamed back to the UI.
7. Conversation state, metadata, and event log are persisted locally.

## 4. Runtime Boundaries

### 4.1 Desktop App

Responsibilities:
1. Windowing and navigation.
2. Chat UI and project UI.
3. Settings and connector management.
4. Status, logs, and import/export flows.

### 4.2 Domain Layer

Responsibilities:
1. Conversation lifecycle.
2. Project lifecycle.
3. Message routing.
4. Retry and fallback decisions.
5. Export/import validation.

### 4.3 Infrastructure Layer

Responsibilities:
1. Local database access.
2. Secret storage.
3. File system access.
4. HTTP and SSE transport.
5. Packaging/runtime integration.

## 5. Proposed Module Layout

Target module layout:

```
liaotao/
  app/
    desktop/                # Compose Desktop entry point, windows, navigation
  domain/
    projects/               # project entities and use cases
    conversations/          # conversation entities and use cases
    routing/                # retries, fallback, compare mode, execution policy
    exportimport/           # export/import policy and validation
  connectors/
    core/                   # connector contracts and shared DTOs
    openaicompat/           # generic OpenAI-compatible connector
    ollama/                 # Ollama-specific defaults and validation
    litellm/                # LiteLLM-specific defaults and validation
    aitao/                  # Aitao integration entrypoint
    mcp/                    # MCP manager, transport, tool wrappers
  persistence/
    database/               # SQLite repositories and migrations
    secrets/                # OS keychain integration
    files/                  # JSON export/import and file IO
  shared/
    logging/                # structured logging and event audit trail
    config/                 # app config and feature flags
    platform/               # OS-specific helpers
  docs/
  scripts/
  tests/
```

## 6. Core Technical Decisions

### 6.1 UI Stack

Use Compose Desktop.

Reasons:
1. Strong desktop-first UX path.
2. Better alignment with Kai’s proven patterns.
3. Cleaner handling of settings-heavy screens than a thin WebView shell.
4. Better chance of avoiding the local Wails friction already observed on this Mac.

### 6.2 Local Persistence

Use SQLite for structured local data.

Store in database:
1. Projects.
2. Conversations.
3. Messages.
4. Connector instances metadata.
5. MCP server definitions.
6. Routing event history.

Do not store secrets in SQLite.

Use OS-native secure storage for:
1. API keys.
2. Authorization headers.
3. Any future secret material.

### 6.3 Export / Import

Use versioned JSON packages.

Rules:
1. Export contains user data and safe references only.
2. Secrets are never exported.
3. Import supports schema version checks.
4. Import supports partial recovery when non-critical sections are malformed.

### 6.4 Connector Strategy

The connector layer is contract-first.

Connector contract responsibilities:
1. Validate configuration.
2. Fetch available models if supported.
3. Execute chat request.
4. Stream response chunks.
5. Normalize provider errors.

Initial connector set:
1. OpenAI-compatible base connector.
2. Ollama connector.
3. LiteLLM connector.
4. Aitao connector.
5. MCP manager for external tools.

### 6.5 Routing Engine

The routing engine owns runtime behavior across providers.

Features:
1. Primary connector selection.
2. Ordered fallback chain.
3. Retry with bounded backoff.
4. Parallel compare mode for selected connectors.
5. Visible execution status in UI.

Rules:
1. Provider failure must not silently erase context.
2. User must see which provider answered.
3. Failed attempts must be inspectable in execution history.

### 6.6 MCP Integration

MCP support is a first-class subsystem, not a one-off add-on.

Scope for v1:
1. Remote MCP over HTTP/SSE only.
2. Background reconnect on app start.
3. Per-server enabled state.
4. Per-tool enabled state.
5. Connection status visible in settings.

Out of scope for v1:
1. stdio transport.
2. local embedded MCP runtime.

## 7. UI Surface Model

Primary desktop surfaces:
1. Chat workspace.
2. Project sidebar.
3. Conversation history/search.
4. Source selector and compare mode.
5. Settings.
6. Import/export dialog.
7. Activity/log drawer.

The app should feel like one coherent workspace, not a bundle of separate tools.

## 8. Data Model Baseline

Core entities:
1. Project.
2. Conversation.
3. Message.
4. ConnectorInstance.
5. McpServer.
6. ExecutionRun.
7. ExportPackage.

Minimum indexing requirements:
1. Search messages by keyword.
2. Filter conversations by project.
3. Filter conversations by source.
4. Sort conversations by last activity.

## 9. Error Handling and Resilience

Principles:
1. Network instability is normal and must be handled explicitly.
2. Fallback must be visible, never hidden.
3. Partial success is better than total failure when safe.
4. Storage corruption should degrade gracefully where possible.

Minimum controls:
1. Per-request timeout.
2. Retry cap.
3. Provider-level health state.
4. User-facing offline/limited-connectivity status.

## 10. Security Model

1. No secrets in Git.
2. No secrets in JSON export.
3. Sensitive values stored in OS secure storage.
4. Logs must avoid secret leakage.
5. MCP authorization headers treated as secrets.

## 11. Packaging and Distribution

Target desktop outputs:
1. macOS: DMG.
2. Windows: MSI.
3. Linux: AppImage and DEB.

Packaging is part of the architecture, not a late-stage concern.

Release validation must include a desktop smoke-test pass on packaged binaries.

## 12. Testing Strategy

### 12.1 Mandatory Checks

1. Integrity checks.
2. Dependency checks.
3. Unit tests for domain rules.
4. Integration tests for connectors.
5. Import/export schema tests.
6. Desktop smoke tests on packaged builds.

### 12.2 High-Risk Scenarios

1. Streaming responses.
2. Provider fallback.
3. MCP reconnect and tool discovery.
4. Project export/import.
5. Search and conversation restoration.

## 13. Migration from Current Repository State

The current repository was regenerated from tao-init using the Wails adapter. That scaffold should be treated as temporary bootstrap only.

Migration plan:
1. Preserve repository governance files and baseline scripts where useful.
2. Remove Wails-specific runtime code.
3. Re-bootstrap the app with a Kotlin/Gradle desktop structure.
4. Recreate docs, scripts, and tests around the new stack.

## 14. Delivery Phases

### Phase 1 - Foundation

1. Compose Desktop app shell.
2. Navigation and theme system.
3. SQLite baseline.
4. Secret storage abstraction.
5. OpenAI-compatible connector base.

### Phase 2 - MVP Core

1. Projects and conversations.
2. Ollama, LiteLLM, and Aitao connectors.
3. Fallback routing.
4. Import/export JSON.
5. Search/filter/history.

### Phase 3 - Advanced Connectivity

1. MCP server management.
2. Tool discovery and execution.
3. Compare mode.
4. Release packaging and smoke tests.

## 15. Engineering Rules

1. Keep source files small and modular.
2. Keep product rules out of UI widgets.
3. Keep provider-specific behavior out of generic domain services.
4. Keep public data separate from secret data.
5. Document architectural decisions before adding cross-cutting complexity.
