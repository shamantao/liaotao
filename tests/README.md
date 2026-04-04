# Testing Conventions

## Mandatory Baseline (all projects)
- `scripts/test-integrity.sh`
- `scripts/test-dependencies.sh`
- `scripts/check-secrets.sh`
- `scripts/healthcheck.sh`

## Unit tests
- Project teams define scope and depth based on domain risk.
- Tests live in `tests/` at the project root (or `src/` for Rust).

## Integration tests
- Recommended for key workflows when relevant.
- Never use production files as test fixtures.

## E2E tests
- Strategy is required in `docs/ARCHITECTURE.md`.
- Framework/tooling choice is project-specific.

## Healthcheck
- `scripts/healthcheck.sh` validates environment setup.
- Run it before and after generating a project.

## Fixtures
- Store small test files in `tests/fixtures/`.
- Never commit large media files.
