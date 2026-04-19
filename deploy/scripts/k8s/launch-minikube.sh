#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUILD_SCRIPT="${SCRIPT_DIR}/build-images.sh"
DEPLOY_INFRA_SCRIPT="${SCRIPT_DIR}/deploy-infra.sh"
DEPLOY_APP_SCRIPT="${SCRIPT_DIR}/deploy-all.sh"
PORT_FORWARD_SCRIPT="${SCRIPT_DIR}/port-forward.sh"
ENSURE_BUCKET_SCRIPT="${SCRIPT_DIR}/ensure-minio-bucket.sh"

usage() {
  cat <<'EOF'
Usage:
  ./launch-minikube.sh [options]

One-shot local Kubernetes launcher for minikube:
1. Starts minikube
2. Builds app images into the minikube Docker daemon
3. Deploys infra via Helm
4. Deploys the app via Helm against that infra release
5. Ensures the MinIO bucket exists
6. Optionally starts port-forward

Core options:
  --release-name <name>             App Helm release. Default: crawler
  --namespace <name>                App namespace. Default: crawler
  --infra-release-name <name>       Infra Helm release. Default: infra
  --infra-namespace <name>          Infra namespace. Default: infra
  --values-env <dev|prod>           Helm values overlay. Default: dev
  --registry <name>                 Image prefix. Default: distributed-crawler
  --tag <tag>                       Image tag. Default: latest
  --queue-secrets-file <path>       JSON file embedded into the app Secret

Secrets and app config:
  --pg-user <name>                  PostgreSQL user. Default: crawler
  --pg-password <pwd>               PostgreSQL password. Default: some-pwd-123
  --pg-database <name>              PostgreSQL database. Default: crawler
  --rabbitmq-user <name>            RabbitMQ user. Default: guest
  --rabbitmq-password <pwd>         RabbitMQ password. Default: guest
  --minio-user <name>               MinIO user. Default: minioadmin
  --minio-password <pwd>            MinIO password. Default: minioadmin
  --minio-bucket <name>             MinIO bucket. Default: pages
  --redis-password <pwd>            Redis password. Default: some_redis_pwd_123
  --grafana-user <name>             Grafana user. Default: admin
  --grafana-password <pwd>          Grafana password. Default: changeme-grafana-password
  --jwt-secret <value>              JWT secret for the API
  --default-user-email <email>      Default admin email. Default: admin@example.com
  --default-user-password <pwd>     Default admin password. Default: 12345678
  --messaging-broker <kind>         rabbitmq | kafka | grpc_memory. Default: rabbitmq
  --cors-origin <origin>            HTTP_CORS_ALLOWED_ORIGINS value. Default: http://localhost:4200

Minikube behavior:
  --skip-minikube-start             Reuse the current cluster
  --no-build                        Skip image build
  --no-bucket                       Skip MinIO bucket creation
  --port-forward                    Start kubectl port-forward after deploy
  --port-forward-services <csv>     Services for port-forward. Example: grpc-server,ui,minio
  --lite                            Disable heavy observability pieces. Default behavior
  --full-observability              Enable Prometheus/Grafana/OpenSearch stack
  --minikube-driver <name>          Default: docker
  --minikube-cpus <n>               Default: 6
  --minikube-memory <mb>            Default: 8192
  --minikube-disk-size <size>       Optional, e.g. 20g

Advanced Helm passthrough:
  --app-values-file <path>          Extra app values file, may be repeated
  --infra-values-file <path>        Extra infra values file, may be repeated
  --app-set <key=value>             Extra helm --set for the app, may be repeated
  --app-set-string <key=value>      Extra helm --set-string for the app, may be repeated
  --app-set-file <key=path>         Extra helm --set-file for the app, may be repeated
  --infra-set <key=value>           Extra helm --set for infra, may be repeated
  --infra-set-string <key=value>    Extra helm --set-string for infra, may be repeated
  --infra-set-file <key=path>       Extra helm --set-file for infra, may be repeated
  --help                            Show this message

Examples:
  ./launch-minikube.sh --pg-password mypwd --jwt-secret supersecret --default-user-password admin123
  ./launch-minikube.sh --full-observability --port-forward
  ./launch-minikube.sh --app-set fetchWorker.replicaCount=3 --infra-set redisinsight.enabled=false
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: required command not found: $1" >&2
    exit 1
  fi
}

