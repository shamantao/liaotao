#!/usr/bin/env bash
# Launch-liaotao.command - macOS double-click launcher for Liaotao.
# Responsibilities: provide a Finder-friendly entrypoint that starts the
# desktop app via the runtime control script.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$PROJECT_DIR"
bash "$PROJECT_DIR/scripts/app-control.sh" start

echo ""
echo "Press Enter to close this window..."
read -r