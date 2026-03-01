#!/usr/bin/env bash
# Deploy a single application component by disabling all others.
# For infrastructure, use deploy-infra.sh instead.
#
# Usage:
#   ./deploy-component.sh <component> [extra helm args...]
#
# Components:
#   grpc-server    – API server + DB migration job
#   fetch-worker   – Fetch worker
#   parser-worker  – Parser worker
#   export-worker  – Export worker
#   ui             – Angular admin UI
#
# Examples:
#   ./deploy-component.sh grpc-server
#   ./deploy-component.sh fetch-worker --set fetchWorker.replicaCount=3
#   VALUES_ENV=prod ./deploy-component.sh parser-worker
#
# Environment variables:
#   NAMESPACE     – Kubernetes namespace (default: crawler)
#   VALUES_ENV    – Values overlay: dev | prod (default: dev)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="$(cd "${SCRIPT_DIR}/../../helm/distributed-crawler" && pwd)"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"

COMPONENT="${1:?Usage: $0 <grpc-server|fetch-worker|parser-worker|export-worker|ui>}"
shift

RELEASE_NAME="crawler-${COMPONENT}"

# Disable all app components and embedded infra subcharts
DISABLE_ALL=(
  --set grpcServer.enabled=false
  --set fetchWorker.enabled=false
  --set parserWorker.enabled=false
  --set exportWorker.enabled=false
  --set ui.enabled=false
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
  ui)
    ENABLE=(--set ui.enabled=true)
    ;;
  *)
    echo "ERROR: Unknown component '${COMPONENT}'." >&2
    echo "Valid: grpc-server, fetch-worker, parser-worker, export-worker, ui" >&2
    echo "For infrastructure, use deploy-infra.sh." >&2
    exit 1
    ;;
esac

echo "==> Deploying component: ${COMPONENT} (release=${RELEASE_NAME}, namespace=${NAMESPACE})"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  -f "${CHART_DIR}/values.yaml" \
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml" \
  "${DISABLE_ALL[@]}" \
  "${ENABLE[@]}" \
  "$@"

echo ""
echo "==> Done. Pods for ${COMPONENT}:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
