#!/usr/bin/env bash
# Deploy the full application stack.
#
# Two modes:
#   Self-contained (default):
#     Deploys app + embedded bitnami subcharts (PostgreSQL, RabbitMQ, MinIO, Redis)
#     in a single Helm release. No separate infra release needed.
#
#   External infra (EXTERNAL_INFRA=true):
#     Deploys app only. Infra is assumed to be running via deploy-infra.sh.
#     Uses values-external-infra.yaml to point the app at the infra services.
#     Run deploy-infra.sh first when using this mode.
#
# Usage:
#   ./deploy-all.sh                              # self-contained, dev
#   EXTERNAL_INFRA=true ./deploy-all.sh          # app only, dev
#   VALUES_ENV=prod EXTERNAL_INFRA=true ./deploy-all.sh
#   ./deploy-all.sh --set grpcServer.replicaCount=2
#
# Environment variables:
#   RELEASE_NAME    – Helm release name (default: crawler)
#   NAMESPACE       – Kubernetes namespace (default: crawler)
#   VALUES_ENV      – Values overlay: dev | prod (default: dev)
#   EXTERNAL_INFRA  – Set to "true" to use separate infra release (default: false)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../helm/distributed-crawler"
RELEASE_NAME="${RELEASE_NAME:-crawler}"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"
EXTERNAL_INFRA="${EXTERNAL_INFRA:-false}"

echo "==> App deploy: release=${RELEASE_NAME}  namespace=${NAMESPACE}  env=${VALUES_ENV}  external-infra=${EXTERNAL_INFRA}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

VALUE_FILES=(
  -f "${CHART_DIR}/values.yaml"
  -f "${CHART_DIR}/values-${VALUES_ENV}.yaml"
)

if [[ "${EXTERNAL_INFRA}" == "true" ]]; then
  VALUE_FILES+=(-f "${CHART_DIR}/values-external-infra.yaml")
  echo "==> External infra mode: skipping helm dependency update"
else
  echo "==> Updating Helm dependencies (bitnami subcharts)..."
  helm dependency update "${CHART_DIR}"
fi

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${VALUE_FILES[@]}" \
  --wait \
  --timeout 10m \
  "$@"

echo ""
echo "==> Deploy complete. Pods:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
