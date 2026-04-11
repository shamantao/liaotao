# liaotao

Date: 2026-04-11
Stack: wails-go

## Overview
liaotao is a local-first desktop AI chat application with:
- OpenAI-compatible providers and Ollama
- MCP server integration
- Conversation history and search
- Settings import/export (TOML)
- Cross-platform desktop builds via Wails

## Requirements
- Go >= 1.22
- Wails CLI (`wails3`)
- Node.js >= 20
- Wails platform prerequisites: https://wails.io/docs/gettingstarted/installation

## Getting Started
```bash
git clone <your-repo-url>
cd liaotao
wails3 dev
```

## Quality Checks (mandatory baseline)
Run these commands before publishing or opening a pull request:

```bash
bash ci/test-integrity.sh
bash ci/test-dependencies.sh
bash ci/check-secrets.sh
bash ci/healthcheck.sh --stack wails-go
```

What this baseline does:
- `test-integrity.sh`: checks project integrity (required docs/files, unresolved template placeholders)
- `test-dependencies.sh`: checks dependency metadata and required package tools
- `check-secrets.sh`: detects obvious credentials/tokens
- `healthcheck.sh`: runs global sanity checks and stack checks

Project-specific tests (business logic, E2E details) remain the responsibility of each project team.

## Configuration
Edit `config/default.toml` to adjust defaults.  
User overrides go in `~/.config/liaotao/user.toml`.

## Local Release Build
```bash
bash ci/build-release.sh
```

Build outputs are written to `build/artifacts/`.

## CI and Versioning
- Workflow file: `.github/workflows/release-build.yml`
- Version resolver: `ci/release-version.sh`
- Tag format: `vMAJOR.MINOR.PATCH`

## Logs
Logs are written to `logs/` (JSON + human-readable).

## Debug vs Normal Mode
- `debug`: originals are never deleted.
- `normal`: originals are moved to Trash after successful processing.

Always start in `debug` mode until you trust the workflow.

## Project Layout
```
liaotao/
  LICENSE
  VERSION
  config/
    default.toml
  ci/
    check-secrets.sh
    test-integrity.sh
    test-dependencies.sh
    healthcheck.sh
    build-release.sh
    release-version.sh
  docs/
    ARCHITECTURE.md
    SECURITY.md
    CHANGELOG.md
  logs/
  reports/
  tests/
  .github/
    workflows/
      release-build.yml
```

## Governance
- Versioning follows SemVer (`MAJOR.MINOR.PATCH`).
- Changelog format follows Keep a Changelog.
- Security reporting rules are documented in `docs/SECURITY.md`.

## Optional CI (Jenkins via Docker)
GitHub Actions is the default CI for this project.
