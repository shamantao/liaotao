# Architecture - liaotao

Date: 2026-05-07
Stack: wails-go
Profile: minimal

## 1. Overview
Describe the project goal, core use cases, and scope.

## 2. Project Structure
Explain the main folders and responsibilities.

Example:
```
liaotao/
  config/      # configuration files
  docs/        # product and technical documentation
  scripts/     # utility and automation scripts
  src/         # core source code
  tests/       # unit/integration tests
```

## 3. Development Flow
- Create small, focused modules.
- Keep files under 400 lines when feasible.
- Use English comments for technical notes in code.
- Run the project healthcheck before each release.

## 4. Testing Strategy
- Baseline integrity and dependency checks are mandatory (`scripts/test-integrity.sh`, `scripts/test-dependencies.sh`).
- Unit/integration tests are defined by each project based on its domain risk.
- E2E strategy is project-defined and not imposed by the template.

E2E decision record:
- Status: `implemented` or `deferred`
- Scope: what E2E validates
- Tooling: selected by project team
- Rationale: why this choice fits the project risk

## 5. Versioning and Releases
- Follow SemVer (`MAJOR.MINOR.PATCH`).
- Document user-visible changes in `CHANGELOG.md`.

## 6. Security Notes
- Never commit secrets.
- Report vulnerabilities via `SECURITY.md`.
