# Liaotao Documentation Index

Date: 2026-05-07
Primary stack: Kotlin + Compose Desktop
Status: Architecture reset in progress

## Reading Order

1. `docs/ARCHITECTURE.md` for the implementation target.
2. `docs/SECURITY.md` for security handling rules.
3. `docs/CHANGELOG.md` for release history.
4. External product references in `docs-liaotao/` for PRD and backlog.

## Local Development Baseline

Run these commands before publishing changes:

```bash
bash scripts/test-integrity.sh
bash scripts/test-dependencies.sh
bash scripts/check-secrets.sh
bash scripts/healthcheck.sh --stack compose-desktop
```

## Current Constraints

1. The repository has been migrated away from Wails at the architecture level.
2. The Gradle wrapper still needs to be generated on a machine with a working JDK.
3. Kotlin Compose Desktop is now the only approved application stack.

## Near-Term Technical Goal

Reach a runnable desktop shell with:
1. `app-desktop` as the executable module.
2. `domain`, `connectors`, `persistence`, and `shared` as initial boundaries.
3. Packaged desktop outputs for macOS, Windows, and Linux.
