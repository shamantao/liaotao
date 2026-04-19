# Contributing

## Code style
- Follow conventions documented in `docs/`.
- Keep modules with single responsibilities.

### Svelte conventions
- One component per `.svelte` file.
- Use scoped `<style>` in every component — no global CSS except design tokens.
- Shared state goes in `stores/` (Svelte writable/derived stores).
- Backend calls go through `lib/bridge.js` — never call `window.wails` directly.
- Reference only semantic tokens from `themes/semantic.css` — never primitives.
- Svelte 5 runes: use `$state`, `$derived`, `$effect`, `$props` where appropriate.
- Keep components under 350 lines; split into sub-components when growing.
- See `docs/THEMING.md` for theming rules and token contract.

## Commits
- Use Conventional Commits (examples: `feat:`, `fix:`, `docs:`, `chore:`).
- Reference issue numbers when applicable.

## Testing
- Run baseline checks before committing:
	- `bash scripts/test-integrity.sh`
	- `bash scripts/test-dependencies.sh`
	- `bash scripts/healthcheck.sh --stack <stack>`
- Project-specific unit/integration/E2E tests are defined by each project team.

## Security checks
- Run `scripts/check-secrets.sh` before opening a pull request.
- Never commit credentials, tokens, or private keys.
