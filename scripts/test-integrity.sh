#!/usr/bin/env bash
# test-integrity.sh — validates baseline project integrity.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0

ok()   { echo "  [OK]  $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

echo ""
echo "=== Integrity Checks ==="
echo ""

[[ -f "$PROJECT_DIR/README.md" ]] && ok "README.md present" || fail "README.md missing"
[[ -f "$PROJECT_DIR/LICENSE" ]] && ok "LICENSE present" || fail "LICENSE missing"
[[ -f "$PROJECT_DIR/docs/ARCHITECTURE.md" ]] && ok "docs/ARCHITECTURE.md present" || fail "docs/ARCHITECTURE.md missing"
[[ -f "$PROJECT_DIR/docs/SECURITY.md" ]] && ok "docs/SECURITY.md present" || fail "docs/SECURITY.md missing"
[[ -f "$PROJECT_DIR/CHANGELOG.md" || -f "$PROJECT_DIR/docs/CHANGELOG.md" ]] && ok "CHANGELOG present" || fail "CHANGELOG missing"

if command -v rg >/dev/null 2>&1; then
  if rg -n "\{\{[^}]+\}\}" "$PROJECT_DIR" \
    --glob '!**/.git/**' \
    --glob '!**/node_modules/**' \
    --glob '!**/target/**' \
    --glob '!**/.venv/**' >/dev/null; then
    fail "Unresolved template placeholders detected"
  else
    ok "No unresolved template placeholders"
  fi
else
  ok "Placeholder scan skipped (rg not installed)"
fi

echo ""
echo "=== Integrity Result: $PASS passed / $FAIL failed ==="
echo ""

[[ $FAIL -eq 0 ]] && exit 0 || exit 1
