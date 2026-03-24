# Deployment Launch Spec

This document describes how to launch `distributed-crawler` from the current repository state using:

- Docker Compose for local development
- Kubernetes + Helm for cluster deployment

It is grounded in the current scripts and configuration files:

- `deploy/scripts/docker/*`
- `deploy/scripts/k8s/*`
- `deploy/scripts/docker/launch.sh`
- `deploy/scripts/k8s/launch-minikube.sh`
- `docker-compose.yaml`
- `docker-compose.app.yaml`
- `deploy/helm/distributed-crawler/*`
- `deploy/helm/infra/*`

## Goals

- Provide one repeatable launch flow for local Docker deployment
- Provide one repeatable launch flow for Kubernetes deployment
- Make the required environment and launch modes explicit
- Show how to validate a deployment before and after startup

## Runtime Components

Application components:

- `grpc-server`
- `fetch-worker`
- `parser-worker`
- `export-worker`
- `ui`
- `migrate` job/container

Infrastructure components:

- PostgreSQL
- RabbitMQ
- MinIO
- Redis
- RedisInsight
- Jaeger
- OpenTelemetry Collector
- Prometheus
- Grafana
- OpenSearch
- OpenSearch Dashboards
- Kafka
- Kafka UI
- Zookeeper

## Canonical Launch Inputs

Application launch is driven by the env contract consumed from `internal/config/env/**`.

The most important deployment-time variables are:

- `PG_DSN`
- `RABBITMQ_URL`
- `RABBITMQ_CRAWL_QUEUE_NAME`
- `RABBITMQ_PARSING_QUEUE_NAME`
- `REDIS_ADDRESS`
- `REDIS_PWD`
- `MINIO_ENDPOINT`
- `MINIO_USER`
- `MINIO_PWD`
- `MINIO_BUCKET_NAME`
- `MESSAGING_BROKER`
- `KAFKA_BROKERS`
- `KAFKA_CONSUMER_GROUP`
- `KAFKA_CRAWL_TOPIC_NAME`
- `KAFKA_PARSING_TOPIC_NAME`
- `MEMORY_BROKER_ADDR`
- `MEMORY_BROKER_CAPACITY`
- `OTEL_*`
- `OPENSEARCH_*`
- `JWT_SECRET`
- `DEFAULT_USER_EMAIL`
- `DEFAULT_USER_PWD`
- `HTTP_CORS_ALLOWED_ORIGINS`
- `QUEUE_SECRETS_*`

For Docker Compose these are generated from `.env` plus compose defaults.

For Helm these are generated from:

- `values.yaml`
- `values-dev.yaml` or `values-prod.yaml`
- `values-external-infra.yaml`
- chart secrets/configmaps

## Docker Compose

### Prerequisites

- Docker Engine with Compose v2
- A populated root `.env` file
- A queue secrets file at `queue-secrets.json.example` or a file with equivalent content

### Pre-launch Validation

Run:

```bash
docker compose -f docker-compose.yaml -f docker-compose.app.yaml config
```

This validates the merged compose model before any containers are started.

### Default Full-stack Launch

The default Docker entrypoint is:

```bash
./deploy/scripts/docker/deploy-all.sh
```

For argument-driven local launch without preparing a root `.env`, use:

```bash
./deploy/scripts/docker/launch.sh \
  --pg-password some-pwd-123 \
  --rabbitmq-password guest \
  --minio-password minioadmin \
  --redis-password some_redis_pwd_123 \
  --jwt-secret your-secret-key-change-this-in-production-make-it-long-and-random \
  --default-user-password 12345678
```

If you want one explicitly named full-stack Docker entrypoint, use:

```bash
./deploy/scripts/docker/deploy-everything.sh \
  --pg-password some-pwd-123 \
  --rabbitmq-password guest \
  --minio-password minioadmin \
  --redis-password some_redis_pwd_123 \
  --jwt-secret your-secret-key-change-this-in-production-make-it-long-and-random \
  --default-user-password 12345678
```

