#!/usr/bin/env bash
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
# Each entry: "alias namespace svc-name port1:targetPort1 [port2:targetPort2 ...]"
#
declare -A SVC_NS SVC_NAME SVC_PORTS

register() {
  local alias="$1" ns="$2" svc="$3"
  shift 3
  SVC_NS[$alias]="$ns"
  SVC_NAME[$alias]="$svc"
  SVC_PORTS[$alias]="$*"
}

register postgresql            "${INFRA_NS}" "${INFRA_REL}-postgresql"              54322:5432
register rabbitmq              "${INFRA_NS}" "${INFRA_REL}-rabbitmq"                5672:5672 15672:15672
register minio                 "${INFRA_NS}" "${INFRA_REL}-minio"                   9000:9000 9001:9001
register redis                 "${INFRA_NS}" "${INFRA_REL}-redis-master"            6379:6379
register redisinsight          "${INFRA_NS}" "${INFRA_REL}-redisinsight"            8001:8001
register jaeger                "${INFRA_NS}" "${INFRA_REL}-jaeger-query"            16686:16686
register prometheus            "${INFRA_NS}" "${INFRA_REL}-prometheus"              9090:9090
register grafana               "${INFRA_NS}" "${INFRA_REL}-grafana"                 3000:3000
register opensearch            "${INFRA_NS}" "${INFRA_REL}-opensearch"              9200:9200
register opensearch-dashboards "${INFRA_NS}" "${INFRA_REL}-opensearch-dashboards"   5601:5601
register grpc-server           "${APP_NS}"   "${APP_FULL}-grpc-server"              8083:8083 8084:8084
register ui                    "${APP_NS}"   "${APP_FULL}-ui"                        4200:8080

ALL_ALIASES=(
  postgresql rabbitmq minio redis redisinsight
  jaeger prometheus grafana opensearch opensearch-dashboards
  grpc-server ui
)

# ---- Select which services to forward ---------------------------------------
if [[ $# -gt 0 ]]; then
  SELECTED=("$@")
  for alias in "${SELECTED[@]}"; do
    if [[ -z "${SVC_NS[$alias]+_}" ]]; then
      echo "ERROR: Unknown service '${alias}'." >&2
      echo "Available: ${ALL_ALIASES[*]}" >&2
      exit 1
    fi
  done
else
  SELECTED=("${ALL_ALIASES[@]}")
fi

# ---- Start port-forwards in background --------------------------------------
PIDS=()

cleanup() {
  echo ""
  echo "==> Stopping port-forwards..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for alias in "${SELECTED[@]}"; do
  ns="${SVC_NS[$alias]}"
  svc="${SVC_NAME[$alias]}"
  # shellcheck disable=SC2206
  ports=(${SVC_PORTS[$alias]})

  echo "==> Forwarding ${alias} (${ns}/${svc}): ${ports[*]}"
  kubectl port-forward --address=0.0.0.0 -n "${ns}" "svc/${svc}" "${ports[@]}" &>/dev/null &
  PIDS+=($!)
done

echo ""
echo "==> All port-forwards active. Press Ctrl-C to stop."
echo ""
echo "  Service                 Local URL"
echo "  ----------------------  ----------------------------------"

print_url() {
  local alias="$1"
  # shellcheck disable=SC2206
  local ports=(${SVC_PORTS[$alias]})
  local first_port="${ports[0]%%:*}"   # left side of first mapping
  case "$alias" in
    rabbitmq)              echo "  rabbitmq (AMQP)         localhost:5672" ;;
    rabbitmq-ui)           : ;;
    minio-api)             : ;;
    *)                     ;;
  esac
}

# Print a clean table of URLs
for alias in "${SELECTED[@]}"; do
  # shellcheck disable=SC2206
  ports=(${SVC_PORTS[$alias]})
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
