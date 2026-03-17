#!/usr/bin/env bash
# Deploy the full application stack in Docker.
#
# Two modes:
#   Self-contained (default):
#     Starts infra (PostgreSQL, RabbitMQ, MinIO, Redis, OTel, etc.) AND
#     all app components (grpc-server, workers, UI) in a single compose project.
#
#   App only (APP_ONLY=true):
#     Starts app components only. Infra must already be running via:
#       docker compose -f docker-compose.yaml up -d
#
# Usage:
#   ./deploy-all.sh                    # infra + all app components
#   APP_ONLY=true ./deploy-all.sh      # app components only
#   NO_BUILD=true ./deploy-all.sh      # skip image build
#   ./deploy-all.sh --scale fetch-worker=3
#
# Environment variables:
#   REGISTRY   – image name prefix    (default: distributed-crawler)
#   TAG        – image tag            (default: latest)
#   APP_ONLY   – skip infra startup   (default: false)
#   NO_BUILD   – skip docker build    (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUILD_SCRIPT="${SCRIPT_DIR}/../k8s/build-images.sh"

REGISTRY="${REGISTRY:-distributed-crawler}"
TAG="${TAG:-latest}"
APP_ONLY="${APP_ONLY:-false}"
NO_BUILD="${NO_BUILD:-false}"

INFRA_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE="-f ${PROJECT_ROOT}/docker-compose.app.yaml"

export REGISTRY TAG

echo "==> Docker deploy: registry=${REGISTRY}  tag=${TAG}  app-only=${APP_ONLY}"

# Build images unless skipped
if [[ "${NO_BUILD}" != "true" ]]; then
  echo "==> Building images..."
  bash "${BUILD_SCRIPT}"
fi

# Start infra if not app-only
if [[ "${APP_ONLY}" != "true" ]]; then
  echo "==> Starting infrastructure..."
  docker compose ${INFRA_COMPOSE} up -d
fi

# Run DB migrations
echo "==> Running migrations..."
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} run --rm migrate

# Start all app components
echo "==> Starting application..."
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} up -d \
  grpc-server fetch-worker parser-worker export-worker ui \
  "$@"

echo ""
echo "==> Deploy complete. Running containers:"
docker compose ${INFRA_COMPOSE} ${APP_COMPOSE} ps

echo ""
echo "==> Endpoints:"
echo "  gRPC API      localhost:${GRPC_PORT:-8083}"
echo "  HTTP gateway  http://localhost:${HTTP_PORT:-8084}"
echo "  Admin UI      http://localhost:${UI_PORT:-8080}"
