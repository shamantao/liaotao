#!/usr/bin/env bash
# bootstrap-gradle-wrapper.sh - generates the Gradle wrapper for the project.
# Responsibilities: detect Java runtime, use local gradle when available, or
# fallback to a temporary open-source Gradle distribution download.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TMP_DIR="$PROJECT_DIR/.tmp"
GRADLE_VERSION="8.10.2"

source "$SCRIPT_DIR/java-env.sh"

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

if [[ -x "$PROJECT_DIR/gradlew" ]]; then
  echo "[OK] Gradle wrapper already present"
  exit 0
fi

if command -v gradle >/dev/null 2>&1; then
  echo "[INFO] Generating Gradle wrapper with local gradle"
  (cd "$PROJECT_DIR" && gradle wrapper)
  echo "[OK] Gradle wrapper generated."
  exit 0
fi

mkdir -p "$TMP_DIR"

GRADLE_DIST_DIR="$TMP_DIR/gradle-$GRADLE_VERSION"
GRADLE_ZIP="$TMP_DIR/gradle-$GRADLE_VERSION-bin.zip"

if [[ ! -x "$GRADLE_DIST_DIR/bin/gradle" ]]; then
  if [[ ! -f "$GRADLE_ZIP" ]]; then
    echo "[INFO] Downloading Gradle $GRADLE_VERSION distribution"
    curl -fL "https://services.gradle.org/distributions/gradle-$GRADLE_VERSION-bin.zip" -o "$GRADLE_ZIP"
  fi

  echo "[INFO] Extracting Gradle distribution"
  rm -rf "$GRADLE_DIST_DIR"
  unzip -q "$GRADLE_ZIP" -d "$TMP_DIR"
fi

cd "$PROJECT_DIR"
"$GRADLE_DIST_DIR/bin/gradle" wrapper
echo "[OK] Gradle wrapper generated."