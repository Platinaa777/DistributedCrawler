#!/bin/zsh
# Open kubectl port-forwards for infrastructure and application services.
# All forwards run in the background; Ctrl-C kills them all cleanly.
#
# Usage:
#   ./port-forward.sh                              # forward everything
#   ./port-forward.sh postgresql rabbitmq minio    # selected services only
#   ./port-forward.sh grpc-server jaeger           # mix of app + infra
#
# Available services:
#   Infra:  postgresql  rabbitmq  minio  redis  redisinsight
#           jaeger  prometheus  grafana  opensearch  opensearch-dashboards
#   App:    grpc-server  ui
#
# Environment variables:
#   INFRA_NAMESPACE  – namespace for infra release (default: infra)
#   INFRA_RELEASE    – infra Helm release name    (default: infra)
#   APP_NAMESPACE    – namespace for app release   (default: crawler)
#   APP_RELEASE      – app Helm release name       (default: crawler)
set -euo pipefail

INFRA_NS="${INFRA_NAMESPACE:-infra}"
INFRA_REL="${INFRA_RELEASE:-infra}"
APP_NS="${APP_NAMESPACE:-crawler}"
APP_REL="${APP_RELEASE:-crawler}"
APP_CHART="${APP_CHART_NAME:-distributed-crawler}"
APP_FULL="${APP_REL}-${APP_CHART}"

# ---- Service registry -------------------------------------------------------
# svc_ns <alias>    → namespace
# svc_name <alias>  → k8s service name
# svc_ports <alias> → space-separated "local:remote" mappings

svc_ns() {
  case "$1" in
    postgresql|rabbitmq|minio|redis|redisinsight|jaeger|prometheus|grafana|opensearch|opensearch-dashboards)
      echo "${INFRA_NS}" ;;
    grpc-server|ui)
      echo "${APP_NS}" ;;
    *) return 1 ;;
  esac
}

svc_name() {
  case "$1" in
    postgresql)              echo "${INFRA_REL}-postgresql" ;;
    rabbitmq)                echo "${INFRA_REL}-rabbitmq" ;;
    minio)                   echo "${INFRA_REL}-minio" ;;
    redis)                   echo "${INFRA_REL}-redis-master" ;;
    redisinsight)            echo "${INFRA_REL}-redisinsight" ;;
    jaeger)                  echo "${INFRA_REL}-jaeger-query" ;;
    prometheus)              echo "${INFRA_REL}-prometheus" ;;
    grafana)                 echo "${INFRA_REL}-grafana" ;;
    opensearch)              echo "${INFRA_REL}-opensearch" ;;
    opensearch-dashboards)   echo "${INFRA_REL}-opensearch-dashboards" ;;
    grpc-server)             echo "${APP_FULL}-grpc-server" ;;
    ui)                      echo "${APP_FULL}-ui" ;;
    *) return 1 ;;
  esac
}

svc_ports() {
  case "$1" in
    postgresql)              echo "54322:5432" ;;
    rabbitmq)                echo "5672:5672 15672:15672" ;;
    minio)                   echo "9000:9000 9001:9001" ;;
    redis)                   echo "6379:6379" ;;
    redisinsight)            echo "8001:5540" ;;
    jaeger)                  echo "16686:16686" ;;
    prometheus)              echo "9090:9090" ;;
    grafana)                 echo "3000:3000" ;;
    opensearch)              echo "9200:9200" ;;
    opensearch-dashboards)   echo "5601:5601" ;;
    grpc-server)             echo "8083:8083 8084:8084" ;;
    ui)                      echo "4200:8080" ;;
    *) return 1 ;;
  esac
}

ALL_ALIASES=(
  postgresql rabbitmq minio redis redisinsight
  jaeger prometheus grafana opensearch opensearch-dashboards
  grpc-server ui
)

# ---- Select which services to forward ---------------------------------------
if [[ $# -gt 0 ]]; then
  SELECTED=("$@")
  for alias in "${SELECTED[@]}"; do
    if ! svc_ns "${alias}" >/dev/null 2>&1; then
      echo "ERROR: Unknown service '${alias}'." >&2
      echo "Available: ${ALL_ALIASES[*]}" >&2
      exit 1
    fi
  done
else
  SELECTED=("${ALL_ALIASES[@]}")
fi

# ---- Start port-forwards with auto-restart ----------------------------------
PIDS=()

cleanup() {
  echo ""
  echo "==> Stopping port-forwards..."
  for pid in "${PIDS[@]+"${PIDS[@]}"}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Runs kubectl port-forward in a retry loop so transient failures
# (pod not ready, connection reset) don't kill the whole session.
forward_with_retry() {
  local alias="$1" ns="$2" svc="$3"
  shift 3
  local ports=("$@")
  while true; do
    kubectl port-forward --address=0.0.0.0 -n "${ns}" "svc/${svc}" "${ports[@]}" 2>&1 \
      | sed "s/^/[${alias}] /" || true
    echo "[${alias}] port-forward exited, retrying in 5s..." >&2
    sleep 5
  done
}

for alias in "${SELECTED[@]}"; do
  ns="$(svc_ns "${alias}")"
  svc="$(svc_name "${alias}")"
  # shellcheck disable=SC2206
  ports=($(svc_ports "${alias}"))

  echo "==> Forwarding ${alias} (${ns}/${svc}): ${ports[*]}"
  forward_with_retry "${alias}" "${ns}" "${svc}" "${ports[@]}" &
  PIDS+=($!)
done

echo ""
echo "==> All port-forwards active. Press Ctrl-C to stop."
echo ""
echo "  Service                 Local URL"
echo "  ----------------------  ----------------------------------"

for alias in "${SELECTED[@]}"; do
  # shellcheck disable=SC2206
  ports=($(svc_ports "${alias}"))
  for mapping in "${ports[@]}"; do
    local_port="${mapping%%:*}"
    case "${local_port}" in
      5672)   printf "  %-26s %s\n" "rabbitmq (AMQP)"        "localhost:${local_port}" ;;
      15672)  printf "  %-26s %s\n" "rabbitmq UI"             "http://localhost:${local_port}  (guest/guest)" ;;
      9000)   printf "  %-26s %s\n" "minio (API)"             "localhost:${local_port}" ;;
      9001)   printf "  %-26s %s\n" "minio (console)"         "http://localhost:${local_port}  (minioadmin/minioadmin)" ;;
      54322)  printf "  %-26s %s\n" "postgresql"              "localhost:${local_port}" ;;
      6379)   printf "  %-26s %s\n" "redis"                   "localhost:${local_port}" ;;
      8001)   printf "  %-26s %s\n" "redisinsight"            "http://localhost:${local_port}" ;;
      16686)  printf "  %-26s %s\n" "jaeger UI"               "http://localhost:${local_port}" ;;
      9090)   printf "  %-26s %s\n" "prometheus"              "http://localhost:${local_port}" ;;
      3000)   printf "  %-26s %s\n" "grafana"                 "http://localhost:${local_port}  (admin/changeme-grafana-password)" ;;
      9200)   printf "  %-26s %s\n" "opensearch"              "http://localhost:${local_port}" ;;
      5601)   printf "  %-26s %s\n" "opensearch dashboards"   "http://localhost:${local_port}" ;;
      4200)   printf "  %-26s %s\n" "admin UI"                "http://localhost:${local_port}" ;;
      8083)   printf "  %-26s %s\n" "grpc API"                "localhost:${local_port}" ;;
      8084)   printf "  %-26s %s\n" "http gateway"            "http://localhost:${local_port}" ;;
    esac
  done
done

wait
