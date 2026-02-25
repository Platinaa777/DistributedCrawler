#!/usr/bin/env bash
# Start all application components in the background.
# Logs are written to <project_root>/logs/, PIDs saved to <project_root>/.pids/.
#
# Usage:
#   ./start-all.sh
#
# Environment variables:
#   CONFIG_PATH    – config file for API servers (default: .env)
#   WORKER_CONFIG  – config file for workers (default: .worker.env)
#   USE_BINARY     – set to 1 to run pre-built binaries instead of go run
#   BIN_DIR        – directory with pre-built binaries (default: <project_root>/bin)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
LOGS_DIR="${PROJECT_ROOT}/logs"
PIDS_DIR="${PROJECT_ROOT}/.pids"
BIN_DIR="${BIN_DIR:-${PROJECT_ROOT}/bin}"

CONFIG_PATH="${CONFIG_PATH:-.env}"
WORKER_CONFIG="${WORKER_CONFIG:-.worker.env}"

mkdir -p "${LOGS_DIR}" "${PIDS_DIR}"

start() {
  local name="$1"
  shift
  local logfile="${LOGS_DIR}/${name}.log"
  local pidfile="${PIDS_DIR}/${name}.pid"

  if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    echo "==> ${name} already running (PID $(cat "$pidfile"))"
    return
  fi

  "$@" >> "${logfile}" 2>&1 &
  echo $! > "${pidfile}"
  echo "==> Started ${name} (PID $!)"
  echo "    log: ${logfile}"
}

run_component() {
  local name="$1"
  shift
  if [[ "${USE_BINARY:-0}" == "1" ]]; then
    start "${name}" "${BIN_DIR}/${name}" "$@"
  else
    start "${name}" go run "${PROJECT_ROOT}/cmd/${name}/main.go" "$@"
  fi
}

run_component grpc_server     --config-path="${CONFIG_PATH}"
run_component fetch_worker    --config-path="${WORKER_CONFIG}"
run_component parser_worker   --config-path="${WORKER_CONFIG}"
run_component export_worker   --config-path="${WORKER_CONFIG}"
run_component scheduler_worker --config-path="${WORKER_CONFIG}"

echo ""
echo "==> All components started."
echo "    Logs : ${LOGS_DIR}/"
echo "    Stop : ${SCRIPT_DIR}/stop-all.sh"
