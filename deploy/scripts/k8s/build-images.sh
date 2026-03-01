#!/usr/bin/env bash
# Build Docker images for all application components.
#
# Usage:
#   ./build-images.sh                          # build all components
#   ./build-images.sh fetch-worker grpc-server # build specific components
#   ./build-images.sh --minikube               # point to minikube's Docker daemon first
#
# Environment variables:
#   DOCKER_REGISTRY  – image name prefix (default: distributed-crawler)
#   IMAGE_TAG        – image tag         (default: latest)
#   NO_CACHE         – set to 1 to pass --no-cache to docker build
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

REGISTRY="${DOCKER_REGISTRY:-distributed-crawler}"
TAG="${IMAGE_TAG:-latest}"
EXTRA_BUILD_ARGS=()
[[ "${NO_CACHE:-0}" == "1" ]] && EXTRA_BUILD_ARGS+=(--no-cache)

# Parse --minikube flag (may appear anywhere in args)
ARGS=()
USE_MINIKUBE=false
for arg in "$@"; do
  if [[ "$arg" == "--minikube" ]]; then
    USE_MINIKUBE=true
  else
    ARGS+=("$arg")
  fi
done
set -- "${ARGS[@]+"${ARGS[@]}"}"

if [[ "$USE_MINIKUBE" == true ]]; then
  echo "==> Switching Docker context to minikube..."
  eval "$(minikube docker-env)"
fi

COMPONENTS=(
  "grpc-server:docker/grpc_server/Dockerfile"
  "fetch-worker:docker/fetch_worker/Dockerfile"
  "parser-worker:docker/parser_worker/Dockerfile"
  "export-worker:docker/export_worker/Dockerfile"
  "ui:docker/ui/Dockerfile"
)

build_image() {
  local name="$1"
  local dockerfile="$2"
  local image="${REGISTRY}/${name}:${TAG}"
  echo "==> Building ${image} ..."
  docker build "${EXTRA_BUILD_ARGS[@]}" -t "$image" -f "${PROJECT_ROOT}/${dockerfile}" "$PROJECT_ROOT"
  echo "==> Done: ${image}"
}

if [[ $# -gt 0 ]]; then
  for arg in "$@"; do
    found=false
    for entry in "${COMPONENTS[@]}"; do
      name="${entry%%:*}"
      dockerfile="${entry##*:}"
      if [[ "$name" == "$arg" ]]; then
        build_image "$name" "$dockerfile"
        found=true
        break
      fi
    done
    if [[ "$found" == false ]]; then
      echo "ERROR: Unknown component '${arg}'." >&2
      echo "Available: ${COMPONENTS[*]%%:*}" >&2
      exit 1
    fi
  done
else
  for entry in "${COMPONENTS[@]}"; do
    build_image "${entry%%:*}" "${entry##*:}"
  done
fi
