#!/usr/bin/env bash
# Stop all components that were started by start-all.sh.
#
# Usage:
#   ./stop-all.sh               # stop all running components
#   ./stop-all.sh grpc_server   # stop a specific component
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PIDS_DIR="${PROJECT_ROOT}/.pids"

stop() {
  local name="$1"
  local pidfile="${PIDS_DIR}/${name}.pid"

  if [[ ! -f "$pidfile" ]]; then
    echo "==> ${name}: no PID file, skipping"
    return
  fi

  local pid
  pid="$(cat "$pidfile")"

  if kill -0 "${pid}" 2>/dev/null; then
    echo "==> Stopping ${name} (PID ${pid})..."
    kill "${pid}"
    # Wait briefly for graceful shutdown
    local i=0
    while kill -0 "${pid}" 2>/dev/null && (( i < 10 )); do
      sleep 0.5
      (( i++ ))
    done
    if kill -0 "${pid}" 2>/dev/null; then
      echo "    Force killing ${name}..."
      kill -9 "${pid}" 2>/dev/null || true
    fi
    echo "    Stopped."
  else
    echo "==> ${name} (PID ${pid}) was not running."
  fi

  rm -f "${pidfile}"
}

if [[ ! -d "${PIDS_DIR}" ]]; then
  echo "==> No .pids directory found. Nothing to stop."
  exit 0
fi

if [[ $# -gt 0 ]]; then
  for arg in "$@"; do
    stop "${arg}"
  done
else
  for pidfile in "${PIDS_DIR}"/*.pid; do
    [[ -f "$pidfile" ]] || continue
    stop "$(basename "${pidfile}" .pid)"
  done
fi

echo "==> Done."
