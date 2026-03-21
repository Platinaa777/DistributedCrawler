#!/usr/bin/env bash
# Run a single component directly (via go run or pre-built binary).
#
# Usage:
#   ./run.sh <component> [extra args...]
#
# Components:
#   grpc_server, http_server, fetch_worker, parser_worker,
#   export_worker, scheduler_worker, memory_broker
#
# Environment variables:
#   CONFIG_PATH        – config file for API servers (default: .env)
#   WORKER_CONFIG      – config file for workers (default: .worker.env)
#   MEMORY_BROKER_ADDR – address for memory_broker (default: :9090)
#   USE_BINARY         – set to 1 to run pre-built binary from BIN_DIR instead of go run
#   BIN_DIR            – directory with pre-built binaries (default: <project_root>/bin)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BIN_DIR="${BIN_DIR:-${PROJECT_ROOT}/bin}"

COMPONENT="${1:?Usage: $0 <component>}"
shift

CONFIG_PATH="${CONFIG_PATH:-.env}"
WORKER_CONFIG="${WORKER_CONFIG:-.worker.env}"
MEMORY_BROKER_ADDR="${MEMORY_BROKER_ADDR:-:9090}"

run_cmd() {
  local name="$1"
  shift
  if [[ "${USE_BINARY:-0}" == "1" ]]; then
    exec "${BIN_DIR}/${name}" "$@"
  else
    exec go run "${PROJECT_ROOT}/cmd/${name}/main.go" "$@"
  fi
}

case "${COMPONENT}" in
  grpc_server)
    run_cmd grpc_server --config-path="${CONFIG_PATH}" "$@"
    ;;
  http_server)
    run_cmd http_server --config-path="${CONFIG_PATH}" "$@"
    ;;
  fetch_worker|parser_worker|export_worker|scheduler_worker)
    run_cmd "${COMPONENT}" --worker-config-path="${WORKER_CONFIG}" "$@"
    ;;
  memory_broker)
    run_cmd memory_broker --addr="${MEMORY_BROKER_ADDR}" "$@"
    ;;
  *)
    echo "ERROR: Unknown component '${COMPONENT}'." >&2
    echo "Available: grpc_server, http_server, fetch_worker, parser_worker, export_worker, scheduler_worker, memory_broker" >&2
    exit 1
    ;;
esac
