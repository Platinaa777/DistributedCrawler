#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

INFRA_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.app.yaml"

INFRA_COMPOSE_ARGS=(-f "${INFRA_COMPOSE_FILE}")
APP_STACK_COMPOSE_ARGS=(-f "${INFRA_COMPOSE_FILE}" -f "${APP_COMPOSE_FILE}")

load_project_env() {
  local env_file="${PROJECT_ROOT}/.env"
  local line key value

  if [[ ! -f "${env_file}" ]]; then
    return 0
  fi

  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line%$'\r'}"

    if [[ -z "${line//[[:space:]]/}" || "${line}" =~ ^[[:space:]]*# ]]; then
      continue
    fi

    if [[ "${line}" != *=* ]]; then
      continue
    fi

    key="${line%%=*}"
    value="${line#*=}"

    key="${key//[[:space:]]/}"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"

    if [[ "${value}" == *" #"* ]]; then
      value="${value%% \#*}"
      value="${value%"${value##*[![:space:]]}"}"
    fi

    if [[ -z "${key}" || -n "${!key:-}" ]]; then
      continue
    fi

    export "${key}=${value}"
  done < "${env_file}"
}

compose_infra() {
  docker compose "${INFRA_COMPOSE_ARGS[@]}" "$@"
}

compose_stack() {
  docker compose "${APP_STACK_COMPOSE_ARGS[@]}" "$@"
}

validate_compose_config() {
  echo "==> Validating Docker Compose configuration..."
  compose_stack config > /dev/null
}

wait_for_tcp() {
  local name="${1}"
  local host="${2}"
  local port="${3}"
  local timeout="${4:-120}"
  local start_ts

  echo "==> Waiting for ${name} on ${host}:${port}..."
  start_ts="$(date +%s)"

  while true; do
    if bash -c "exec 3<>/dev/tcp/${host}/${port}" >/dev/null 2>&1; then
      return 0
    fi

    if (( $(date +%s) - start_ts >= timeout )); then
      echo "ERROR: Timed out waiting for ${name} on ${host}:${port}" >&2
      return 1
    fi

    sleep 2
  done
}

wait_for_core_infra() {
  wait_for_tcp "PostgreSQL" "127.0.0.1" "${PG_PORT:-54322}"
  wait_for_tcp "RabbitMQ" "127.0.0.1" "5672"
  wait_for_tcp "MinIO" "127.0.0.1" "9000"
  wait_for_tcp "Redis" "127.0.0.1" "6379"
}

wait_for_grpc_server() {
  wait_for_tcp "gRPC server" "127.0.0.1" "${GRPC_PORT:-8083}"
  wait_for_tcp "HTTP gateway" "127.0.0.1" "${HTTP_PORT:-8084}"
}
