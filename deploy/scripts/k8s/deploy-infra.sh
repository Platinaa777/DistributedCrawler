#!/usr/bin/env bash
# Deploy the shared infrastructure Helm release.
#
# Environment variables:
#   RELEASE_NAME  - infra Helm release name (default: infra)
#   NAMESPACE     - infra namespace (default: infra)
#   VALUES_ENV    - values overlay: dev | prod (default: dev)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../helm/infra"
RELEASE_NAME="${RELEASE_NAME:-infra}"
NAMESPACE="${NAMESPACE:-infra}"
VALUES_ENV="${VALUES_ENV:-dev}"

VALUE_FILES=(
  -f "${CHART_DIR}/values.yaml"
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml"
)

echo "==> Infra deploy: release=${RELEASE_NAME} namespace=${NAMESPACE} env=${VALUES_ENV}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Validating infra chart with helm template..."
helm template "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${VALUE_FILES[@]}" \
  "$@" > /dev/null

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${VALUE_FILES[@]}" \
  --wait \
  --timeout 15m \
  "$@"

echo ""
echo "==> Infrastructure deployed. Pods:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"

echo ""
echo "==> Services:"
kubectl get svc -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
