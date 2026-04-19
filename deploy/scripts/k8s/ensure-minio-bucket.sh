#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./ensure-minio-bucket.sh [options]

Options:
  --release-name <name>     Infra Helm release name. Default: infra
  --namespace <name>        Infra namespace. Default: infra
  --bucket <name>           MinIO bucket to create. Default: pages
  --minio-user <user>       MinIO root user. Default: minioadmin
  --minio-password <pwd>    MinIO root password. Required unless default is acceptable
  --mc-image <image>        Image used for the temporary mc pod. Default: minio/mc:latest
  --timeout-seconds <sec>   Rollout wait timeout. Default: 180
  --help                    Show this message

The script waits for the MinIO deployment and then creates the bucket if it does
not exist yet.
EOF
}

RELEASE_NAME="infra"
NAMESPACE="infra"
BUCKET_NAME="pages"
MINIO_USER="minioadmin"
MINIO_PASSWORD="minioadmin"
MC_IMAGE="minio/mc:latest"
TIMEOUT_SECONDS="180"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release-name)
      RELEASE_NAME="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --bucket)
      BUCKET_NAME="$2"
      shift 2
      ;;
    --minio-user)
      MINIO_USER="$2"
      shift 2
      ;;
    --minio-password)
      MINIO_PASSWORD="$2"
      shift 2
      ;;
    --mc-image)
      MC_IMAGE="$2"
      shift 2
      ;;
    --timeout-seconds)
      TIMEOUT_SECONDS="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: Unknown argument '$1'." >&2
      usage >&2
      exit 1
      ;;
  esac
done

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl is required." >&2
  exit 1
fi

MINIO_DEPLOYMENT="${RELEASE_NAME}-minio"
MINIO_SERVICE="${RELEASE_NAME}-minio"
SANITIZED_BUCKET="$(printf '%s' "${BUCKET_NAME}" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-')"
MC_POD_NAME="mc-${SANITIZED_BUCKET}-$(date +%s)"

echo "==> Waiting for MinIO deployment '${MINIO_DEPLOYMENT}' in namespace '${NAMESPACE}'..."
kubectl rollout status "deployment/${MINIO_DEPLOYMENT}" -n "${NAMESPACE}" --timeout="${TIMEOUT_SECONDS}s"

echo "==> Ensuring MinIO bucket '${BUCKET_NAME}' exists..."
kubectl run "${MC_POD_NAME}" \
  -n "${NAMESPACE}" \
  --image "${MC_IMAGE}" \
  --restart=Never \
  --rm \
  --attach \
  --env "MINIO_ENDPOINT=http://${MINIO_SERVICE}:9000" \
  --env "MINIO_USER=${MINIO_USER}" \
  --env "MINIO_PASSWORD=${MINIO_PASSWORD}" \
  --env "MINIO_BUCKET=${BUCKET_NAME}" \
  --command -- sh -c '
    set -e
    until mc alias set local "${MINIO_ENDPOINT}" "${MINIO_USER}" "${MINIO_PASSWORD}" >/dev/null 2>&1; do
      sleep 2
    done
    mc mb --ignore-existing "local/${MINIO_BUCKET}"
  '

echo "==> Bucket ready: ${BUCKET_NAME}"
