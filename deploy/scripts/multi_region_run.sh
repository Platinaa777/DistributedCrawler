#!/usr/bin/env bash
# One-command launcher for the distributed-crawler stack with regional fetch workers.
#
# Each region gets its own pool of fetch workers. The queue connection details
# (broker URL, queue name) for each regional pool must be provided entirely
# through env vars or the worker config file — there is no database-driven
# queue endpoint discovery at runtime.
#
# Parser workers always run in a single shared pool and are not region-aware.
#
# Modes:
#   local   – run base stack as background processes, then one fetch-worker
#             process per region (each with WORKER_REGION set)
#   docker  – start base stack via Docker Compose (no fetch-workers), then
#             launch one fetch-worker container per region
#   k8s     – deploy via minikube + Helm with fetchWorker.regions set
#
# Usage:
#   ./multi_region_run.sh --regions <csv> [--mode local|docker|k8s] [options]
#
# Required:
#   --regions <csv>             Comma-separated region names, e.g. us-east,eu-west
#
# Options (all modes):
#   --mode <local|docker|k8s>   Deployment mode (default: docker)
#
# Options (local):
#   --config <path>             API server config file  (default: .env)
#   --worker-config <path>      Worker config file for non-fetch workers (default: .worker.env)
#   --build                     Build Go binaries before starting (USE_BINARY=1)
#
# Options (docker):
#   --no-build                  Skip docker build
#   --tag <tag>                 Image tag (default: latest)
#   --registry <name>           Image registry prefix (default: distributed-crawler)
#
# Options (k8s):
#   --no-build                  Skip image build
#   --no-bucket                 Skip MinIO bucket creation
#   --port-forward              Start port-forward after deploy
#   --full-observability        Enable Prometheus/Grafana/OpenSearch
#   --tag <tag>                 Image tag (default: latest)
#   --jwt-secret <value>        JWT secret
#   --pg-password <pwd>         PostgreSQL password
#   --default-user-password <pwd> Default admin UI password
#   --messaging-broker <kind>   rabbitmq | kafka | grpc_memory (default: rabbitmq)
#
# Any extra arguments after -- are forwarded verbatim to the underlying script.
#
# Examples:
#   ./multi_region_run.sh --regions us-east,eu-west
#   ./multi_region_run.sh --regions us-east,eu-west --mode docker --no-build
#   ./multi_region_run.sh --regions us-east,eu-west --mode k8s --port-forward
#   ./multi_region_run.sh --regions us-east,eu-west --mode k8s -- --jwt-secret supersecret
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

INFRA_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yaml"
APP_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.app.yaml"

# ── defaults ──────────────────────────────────────────────────────────────────
MODE="docker"
REGIONS_CSV=""
CONFIG_PATH=".env"
WORKER_CONFIG=".worker.env"
BUILD_LOCAL=false
PASSTHROUGH=()

# ── argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --regions)       REGIONS_CSV="$2";    shift 2 ;;
    --mode)          MODE="$2";           shift 2 ;;
    --config)        CONFIG_PATH="$2";    shift 2 ;;
    --worker-config) WORKER_CONFIG="$2";  shift 2 ;;
    --build)         BUILD_LOCAL=true;    shift ;;
    --)              shift; PASSTHROUGH+=("$@"); break ;;
    --help|-h)
      sed -n '2,/^set -/{ /^set -/d; s/^# \{0,1\}//; p }' "$0"
      exit 0
      ;;
    *)               PASSTHROUGH+=("$1"); shift ;;
  esac
done

if [[ -z "${REGIONS_CSV}" ]]; then
  echo "ERROR: --regions is required. Example: --regions us-east,eu-west" >&2
  exit 1
fi