On Windows PowerShell, use:

```powershell
.\deploy\scripts\docker\deploy-everything.ps1 `
  -PgPassword some-pwd-123 `
  -RabbitMqPassword guest `
  -MinioPassword minioadmin `
  -RedisPassword some_redis_pwd_123 `
  -JwtSecret your-secret-key-change-this-in-production-make-it-long-and-random `
  -DefaultUserPassword 12345678
```

Behavior:

1. Loads `.env` if present.
2. Validates the compose configuration.
3. Builds images unless `NO_BUILD=true`.
4. Starts infrastructure from `docker-compose.yaml`.
5. Waits for PostgreSQL, RabbitMQ, MinIO, and Redis to accept connections.
6. Runs database migrations through the `migrate` service.
7. Starts `grpc-server`.
8. Waits for gRPC and HTTP gateway ports to become reachable.
9. Starts `fetch-worker`, `parser-worker`, `export-worker`, and `ui`.

### Docker Launch Interfaces

Supported environment variables:

- `REGISTRY`
- `TAG`
- `APP_ONLY`
- `NO_BUILD`

Examples:

```bash
./deploy/scripts/docker/deploy-all.sh
APP_ONLY=true ./deploy/scripts/docker/deploy-all.sh
NO_BUILD=true ./deploy/scripts/docker/deploy-all.sh
REGISTRY=myrepo TAG=v1 ./deploy/scripts/docker/deploy-all.sh
./deploy/scripts/docker/deploy-all.sh --scale fetch-worker=3
```

Notes:

- `REGISTRY` and `TAG` are passed through to image build and compose image references.
- `APP_ONLY=true` assumes infrastructure is already running in the same compose project.
- Extra CLI args are passed through to `docker compose up`.

### Launching One Docker Component

Use:

```bash
./deploy/scripts/docker/deploy-component.sh <component>
```

Supported components:

- `grpc-server`
- `fetch-worker`
- `parser-worker`
- `export-worker`
- `ui`

Examples:

```bash
./deploy/scripts/docker/deploy-component.sh grpc-server
./deploy/scripts/docker/deploy-component.sh fetch-worker
REGISTRY=myrepo TAG=v1 ./deploy/scripts/docker/deploy-component.sh parser-worker
NO_BUILD=true ./deploy/scripts/docker/deploy-component.sh ui
```

Behavior:

- Validates compose first.
- Waits for core infrastructure.
- Runs migrations automatically when launching `grpc-server`.
- Waits for the API when launching workers or the UI.

### Docker Runtime Endpoints

After a successful full launch, the primary endpoints are:

- gRPC API: `localhost:8083`
- HTTP gateway: `http://localhost:8084`
- Admin UI: `http://localhost:18080`
- RabbitMQ UI: `http://localhost:15672`
- MinIO console: `http://localhost:9001`
- RedisInsight: `http://localhost:5540`
- Jaeger UI: `http://localhost:16686`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- OpenSearch: `http://localhost:9200`
- OpenSearch Dashboards: `http://localhost:5601`

### Docker Post-launch Checks

Check container state:

```bash
docker compose -f docker-compose.yaml -f docker-compose.app.yaml ps
```

Check logs:

```bash
docker compose -f docker-compose.yaml -f docker-compose.app.yaml logs -f grpc-server
docker compose -f docker-compose.yaml -f docker-compose.app.yaml logs -f fetch-worker
```

Check login path:

```bash
curl -X POST http://localhost:8084/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"12345678"}'
```

### Docker Teardown

Stop app only:

```bash
./deploy/scripts/docker/teardown.sh
```

Stop app and infra:

```bash
INFRA=true ./deploy/scripts/docker/teardown.sh
```

Remove volumes too:

```bash
INFRA=true VOLUMES=true ./deploy/scripts/docker/teardown.sh
```

## Kubernetes + Helm

### Prerequisites

