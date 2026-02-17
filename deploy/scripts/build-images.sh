#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

REGISTRY="${DOCKER_REGISTRY:-distributed-crawler}"
TAG="${IMAGE_TAG:-latest}"

COMPONENTS=(
  "export-worker:docker/export_worker/Dockerfile"
  "fetch-worker:docker/fetch_worker/Dockerfile"
  "parser-worker:docker/parser_worker/Dockerfile"
  "grpc-server:docker/grpc_server/Dockerfile"
)

build_image() {
  local name="$1"
  local dockerfile="$2"
  local image="${REGISTRY}/${name}:${TAG}"

  echo "==> Building ${image} ..."
  docker build --no-cache -t "$image" -f "${PROJECT_ROOT}/${dockerfile}" "$PROJECT_ROOT"
  echo "==> Done: ${image}"
}

# If arguments provided, build only those components
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
      echo "ERROR: Unknown component '$arg'. Available: ${COMPONENTS[*]%%:*}" >&2
      exit 1
    fi
  done
else
  # Build all
  for entry in "${COMPONENTS[@]}"; do
    name="${entry%%:*}"
    dockerfile="${entry##*:}"
    build_image "$name" "$dockerfile"
  done
fi
