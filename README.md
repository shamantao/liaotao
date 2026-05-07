# Liaotao

Desktop AI workspace for multi-source conversations, provider fallback, MCP connectivity, and portable local history.

## Current State

This repository is being reset from an old Wails bootstrap to a Kotlin + Compose Desktop baseline.

P0.1 and P0.2 are now in place:
1. Wails is no longer the target runtime architecture.
2. A Gradle multi-module desktop skeleton exists.
3. Module boundaries are defined for `app-desktop`, `domain`, `connectors`, `persistence`, and `shared`.

## Stack

1. Kotlin JVM
2. Compose Desktop
3. Gradle multi-module build
4. SQLite planned for local persistence
5. OS secure storage planned for secrets

## Repository Layout

```text
liaotao/
	app-desktop/
	connectors/
	domain/
	persistence/
	shared/
	config/
	docs/
	logs/
	reports/
	scripts/
	tests/
```

## Local Requirements

Current minimum local prerequisites for development:
1. JDK 21
2. Gradle or the ability to generate the Gradle wrapper locally

The current machine does not yet provide a working Java runtime, so the project structure has been prepared but not fully executable.

## First Bootstrap After JDK Installation

```bash
bash scripts/bootstrap-gradle-wrapper.sh
./gradlew tasks
./gradlew :app-desktop:run
```

## Installation and Runtime Scripts

1. Desktop install baseline:

```bash
bash scripts/install-desktop.sh
```

2. Runtime control:

```bash
bash scripts/app-control.sh start
bash scripts/app-control.sh status
bash scripts/app-control.sh stop
```

3. macOS double-click launcher:

Use `Launch-liaotao.command` from Finder.

4. End-to-end acceptance routine:

```bash
bash scripts/recette-desktop.sh
```

This routine validates integrity, installs prerequisites, runs healthcheck, and verifies start/status/stop.

## Open Source Runtime Policy

1. JDK: OpenJDK 21 recommended (Homebrew or equivalent open-source distribution).
2. Gradle: wrapper-first workflow (`gradlew`) to avoid global tooling lock-in.
3. Python dependencies (if introduced): must be managed via `uv`.

## Quality Checks

```bash
bash scripts/test-integrity.sh
bash scripts/test-dependencies.sh
bash scripts/check-secrets.sh
bash scripts/healthcheck.sh --stack compose-desktop
```

## Documentation

1. Product scope: `docs-liaotao/PRD-liaotao.md`
2. Technical target: `docs/ARCHITECTURE.md`
3. Delivery order: `docs-liaotao/BACKLOG-liaotao.md`
