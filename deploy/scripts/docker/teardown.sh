#!/usr/bin/env bash
# Stop and remove all Docker containers and all Docker volumes on the host.
#
# Usage:
#   ./teardown.sh
#   FORCE=true ./teardown.sh
#
# Environment variables:
#   FORCE  - set to "true" to skip the interactive confirmation prompt
set -euo pipefail

FORCE="${FORCE:-false}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is not installed or not available in PATH." >&2
  exit 1
fi

if [[ "${FORCE}" != "true" ]] && [[ -t 0 ]]; then
  echo "WARNING: This will delete ALL Docker containers and ALL Docker volumes on this host."
  read -r -p "Type 'delete-all' to continue: " CONFIRM
  if [[ "${CONFIRM}" != "delete-all" ]]; then
    echo "Aborted."
    exit 1
  fi
fi

container_ids="$(docker ps -aq)"
if [[ -n "${container_ids}" ]]; then
  echo "==> Removing all containers..."
  docker rm -f ${container_ids}
else
  echo "==> No containers found."
fi

volume_ids="$(docker volume ls -q)"
if [[ -n "${volume_ids}" ]]; then
  echo "==> Removing all volumes..."
  docker volume rm ${volume_ids}
else
  echo "==> No volumes found."
fi

echo ""
echo "==> Remaining containers:"
docker ps -a || true

echo ""
echo "==> Remaining volumes:"
docker volume ls || true
