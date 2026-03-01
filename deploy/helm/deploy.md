# Helm Deployment Guide

Scripts live in `deploy/scripts/k8s/`. All scripts are idempotent (`helm upgrade --install`).

---

## 1. Build Images

```bash
# Build all images (local Docker)
./deploy/scripts/k8s/build-images.sh

# Build into minikube's Docker daemon (no push needed)
./deploy/scripts/k8s/build-images.sh --minikube

# Build specific components
./deploy/scripts/k8s/build-images.sh grpc-server fetch-worker

# Force rebuild without cache
NO_CACHE=1 ./deploy/scripts/k8s/build-images.sh --minikube
```

---

## 2. Deploy Infrastructure

Deploys: PostgreSQL, RabbitMQ, MinIO, Redis, RedisInsight, OTel Collector, Jaeger, Prometheus, Grafana, OpenSearch, OpenSearch Dashboards into the `infra` namespace.

```bash
# Dev (minimal resources, Prometheus/Grafana/OpenSearch disabled)
./deploy/scripts/k8s/deploy-infra.sh

# Production
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh

# Custom namespace
NAMESPACE=infra VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh
```

---

## 3. Deploy Application

The app chart deploys these components:

| Component      | Description                                        |
|----------------|----------------------------------------------------|
| `grpc-server`  | API server — gRPC `:8083` + HTTP gateway `:8084`   |
| `fetch-worker` | Fetches pages, uploads to MinIO                    |
| `parser-worker`| Parses pages, extracts records, discovers links    |
| `export-worker`| Generates export files in MinIO for completed jobs |
| `ui`           | Angular admin UI — HTTP `:80` (ClusterIP)          |
| `migrations`   | DB migration Job (runs alongside `grpc-server`)    |

### Option A — External infra (recommended for dev)

Requires infra deployed first (step 2). App points at the `infra` release services.

```bash
EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```

### Option B — Self-contained

App chart includes embedded bitnami subcharts (PostgreSQL, RabbitMQ, MinIO, Redis). No separate infra release needed.

```bash
./deploy/scripts/k8s/deploy-all.sh
```

### Deploy a single component

```bash
./deploy/scripts/k8s/deploy-component.sh grpc-server
./deploy/scripts/k8s/deploy-component.sh fetch-worker
./deploy/scripts/k8s/deploy-component.sh parser-worker
./deploy/scripts/k8s/deploy-component.sh export-worker
./deploy/scripts/k8s/deploy-component.sh ui

# With extra helm args
./deploy/scripts/k8s/deploy-component.sh fetch-worker --set fetchWorker.replicaCount=3
```

---

## 4. Port-Forwards

Opens local ports to all services. Ctrl-C stops everything cleanly.

```bash
# Forward all services
./deploy/scripts/k8s/port-forward.sh

# Forward selected services only
./deploy/scripts/k8s/port-forward.sh postgresql rabbitmq minio
./deploy/scripts/k8s/port-forward.sh grpc-server jaeger grafana
```

Available service names:

| Name                   | Local port(s)            |
|------------------------|--------------------------|
| `postgresql`           | 54322                    |
| `rabbitmq`             | 5672, 15672 (UI)         |
| `minio`                | 9000, 9001 (UI)          |
| `redis`                | 6379                     |
| `redisinsight`         | 8001                     |
| `jaeger`               | 16686                    |
| `prometheus`           | 9090                     |
| `grafana`              | 3000                     |
| `opensearch`           | 9200                     |
| `opensearch-dashboards`| 5601                     |
| `grpc-server`          | 8083 (gRPC), 8084 (HTTP) |
| `ui`                   | 8080 → 80 (HTTP)         |

---

## 5. Push Images (remote registry)

```bash
TARGET_REGISTRY=ghcr.io/myorg IMAGE_TAG=v1.2.3 ./deploy/scripts/k8s/push-images.sh

# Push specific components
TARGET_REGISTRY=ghcr.io/myorg ./deploy/scripts/k8s/push-images.sh grpc-server fetch-worker
```

---

## 6. Teardown

```bash
# Remove both app and infra releases
./deploy/scripts/k8s/teardown.sh

# App only
APP_ONLY=true ./deploy/scripts/k8s/teardown.sh

# Infra only
INFRA_ONLY=true ./deploy/scripts/k8s/teardown.sh
```

---

## Headlamp (cluster UI)

See `headlamp.md` for setup.

```bash
minikube service headlamp -n headlamp
```

---

## Quick Reference — Service URLs

| Service               | URL                            | Credentials                        |
|-----------------------|--------------------------------|------------------------------------|
| Admin UI              | `http://localhost:8080`        | JWT (admin email in values)        |
| gRPC API              | `localhost:8083`               | JWT (admin email in values)        |
| HTTP gateway          | `http://localhost:8084`        | same                               |
| RabbitMQ UI           | `http://localhost:15672`       | guest / guest                      |
| MinIO console         | `http://localhost:9001`        | minioadmin / minioadmin            |
| RedisInsight          | `http://localhost:8001`        | —                                  |
| Jaeger UI             | `http://localhost:16686`       | —                                  |
| Prometheus            | `http://localhost:9090`        | —                                  |
| Grafana               | `http://localhost:3000`        | admin / changeme-grafana-password  |
| OpenSearch            | `http://localhost:9200`        | —                                  |
| OpenSearch Dashboards | `http://localhost:5601`        | —                                  |
