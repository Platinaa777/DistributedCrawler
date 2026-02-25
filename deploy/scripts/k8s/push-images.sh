#!/usr/bin/env bash
# Push Docker images to a remote registry.
# Must be run after build-images.sh (or set DOCKER_REGISTRY/IMAGE_TAG to match).
#
# Usage:
#   ./push-images.sh                           # push all components
#   ./push-images.sh fetch-worker grpc-server  # push specific components
#
# Environment variables:
#   DOCKER_REGISTRY  – source registry used when building (default: distributed-crawler)
#   TARGET_REGISTRY  – destination registry to push to (required)
#   IMAGE_TAG        – image tag (default: latest)
#
# Example:
#   TARGET_REGISTRY=myregistry.io/myorg IMAGE_TAG=v1.2.3 ./push-images.sh
set -euo pipefail

DOCKER_REGISTRY="${DOCKER_REGISTRY:-distributed-crawler}"
TARGET_REGISTRY="${TARGET_REGISTRY:?TARGET_REGISTRY is required (e.g. ghcr.io/myorg or docker.io/myorg)}"
TAG="${IMAGE_TAG:-latest}"

COMPONENTS=(
  "export-worker"
  "fetch-worker"
  "parser-worker"
  "grpc-server"
)

push_image() {
  local name="$1"
  local src="${DOCKER_REGISTRY}/${name}:${TAG}"
  local dst="${TARGET_REGISTRY}/${name}:${TAG}"

  echo "==> Tagging ${src} -> ${dst}"
  docker tag "${src}" "${dst}"

  echo "==> Pushing ${dst} ..."
  docker push "${dst}"
  echo "==> Done: ${dst}"
}

if [[ $# -gt 0 ]]; then
  for arg in "$@"; do
    found=false
    for name in "${COMPONENTS[@]}"; do
      if [[ "${name}" == "${arg}" ]]; then
        push_image "${name}"
        found=true
        break
      fi
    done
    if [[ "${found}" == false ]]; then
      echo "ERROR: Unknown component '${arg}'. Available: ${COMPONENTS[*]}" >&2
      exit 1
    fi
  done
else
  for name in "${COMPONENTS[@]}"; do
    push_image "${name}"
  done
fi
