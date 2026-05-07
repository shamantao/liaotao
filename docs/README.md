# liaotao

Date: 2026-05-07
Stack: wails-go
Template version: 1.0.0
Profile: minimal

## Overview
<!-- Describe your project here -->

## Requirements
<!-- List runtime dependencies (ffmpeg, ffprobe, etc.) -->

## Getting Started
```bash
# see adapter-specific instructions in the adapter README
```

## Quality Checks (mandatory baseline)
Run these commands before publishing or opening a pull request:

```bash
bash scripts/test-integrity.sh
bash scripts/test-dependencies.sh
bash scripts/check-secrets.sh
bash scripts/healthcheck.sh --stack wails-go
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
  config/
    default.toml
  docs/
    ARCHITECTURE.md
    SECURITY.md
  logs/
  reports/
  scripts/
    check-secrets.sh
    test-integrity.sh
    test-dependencies.sh
    healthcheck.sh
  tests/
  Jenkinsfile           ← (only if CI profile = jenkins-docker)
  ci/jenkins/
    docker-compose.jenkins.yml
  src-tauri/            ← (tauri-rust only)
  src/                  ← (tauri-rust only)
```

## Governance
- Versioning follows SemVer (`MAJOR.MINOR.PATCH`).
- Changelog format follows Keep a Changelog.
- Security reporting rules are documented in `docs/SECURITY.md`.

## Optional CI (Jenkins via Docker)
If your project was generated with CI profile `jenkins-docker`:

```bash
docker compose -f ci/jenkins/docker-compose.jenkins.yml up -d
```

Then configure Jenkins to run the pipeline using `Jenkinsfile`.

## Template
This project was generated from `devwww/tao-init`.  
See `devwww/tao-init.sh` to generate new projects.
