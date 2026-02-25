#!/usr/bin/env bash
# Deploy the infrastructure Helm release (databases, messaging, observability).
# This release is intentionally separate from the application release so that
# infra can be upgraded/rolled back independently.
#
# Usage:
#   ./deploy-infra.sh                        # dev environment (default)
#   VALUES_ENV=prod ./deploy-infra.sh        # production environment
#   VALUES_ENV=prod NAMESPACE=infra ./deploy-infra.sh
#
# Extra Helm args are passed through:
#   ./deploy-infra.sh --set postgresql.auth.password=secret123
#
# Environment variables:
#   RELEASE_NAME  – Helm release name (default: infra)
#   NAMESPACE     – Kubernetes namespace  (default: crawler)
#   VALUES_ENV    – Values overlay: dev | prod  (default: dev)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../helm/infra"
RELEASE_NAME="${RELEASE_NAME:-infra}"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"

echo "==> Infra deploy: release=${RELEASE_NAME}  namespace=${NAMESPACE}  env=${VALUES_ENV}"

# ---- 1. Ensure namespace exists -------------------------------------------
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# ---- 2. Remove stale subchart lock (chart has no dependencies) -------------
rm -f "${CHART_DIR}/Chart.lock"

# ---- 3. Deploy / upgrade --------------------------------------------------
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  -f "${CHART_DIR}/values.yaml" \
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml" \
  --wait \
  --timeout 15m \
  "$@"

echo ""
echo "==> Infrastructure deployed. Pods:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"

echo ""
echo "==> Service endpoints:"
kubectl get svc -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
