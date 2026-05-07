# Release Checklist

Date: 2026-05-07
Scope: Mandatory release checks for desktop package targets.

## 1. Common Release Gates

1. `bash scripts/test-integrity.sh` passes.
2. `bash scripts/test-dependencies.sh` passes.
3. `bash scripts/check-secrets.sh` passes.
4. `./gradlew :domain:test :connectors:test :persistence:test` passes.
5. `./gradlew :app-desktop:compileKotlin` passes.

## 2. Packaging Targets

### 2.1 DMG (macOS)

1. Build DMG artifact.
2. Install on clean macOS profile.
3. Launch app and run smoke checklist.

### 2.2 MSI (Windows)

1. Build MSI artifact.
2. Install on clean Windows profile.
3. Launch app and run smoke checklist.

### 2.3 AppImage (Linux)

1. Build AppImage artifact.
2. Mark executable and launch on clean Linux profile.
3. Run smoke checklist.

### 2.4 DEB (Linux)

1. Build DEB artifact.
2. Install using package manager on clean Linux profile.
3. Launch app and run smoke checklist.

## 3. Sign-Off

1. Confirm `tests/desktop-packaged-smoke-checklist.md` completed for each target.
2. Confirm changelog updated.
3. Confirm no credentials are present in exported JSON samples.
4. Record final release decision: GO / NO-GO.