# Contributing

## Code style
- Follow conventions documented in `docs/`.
- Keep modules with single responsibilities.

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
