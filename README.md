# Distributed Crawler

Distributed web crawling platform built with Go. Uses RabbitMQ (or Kafka / gRPC-memory) for task distribution, PostgreSQL for persistence, MinIO for blob storage, Redis for rate limiting/caching, and OpenTelemetry for observability.

## Deploy

### Docker (local / CI)

One-click — builds images, starts infra, runs migrations, starts all app components:

```bash
make docker-deploy
```

Override credentials or broker:

```bash
make docker-deploy ARGS="--pg-password mypwd --jwt-secret supersecret --default-user-password admin123"
make docker-deploy ARGS="--messaging-broker kafka --no-build"
```

Deploy a single component (infra must already be running):

```bash
make docker-deploy-component COMPONENT=grpc-server
make docker-deploy-component COMPONENT=fetch-worker
```

Tear down:

```bash
make docker-teardown            # stops app containers, keeps infra volumes
make docker-teardown INFRA=true # stops everything
```

View logs:

```bash
make docker-logs                                   # infra services
./deploy/scripts/docker/logs.sh                    # all app components
./deploy/scripts/docker/logs.sh grpc-server        # single component
```

Common `--` flags for `make docker-deploy ARGS="..."`:

| Flag | Default |
|------|---------|
| `--pg-password` | `some-pwd-123` |
| `--rabbitmq-password` | `guest` |
| `--minio-password` | `minioadmin` |
| `--redis-password` | `some_redis_pwd_123` |
| `--jwt-secret` | (insecure default) |
| `--default-user-email` | `admin@example.com` |
| `--default-user-password` | `12345678` |
| `--messaging-broker` | `rabbitmq` |
| `--app-only` | skip infra startup |
| `--no-build` | skip image build |

---

### Kubernetes — minikube (local)

One-click — starts minikube, builds images inside the cluster, deploys infra and app via Helm:

```bash
make k8s-deploy
```

Override secrets or enable full observability stack:

```bash
make k8s-deploy ARGS="--pg-password mypwd --jwt-secret supersecret --default-user-password admin123"
make k8s-deploy ARGS="--full-observability --port-forward"
```

Deploy a single component:

```bash
make k8s-deploy-component COMPONENT=grpc-server
make k8s-deploy-component COMPONENT=fetch-worker
```

Open port-forwards to all services after deploy:

```bash
make k8s-port-forward
```

Tear down all Helm releases:

```bash
make k8s-teardown
```

Common `--` flags for `make k8s-deploy ARGS="..."`:

| Flag | Default |
|------|---------|
| `--pg-password` | `some-pwd-123` |
| `--rabbitmq-password` | `guest` |
| `--minio-password` | `minioadmin` |
| `--redis-password` | `some_redis_pwd_123` |
| `--jwt-secret` | (insecure default) |
| `--default-user-email` | `admin@example.com` |
| `--default-user-password` | `12345678` |
| `--messaging-broker` | `rabbitmq` |
| `--values-env` | `dev` (`dev` or `prod`) |
| `--full-observability` | enables Prometheus/Grafana/OpenSearch |
| `--port-forward` | start port-forward after deploy |
| `--no-build` | skip image build |
| `--skip-minikube-start` | reuse existing cluster |

### Multi-region workers (Helm)

`fetchWorker.regions` and `parserWorker.regions` are lists of region strings. One Deployment is created per entry; all share the same `replicaCount`.

```yaml
# values.yaml override — one Deployment per region, 3 replicas each
fetchWorker:
  replicaCount: 3
  regions:
    - us-east
    - eu-west

parserWorker:
  replicaCount: 3
  regions:
    - us-east
    - eu-west
```

If `regions` is empty (default), a single Deployment is created with no `WORKER_REGION` set (uses static queue names from config).

---

### Local (no Docker)

Start infrastructure services first:

```bash
docker compose -f docker-compose.yaml up -d
```

Run all components (logs go to `logs/`, PIDs to `.pids/`):

```bash
./deploy/scripts/local/start-all.sh
./deploy/scripts/local/stop-all.sh
```

Or run individual components:

```bash
make run-grpc-server
make run-fetcher
make run-parser
make run-export
```

---

## Database migrations

```bash
make local-migration-up        # apply all pending migrations
make local-migration-down      # roll back last migration
make local-migration-status    # show migration state
make local-migration-create NAME=add_something   # create new migration file
```

## Build

```bash
make build                     # build gRPC server binary → ./bin/distributed-crawler
```

## Test

```bash
make test                      # all tests, 5 iterations, with coverage
make test-coverage             # HTML coverage report
```

## Code generation

```bash
make .bin-deps   # install protoc plugins, goose, statik
make generate    # buf generate + statik embed
go generate ./... # regenerate mocks (minimock)
```