yaml_escape() {
  local value="${1//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "${value}"
}

write_file_block() {
  local indent="$1"
  local file_path="$2"
  local pad
  pad="$(printf '%*s' "${indent}" '')"

  while IFS= read -r line || [[ -n "${line}" ]]; do
    printf '%s%s\n' "${pad}" "${line}"
  done < "${file_path}"
}

csv_to_array() {
  local csv="$1"
  local out_name="$2"
  local old_ifs
  local raw_items=()
  old_ifs="$IFS"
  IFS=','
  read -r -a raw_items <<< "${csv}"
  IFS="${old_ifs}"
  local item
  for item in "${raw_items[@]+"${raw_items[@]}"}"; do
    eval "${out_name}+=($(printf '%q' "${item}"))"
  done
}

VALUES_ENV="dev"
APP_RELEASE_NAME="crawler"
APP_NAMESPACE="crawler"
INFRA_RELEASE_NAME="infra"
INFRA_NAMESPACE="infra"
REGISTRY="distributed-crawler"
TAG="latest"
QUEUE_SECRETS_FILE="${PROJECT_ROOT}/queue-secrets.json.example"

PG_USER="crawler"
PG_PASSWORD="some-pwd-123"
PG_DATABASE_NAME="crawler"
RMQ_USER="guest"
RMQ_PWD="guest"
MINIO_USER="minioadmin"
MINIO_PWD="minioadmin"
MINIO_BUCKET_NAME="pages"
REDIS_PWD="some_redis_pwd_123"
GRAFANA_USER="admin"
GRAFANA_PWD="changeme-grafana-password"
JWT_SECRET="your-secret-key-change-this-in-production-make-it-long-and-random"
DEFAULT_USER_EMAIL="admin@example.com"
DEFAULT_USER_PWD="12345678"
MESSAGING_BROKER="rabbitmq"
HTTP_CORS_ALLOWED_ORIGINS="http://localhost:4200"

START_MINIKUBE="true"
BUILD_IMAGES="true"
CREATE_BUCKET="true"
PORT_FORWARD="false"
LITE_MODE="true"
MINIKUBE_DRIVER="docker"
MINIKUBE_CPUS="6"
MINIKUBE_MEMORY="8192"
MINIKUBE_DISK_SIZE=""
PORT_FORWARD_SERVICES_CSV=""

