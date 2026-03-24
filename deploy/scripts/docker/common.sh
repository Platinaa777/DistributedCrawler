#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

INFRA_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.app.yaml"

ALL_APP_SERVICES=(grpc-server fetch-worker parser-worker export-worker ui)

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

trim_value() {
  local value="${1:-}"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

array_contains() {
  local needle="$1"
  shift

  local item
  for item in "$@"; do
    if [[ "${item}" == "${needle}" ]]; then
      return 0
    fi
  done

  return 1
}

append_unique() {
  local -n arr_ref="$1"
  local value="$2"

  if ! array_contains "${value}" "${arr_ref[@]}"; then
    arr_ref+=("${value}")
  fi
}

normalize_broker() {
  local broker
  broker="$(trim_value "${1:-rabbitmq}")"
  broker="${broker,,}"

  case "${broker}" in
    kafka|grpc_memory)
      printf '%s' "${broker}"
      ;;
    *)
      printf '%s' "rabbitmq"
      ;;
  esac
}

normalize_fetcher_type() {
  local fetcher_type
  fetcher_type="$(trim_value "${1:-http}")"
  fetcher_type="${fetcher_type,,}"

  if [[ "${fetcher_type}" == "browser" ]]; then
    printf '%s' "browser"
    return
  fi

  printf '%s' "http"
}

parse_csv_into_array() {
  local csv="$1"
  local -n out_ref="$2"
  local old_ifs
  local raw_items=()
  local item

  old_ifs="$IFS"
  IFS=','
  read -r -a raw_items <<< "${csv}"
  IFS="${old_ifs}"

  for item in "${raw_items[@]}"; do
    item="$(trim_value "${item}")"
    if [[ -n "${item}" ]]; then
      out_ref+=("${item}")
    fi
  done
}

validate_app_service() {
  local service="$1"

  if ! array_contains "${service}" "${ALL_APP_SERVICES[@]}"; then
    echo "ERROR: Unknown app service '${service}'." >&2
    echo "Valid: ${ALL_APP_SERVICES[*]}" >&2
    exit 1
  fi
}

resolve_launch_selection() {
  local requested_services=()
  local ordered_services=()
  local service

  if [[ -n "${APP_COMPONENTS_CSV:-}" ]]; then
    parse_csv_into_array "${APP_COMPONENTS_CSV}" requested_services
  elif [[ -n "${APP_COMPONENTS:-}" ]]; then
    parse_csv_into_array "${APP_COMPONENTS}" requested_services
  fi

  if [[ "${#requested_services[@]}" -eq 0 ]]; then
    requested_services=("${ALL_APP_SERVICES[@]}")
  fi

  for service in "${requested_services[@]}"; do
    validate_app_service "${service}"
  done

  SELECTED_APP_SERVICES=()
  for service in "${requested_services[@]}"; do
    append_unique SELECTED_APP_SERVICES "${service}"
  done

  if array_contains "fetch-worker" "${SELECTED_APP_SERVICES[@]}" || \
     array_contains "parser-worker" "${SELECTED_APP_SERVICES[@]}" || \
     array_contains "export-worker" "${SELECTED_APP_SERVICES[@]}" || \
     array_contains "ui" "${SELECTED_APP_SERVICES[@]}"; then
    append_unique SELECTED_APP_SERVICES "grpc-server"
  fi

  ordered_services=()
  for service in "${ALL_APP_SERVICES[@]}"; do
    if array_contains "${service}" "${SELECTED_APP_SERVICES[@]}"; then
      ordered_services+=("${service}")
    fi
  done
  SELECTED_APP_SERVICES=("${ordered_services[@]}")

  SELECTED_INFRA_SERVICES=(
    pg
    minio
    redis
    jaeger
    otel-collector
    opensearch
    opensearch-dashboards
    prometheus
    grafana
    redisinsight
  )

  case "$(normalize_broker "${MESSAGING_BROKER:-rabbitmq}")" in
    kafka)
      append_unique SELECTED_INFRA_SERVICES "zookeeper"
      append_unique SELECTED_INFRA_SERVICES "kafka"
      append_unique SELECTED_INFRA_SERVICES "kafka-ui"
      ;;
    grpc_memory)
      ;;
    *)
      append_unique SELECTED_INFRA_SERVICES "rabbitmq"
      ;;
  esac

  if array_contains "fetch-worker" "${SELECTED_APP_SERVICES[@]}" && \
     [[ "$(normalize_fetcher_type "${FETCHER_TYPE:-http}")" == "browser" ]]; then
    append_unique SELECTED_INFRA_SERVICES "chrome"
  fi
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

wait_for_compose_health() {
  local service="${1}"
  local timeout="${2:-120}"
  local start_ts
  local container_id
  local health_status

  echo "==> Waiting for ${service} health..."
  start_ts="$(date +%s)"

  while true; do
    container_id="$(compose_stack ps -q "${service}" 2>/dev/null | head -n 1)"

    if [[ -n "${container_id}" ]]; then
      health_status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${container_id}" 2>/dev/null || true)"

      case "${health_status}" in
        healthy|none)
          return 0
          ;;
        unhealthy)
          echo "ERROR: Service '${service}' became unhealthy" >&2
          return 1
          ;;
      esac
    fi

    if (( $(date +%s) - start_ts >= timeout )); then
      echo "ERROR: Timed out waiting for ${service} health" >&2
      return 1
    fi

    sleep 2
  done
}

wait_for_core_infra() {
  wait_for_tcp "PostgreSQL" "127.0.0.1" "${PG_PORT:-54322}"
  wait_for_compose_health "pg"
  wait_for_tcp "MinIO" "127.0.0.1" "9000"
  wait_for_tcp "Redis" "127.0.0.1" "6379"
  wait_for_compose_health "redis"

  case "$(normalize_broker "${MESSAGING_BROKER:-rabbitmq}")" in
    kafka)
      wait_for_tcp "Kafka" "127.0.0.1" "9091"
      wait_for_compose_health "kafka"
      ;;
    grpc_memory)
      ;;
    *)
      wait_for_tcp "RabbitMQ" "127.0.0.1" "5672"
      wait_for_compose_health "rabbitmq"
      ;;
  esac
}

wait_for_optional_services() {
  if [[ "$(normalize_fetcher_type "${FETCHER_TYPE:-http}")" == "browser" ]] && \
     array_contains "fetch-worker" "${SELECTED_APP_SERVICES[@]:-}"; then
    wait_for_tcp "Chrome" "127.0.0.1" "9222"
  fi
}

wait_for_grpc_server() {
  wait_for_tcp "gRPC server" "127.0.0.1" "${GRPC_PORT:-8083}"
  wait_for_tcp "HTTP gateway" "127.0.0.1" "${HTTP_PORT:-8084}"
}
