#!/usr/bin/env bash
# test-dependencies.sh — validates dependency metadata and tool availability.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PASS=0
FAIL=0

source "$SCRIPT_DIR/java-env.sh"

ok()   { echo "  [OK]  $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }
note() { echo "  [INFO] $1"; }

echo ""
echo "=== Dependency Checks ==="
echo ""

detected=0

# Kotlin / Gradle ecosystem
if [[ -f "$PROJECT_DIR/settings.gradle.kts" || -f "$PROJECT_DIR/build.gradle.kts" ]]; then
  detected=$((detected+1))
  note "Gradle/Kotlin project detected."

  if ensure_java_21_runtime; then
    ok "JDK 21 runtime found"
  else
    if resolve_java_runtime; then
      fail "Java runtime found but not JDK 21 (found major $(java_major_version || echo unknown))"
    else
      fail "Java runtime not found or not configured"
    fi
  fi

  if [[ -x "$PROJECT_DIR/gradlew" ]]; then
    ok "Gradle wrapper present"
  elif command -v gradle >/dev/null 2>&1; then
    ok "gradle found"
  else
    fail "Neither Gradle wrapper nor gradle command found"
  fi

  [[ -f "$PROJECT_DIR/settings.gradle.kts" ]] && ok "settings.gradle.kts present" || fail "settings.gradle.kts missing"
  [[ -f "$PROJECT_DIR/build.gradle.kts" ]] && ok "build.gradle.kts present" || fail "build.gradle.kts missing"
fi

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
  if command -v uv >/dev/null 2>&1; then
    ok "uv found"
  else
    fail "uv not found"
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
