#!/usr/bin/env bash
# Deploy the infrastructure Helm release (PostgreSQL, RabbitMQ, MinIO, Redis,
# RedisInsight, OTel Collector, Jaeger, Prometheus, Grafana, OpenSearch).
#
# This release is intentionally separate from the application release so infra
# can be upgraded or rolled back independently.
#
# Usage:
#   ./deploy-infra.sh                          # dev (default)
#   VALUES_ENV=prod ./deploy-infra.sh          # production
#   NAMESPACE=infra ./deploy-infra.sh          # custom namespace
#   ./deploy-infra.sh --set postgresql.auth.password=secret123
#
# Environment variables:
#   RELEASE_NAME  – Helm release name (default: infra)
#   NAMESPACE     – Kubernetes namespace (default: infra)
#   VALUES_ENV    – Values overlay: dev | prod  (default: dev)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../helm/infra"
RELEASE_NAME="${RELEASE_NAME:-infra}"
NAMESPACE="${NAMESPACE:-infra}"
VALUES_ENV="${VALUES_ENV:-dev}"

echo "==> Infra deploy: release=${RELEASE_NAME}  namespace=${NAMESPACE}  env=${VALUES_ENV}"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

rm -f "${CHART_DIR}/Chart.lock"

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
echo "==> Services:"
kubectl get svc -n "${NAMESPACE}" -l "app.kubernetes.io/instance=${RELEASE_NAME}"
