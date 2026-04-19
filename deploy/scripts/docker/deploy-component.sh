#!/usr/bin/env bash
# Build and start a single application component in Docker.
# Infra must already be running in the same compose project.
#
# Usage:
#   ./deploy-component.sh <component> [extra docker compose args...]
#
# Components:
#   grpc-server    - API server (runs migrations first)
#   fetch-worker   - Fetch worker
#   parser-worker  - Parser worker
#   export-worker  - Export worker
#   ui             - Angular admin UI
#
# Environment variables:
#   REGISTRY   - image name prefix    (default: distributed-crawler)
#   TAG        - image tag            (default: latest)
#   NO_BUILD   - skip docker build    (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUILD_SCRIPT="${SCRIPT_DIR}/../k8s/build-images.sh"

source "${SCRIPT_DIR}/common.sh"

load_project_env

REGISTRY="${REGISTRY:-distributed-crawler}"
TAG="${TAG:-latest}"
NO_BUILD="${NO_BUILD:-false}"

COMPONENT="${1:?Usage: $0 <grpc-server|fetch-worker|parser-worker|export-worker|ui>}"
shift

case "${COMPONENT}" in
  grpc-server)
    SERVICE="grpc-server"
    DOCKERFILE="docker/grpc_server/Dockerfile"
    RUN_MIGRATE=true
    WAIT_FOR_GRPC=false
    ;;
  fetch-worker)
    SERVICE="fetch-worker"
    DOCKERFILE="docker/fetch_worker/Dockerfile"
    RUN_MIGRATE=false
    WAIT_FOR_GRPC=true
    ;;
  parser-worker)
    SERVICE="parser-worker"
    DOCKERFILE="docker/parser_worker/Dockerfile"
    RUN_MIGRATE=false
    WAIT_FOR_GRPC=true
    ;;
  export-worker)
    SERVICE="export-worker"
    DOCKERFILE="docker/export_worker/Dockerfile"
    RUN_MIGRATE=false
    WAIT_FOR_GRPC=true
    ;;
  ui)
    SERVICE="ui"
    DOCKERFILE="docker/ui/Dockerfile"
    RUN_MIGRATE=false
    WAIT_FOR_GRPC=true
    ;;
  *)
    echo "ERROR: Unknown component '${COMPONENT}'." >&2
    echo "Valid: grpc-server, fetch-worker, parser-worker, export-worker, ui" >&2
    exit 1
    ;;
esac

export REGISTRY TAG
export DOCKER_REGISTRY="${REGISTRY}"
export IMAGE_TAG="${TAG}"

echo "==> Deploying component: ${COMPONENT}  registry=${REGISTRY}  tag=${TAG}"

validate_compose_config
APP_COMPONENTS_CSV="${SERVICE}"
resolve_launch_selection
wait_for_core_infra
wait_for_optional_services

if [[ "${WAIT_FOR_GRPC}" == "true" ]]; then
  wait_for_grpc_server
fi

if [[ "${NO_BUILD}" != "true" ]]; then
  echo "==> Building ${REGISTRY}/${COMPONENT}:${TAG} ..."
  docker build -t "${REGISTRY}/${COMPONENT}:${TAG}" \
    -f "${PROJECT_ROOT}/${DOCKERFILE}" "${PROJECT_ROOT}"
fi

if [[ "${RUN_MIGRATE}" == "true" ]]; then
  echo "==> Running migrations..."
  compose_stack run --rm migrate
fi

echo "==> Starting ${SERVICE}..."
compose_stack up -d "${SERVICE}" "$@"

if [[ "${SERVICE}" == "grpc-server" ]]; then
  wait_for_grpc_server
fi

echo ""
echo "==> Done. Container status:"
compose_stack ps "${SERVICE}"
