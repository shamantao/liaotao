#!/usr/bin/env bash
# Lancer-liaotao.command -- Double-click launcher for liaotao.
# It validates prerequisites, then starts the app directly with Go.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

# Ensure common Go paths are available when launched from Finder.
export PATH="$PATH:$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin"

echo ""
echo "=== liaotao launcher ==="
echo "Project: $PROJECT_DIR"
echo ""

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] Go is not installed."
  echo "Install Go: https://go.dev/dl/"
  read -r -p "Press Enter to close..." _
  exit 1
fi

echo "[OK] Go: $(go version)"
echo ""
echo "Starting app..."
echo "(First launch may take up to 1-2 minutes while Go compiles dependencies.)"

go run .
