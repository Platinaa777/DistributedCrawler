#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy-all.sh"

usage() {
  cat <<'EOF'
Usage:
  ./launch.sh [options]

Simple Docker entrypoint for local development. It exports the required env
variables and then runs deploy-all.sh.

Common options:
  --registry <name>               Image prefix. Default: distributed-crawler
  --tag <tag>                     Image tag. Default: latest
  --components <csv>              App services to launch. Default: grpc-server,fetch-worker,parser-worker,export-worker,ui
  --component <name>              App service to launch, may be repeated
  --pg-user <name>                PostgreSQL user. Default: crawler
  --pg-password <pwd>             PostgreSQL password. Default: some-pwd-123
  --pg-database <name>            PostgreSQL database. Default: crawler
  --pg-port <port>                PostgreSQL host port. Default: 54322
  --rabbitmq-user <name>          RabbitMQ user. Default: guest
  --rabbitmq-password <pwd>       RabbitMQ password. Default: guest
  --minio-user <name>             MinIO user. Default: minioadmin
  --minio-password <pwd>          MinIO password. Default: minioadmin
  --minio-bucket <name>           MinIO bucket name. Default: pages
  --redis-password <pwd>          Redis password. Default: some_redis_pwd_123
  --grafana-user <name>           Grafana admin user. Default: admin
  --grafana-password <pwd>        Grafana admin password. Default: changeme-grafana-password
  --jwt-secret <value>            JWT secret for the API
  --default-user-email <email>    Default admin email. Default: admin@example.com
  --default-user-password <pwd>   Default admin password. Default: 12345678
  --messaging-broker <kind>       rabbitmq | kafka | grpc_memory. Default: rabbitmq
  --cors-origin <origin>          HTTP_CORS_ALLOWED_ORIGINS value. Default: http://localhost:4200
  --queue-secrets-file <path>     Host path mounted as /etc/crawler/queue-secrets.json
  --app-only                      Skip infra startup
  --no-build                      Skip docker image build
  --env <KEY=VALUE>               Extra environment variable, may be repeated
  --compose-arg <arg>             Extra argument passed to docker compose up, may be repeated
  --help                          Show this message

Examples:
  ./launch.sh --pg-password mypwd --jwt-secret supersecret --default-user-password admin123
  ./launch.sh --no-build --compose-arg=--scale --compose-arg=fetch-worker=3
EOF
}

REGISTRY="distributed-crawler"
TAG="latest"
APP_COMPONENTS=()
APP_COMPONENTS_CSV=""
PG_USER="crawler"
PG_PASSWORD="some-pwd-123"
PG_DATABASE_NAME="crawler"
PG_PORT="54322"
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
QUEUE_SECRETS_HOST_PATH="${PROJECT_ROOT}/queue-secrets.json.example"
APP_ONLY="false"
NO_BUILD="false"
EXTRA_ENV=()
COMPOSE_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --tag)
      TAG="$2"
      shift 2
      ;;
    --components)
      IFS=',' read -r -a parsed_components <<< "$2"
      for component in "${parsed_components[@]}"; do
        component="${component#"${component%%[![:space:]]*}"}"
        component="${component%"${component##*[![:space:]]}"}"
        if [[ -n "${component}" ]]; then
          APP_COMPONENTS+=("${component}")
        fi
      done
      shift 2
      ;;
    --component)
      APP_COMPONENTS+=("$2")
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
    --pg-port)
      PG_PORT="$2"
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
    --queue-secrets-file)
      QUEUE_SECRETS_HOST_PATH="$2"
      shift 2
      ;;
    --env)
      EXTRA_ENV+=("$2")
      shift 2
      ;;
    --compose-arg)
      COMPOSE_ARGS+=("$2")
      shift 2
      ;;
    --app-only)
      APP_ONLY="true"
      shift
      ;;
    --no-build)
      NO_BUILD="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    --*=*)
      key="${1%%=*}"
      value="${1#*=}"
      case "${key}" in
        --components)
          IFS=',' read -r -a parsed_components <<< "${value}"
          for component in "${parsed_components[@]}"; do
            component="${component#"${component%%[![:space:]]*}"}"
            component="${component%"${component##*[![:space:]]}"}"
            if [[ -n "${component}" ]]; then
              APP_COMPONENTS+=("${component}")
            fi
          done
          ;;
        --component)
          APP_COMPONENTS+=("${value}")
          ;;
        --compose-arg)
          COMPOSE_ARGS+=("${value}")
          ;;
        --env)
          EXTRA_ENV+=("${value}")
          ;;
        *)
          echo "ERROR: Unknown argument '${1}'." >&2
          usage >&2
          exit 1
          ;;
      esac
      shift
      ;;
    *)
      echo "ERROR: Unknown argument '${1}'." >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ ! -f "${QUEUE_SECRETS_HOST_PATH}" ]]; then
  echo "ERROR: queue secrets file not found: ${QUEUE_SECRETS_HOST_PATH}" >&2
  exit 1
fi

for kv in "${EXTRA_ENV[@]}"; do
  if [[ "${kv}" != *=* ]]; then
    echo "ERROR: --env expects KEY=VALUE, got '${kv}'." >&2
    exit 1
  fi
  export "${kv}"
done

export REGISTRY TAG APP_ONLY NO_BUILD
if [[ "${#APP_COMPONENTS[@]}" -gt 0 ]]; then
  APP_COMPONENTS_CSV="$(IFS=,; printf '%s' "${APP_COMPONENTS[*]}")"
fi
export APP_COMPONENTS_CSV
export PG_USER PG_PASSWORD PG_DATABASE_NAME PG_PORT
export RMQ_USER RMQ_PWD
export MINIO_USER MINIO_PWD MINIO_BUCKET_NAME
export REDIS_PWD
export GRAFANA_USER GRAFANA_PWD
export JWT_SECRET DEFAULT_USER_EMAIL DEFAULT_USER_PWD
export MESSAGING_BROKER HTTP_CORS_ALLOWED_ORIGINS
export QUEUE_SECRETS_HOST_PATH

echo "==> Launching Docker stack with argument-driven configuration..."
bash "${DEPLOY_SCRIPT}" "${COMPOSE_ARGS[@]}"
