#!/usr/bin/env bash
# app-control.sh - controls Liaotao desktop runtime lifecycle.
# Responsibilities: start, stop, and report status for local desktop execution
# using a PID file and log file to support repeatable local operations.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$PROJECT_DIR/.tmp/liaotao.pid"
LOG_FILE="$PROJECT_DIR/logs/liaotao-app.log"

source "$SCRIPT_DIR/java-env.sh"

mkdir -p "$PROJECT_DIR/.tmp" "$PROJECT_DIR/logs"

is_running() {
  if [[ ! -f "$PID_FILE" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "$PID_FILE")"
  [[ -n "$pid" ]] && ps -p "$pid" >/dev/null 2>&1
}

ensure_runtime_ready() {
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

  if [[ ! -x "$PROJECT_DIR/gradlew" ]]; then
    echo "[INFO] Gradle wrapper missing, bootstrapping..."
    bash "$PROJECT_DIR/scripts/bootstrap-gradle-wrapper.sh"
  fi
}

start_app() {
  if is_running; then
    echo "[INFO] Liaotao already running (PID $(cat "$PID_FILE"))"
    return 0
  fi

  ensure_runtime_ready

  echo "[INFO] Starting Liaotao desktop app..."
  echo "[INFO] First launch may take several minutes (Gradle download + initial build)."
  echo "[INFO] Live logs: tail -f $LOG_FILE"
  (
    cd "$PROJECT_DIR"
    nohup ./gradlew :app-desktop:run >>"$LOG_FILE" 2>&1 &
    echo $! >"$PID_FILE"
  )

  sleep 1
  if is_running; then
    echo "[OK] Liaotao started (PID $(cat "$PID_FILE"))"
    echo "[INFO] Logs: $LOG_FILE"
  else
    echo "[FAIL] Start failed, check logs: $LOG_FILE"
    rm -f "$PID_FILE"
    exit 1
  fi
}

stop_app() {
  if ! is_running; then
    echo "[INFO] Liaotao is not running"
    rm -f "$PID_FILE"
    return 0
  fi

  local pid
  pid="$(cat "$PID_FILE")"
  echo "[INFO] Stopping Liaotao (PID $pid)..."
  kill "$pid" >/dev/null 2>&1 || true

  for _ in 1 2 3 4 5; do
    if ! ps -p "$pid" >/dev/null 2>&1; then
      rm -f "$PID_FILE"
      echo "[OK] Liaotao stopped"
      return 0
    fi
    sleep 1
  done

  echo "[WARN] Process still running, forcing stop"
  kill -9 "$pid" >/dev/null 2>&1 || true
  rm -f "$PID_FILE"
  echo "[OK] Liaotao stopped (forced)"
}

status_app() {
  if is_running; then
    echo "[OK] Liaotao running (PID $(cat "$PID_FILE"))"
    echo "[INFO] Logs: $LOG_FILE"
  else
    echo "[INFO] Liaotao not running"
  fi
}

usage() {
  echo "Usage: $0 {start|stop|status|restart}"
}

COMMAND="${1:-}"

case "$COMMAND" in
  start)
    start_app
    ;;
  stop)
    stop_app
    ;;
  status)
    status_app
    ;;
  restart)
    stop_app
    start_app
    ;;
  *)
    usage
    exit 1
    ;;
esac