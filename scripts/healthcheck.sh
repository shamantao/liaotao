#!/usr/bin/env bash
# healthcheck.sh — verifies the project structure and selected stack baseline.
# Usage: bash scripts/healthcheck.sh [--stack tauri-rust|bash-shell|python|compose-desktop]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/java-env.sh"

STACK="auto"
if [[ "${1:-}" == "--stack" && -n "${2:-}" ]]; then
  STACK="$2"
elif [[ -n "${1:-}" ]]; then
  STACK="$1"
fi
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PROJECT_NAME="$(basename "$PROJECT_DIR")"
PASS=0
FAIL=0

ok()   { echo "  [OK]  $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

echo ""
echo "=== Healthcheck: $PROJECT_NAME ==="
echo ""

# --- Common checks ---
echo "-- Core structure --"
[[ -f "$PROJECT_DIR/config/default.toml" ]] && ok "config/default.toml exists" || fail "config/default.toml missing"
[[ -d "$PROJECT_DIR/logs" ]]                && ok "logs/ directory exists"      || fail "logs/ missing (run init first)"
[[ -d "$PROJECT_DIR/reports" ]]             && ok "reports/ directory exists"   || fail "reports/ missing (run init first)"
[[ -f "$PROJECT_DIR/README.md" ]]           && ok "README.md exists"            || fail "README.md missing"
[[ -f "$PROJECT_DIR/LICENSE" ]]             && ok "LICENSE exists"              || fail "LICENSE missing"
[[ -f "$PROJECT_DIR/docs/ARCHITECTURE.md" ]] && ok "ARCHITECTURE.md exists"     || fail "docs/ARCHITECTURE.md missing"
[[ -f "$PROJECT_DIR/docs/SECURITY.md" ]]     && ok "SECURITY.md exists"         || fail "docs/SECURITY.md missing"
[[ -f "$PROJECT_DIR/scripts/check-secrets.sh" ]] && ok "check-secrets.sh exists" || fail "scripts/check-secrets.sh missing"
[[ -f "$PROJECT_DIR/scripts/test-integrity.sh" ]] && ok "test-integrity.sh exists" || fail "scripts/test-integrity.sh missing"
[[ -f "$PROJECT_DIR/scripts/test-dependencies.sh" ]] && ok "test-dependencies.sh exists" || fail "scripts/test-dependencies.sh missing"

echo ""
echo "-- Baseline checks --"
bash "$PROJECT_DIR/scripts/test-integrity.sh" && ok "integrity checks passed" || fail "integrity checks failed"
bash "$PROJECT_DIR/scripts/test-dependencies.sh" && ok "dependency checks passed" || fail "dependency checks failed"

# --- Stack-specific checks ---
if [[ "$STACK" == "tauri-rust" ]] || [[ -f "$PROJECT_DIR/src-tauri/Cargo.toml" ]]; then
  echo ""
  echo "-- Stack: tauri-rust --"
  command -v rustc  >/dev/null 2>&1 && ok "rustc found ($(rustc --version))" || fail "rustc not found"
  command -v cargo  >/dev/null 2>&1 && ok "cargo found"                       || fail "cargo not found"
  command -v node   >/dev/null 2>&1 && ok "node found ($(node --version))"   || fail "node not found"
  [[ -f "$PROJECT_DIR/src-tauri/Cargo.toml" ]] && ok "Cargo.toml found"       || fail "src-tauri/Cargo.toml missing"
fi

if [[ "$STACK" == "bash-shell" ]] || [[ -f "$PROJECT_DIR/scripts/main.sh" ]]; then
  echo ""
  echo "-- Stack: bash-shell --"
  bash_ver=$(bash --version | head -n1 | grep -oE '[0-9]+\.[0-9]+' | head -1)
  major="${bash_ver%%.*}"
  (( major >= 4 )) && ok "bash >= 4 ($bash_ver)" || fail "bash < 4 required (found $bash_ver)"
  [[ -f "$PROJECT_DIR/scripts/main.sh" ]] && ok "scripts/main.sh found"       || fail "scripts/main.sh missing"
fi

if [[ "$STACK" == "python" ]] || [[ -f "$PROJECT_DIR/pyproject.toml" ]]; then
  echo ""
  echo "-- Stack: python --"
  command -v uv      >/dev/null 2>&1 && ok "uv found ($(uv --version))"             || fail "uv not found — installer: curl -LsSf https://astral.sh/uv/install.sh | sh"
  command -v python3 >/dev/null 2>&1 && ok "python3 found ($(python3 --version))"   || fail "python3 not found"
  [[ -f "$PROJECT_DIR/pyproject.toml" ]]   && ok "pyproject.toml found"             || fail "pyproject.toml missing"
  [[ -f "$PROJECT_DIR/.python-version" ]]  && ok ".python-version found ($(cat "$PROJECT_DIR/.python-version"))" || fail ".python-version missing"
fi

if [[ "$STACK" == "compose-desktop" ]] || [[ -f "$PROJECT_DIR/settings.gradle.kts" ]]; then
  echo ""
  echo "-- Stack: compose-desktop --"
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
    ok "Gradle wrapper found"
  elif command -v gradle >/dev/null 2>&1; then
    ok "gradle found"
  else
    fail "Neither Gradle wrapper nor gradle command found"
  fi
  [[ -f "$PROJECT_DIR/settings.gradle.kts" ]] && ok "settings.gradle.kts found" || fail "settings.gradle.kts missing"
  [[ -f "$PROJECT_DIR/build.gradle.kts" ]] && ok "build.gradle.kts found" || fail "build.gradle.kts missing"
  [[ -f "$PROJECT_DIR/app-desktop/build.gradle.kts" ]] && ok "app-desktop/build.gradle.kts found" || fail "app-desktop/build.gradle.kts missing"
  [[ -f "$PROJECT_DIR/app-desktop/src/main/kotlin/io/liaotao/appdesktop/Main.kt" ]] && ok "desktop main entry point found" || fail "desktop main entry point missing"
fi

if [[ "$STACK" == "wails-go" ]]; then
  echo ""
  echo "-- Stack: wails-go --"
  fail "wails-go is no longer an approved stack for this repository"
fi

# --- Summary ---
echo ""
echo "=== Result: $PASS passed / $FAIL failed ==="
echo ""
[[ $FAIL -eq 0 ]] && exit 0 || exit 1