APP_HELM_ARGS=()
INFRA_HELM_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release-name)
      APP_RELEASE_NAME="$2"
      shift 2
      ;;
    --namespace)
      APP_NAMESPACE="$2"
      shift 2
      ;;
    --infra-release-name)
      INFRA_RELEASE_NAME="$2"
      shift 2
      ;;
    --infra-namespace)
      INFRA_NAMESPACE="$2"
      shift 2
      ;;
    --values-env)
      VALUES_ENV="$2"
      shift 2
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --tag)
      TAG="$2"
      shift 2
      ;;
    --queue-secrets-file)
      QUEUE_SECRETS_FILE="$2"
      shift 2
      ;;
    --pg-user)
      PG_USER="$2"
      shift 2
      ;;
    --pg-password)
      PG_PASSWORD="$2"
      shift 2
      ;;
    --pg-database)
      PG_DATABASE_NAME="$2"
      shift 2
      ;;
    --rabbitmq-user)
      RMQ_USER="$2"
      shift 2
      ;;
    --rabbitmq-password)
      RMQ_PWD="$2"
      shift 2
      ;;
    --minio-user)
      MINIO_USER="$2"
      shift 2
      ;;
    --minio-password)
      MINIO_PWD="$2"
      shift 2
      ;;
    --minio-bucket)
      MINIO_BUCKET_NAME="$2"
      shift 2
      ;;
    --redis-password)
      REDIS_PWD="$2"
      shift 2
      ;;
    --grafana-user)
      GRAFANA_USER="$2"
      shift 2
      ;;
    --grafana-password)
      GRAFANA_PWD="$2"
      shift 2
      ;;
    --jwt-secret)
      JWT_SECRET="$2"
      shift 2
      ;;
    --default-user-email)
      DEFAULT_USER_EMAIL="$2"
      shift 2
      ;;
    --default-user-password)
      DEFAULT_USER_PWD="$2"
      shift 2
      ;;
    --messaging-broker)
      MESSAGING_BROKER="$2"
      shift 2
      ;;
    --cors-origin)
      HTTP_CORS_ALLOWED_ORIGINS="$2"
      shift 2
      ;;
    --skip-minikube-start)
      START_MINIKUBE="false"
      shift
      ;;
    --no-build)
      BUILD_IMAGES="false"
      shift
      ;;
    --no-bucket)
      CREATE_BUCKET="false"
      shift
      ;;
    --port-forward)
      PORT_FORWARD="true"
      shift
      ;;
    --port-forward-services)
      PORT_FORWARD_SERVICES_CSV="$2"
      shift 2
      ;;
    --lite)
      LITE_MODE="true"
      shift
      ;;
    --full-observability)
      LITE_MODE="false"
      shift
      ;;
    --minikube-driver)
      MINIKUBE_DRIVER="$2"
      shift 2
      ;;
    --minikube-cpus)
      MINIKUBE_CPUS="$2"
      shift 2
      ;;
    --minikube-memory)
      MINIKUBE_MEMORY="$2"
      shift 2
      ;;
    --minikube-disk-size)
      MINIKUBE_DISK_SIZE="$2"
      shift 2
      ;;
    --app-values-file)
      APP_HELM_ARGS+=("-f" "$2")
      shift 2
      ;;
    --infra-values-file)
      INFRA_HELM_ARGS+=("-f" "$2")
      shift 2
      ;;
    --app-set)
      APP_HELM_ARGS+=("--set" "$2")
      shift 2
      ;;
    --app-set-string)
      APP_HELM_ARGS+=("--set-string" "$2")
      shift 2
      ;;
    --app-set-file)
      APP_HELM_ARGS+=("--set-file" "$2")
      shift 2
      ;;
    --infra-set)
      INFRA_HELM_ARGS+=("--set" "$2")
      shift 2
      ;;
    --infra-set-string)
      INFRA_HELM_ARGS+=("--set-string" "$2")
      shift 2
      ;;
    --infra-set-file)
      INFRA_HELM_ARGS+=("--set-file" "$2")
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

if [[ ! -f "${QUEUE_SECRETS_FILE}" ]]; then
  echo "ERROR: queue secrets file not found: ${QUEUE_SECRETS_FILE}" >&2
  exit 1
fi

require_command kubectl
require_command helm
require_command bash
if [[ "${BUILD_IMAGES}" == "true" ]]; then
  require_command docker
fi
if [[ "${START_MINIKUBE}" == "true" || "${BUILD_IMAGES}" == "true" ]]; then
  require_command minikube
fi

INFRA_VALUES_FILE="$(mktemp)"
APP_VALUES_FILE="$(mktemp)"
cleanup() {
  rm -f "${INFRA_VALUES_FILE}" "${APP_VALUES_FILE}"
}
trap cleanup EXIT

INFRA_PROMETHEUS_ENABLED="true"
INFRA_GRAFANA_ENABLED="true"
INFRA_OPENSEARCH_ENABLED="true"
INFRA_OPENSEARCH_DASHBOARDS_ENABLED="true"
APP_OTEL_ENABLED="true"
APP_OPENSEARCH_ENABLED="true"

if [[ "${LITE_MODE}" == "true" ]]; then
  INFRA_PROMETHEUS_ENABLED="false"
  INFRA_GRAFANA_ENABLED="false"
  INFRA_OPENSEARCH_ENABLED="false"
  INFRA_OPENSEARCH_DASHBOARDS_ENABLED="false"
  APP_OTEL_ENABLED="false"
  APP_OPENSEARCH_ENABLED="false"