- Kubernetes cluster
- `kubectl` configured to the target cluster
- Helm 3
- Docker for image builds
- Either:
  - a cluster-local image runtime such as minikube, or
  - a reachable image registry

### Deployment Model

The current Helm flow is infra-first by contract.

The app chart always targets an explicit infra release for:

- PostgreSQL
- RabbitMQ
- MinIO
- Redis
- OpenTelemetry Collector
- OpenSearch

Two supported modes exist:

1. Self-contained launch via script
   The deploy script installs infra and app together, but as two separate Helm releases.
2. External infra launch
   The infra release already exists; the app release is installed against it.

### Image Build

Build images:

```bash
./deploy/scripts/k8s/build-images.sh
```

Build into minikube’s Docker daemon:

```bash
./deploy/scripts/k8s/build-images.sh --minikube
```

Build with explicit image coordinates:

```bash
REGISTRY=myrepo TAG=v1 ./deploy/scripts/k8s/build-images.sh
```

Legacy aliases still supported by the script:

- `DOCKER_REGISTRY`
- `IMAGE_TAG`

### Pre-launch Validation

Infra chart:

```bash
helm template infra ./deploy/helm/infra \
  -f ./deploy/helm/infra/values.yaml \
  -f ./deploy/helm/infra/values-dev.yaml
```

App chart:

```bash
helm template crawler ./deploy/helm/distributed-crawler \
  -f ./deploy/helm/distributed-crawler/values.yaml \
  -f ./deploy/helm/distributed-crawler/values-dev.yaml \
  -f ./deploy/helm/distributed-crawler/values-external-infra.yaml \
  --set-string infra.releaseName=infra \
  --set-string infra.namespace=infra
```

The deployment scripts run equivalent `helm template` validation before install/upgrade.

### Kubernetes Full Launch: Self-contained Script Mode

Use:

```bash
./deploy/scripts/k8s/deploy-all.sh
```

For the simplest local `minikube + Helm` flow with arguments instead of env vars,
use the dedicated wrapper:

```bash
./deploy/scripts/k8s/launch-minikube.sh \
  --pg-password some-pwd-123 \
  --rabbitmq-password guest \
  --minio-password minioadmin \
  --redis-password some_redis_pwd_123 \
  --jwt-secret your-secret-key-change-this-in-production-make-it-long-and-random \
  --default-user-password 12345678 \
  --port-forward
```

Wrapper behavior:

1. Starts minikube unless `--skip-minikube-start` is used.
2. Builds images into minikube unless `--no-build` is used.
3. Deploys infra as a separate Helm release.
4. Deploys the app release against that infra.
5. Creates the MinIO bucket automatically unless `--no-bucket` is used.
6. Starts port-forward when `--port-forward` is requested.

Behavior:

1. Creates namespaces if needed.
2. Templates infra and app charts.
3. Deploys infra release first.
4. Deploys app release pointed at that infra release.
5. Waits for resources and migration jobs.

Default release mapping in this mode:

- App release: `crawler`
- App namespace: `crawler`
- Infra release: `crawler-infra`
- Infra namespace: `crawler`

Production example:

```bash
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-all.sh
```

Custom release/namespace example:

```bash
RELEASE_NAME=mycrawler NAMESPACE=apps INFRA_RELEASE_NAME=mycrawler-infra INFRA_NAMESPACE=infra ./deploy/scripts/k8s/deploy-all.sh
```

### Kubernetes Split Launch: External Infra

Deploy infra:

```bash
./deploy/scripts/k8s/deploy-infra.sh
```

Then deploy app against that infra release:

```bash
EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```

Default release mapping in this mode:

- App release: `crawler`
- App namespace: `crawler`
- Infra release: `infra`
- Infra namespace: `infra`

Production example:

```bash
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh
VALUES_ENV=prod EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```

### Helm Script Interfaces

`deploy-all.sh` supports:

- `RELEASE_NAME`
- `NAMESPACE`
- `VALUES_ENV`
- `EXTERNAL_INFRA`
- `INFRA_RELEASE_NAME`
- `INFRA_NAMESPACE`
- pass-through Helm args

