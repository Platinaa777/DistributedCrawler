#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-crawler}"

echo "==> Uninstalling all Helm releases in namespace ${NAMESPACE}..."

for release in $(helm list -n "${NAMESPACE}" -q 2>/dev/null); do
  echo "    Uninstalling ${release}..."
  helm uninstall "${release}" -n "${NAMESPACE}"
done

echo "==> Remaining resources in ${NAMESPACE}:"
kubectl get all -n "${NAMESPACE}" 2>/dev/null || true
