#!/usr/bin/env bash
# Build and start a single application component in Docker.
# Infra must already be running (docker compose -f docker-compose.yaml up -d).
#
# Usage:
#   ./deploy-component.sh <component> [extra docker compose args...]
#
# Components:
#   grpc-server    – API server (runs migrations first)
#   fetch-worker   – Fetch worker
#   parser-worker  – Parser worker
#   export-worker  – Export worker
#   ui             – Angular admin UI
#
# Examples:
#   ./deploy-component.sh grpc-server
#   ./deploy-component.sh fetch-worker --scale fetch-worker=3
#   NO_BUILD=true ./deploy-component.sh parser-worker
#   REGISTRY=myregistry TAG=v1.2.3 ./deploy-component.sh grpc-server
#
# Environment variables:
#   REGISTRY   – image name prefix    (default: distributed-crawler)
#   TAG        – image tag            (default: latest)
#   NO_BUILD   – skip docker build    (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUILD_SCRIPT="${SCRIPT_DIR}/../k8s/build-images.sh"

REGISTRY="${REGISTRY:-distributed-crawler}"
TAG="${TAG:-latest}"
NO_BUILD="${NO_BUILD:-false}"

INFRA_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.app.yaml"

COMPONENT="${1:?Usage: $0 <grpc-server|fetch-worker|parser-worker|export-worker|ui>}"
shift

# Map component name to docker-compose service name and Dockerfile
case "${COMPONENT}" in
  grpc-server)
    SERVICE="grpc-server"
    DOCKERFILE="docker/grpc_server/Dockerfile"
    RUN_MIGRATE=true
    ;;
  fetch-worker)
    SERVICE="fetch-worker"
    DOCKERFILE="docker/fetch_worker/Dockerfile"
    RUN_MIGRATE=false
    ;;
  parser-worker)
    SERVICE="parser-worker"
    DOCKERFILE="docker/parser_worker/Dockerfile"
    RUN_MIGRATE=false
    ;;
  export-worker)
    SERVICE="export-worker"
    DOCKERFILE="docker/export_worker/Dockerfile"
    RUN_MIGRATE=false
    ;;
  ui)
    SERVICE="ui"
    DOCKERFILE="docker/ui/Dockerfile"
    RUN_MIGRATE=false
    ;;
  *)
    echo "ERROR: Unknown component '${COMPONENT}'." >&2
    echo "Valid: grpc-server, fetch-worker, parser-worker, export-worker, ui" >&2
    exit 1
    ;;
esac

export REGISTRY TAG

echo "==> Deploying component: ${COMPONENT}  registry=${REGISTRY}  tag=${TAG}"

# Build image unless skipped
if [[ "${NO_BUILD}" != "true" ]]; then
  echo "==> Building ${REGISTRY}/${COMPONENT}:${TAG} ..."
  docker build -t "${REGISTRY}/${COMPONENT}:${TAG}" \
    -f "${PROJECT_ROOT}/${DOCKERFILE}" "${PROJECT_ROOT}"
fi

# Run migrations when deploying grpc-server
if [[ "${RUN_MIGRATE}" == "true" ]]; then
  echo "==> Running migrations..."
  docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} run --rm migrate
fi

# Start the component
echo "==> Starting ${SERVICE}..."
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} up -d "${SERVICE}" "$@"

echo ""
echo "==> Done. Container status:"
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} ps "${SERVICE}"
