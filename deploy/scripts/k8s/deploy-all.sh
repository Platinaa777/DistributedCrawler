#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../helm/distributed-crawler"
RELEASE_NAME="${RELEASE_NAME:-distributed-crawler}"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"

echo "==> Deploying full stack: ${RELEASE_NAME} in namespace ${NAMESPACE} (env: ${VALUES_ENV})"

# Ensure namespace exists
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# Build helm dependencies (bitnami subcharts)
echo "==> Updating Helm dependencies..."
helm dependency update "${CHART_DIR}"

# Idempotent deploy: upgrade --install
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  -f "${CHART_DIR}/values.yaml" \
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml" \
  --wait \
  --timeout 10m \
  "$@"

echo "==> Deploy complete. Pods:"
kubectl get pods -n "${NAMESPACE}"
