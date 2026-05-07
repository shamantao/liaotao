#!/usr/bin/env bash
# recette-desktop.sh - end-to-end local acceptance routine for desktop baseline.
# Responsibilities: run prerequisite checks, bootstrap runtime tooling, and
# verify operational lifecycle (start/status/stop) for a reproducible validation.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Liaotao Desktop Acceptance ==="

cd "$PROJECT_DIR"

echo "[STEP] Integrity"
bash scripts/test-integrity.sh

echo "[STEP] Install baseline"
bash scripts/install-desktop.sh

echo "[STEP] Dependencies"
bash scripts/test-dependencies.sh

echo "[STEP] Healthcheck"
bash scripts/healthcheck.sh --stack compose-desktop

echo "[STEP] Runtime start"
bash scripts/app-control.sh start

echo "[STEP] Runtime status"
bash scripts/app-control.sh status

echo "[STEP] Runtime stop"
bash scripts/app-control.sh stop

echo "[OK] Desktop acceptance routine completed"