# CLAUDE.md

This file gives coding agents the shortest accurate orientation for working in this repository.

## Overview

Distributed Crawler is a Go-based distributed web crawling platform.

- API entrypoint: `cmd/grpc_server`
- Workers: `cmd/fetch_worker`, `cmd/parser_worker`, `cmd/export_worker`, `cmd/scheduler_worker`
- Dev broker: `cmd/memory_broker`
- DB migration CLI: `cmd/migrate`
- UI: `ui/` (Angular, served by nginx in containerized deploys)

Main infrastructure:

- PostgreSQL for jobs, tasks, users, outbox, previews
- MinIO for fetched page payloads and export artifacts
- RabbitMQ, Kafka, or gRPC memory broker for task delivery
- Redis for rate limiting and robots/cache support
- OpenTelemetry, Prometheus, Grafana, Jaeger, OpenSearch for observability

## Fast Start

### Preferred launchers

```bash
# Single-region full stack
./deploy/scripts/default_run.sh
./deploy/scripts/default_run.sh --mode docker
./deploy/scripts/default_run.sh --mode k8s

# Multi-region fetch workers
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west
```

### Common development commands

```bash
make build

make run-grpc-server
make run-fetcher
make run-parser
make run-export

make docker-deploy
make k8s-deploy

make test
make test-coverage

make local-migration-up
make local-migration-down
make local-migration-status

make .bin-deps
make generate
go generate ./...
```

### Local process mode

```bash
docker compose -f docker-compose.yaml up -d
./deploy/scripts/local/start-all.sh
./deploy/scripts/local/stop-all.sh
```

Logs go to `logs/`, PID files to `.pids/`.

## Real Binary Flags

The app binaries take very few flags directly; most configuration comes from dotenv files or env vars.

```bash
go run ./cmd/grpc_server/main.go --config-path=.env
go run ./cmd/fetch_worker/main.go --worker-config-path=.worker.env
go run ./cmd/parser_worker/main.go --worker-config-path=.worker.env
go run ./cmd/export_worker/main.go --worker-config-path=.worker.env
go run ./cmd/scheduler_worker/main.go --worker-config-path=.worker.env
go run ./cmd/memory_broker/main.go --addr :9095 --capacity 1000
go run ./cmd/migrate/main.go --dsn "$PG_DSN" status
```

`internal/config/config.go` skips dotenv loading when `CONFIG_SOURCE=env`, which is how Docker Compose and Helm deployments inject config.

## Architecture Notes

### Coordinator

`grpc-server` is the coordinator service.

- Serves gRPC and grpc-gateway HTTP APIs
- Seeds the default admin user
- Registers worker health/control service
- Runs the outbox publisher in-process
- Runs the schedule worker in-process

Important implication: there is also a standalone `scheduler_worker` binary, but the default API deployment already starts scheduling internally.

### Worker pipeline

```text
create job
-> save tasks + outbox events in PostgreSQL
-> outbox publisher sends crawl tasks to broker
-> fetch-worker downloads pages and stores HTML in MinIO
-> parser-worker extracts data, discovers more links, stores results
-> export-worker aggregates completed results into JSON and CSV
```

### Storage and transport

- Domain state lives in `internal/domain/`
- Use cases live in `internal/application/service/`
- gRPC handlers live in `internal/api/`
- HTTP-only handlers live in `internal/interfaces/http/`
- Infra adapters live in `internal/infra/`
- App wiring lives in `internal/app/`
- Worker implementations live in `internal/worker/`

Important patterns:

- Outbox pattern for reliable publish after DB state changes
- Repository plus converter plus snapshot layering for persistence
- Messaging abstraction in `internal/infra/messaging/`
- Optional PostgreSQL sharding
- Worker heartbeat and drain control through `WorkerMonitor`

## Messaging And Regions

- Broker is selected by `MESSAGING_BROKER`: `rabbitmq`, `kafka`, or `grpc_memory`
- Default crawl queue name comes from `RABBITMQ_CRAWL_QUEUE_NAME` or Kafka equivalent
- Multi-region mode uses `RABBITMQ_CRAWL_QUEUE_NAMES` plus `WORKER_REGION`
- `multi_region_run.sh` creates one fetch-worker pool per region
- Parser workers are shared and not region-bound

## Config Surfaces To Know

### Core env vars

- `PG_DSN`
- `LOG_LEVEL`
- `LOG_ENV`
- `MESSAGING_BROKER`

### API env vars

- `GRPC_HOST`, `GRPC_PORT`
- `HTTP_HOST`, `HTTP_PORT`
- `JWT_SECRET`
- `DEFAULT_USER_EMAIL`, `DEFAULT_USER_PWD`
- `HTTP_CORS_ALLOWED_ORIGINS`

### Worker env vars

- `MINIO_ENDPOINT`, `MINIO_USER`, `MINIO_PWD`, `MINIO_BUCKET_NAME`
- `REDIS_ADDRESS`, `REDIS_PWD`, `REDIS_DB`
- `LIMITER_TYPE`
- `WORKER_REGION`
- `FETCHER_TYPE`
- `CHROME_REMOTE_URL`

### Queue-secret support

- `QUEUE_SECRETS_FILE_PATH`
- `QUEUE_SECRETS_WATCH_ENABLED`
- `QUEUE_SECRETS_RELOAD_INTERVAL`

The example file is [queue-secrets.json.example](/Users/denis/projects/go/DistributedCrawler/queue-secrets.json.example).

## Files And Directories Worth Checking First

- [README.md](/Users/denis/projects/go/DistributedCrawler/README.md)
- [docs/operator-manual.md](/Users/denis/projects/go/DistributedCrawler/docs/operator-manual.md)
- [docs/parsing-syntax-spec.md](/Users/denis/projects/go/DistributedCrawler/docs/parsing-syntax-spec.md)
- [Makefile](/Users/denis/projects/go/DistributedCrawler/Makefile)
- [docker-compose.yaml](/Users/denis/projects/go/DistributedCrawler/docker-compose.yaml)
- [docker-compose.app.yaml](/Users/denis/projects/go/DistributedCrawler/docker-compose.app.yaml)
- [internal/app/api_app.go](/Users/denis/projects/go/DistributedCrawler/internal/app/api_app.go)
- [internal/app/worker_app.go](/Users/denis/projects/go/DistributedCrawler/internal/app/worker_app.go)

## Working Conventions

- Prefer `rg` for searches
- Prefer `apply_patch` for edits
- Do not assume the old docs are correct; verify against `cmd/`, `internal/app/`, and deployment scripts
- Be careful with `deploy/scripts/docker/teardown.sh`: it removes all Docker containers and all Docker volumes on the host
- Respect existing changes in the worktree; do not revert unrelated user work

## Testing And Regeneration

Use these after changing API or generated assets:

```bash
make test
make generate
go generate ./...
```

Generated sources:

- Protos: `api/v1/`
- Generated Go: `pkg/v1/`
- Embedded Swagger assets: `statik/statik.go`
