#!/usr/bin/env bash
# Deploy the full application stack in Docker.
#
# Two modes:
#   Self-contained (default):
#     Starts infra, waits for the core dependencies, runs migrations, then
#     starts the API before the workers and UI.
#
#   App only (APP_ONLY=true):
#     Starts only the application components. Infra must already be running in
#     the same compose project.
#
# Usage:
#   ./deploy-all.sh
#   APP_ONLY=true ./deploy-all.sh
#   NO_BUILD=true ./deploy-all.sh
#   ./deploy-all.sh --scale fetch-worker=3
#
# Environment variables:
#   REGISTRY   - image name prefix    (default: distributed-crawler)
#   TAG        - image tag            (default: latest)
#   APP_ONLY   - skip infra startup   (default: false)
#   NO_BUILD   - skip docker build    (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="${SCRIPT_DIR}/../k8s/build-images.sh"

source "${SCRIPT_DIR}/common.sh"

load_project_env

REGISTRY="${REGISTRY:-distributed-crawler}"
TAG="${TAG:-latest}"
APP_ONLY="${APP_ONLY:-false}"
NO_BUILD="${NO_BUILD:-false}"

export REGISTRY TAG
export DOCKER_REGISTRY="${REGISTRY}"
export IMAGE_TAG="${TAG}"

resolve_launch_selection

echo "==> Docker deploy: registry=${REGISTRY}  tag=${TAG}  app-only=${APP_ONLY}"
echo "==> App services: ${SELECTED_APP_SERVICES[*]}"
echo "==> Infra services: ${SELECTED_INFRA_SERVICES[*]}"

validate_compose_config

if [[ "${NO_BUILD}" != "true" ]]; then
  echo "==> Building images..."
  bash "${BUILD_SCRIPT}" "${SELECTED_APP_SERVICES[@]}"
fi

if [[ "${APP_ONLY}" != "true" ]]; then
  echo "==> Starting infrastructure..."
  compose_infra up -d "${SELECTED_INFRA_SERVICES[@]}"
fi

wait_for_core_infra
wait_for_optional_services

if array_contains "grpc-server" "${SELECTED_APP_SERVICES[@]}"; then
  echo "==> Running migrations..."
  compose_stack run --rm migrate

  echo "==> Starting gRPC server..."
  compose_stack up -d grpc-server

  wait_for_grpc_server
fi

remaining_app_services=()
for service in "${SELECTED_APP_SERVICES[@]}"; do
  if [[ "${service}" != "grpc-server" ]]; then
    remaining_app_services+=("${service}")
  fi
done

if [[ "${#remaining_app_services[@]}" -gt 0 ]]; then
  echo "==> Starting remaining application components..."
  compose_stack up -d "${remaining_app_services[@]}" "$@"
fi

echo ""
echo "==> Deploy complete. Running containers:"
compose_stack ps

echo ""
echo "==> Endpoints:"
if array_contains "grpc-server" "${SELECTED_APP_SERVICES[@]}"; then
  echo "  gRPC API      localhost:${GRPC_PORT:-8083}"
  echo "  HTTP gateway  http://localhost:${HTTP_PORT:-8084}"
fi
if array_contains "ui" "${SELECTED_APP_SERVICES[@]}"; then
  echo "  Admin UI      http://localhost:${UI_PORT:-18080}"
fi
