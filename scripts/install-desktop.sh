#!/usr/bin/env bash
# install-desktop.sh - prepares local desktop execution prerequisites.
# Responsibilities: validate Java/Gradle prerequisites, generate Gradle wrapper
# when possible, enforce uv for Python dependencies, and normalize script rights.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

source "$SCRIPT_DIR/java-env.sh"

echo "=== Liaotao Desktop Install Check ==="

if ! ensure_java_21_runtime; then
  if resolve_java_runtime; then
    echo "[FAIL] Java runtime detected but not JDK 21."
    echo "[INFO] Current Java major: $(java_major_version || echo unknown)"
    echo "[INFO] Install OpenJDK 21 (e.g. openjdk@21) and ensure it is selected."
  else
    echo "[FAIL] Java runtime not found. Install JDK 21 first."
  fi
  exit 1
fi
echo "[OK] JDK 21 detected"

if [[ ! -x "$PROJECT_DIR/gradlew" ]]; then
  echo "[INFO] Gradle wrapper missing, bootstrapping..."
  bash "$PROJECT_DIR/scripts/bootstrap-gradle-wrapper.sh"
else
  echo "[OK] Gradle wrapper detected"
fi

if [[ -f "$PROJECT_DIR/pyproject.toml" || -f "$PROJECT_DIR/requirements.txt" || -f "$PROJECT_DIR/requirements-dev.txt" ]]; then
  if command -v uv >/dev/null 2>&1; then
    echo "[OK] uv detected for Python dependency management"
  else
    echo "[FAIL] Python dependencies detected but uv is not installed"
    exit 1
  fi
else
  echo "[INFO] No Python dependency manifests detected"
fi

chmod +x "$PROJECT_DIR/scripts/app-control.sh"
chmod +x "$PROJECT_DIR/scripts/bootstrap-gradle-wrapper.sh"
chmod +x "$PROJECT_DIR/scripts/java-env.sh"
chmod +x "$PROJECT_DIR/scripts/recette-desktop.sh"
chmod +x "$PROJECT_DIR/scripts/install-desktop.sh"
chmod +x "$PROJECT_DIR/Launch-liaotao.command"

echo "[OK] Script permissions normalized"
echo "[OK] Desktop install baseline ready"