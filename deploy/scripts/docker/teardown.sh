#!/usr/bin/env bash
# Stop and remove application and/or infrastructure Docker containers.
#
# Usage:
#   ./teardown.sh              # stop app components only (preserve infra)
#   INFRA=true ./teardown.sh   # stop app + infra
#   APP_ONLY=true ./teardown.sh  # same as default (explicit)
#
# Flags:
#   -v / VOLUMES=true    – also remove named volumes (destructive — data loss!)
#
# Environment variables:
#   INFRA      – set to "true" to also stop infra services (default: false)
#   APP_ONLY   – set to "true" to stop app only, ignored if INFRA=true (default: true)
#   VOLUMES    – set to "true" to remove volumes (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

INFRA="${INFRA:-false}"
VOLUMES="${VOLUMES:-false}"

INFRA_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.app.yaml"

VOLUME_FLAG=()
[[ "${VOLUMES}" == "true" ]] && VOLUME_FLAG=(-v)

APP_SERVICES=(grpc-server fetch-worker parser-worker export-worker ui migrate)

echo "==> Stopping application components..."
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} \
  stop "${APP_SERVICES[@]}" 2>/dev/null || true

docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} \
  rm -f "${VOLUME_FLAG[@]}" "${APP_SERVICES[@]}" 2>/dev/null || true

if [[ "${INFRA}" == "true" ]]; then
  echo "==> Stopping infrastructure..."
  docker compose ${INFRA_COMPOSE} down "${VOLUME_FLAG[@]}"
fi

echo ""
echo "==> Remaining containers (project):"
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} ps 2>/dev/null || true
