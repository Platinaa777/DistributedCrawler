#!/usr/bin/env bash
# Tail logs for application components running in Docker.
#
# Usage:
#   ./logs.sh                              # all app components
#   ./logs.sh grpc-server                  # single component
#   ./logs.sh fetch-worker parser-worker   # multiple components
#   FOLLOW=false ./logs.sh grpc-server     # print and exit
#
# Environment variables:
#   FOLLOW  – follow log output (default: true)
#   TAIL    – number of lines to show from end (default: 50)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

FOLLOW="${FOLLOW:-true}"
TAIL="${TAIL:-50}"

INFRA_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.app.yaml"

ALL_APP_SERVICES=(grpc-server fetch-worker parser-worker export-worker ui)

VALID_SERVICES=("${ALL_APP_SERVICES[@]}")

# Validate requested services
if [[ $# -gt 0 ]]; then
  for svc in "$@"; do
    found=false
    for valid in "${VALID_SERVICES[@]}"; do
      [[ "$svc" == "$valid" ]] && found=true && break
    done
    if [[ "$found" == false ]]; then
      echo "ERROR: Unknown service '${svc}'." >&2
      echo "Available: ${VALID_SERVICES[*]}" >&2
      exit 1
    fi
  done
  SERVICES=("$@")
else
  SERVICES=("${ALL_APP_SERVICES[@]}")
fi

FLAGS=(--tail "${TAIL}")
[[ "${FOLLOW}" == "true" ]] && FLAGS+=(-f)

echo "==> Logs: ${SERVICES[*]}"
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} logs "${FLAGS[@]}" "${SERVICES[@]}"
