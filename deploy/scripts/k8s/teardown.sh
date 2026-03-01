#!/usr/bin/env bash
# Uninstall all Helm releases from app and/or infra namespaces.
#
# Usage:
#   ./teardown.sh              # both namespaces
#   APP_ONLY=true  ./teardown.sh
#   INFRA_ONLY=true ./teardown.sh
#
# Environment variables:
#   APP_NAMESPACE    – Application namespace (default: crawler)
#   INFRA_NAMESPACE  – Infrastructure namespace (default: infra)
#   APP_ONLY         – Set to "true" to skip infra namespace
#   INFRA_ONLY       – Set to "true" to skip app namespace
set -euo pipefail

APP_NAMESPACE="${APP_NAMESPACE:-crawler}"
INFRA_NAMESPACE="${INFRA_NAMESPACE:-infra}"
APP_ONLY="${APP_ONLY:-false}"
INFRA_ONLY="${INFRA_ONLY:-false}"

uninstall_namespace() {
  local ns="$1"
  local releases
  releases=$(helm list -n "${ns}" -q 2>/dev/null || true)

  if [[ -z "$releases" ]]; then
    echo "==> No Helm releases found in namespace '${ns}'"
    return
  fi

  echo "==> Uninstalling releases in namespace '${ns}':"
  while IFS= read -r release; do
    echo "    - ${release}"
    helm uninstall "${release}" -n "${ns}"
  done <<< "$releases"
}

if [[ "${INFRA_ONLY}" != "true" ]]; then
  uninstall_namespace "${APP_NAMESPACE}"
fi

if [[ "${APP_ONLY}" != "true" ]]; then
  uninstall_namespace "${INFRA_NAMESPACE}"
fi

echo ""
echo "==> Remaining resources:"
for ns in "${APP_NAMESPACE}" "${INFRA_NAMESPACE}"; do
  echo "--- namespace: ${ns} ---"
  kubectl get all -n "${ns}" 2>/dev/null || true
done
