# Security Policy

## Supported Versions
This project follows active maintenance on the latest stable release branch.

## Reporting a Vulnerability
- Do not open a public issue for sensitive vulnerabilities.
- Report privately to the maintainer first.
- Include reproduction steps, impact, and affected versions.

## Secrets Policy
- Never commit API keys, tokens, passwords, or private keys.
- Keep secrets in local environment variables or secret managers.
- Run `scripts/check-secrets.sh` before opening a pull request.

## Dependency Hygiene
- Review and update dependencies on a regular schedule.
- Validate updates with tests before release.
