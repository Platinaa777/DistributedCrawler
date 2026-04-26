#!/usr/bin/env bash
# One-command launcher for the distributed-crawler stack in default (single-region) mode.
#
# Modes:
#   local   – run all components as background processes via go run (for development)
#   docker  – run the full stack via Docker Compose
#   k8s     – deploy to minikube via Helm
#
# Usage:
#   ./default_run.sh [--mode local|docker|k8s] [options]
#
# Options (local):
#   --config <path>             API server config file  (default: .env)
#   --worker-config <path>      Worker config file      (default: .worker.env)
#   --build                     Build Go binaries before starting (USE_BINARY=1)
#
# Options (docker):
#   --no-build                  Skip docker build
#   --app-only                  Skip infra startup (infra must already be running)
#   --tag <tag>                 Image tag (default: latest)
#   --registry <name>           Image registry prefix (default: distributed-crawler)
#   --redis-limiter             Use Redis for rate limiting across workers (default: inmemory)
#
# Options (k8s):
#   --no-build                  Skip image build
#   --no-bucket                 Skip MinIO bucket creation
#   --port-forward              Start port-forward after deploy
#   --full-observability        Enable Prometheus/Grafana/OpenSearch
#   --tag <tag>                 Image tag (default: latest)
#   --jwt-secret <value>        JWT secret (recommended: override the default)
#   --pg-password <pwd>         PostgreSQL password
#   --default-user-password <pwd> Default admin UI password
#   --messaging-broker <kind>   rabbitmq | kafka | grpc_memory (default: rabbitmq)
#
# Any extra arguments after -- are forwarded verbatim to the underlying script.
#
# Examples:
#   ./default_run.sh
#   ./default_run.sh --mode docker --no-build
#   ./default_run.sh --mode k8s --jwt-secret supersecret --port-forward
#   ./default_run.sh --mode k8s -- --app-set grpcServer.replicaCount=2
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ── defaults ──────────────────────────────────────────────────────────────────
MODE="local"
CONFIG_PATH=".env"
WORKER_CONFIG=".worker.env"
BUILD_LOCAL=false
PASSTHROUGH=()
REDIS_LIMITER=false

# ── argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)         MODE="$2";          shift 2 ;;
    --config)       CONFIG_PATH="$2";   shift 2 ;;
    --worker-config) WORKER_CONFIG="$2"; shift 2 ;;
    --build)        BUILD_LOCAL=true;   shift ;;
    --redis-limiter) REDIS_LIMITER=true; shift ;;
    --)             shift; PASSTHROUGH+=("$@"); break ;;
    --help|-h)
      sed -n '2,/^set -/{ /^set -/d; s/^# \{0,1\}//; p }' "$0"
      exit 0
      ;;
    *)              PASSTHROUGH+=("$1"); shift ;;
  esac
done

# ── apply flags ───────────────────────────────────────────────────────────────
if [[ "${REDIS_LIMITER}" == true ]]; then
  export LIMITER_TYPE=redis
fi

# ── mode: local ───────────────────────────────────────────────────────────────
run_local() {
  if [[ "${BUILD_LOCAL}" == true ]]; then
    echo "==> Building binaries..."
    bash "${SCRIPT_DIR}/local/build.sh"
    export USE_BINARY=1
  fi

  CONFIG_PATH="${CONFIG_PATH}" \
  WORKER_CONFIG="${WORKER_CONFIG}" \
  bash "${SCRIPT_DIR}/local/start-all.sh" "${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}"
}

# ── mode: docker ──────────────────────────────────────────────────────────────
run_docker() {
  bash "${SCRIPT_DIR}/docker/deploy-all.sh" "${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}"
}

# ── mode: k8s ─────────────────────────────────────────────────────────────────
run_k8s() {
  bash "${SCRIPT_DIR}/k8s/launch-minikube.sh" "${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}"
}

# ── dispatch ──────────────────────────────────────────────────────────────────
echo "==> default_run.sh  mode=${MODE}"

case "${MODE}" in
  local)  run_local  ;;
  docker) run_docker ;;
  k8s)    run_k8s    ;;
  *)
    echo "ERROR: Unknown mode '${MODE}'. Use local, docker, or k8s." >&2
    exit 1
    ;;
esac
