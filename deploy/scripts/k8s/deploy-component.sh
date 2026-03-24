#!/usr/bin/env bash
# Deploy a single application component against an existing infra release.
#
# Environment variables:
#   NAMESPACE          - app namespace (default: crawler)
#   VALUES_ENV         - values overlay: dev | prod (default: dev)
#   INFRA_RELEASE_NAME - infra release name (default: infra)
#   INFRA_NAMESPACE    - infra namespace (default: infra)
#   BASE_RELEASE_NAME  - release name hosting the shared grpc-server (default: crawler)
#   GRPC_SERVER_HOST   - explicit grpc-server service host override
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="$(cd "${SCRIPT_DIR}/../../helm/distributed-crawler" && pwd)"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"
INFRA_RELEASE_NAME="${INFRA_RELEASE_NAME:-infra}"
INFRA_NAMESPACE="${INFRA_NAMESPACE:-infra}"
BASE_RELEASE_NAME="${BASE_RELEASE_NAME:-crawler}"

COMPONENT="${1:?Usage: $0 <grpc-server|fetch-worker|parser-worker|export-worker|ui>}"
shift

RELEASE_NAME="crawler-${COMPONENT}"
GRPC_SERVER_HOST="${GRPC_SERVER_HOST:-${BASE_RELEASE_NAME}-distributed-crawler-grpc-server}"

VALUE_FILES=(
  -f "${CHART_DIR}/values.yaml"
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml"
  -f "${CHART_DIR}/values-external-infra.yaml"
)

DISABLE_ALL=(
  --set grpcServer.enabled=false
  --set fetchWorker.enabled=false
  --set parserWorker.enabled=false
  --set exportWorker.enabled=false
  --set ui.enabled=false
  --set migrations.enabled=false
  --set-string "infra.releaseName=${INFRA_RELEASE_NAME}"
  --set-string "infra.namespace=${INFRA_NAMESPACE}"
  --set-string "grpcServer.hostOverride=${GRPC_SERVER_HOST}"
)

case "${COMPONENT}" in
  grpc-server)
    ENABLE=(
      --set grpcServer.enabled=true
      --set migrations.enabled=true
      --set-string "grpcServer.hostOverride="
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
    exit 1
    ;;
esac

echo "==> Deploying component: ${COMPONENT} release=${RELEASE_NAME} namespace=${NAMESPACE}"
echo "==> Infra target: release=${INFRA_RELEASE_NAME} namespace=${INFRA_NAMESPACE}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Validating component chart with helm template..."
helm template "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${VALUE_FILES[@]}" \
  "${DISABLE_ALL[@]}" \
  "${ENABLE[@]}" \
  "$@" > /dev/null

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${VALUE_FILES[@]}" \
  "${DISABLE_ALL[@]}" \
  "${ENABLE[@]}" \
  --wait \
  --wait-for-jobs \
  --timeout 15m \
  "$@"

echo ""
echo "==> Pods for ${COMPONENT}:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
