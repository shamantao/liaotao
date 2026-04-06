# GitHub Copilot Instructions — liaotao

You are an expert AI programming assistant. Follow these rules.

## 1. Language Standards

- **Chat & Explanations**: Always answer, explain, and discuss in **French** (Français).
- **Code Comments**: All code comments, docstrings, and inline documentation must be in **English**.
- **Documentation Files**: `README.md` and other documentation files must be written in **English**.

## 2. File Structure & Quality

- **File Header**: Every source code file MUST start with a comment block explaining the file's purpose and responsibilities.
- **Size Limitation**: Keep files concise. A file should NOT exceed **350-400 lines** of actual code (excluding comments and whitespace). If a file grows larger, refactor and split it into sub-modules.

## 3. Testing Rules

### 3.1 Bug Fixes

Every bug fix MUST be accompanied by a unit test in the **same commit** that:
- Reproduces the exact failure condition (test would have been RED before the fix).
- Passes after the fix (GREEN).
- Is named `Test<Component>_<what_was_broken>`, e.g. `TestStdioTransport_ContextTimeout`.

Never commit a fix without its regression test.

### 3.2 Epics — Full Coverage

Each Epic in `docs/backlog-liaotao.md` must have a corresponding test file (or test section) that covers **all its User Stories**:

| Epic | Backlog prefix | Test file |
|------|---------------|-----------|
| ARCH | `ARCH-*` | `internal/*/arch_test.go` |
| Chat | `UI-*` / `CHAT-*` | `internal/bindings/chat_test.go` |
| Providers | `PROV-*` / `OLL-*` | `internal/bindings/provider_test.go` |
| Smart Router | `ROUTER-*` | `internal/bindings/router_test.go` |
| MCP | `MCP-*` | `internal/bindings/mcp_transport_test.go` |
| Model mgmt | `MOD-*` | `internal/bindings/model_test.go` |
| Conversations | `CONV-*` | `internal/bindings/conversation_test.go` |
| Settings | `SET-*` | `internal/bindings/settings_test.go` |
| i18n | `I18N-*` | `frontend/i18n/i18n_test.go` (JS) |
| Build | `BUILD-*` | CI pipeline checks |

Before marking a US `[x]` in the backlog, verify the corresponding test exists and passes.

### 3.3 Test Execution

Run the full test suite before committing any change touching `internal/`:

```bash
go test ./internal/... -v -timeout 60s
```

For frontend-only changes, run the relevant JS tests (when test harness is set up).

### 3.4 Test Quality

- Use **in-memory mocks** (pipes, `httptest.Server`) — no external processes or network calls in unit tests.
- Tests must complete in **< 5s** individually. Use `context.WithTimeout` to enforce this.
- One test per US minimum; group related assertions within the same test function using `t.Run` sub-tests.
