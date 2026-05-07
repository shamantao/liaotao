#!/usr/bin/env bash
# test-dependencies.sh — validates dependency metadata and tool availability.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0

ok()   { echo "  [OK]  $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }
note() { echo "  [INFO] $1"; }

echo ""
echo "=== Dependency Checks ==="
echo ""

detected=0

# Node ecosystem
if [[ -f "$PROJECT_DIR/package.json" ]]; then
  detected=$((detected+1))
  note "Node project detected (package.json)."
  if command -v npm >/dev/null 2>&1 || command -v pnpm >/dev/null 2>&1 || command -v yarn >/dev/null 2>&1; then
    ok "Node package manager found"
  else
    fail "No Node package manager found (npm/pnpm/yarn)"
  fi

  if [[ -f "$PROJECT_DIR/package-lock.json" || -f "$PROJECT_DIR/pnpm-lock.yaml" || -f "$PROJECT_DIR/yarn.lock" ]]; then
    ok "Node lockfile present"
  else
    fail "Node lockfile missing (package-lock.json/pnpm-lock.yaml/yarn.lock)"
  fi
fi

# Python ecosystem
if [[ -f "$PROJECT_DIR/pyproject.toml" || -f "$PROJECT_DIR/requirements.txt" || -f "$PROJECT_DIR/requirements-dev.txt" ]]; then
  detected=$((detected+1))
  note "Python project detected."
  if command -v python3 >/dev/null 2>&1; then
    ok "python3 found"
  else
    fail "python3 not found"
  fi

  if [[ -f "$PROJECT_DIR/pyproject.toml" || -f "$PROJECT_DIR/requirements.txt" ]]; then
    ok "Python dependency manifest present"
  else
    fail "Python dependency manifest missing"
  fi
fi

# Rust ecosystem
if [[ -f "$PROJECT_DIR/Cargo.toml" || -f "$PROJECT_DIR/src-tauri/Cargo.toml" ]]; then
  detected=$((detected+1))
  note "Rust project detected."
  if command -v cargo >/dev/null 2>&1; then
    ok "cargo found"
  else
    fail "cargo not found"
  fi
fi

if [[ "$detected" -eq 0 ]]; then
  note "No known dependency ecosystem detected."
fi

echo ""
echo "=== Dependency Result: $PASS passed / $FAIL failed ==="
echo ""

[[ $FAIL -eq 0 ]] && exit 0 || exit 1