# Parse regions CSV into array
IFS=',' read -r -a REGIONS <<< "${REGIONS_CSV}"
trimmed=()
for r in "${REGIONS[@]}"; do
  r="${r#"${r%%[![:space:]]*}"}"
  r="${r%"${r##*[![:space:]]}"}"
  [[ -n "${r}" ]] && trimmed+=("${r}")
done
REGIONS=("${trimmed[@]}")

if [[ "${#REGIONS[@]}" -eq 0 ]]; then
  echo "ERROR: --regions produced an empty list after parsing '${REGIONS_CSV}'" >&2
  exit 1
fi

echo "==> multi_region_run.sh  mode=${MODE}  regions=(${REGIONS[*]})"

# ── mode: local ───────────────────────────────────────────────────────────────
run_local() {
  if [[ "${BUILD_LOCAL}" == true ]]; then
    echo "==> Building binaries..."
    bash "${SCRIPT_DIR}/local/build.sh"
    export USE_BINARY=1
  fi

  LOGS_DIR="${PROJECT_ROOT}/logs"
  PIDS_DIR="${PROJECT_ROOT}/.pids"
  BIN_DIR="${BIN_DIR:-${PROJECT_ROOT}/bin}"
  mkdir -p "${LOGS_DIR}" "${PIDS_DIR}"

  start_bg() {
    local name="$1"
    shift
    local logfile="${LOGS_DIR}/${name}.log"
    local pidfile="${PIDS_DIR}/${name}.pid"
    if [[ -f "${pidfile}" ]] && kill -0 "$(cat "${pidfile}")" 2>/dev/null; then
      echo "==> ${name} already running (PID $(cat "${pidfile}"))"
      return
    fi
    "$@" >> "${logfile}" 2>&1 &
    echo $! > "${pidfile}"
    echo "==> Started ${name} (PID $!)"
    echo "    log: ${logfile}"
  }

  run_component() {
    local name="$1"; shift
    if [[ "${USE_BINARY:-0}" == "1" ]]; then
      start_bg "${name}" "${BIN_DIR}/${name}" "$@"
    else
      start_bg "${name}" go run "${PROJECT_ROOT}/cmd/${name}/main.go" "$@"
    fi
  }

  # Start shared components (no fetch-worker here)
  run_component grpc_server      --config-path="${CONFIG_PATH}"
  run_component parser_worker    --worker-config-path="${WORKER_CONFIG}"
  run_component export_worker    --worker-config-path="${WORKER_CONFIG}"
  run_component scheduler_worker --worker-config-path="${WORKER_CONFIG}"

  # Start one fetch-worker per region
  for region in "${REGIONS[@]}"; do
    name="fetch-worker-${region}"
    if [[ "${USE_BINARY:-0}" == "1" ]]; then
      WORKER_REGION="${region}" start_bg "${name}" \
        "${BIN_DIR}/fetch_worker" --worker-config-path="${WORKER_CONFIG}"
    else
      WORKER_REGION="${region}" start_bg "${name}" \
        go run "${PROJECT_ROOT}/cmd/fetch_worker/main.go" --worker-config-path="${WORKER_CONFIG}"
    fi
  done

  echo ""
  echo "==> All components started."
  echo "    Regions : ${REGIONS[*]}"
  echo "    Logs    : ${LOGS_DIR}/"
  echo "    Stop    : ${SCRIPT_DIR}/local/stop-all.sh"
  echo ""
  echo "==> Access points:"
  echo "    UI                 http://localhost:18080"
  echo "    HTTP API gateway   http://localhost:8084"
  echo "    gRPC API           localhost:8083"
  echo "    Swagger UI         http://localhost:8084/swagger-ui"
  echo "    MinIO console      http://localhost:9001"
  echo "    RabbitMQ UI        http://localhost:15672"
}

# ── mode: docker ──────────────────────────────────────────────────────────────
run_docker() {
  if [[ ! -f "${APP_COMPOSE_FILE}" ]]; then
    echo "ERROR: Docker Compose app file not found: ${APP_COMPOSE_FILE}" >&2
    exit 1
  fi

  COMPOSE_FILES=(-f "${INFRA_COMPOSE_FILE}" -f "${APP_COMPOSE_FILE}")

  # Start base stack without fetch-workers so each region gets its own container
  echo "==> Starting base stack (grpc-server, parser-worker, export-worker, ui)..."
  APP_COMPONENTS_CSV="grpc-server,parser-worker,export-worker,ui" \
  bash "${SCRIPT_DIR}/docker/deploy-all.sh" "${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}"

  # Start one detached fetch-worker container per region
  for region in "${REGIONS[@]}"; do
    echo "==> Starting fetch-worker for region: ${region}"
    docker compose "${COMPOSE_FILES[@]}" \
      run --detach --no-deps \
      -e WORKER_REGION="${region}" \
      fetch-worker
  done

  echo ""
  echo "==> Multi-region fetch workers started."
  echo "    Regions      : ${REGIONS[*]}"
  echo "    Running containers:"
  docker compose "${COMPOSE_FILES[@]}" ps
  echo ""
  echo "==> Access points:"
  echo "    UI                 http://localhost:18080"
  echo "    HTTP API gateway   http://localhost:8084"
  echo "    gRPC API           localhost:8083"
  echo "    Swagger UI         http://localhost:8084/swagger-ui"
  echo "    MinIO console      http://localhost:9001"
  echo "    RabbitMQ UI        http://localhost:15672"
}

# ── mode: k8s ─────────────────────────────────────────────────────────────────
run_k8s() {
  # Build helm --set-string value for fetchWorker.regions as YAML sequence:
  # e.g. regions=["us-east","eu-west"]  →  --set 'fetchWorker.regions={us-east,eu-west}'
  local regions_helm
  regions_helm="{$(IFS=','; echo "${REGIONS[*]}")}"

  echo "==> Deploying to k8s with fetchWorker.regions=${regions_helm}"

  bash "${SCRIPT_DIR}/k8s/launch-minikube.sh" \
    --app-set "fetchWorker.regions=${regions_helm}" \
    "${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}"

  echo ""
  echo "==> Access points (requires port-forward or minikube tunnel):"
  echo "    UI                 http://localhost:18080"
  echo "    HTTP API gateway   http://localhost:8084"
  echo "    gRPC API           localhost:8083"
  echo "    Swagger UI         http://localhost:8084/swagger-ui"
  echo "    MinIO console      http://localhost:9001"
  echo "    RabbitMQ UI        http://localhost:15672"
}

# ── dispatch ──────────────────────────────────────────────────────────────────
case "${MODE}" in
  local)  run_local  ;;
  docker) run_docker ;;
  k8s)    run_k8s    ;;
  *)
    echo "ERROR: Unknown mode '${MODE}'. Use local, docker, or k8s." >&2
    exit 1
    ;;
esac
