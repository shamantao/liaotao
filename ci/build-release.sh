#!/usr/bin/env bash
# build-release.sh -- Cross-platform release build orchestration for CI.
# Responsibilities: derive version, build native binary, package portable artifacts.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

export PATH="$PATH:$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin"

VERSION="$(bash "$PROJECT_DIR/ci/release-version.sh")"
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_NAME="$(uname -m)"
ARTIFACT_DIR="$PROJECT_DIR/build/artifacts"
BIN_DIR="$PROJECT_DIR/build/bin"
mkdir -p "$ARTIFACT_DIR" "$BIN_DIR"

echo "[INFO] Building liaotao v$VERSION on $OS_NAME/$ARCH_NAME"

# Ensure app version can be overridden by CI without committing config changes.
export APP__APP__VERSION="$VERSION"

build_binary() {
  local out_name="liaotao"
  if [[ "$OS_NAME" == mingw* || "$OS_NAME" == msys* || "$OS_NAME" == cygwin* ]]; then
    out_name="liaotao.exe"
  fi
  local out_path="$BIN_DIR/$out_name"
  echo "[INFO] Running go build -> $out_path"
  go build -o "$out_path" ./main.go
  echo "[OK] Built binary: $out_path"
}

package_portable() {
  local arch_label="$ARCH_NAME"
  case "$arch_label" in
    x86_64) arch_label="x86_64" ;;
    amd64) arch_label="x86_64" ;;
    aarch64) arch_label="arm64" ;;
    arm64) arch_label="arm64" ;;
  esac

  local base="liaotao-v${VERSION}-${OS_NAME}-${arch_label}"

  if [[ "$OS_NAME" == "darwin" ]]; then
    local bin_path="$BIN_DIR/liaotao"
    if [[ -f "$bin_path" ]]; then
      ditto -c -k --sequesterRsrc --keepParent "$bin_path" "$ARTIFACT_DIR/${base}.zip"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.zip"
      return
    fi
  fi

  if [[ "$OS_NAME" == "linux" ]]; then
    local bin_path="$BIN_DIR/liaotao"
    if [[ -f "$bin_path" ]]; then
      tar -czf "$ARTIFACT_DIR/${base}.tar.gz" -C "$BIN_DIR" "liaotao"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.tar.gz"
      return
    fi
  fi

  if [[ "$OS_NAME" == mingw* || "$OS_NAME" == msys* || "$OS_NAME" == cygwin* || "$OS_NAME" == "windows_nt" ]]; then
    local exe_path="$BIN_DIR/liaotao.exe"
    if [[ -f "$exe_path" ]]; then
      # Convert bash paths to Windows format for PowerShell
      local win_exe_path win_artifact_path
      if command -v cygpath >/dev/null 2>&1; then
        win_exe_path=$(cygpath -w "$exe_path")
        win_artifact_path=$(cygpath -w "$ARTIFACT_DIR/${base}.zip")
      else
        # Fallback if cygpath not available: just use the paths as-is
        win_exe_path="$exe_path"
        win_artifact_path="$ARTIFACT_DIR/${base}.zip"
      fi
      powershell -NoProfile -Command "Compress-Archive -Force -Path '$win_exe_path' -DestinationPath '$win_artifact_path'"
      echo "[OK] Packaged: $ARTIFACT_DIR/${base}.zip"
      return
    fi
  fi

  echo "[WARN] No known binary found to package"
}

build_binary
package_portable