Examples:

```bash
EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-all.sh
EXTERNAL_INFRA=true INFRA_RELEASE_NAME=shared INFRA_NAMESPACE=platform ./deploy/scripts/k8s/deploy-all.sh
./deploy/scripts/k8s/deploy-all.sh --set fetchWorker.replicaCount=3
```

### Deploying a Single Kubernetes Component

Use:

```bash
./deploy/scripts/k8s/deploy-component.sh <component>
```

Supported components:

- `grpc-server`
- `fetch-worker`
- `parser-worker`
- `export-worker`
- `ui`

Examples:

```bash
./deploy/scripts/k8s/deploy-component.sh grpc-server
./deploy/scripts/k8s/deploy-component.sh fetch-worker
INFRA_RELEASE_NAME=infra INFRA_NAMESPACE=infra ./deploy/scripts/k8s/deploy-component.sh parser-worker
BASE_RELEASE_NAME=crawler ./deploy/scripts/k8s/deploy-component.sh ui
```

Notes:

- Single-component deploys assume infra already exists.
- Worker and UI component deploys point to a shared gRPC service host.
- The script validates with `helm template` before deployment.

### Kubernetes Post-launch Checks

Check app pods:

```bash
kubectl get pods -n crawler
```

Check infra services:

```bash
kubectl get svc -n infra
```

Check logs:

```bash
kubectl logs -n crawler -l app.kubernetes.io/component=grpc-server -f
kubectl logs -n crawler -l app.kubernetes.io/component=fetch-worker -f
```

The current templates also gate startup with init containers, so:

- migrations wait for PostgreSQL
- `grpc-server` waits for PostgreSQL, RabbitMQ, MinIO, Redis
- workers wait for infra plus `grpc-server`
- `ui` waits for the HTTP gateway

### Port Forwarding

Use:

```bash
./deploy/scripts/k8s/port-forward.sh
```

Or selected services:

```bash
./deploy/scripts/k8s/port-forward.sh postgresql rabbitmq minio
./deploy/scripts/k8s/port-forward.sh grpc-server grafana jaeger
```

Typical local endpoints after port-forward:

- UI: `http://localhost:18080`
- gRPC: `localhost:8083`
- HTTP gateway: `http://localhost:8084`
- RabbitMQ UI: `http://localhost:15672`
- MinIO console: `http://localhost:9001`
- Grafana: `http://localhost:3000`
- Jaeger: `http://localhost:16686`

### Kubernetes Secrets

App secrets are supplied through chart values or an existing secret reference.

Current supported patterns:

1. Create from values:

```yaml
secrets:
  create: true
  postgres:
    password: "..."
  rabbitmq:
    password: "..."
  minio:
    user: "minioadmin"
    password: "..."
  redis:
    password: "..."
  auth:
    jwtSecret: "..."
    defaultPassword: "..."
```

2. Use an existing secret:

```yaml
secrets:
  create: false
  existingSecret: "crawler-secrets"
```

Infra secrets are configured independently through the infra chart values.

In external-infra mode, app secrets and infra secrets must match for:

- PostgreSQL password
- RabbitMQ password
- MinIO credentials
- Redis password

### Kubernetes Teardown

Remove app and infra:

```bash
./deploy/scripts/k8s/teardown.sh
```

Remove app only:

```bash
APP_ONLY=true ./deploy/scripts/k8s/teardown.sh
```

Remove infra only:

```bash
INFRA_ONLY=true ./deploy/scripts/k8s/teardown.sh
```

## Recommended Launch Paths

For local development with Docker:

```bash
./deploy/scripts/docker/deploy-all.sh
```

For local development on a cluster:

```bash
./deploy/scripts/k8s/build-images.sh --minikube
./deploy/scripts/k8s/deploy-all.sh
```

For shared-cluster deployment with separately managed infra:

```bash
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh
VALUES_ENV=prod EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```
