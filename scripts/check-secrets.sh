#!/usr/bin/env bash
# check-secrets.sh — lightweight secret detection helper.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo ""
echo "=== Secret scan ==="
echo ""

if command -v gitleaks >/dev/null 2>&1; then
  echo "Using gitleaks..."
  gitleaks detect \
    --source "$PROJECT_DIR" \
    --config "$PROJECT_DIR/.gitleaks.toml" \
    --no-git
  echo "No secrets detected by gitleaks."
  exit 0
fi

echo "gitleaks not found; using fallback regex scan..."

if command -v rg >/dev/null 2>&1; then
  if rg -n -i \
    "(api[_-]?key|secret|token|password|private[_-]?key|BEGIN[[:space:]]+RSA[[:space:]]+PRIVATE[[:space:]]+KEY)" \
    "$PROJECT_DIR" \
    --glob '!logs/**' \
    --glob '!.tmp/**' \
    --glob '!reports/**' \
    --glob '!.git/**' \
    --glob '!tests/fixtures/**'; then
    echo "Potential secrets detected. Review before commit."
    exit 1
  fi
else
  if grep -RInE \
    "api[_-]?key|secret|token|password|private[_-]?key|BEGIN[[:space:]]+RSA[[:space:]]+PRIVATE[[:space:]]+KEY" \
    "$PROJECT_DIR"; then
    echo "Potential secrets detected. Review before commit."
    exit 1
  fi
fi

echo "Fallback scan found no obvious secrets."
