#!/usr/bin/env bash
# build-release.sh -- Cross-platform release build orchestration for CI.
# Responsibilities: derive version, run Wails build, package portable artifacts.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

export PATH="$PATH:$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin"

if ! command -v wails3 >/dev/null 2>&1; then
  echo "[ERROR] wails3 not found"
  exit 1
fi

VERSION="$(bash "$PROJECT_DIR/ci/release-version.sh")"
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_NAME="$(uname -m)"
ARTIFACT_DIR="$PROJECT_DIR/build/artifacts"
mkdir -p "$ARTIFACT_DIR"
BUILD_UNIVERSAL="${BUILD_UNIVERSAL:-0}"

echo "[INFO] Building liaotao v$VERSION on $OS_NAME/$ARCH_NAME"

# Ensure app version can be overridden by CI without committing config changes.
export APP__APP__VERSION="$VERSION"

build_once() {
  local target="$1"
  echo "[INFO] Running wails3 build (${target})"
  # Wails v3 uses config-driven builds and does not expose -clean / -platform flags.
  if [[ "$OS_NAME" == "darwin" && "$BUILD_UNIVERSAL" == "1" ]]; then
    echo "[WARN] Universal override requested, but current Wails v3 CLI is config-driven; running native build"
  fi
  wails3 build
}

collect_native_outputs() {
  find "$PROJECT_DIR/build" -type f \( -name "*.dmg" -o -name "*.pkg" -o -name "*.msi" -o -name "*.exe" -o -name "*.AppImage" -o -name "*.deb" -o -name "*.rpm" \) -print0 \
    | while IFS= read -r -d '' file; do
      cp "$file" "$ARTIFACT_DIR/"
      echo "[OK] Collected native artifact: $(basename "$file")"
    done
}

package_portable() {
  local arch_label="$ARCH_NAME"
  if [[ "$OS_NAME" == "darwin" && "$BUILD_UNIVERSAL" == "1" ]]; then
    arch_label="universal"
  fi
  local base="liaotao-v${VERSION}-${OS_NAME}-${arch_label}"
  if [[ "$OS_NAME" == "darwin" ]]; then
    local app_path
    app_path="$(find "$PROJECT_DIR/build" -type d -name "*.app" | head -n1 || true)"
    if [[ -n "$app_path" ]]; then
      ditto -c -k --sequesterRsrc --keepParent "$app_path" "$ARTIFACT_DIR/${base}.zip"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.zip"
      return
    fi
  fi

  if [[ "$OS_NAME" == "linux" ]]; then
    local bin_path
    bin_path="$(find "$PROJECT_DIR/build" -type f -name "liaotao" | head -n1 || true)"
    if [[ -n "$bin_path" ]]; then
      tar -czf "$ARTIFACT_DIR/${base}.tar.gz" -C "$(dirname "$bin_path")" "$(basename "$bin_path")"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.tar.gz"
      return
    fi
  fi

  if [[ "$OS_NAME" == mingw* || "$OS_NAME" == msys* || "$OS_NAME" == cygwin* || "$OS_NAME" == "windows_nt" ]]; then
    local exe_path
    exe_path="$(find "$PROJECT_DIR/build" -type f -name "*.exe" | head -n1 || true)"
    if [[ -n "$exe_path" ]]; then
      powershell -NoProfile -Command "Compress-Archive -Force -Path '$exe_path' -DestinationPath '$ARTIFACT_DIR/${base}.zip'"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.zip"
      return
    fi
  fi

  echo "[WARN] No known binary/app found to package"
}

build_once "native"
collect_native_outputs
package_portable
