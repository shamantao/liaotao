# P1.101 Regression Checklist

## Scope

- Provider CRUD persisted in SQLite
- Chat selector driven by enabled providers
- Export actions and attachment flow
- Theme and path abstraction consistency

## Functional Checks

1. Create, edit, disable, and delete provider in Settings.
2. Restart app and verify provider state is preserved.
3. Disable a provider and verify it is absent from chat model selector.
4. Export all folders and validate generated JSON imports without errors.
5. Export current folder and verify only matching project conversations are present.
6. Attach a text file and verify content context is appended to outbound prompt.

## Architecture Checks

1. No feature-screen hardcoded color constants outside theme manager usage.
2. No feature code hardcodes app data paths; path resolution goes through path manager.
3. Provider selector rollout flag works:
   - `-Dliaotao.feature.persistedProviderSelector=true`
   - `-Dliaotao.feature.persistedProviderSelector=false`

## Automated Test Checks

1. `:persistence:test` includes provider CRUD lifecycle and export/import coverage.
2. `:app-desktop:test` includes provider enabled-filter and path manager checks.
3. `:app-desktop:compileKotlin` completes with no deprecated logo loader warning.