fi

cat > "${INFRA_VALUES_FILE}" <<EOF
postgresql:
  auth:
    username: $(yaml_escape "${PG_USER}")
    password: $(yaml_escape "${PG_PASSWORD}")
    database: $(yaml_escape "${PG_DATABASE_NAME}")
rabbitmq:
  auth:
    username: $(yaml_escape "${RMQ_USER}")
    password: $(yaml_escape "${RMQ_PWD}")
minio:
  auth:
    rootUser: $(yaml_escape "${MINIO_USER}")
    rootPassword: $(yaml_escape "${MINIO_PWD}")
redis:
  auth:
    password: $(yaml_escape "${REDIS_PWD}")
prometheus:
  enabled: ${INFRA_PROMETHEUS_ENABLED}
grafana:
  enabled: ${INFRA_GRAFANA_ENABLED}
  auth:
    adminUser: $(yaml_escape "${GRAFANA_USER}")
    adminPassword: $(yaml_escape "${GRAFANA_PWD}")
opensearch:
  enabled: ${INFRA_OPENSEARCH_ENABLED}
opensearch-dashboards:
  enabled: ${INFRA_OPENSEARCH_DASHBOARDS_ENABLED}
EOF

{
  cat <<EOF
grpcServer:
  image:
    repository: $(yaml_escape "${REGISTRY}/grpc-server")
    tag: $(yaml_escape "${TAG}")
fetchWorker:
  image:
    repository: $(yaml_escape "${REGISTRY}/fetch-worker")
    tag: $(yaml_escape "${TAG}")
parserWorker:
  image:
    repository: $(yaml_escape "${REGISTRY}/parser-worker")
    tag: $(yaml_escape "${TAG}")
exportWorker:
  image:
    repository: $(yaml_escape "${REGISTRY}/export-worker")
    tag: $(yaml_escape "${TAG}")
ui:
  image:
    repository: $(yaml_escape "${REGISTRY}/ui")
    tag: $(yaml_escape "${TAG}")
migrations:
  image:
    repository: $(yaml_escape "${REGISTRY}/grpc-server")
    tag: $(yaml_escape "${TAG}")
config:
  postgres:
    user: $(yaml_escape "${PG_USER}")
    database: $(yaml_escape "${PG_DATABASE_NAME}")
  rabbitmq:
    user: $(yaml_escape "${RMQ_USER}")
  minio:
    bucketName: $(yaml_escape "${MINIO_BUCKET_NAME}")
  auth:
    defaultUserEmail: $(yaml_escape "${DEFAULT_USER_EMAIL}")
  messaging:
    broker: $(yaml_escape "${MESSAGING_BROKER}")
  otel:
    enabled: ${APP_OTEL_ENABLED}
  opensearch:
    enabled: ${APP_OPENSEARCH_ENABLED}
  corsAllowedOrigins: $(yaml_escape "${HTTP_CORS_ALLOWED_ORIGINS}")
secrets:
  postgres:
    password: $(yaml_escape "${PG_PASSWORD}")
  rabbitmq:
    password: $(yaml_escape "${RMQ_PWD}")
  minio:
    user: $(yaml_escape "${MINIO_USER}")
    password: $(yaml_escape "${MINIO_PWD}")
  redis:
    password: $(yaml_escape "${REDIS_PWD}")
  auth:
    jwtSecret: $(yaml_escape "${JWT_SECRET}")
    defaultPassword: $(yaml_escape "${DEFAULT_USER_PWD}")
  queueSecrets:
    content: |
EOF
  write_file_block 6 "${QUEUE_SECRETS_FILE}"
} > "${APP_VALUES_FILE}"

