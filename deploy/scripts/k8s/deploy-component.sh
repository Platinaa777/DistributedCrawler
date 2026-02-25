#!/usr/bin/env bash
set -euo pipefail

# Deploy a single component by disabling all others.
# Usage: ./deploy-component.sh <component> [extra helm args...]
# Components: grpc-server, fetch-worker, parser-worker, export-worker, infra
#
# Examples:
#   ./deploy-component.sh grpc-server
#   ./deploy-component.sh fetch-worker --set fetchWorker.replicaCount=3
#   ./deploy-component.sh infra          # only postgresql, rabbitmq, minio, redis

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="$(cd "${SCRIPT_DIR}/../../helm/distributed-crawler" && pwd)"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"

COMPONENT="${1:?Usage: $0 <grpc-server|fetch-worker|parser-worker|export-worker|infra>}"
shift

RELEASE_NAME="crawler-${COMPONENT}"

# Base: disable everything
DISABLE_ALL=(
  --set grpcServer.enabled=false
  --set fetchWorker.enabled=false
  --set parserWorker.enabled=false
  --set exportWorker.enabled=false
  --set migrations.enabled=false
  --set postgresql.enabled=false
  --set rabbitmq.enabled=false
  --set minio.enabled=false
  --set redis.enabled=false
)

case "${COMPONENT}" in
  grpc-server)
    ENABLE=(
      --set grpcServer.enabled=true
      --set migrations.enabled=true
    )
    ;;
  fetch-worker)
    ENABLE=(--set fetchWorker.enabled=true)
    ;;
  parser-worker)
    ENABLE=(--set parserWorker.enabled=true)
    ;;
  export-worker)
    ENABLE=(--set exportWorker.enabled=true)
    ;;
  infra)
    ENABLE=(
      --set postgresql.enabled=true
      --set rabbitmq.enabled=true
      --set minio.enabled=true
      --set redis.enabled=true
    )
    ;;
  *)
    echo "Unknown component: ${COMPONENT}"
    echo "Valid: grpc-server, fetch-worker, parser-worker, export-worker, infra"
    exit 1
    ;;
esac

echo "==> Deploying component: ${COMPONENT} as release ${RELEASE_NAME}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

helm dependency update "${CHART_DIR}"

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  -f "${CHART_DIR}/values.yaml" \
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml" \
  "${DISABLE_ALL[@]}" \
  "${ENABLE[@]}" \
  "$@"

echo "==> Done. Pods for ${COMPONENT}:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
