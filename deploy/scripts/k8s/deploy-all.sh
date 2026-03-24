#!/usr/bin/env bash
# Deploy the full application stack to Kubernetes.
#
# Two modes:
#   Self-contained (default):
#     Deploys the infra chart first, then deploys the app chart pointed at that
#     infra release in the same namespace.
#
#   External infra (EXTERNAL_INFRA=true):
#     Deploys only the app chart and assumes the infra release already exists.
#
# Environment variables:
#   RELEASE_NAME       - app Helm release name (default: crawler)
#   NAMESPACE          - app namespace (default: crawler)
#   VALUES_ENV         - values overlay: dev | prod (default: dev)
#   EXTERNAL_INFRA     - skip infra install when true (default: false)
#   INFRA_RELEASE_NAME - optional explicit infra release name
#   INFRA_NAMESPACE    - optional explicit infra namespace
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_CHART_DIR="${SCRIPT_DIR}/../../helm/distributed-crawler"
INFRA_CHART_DIR="${SCRIPT_DIR}/../../helm/infra"

RELEASE_NAME="${RELEASE_NAME:-crawler}"
NAMESPACE="${NAMESPACE:-crawler}"
VALUES_ENV="${VALUES_ENV:-dev}"
EXTERNAL_INFRA="${EXTERNAL_INFRA:-false}"

if [[ "${EXTERNAL_INFRA}" == "true" ]]; then
  INFRA_RELEASE_NAME="${INFRA_RELEASE_NAME:-infra}"
  INFRA_NAMESPACE="${INFRA_NAMESPACE:-infra}"
else
  INFRA_RELEASE_NAME="${INFRA_RELEASE_NAME:-${RELEASE_NAME}-infra}"
  INFRA_NAMESPACE="${INFRA_NAMESPACE:-${NAMESPACE}}"
fi

APP_VALUE_FILES=(
  -f "${APP_CHART_DIR}/values.yaml"
  -f "${APP_CHART_DIR}/values-${VALUES_ENV}.yaml"
  -f "${APP_CHART_DIR}/values-external-infra.yaml"
)

INFRA_VALUE_FILES=(
  -f "${INFRA_CHART_DIR}/values.yaml"
  -f "${INFRA_CHART_DIR}/values-${VALUES_ENV}.yaml"
)

APP_SET_ARGS=(
  --set-string "infra.releaseName=${INFRA_RELEASE_NAME}"
  --set-string "infra.namespace=${INFRA_NAMESPACE}"
)

echo "==> App deploy: release=${RELEASE_NAME} namespace=${NAMESPACE} env=${VALUES_ENV} external-infra=${EXTERNAL_INFRA}"
echo "==> Infra target: release=${INFRA_RELEASE_NAME} namespace=${INFRA_NAMESPACE}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace "${INFRA_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Validating infra chart with helm template..."
helm template "${INFRA_RELEASE_NAME}" "${INFRA_CHART_DIR}" \
  --namespace "${INFRA_NAMESPACE}" \
  "${INFRA_VALUE_FILES[@]}" > /dev/null

echo "==> Validating app chart with helm template..."
helm template "${RELEASE_NAME}" "${APP_CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${APP_VALUE_FILES[@]}" \
  "${APP_SET_ARGS[@]}" \
  "$@" > /dev/null

if [[ "${EXTERNAL_INFRA}" != "true" ]]; then
  echo "==> Deploying infrastructure release..."
  helm upgrade --install "${INFRA_RELEASE_NAME}" "${INFRA_CHART_DIR}" \
    --namespace "${INFRA_NAMESPACE}" \
    "${INFRA_VALUE_FILES[@]}" \
    --wait \
    --timeout 15m
fi

echo "==> Deploying application release..."
helm upgrade --install "${RELEASE_NAME}" "${APP_CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  "${APP_VALUE_FILES[@]}" \
  "${APP_SET_ARGS[@]}" \
  --wait \
  --wait-for-jobs \
  --timeout 15m \
  "$@"

echo ""
echo "==> App pods:"
kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"

echo ""
echo "==> Infra services:"
kubectl get svc -n "${INFRA_NAMESPACE}" -l "app.kubernetes.io/instance=${INFRA_RELEASE_NAME}"