if [[ "${START_MINIKUBE}" == "true" ]]; then
  MINIKUBE_ARGS=(
    start
    "--driver=${MINIKUBE_DRIVER}"
    "--cpus=${MINIKUBE_CPUS}"
    "--memory=${MINIKUBE_MEMORY}"
  )
  if [[ -n "${MINIKUBE_DISK_SIZE}" ]]; then
    MINIKUBE_ARGS+=("--disk-size=${MINIKUBE_DISK_SIZE}")
  fi

  echo "==> Starting minikube..."
  minikube "${MINIKUBE_ARGS[@]}"
fi

kubectl config use-context minikube >/dev/null

if [[ "${BUILD_IMAGES}" == "true" ]]; then
  echo "==> Building images inside minikube..."
  REGISTRY="${REGISTRY}" TAG="${TAG}" bash "${BUILD_SCRIPT}" --minikube
fi

echo "==> Deploying infra release..."
RELEASE_NAME="${INFRA_RELEASE_NAME}" \
NAMESPACE="${INFRA_NAMESPACE}" \
VALUES_ENV="${VALUES_ENV}" \
bash "${DEPLOY_INFRA_SCRIPT}" -f "${INFRA_VALUES_FILE}" "${INFRA_HELM_ARGS[@]+"${INFRA_HELM_ARGS[@]}"}"

echo "==> Deploying app release..."
RELEASE_NAME="${APP_RELEASE_NAME}" \
NAMESPACE="${APP_NAMESPACE}" \
VALUES_ENV="${VALUES_ENV}" \
EXTERNAL_INFRA="true" \
INFRA_RELEASE_NAME="${INFRA_RELEASE_NAME}" \
INFRA_NAMESPACE="${INFRA_NAMESPACE}" \
bash "${DEPLOY_APP_SCRIPT}" -f "${APP_VALUES_FILE}" "${APP_HELM_ARGS[@]+"${APP_HELM_ARGS[@]}"}"

if [[ "${CREATE_BUCKET}" == "true" ]]; then
  bash "${ENSURE_BUCKET_SCRIPT}" \
    --release-name "${INFRA_RELEASE_NAME}" \
    --namespace "${INFRA_NAMESPACE}" \
    --bucket "${MINIO_BUCKET_NAME}" \
    --minio-user "${MINIO_USER}" \
    --minio-password "${MINIO_PWD}"
fi

echo ""
echo "==> Deployment complete."
echo "    App release:   ${APP_RELEASE_NAME} (${APP_NAMESPACE})"
echo "    Infra release: ${INFRA_RELEASE_NAME} (${INFRA_NAMESPACE})"
echo "    UI login:      ${DEFAULT_USER_EMAIL} / ${DEFAULT_USER_PWD}"

if [[ "${PORT_FORWARD}" == "true" ]]; then
  if [[ -n "${PORT_FORWARD_SERVICES_CSV}" ]]; then
    declare -a PORT_FORWARD_SERVICES
    csv_to_array "${PORT_FORWARD_SERVICES_CSV}" PORT_FORWARD_SERVICES
  else
    PORT_FORWARD_SERVICES=(grpc-server ui rabbitmq minio jaeger)
    if [[ "${LITE_MODE}" != "true" ]]; then
      PORT_FORWARD_SERVICES+=(grafana)
    fi
  fi

  echo "==> Starting port-forward..."
  INFRA_NAMESPACE="${INFRA_NAMESPACE}" \
  INFRA_RELEASE="${INFRA_RELEASE_NAME}" \
  APP_NAMESPACE="${APP_NAMESPACE}" \
  APP_RELEASE="${APP_RELEASE_NAME}" \
  zsh "${PORT_FORWARD_SCRIPT}" "${PORT_FORWARD_SERVICES[@]}"
else
  echo "==> To access services locally, run:"
  echo "    INFRA_NAMESPACE=${INFRA_NAMESPACE} INFRA_RELEASE=${INFRA_RELEASE_NAME} APP_NAMESPACE=${APP_NAMESPACE} APP_RELEASE=${APP_RELEASE_NAME} bash deploy/scripts/k8s/port-forward.sh grpc-server ui rabbitmq minio"
fi
