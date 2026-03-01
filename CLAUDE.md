# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Distributed web crawling platform built with Go. Uses RabbitMQ (or Kafka/gRPC-memory) for task distribution, PostgreSQL for persistence, MinIO for blob storage, Redis for rate limiting/caching, and OpenTelemetry for observability.

## Development Commands

### Infrastructure

```bash
# Start all infra services (Postgres, RabbitMQ, MinIO, Redis)
docker compose -f docker/docker-compose.yaml up -d
docker compose -f docker/docker-compose.yaml down
```

Ports: PostgreSQL `:54322`, RabbitMQ `:5672` (UI `:15672`), MinIO `:9000` (UI `:9001`), Redis `:6379`

### Build & Run

```bash
make build              # → ./bin/distributed-crawler (gRPC server)

make run-grpc-server    # gRPC :8083 + HTTP gateway :8084
make run-fetcher        # Fetch worker
make run-parser         # Parser worker
make run-export         # Export worker

# Memory broker (standalone gRPC queue service)
go run ./cmd/memory_broker --addr :9095
```

Workers use `--worker-config-path` flag (default `.worker.env`); the API server uses `--config-path` (default `.env`).

### Database Migrations

```bash
make local-migration-status
make local-migration-up
make local-migration-down
make local-migration-create NAME=add_something   # creates new .sql file
```

Migration directory: `internal/infra/persistence/postgres/migrations/`

### Code Generation

```bash
make .bin-deps     # install protoc plugins, goose, statik
make generate      # buf generate + statik embed (runs tidy first)
go generate ./...  # regenerate mocks (minimock)
```

Proto definitions: `api/v1/`. Generated Go: `pkg/v1/` (package `crawlergrpc`).

### Testing

```bash
make test                  # all tests, 5 iterations, with coverage
make test-coverage         # HTML coverage report (excludes mocks/config)

go test ./internal/api/crawl_job/tests/...
go test -run TestCreateCrawlJob ./internal/api/crawl_job/tests/
```

## Architecture

### Entry Points (`cmd/`)

| Binary | Description |
|--------|-------------|
| `grpc_server` | Main API: gRPC + HTTP (grpc-gateway) + outbox publisher + schedule worker, all in one process |
| `fetch_worker` | Consumes `crawl_queue`, fetches pages, uploads to MinIO, publishes to `parsing_queue` |
| `parser_worker` | Consumes `parsing_queue`, extracts data, stores `ExtractedRecord`, handles link discovery |
| `export_worker` | Polls for completed jobs, generates export files in MinIO |
| `scheduler_worker` | Polls for scheduled jobs, creates new crawl cycles |
| `memory_broker` | Standalone gRPC in-memory message broker (alternative to RabbitMQ/Kafka) |
| `grpc_service` | gRPC client utility |

### Layer Structure

```
internal/
  app/                    # Wiring: api_app.go, worker_app.go, service_provider.go
  domain/
    crawl/                # Core crawling domain (jobs, tasks, snapshots, records, outbox)
    auth/                 # Authentication domain (users, refresh tokens)
    queue/                # Queue endpoint admin domain
    shared/               # Shared clock, errors
  application/service/    # Use cases (crawl_job, crawl_task, auth, preview, user, queue)
  api/                    # gRPC service implementations (crawl_job, auth, preview, user, worker, queue_admin)
  interfaces/http/        # Additional HTTP handlers (snapshots, tasks, records)
  infra/
    persistence/postgres/ # Repos, snapshots (DB structs), converters, transactions
    messaging/            # messaging.Client interface + rabbitmq/kafka/memory/broker impls
    services/             # Fetcher, parser, contentstore (MinIO), sanitizer
    cache/                # Redis client, rate limiter, robots.txt cache
    secrets/              # File-based polling secrets store (queue credentials)
    logger/               # Zap logger + OpenSearch core
  auth/                   # JWT service, middleware, RBAC interceptor
  interceptor/            # gRPC: validate, log, shard-key, JWT, RBAC
  worker/                 # FetchWorker, ParserWorker, ExportWorker, ScheduleWorker, WorkerMonitor
  telemetry/              # OpenTelemetry provider + metrics
  config/env/             # One file per config type (pg.go, rabbitmq.go, redis.go, …)
```

### Key Architectural Patterns

**Single API process**: `grpc_server` runs gRPC, HTTP (grpc-gateway), outbox publisher goroutine, and schedule worker goroutine together in `internal/app/api_app.go`.

**Messaging abstraction**: `internal/infra/messaging/client.go` defines `messaging.Client`. Broker selected by `MESSAGING_BROKER` env var (`rabbitmq` | `kafka` | `grpc_memory`). Queue/topic names come from `RABBITMQ_CRAWL_QUEUE_NAME` / `KAFKA_CRAWL_TOPIC_NAME`, etc.

**Outbox pattern**: `TaskEnqueuedEvent` and similar events are stored in `outbox_events` in the same DB transaction as business data. A background goroutine in the API process reads unprocessed events and publishes them to the broker.

**Worker monitoring**: Each worker process runs `WorkerMonitor`, which registers with the API server via `WorkerServiceServer`. The API server can remotely drain or force-kill workers.

**Transaction management**: `TxManager` in `internal/infra/persistence/postgres/transaction/` propagates transactions via `context.Context`. Repos detect and reuse an active tx from ctx automatically.

**Repository / Converter pattern**: DB structs (`*Snapshot`) live in `infra/persistence/postgres/snapshots/`. Converters in `infra/persistence/postgres/converters/` translate between domain models and DB structs. Repos work only with domain models.

**Queue routing**: `internal/worker/routing/queue_routing_policy.go` supports weighted FNV-hash routing across admin-managed queue endpoints (`QueueAdminService`). Queue credentials loaded from a polling file secrets store (`QUEUE_SECRETS_FILE_PATH`).

**Auth**: JWT-based with roles. gRPC interceptor chain: `Log → Validate → ShardKey → JWTAuth → RBAC`. Default admin seeded on startup from `DEFAULT_USER_EMAIL` / `DEFAULT_USER_PWD`.

**Sharding**: Optional PostgreSQL sharding. `PG_SHARDING_ENABLED=true` requires `PG_SHARD_DSNS` (comma-separated DSNs). `ShardKeyInterceptor` reads shard key from gRPC metadata.

### Configuration

Config is loaded from a dotenv file by `config.Load(path)`. All typed config readers are in `internal/config/env/` — one file per config type.

**Required env vars by component:**

| Component | Required vars |
|-----------|--------------|
| All | `PG_DSN`, `LOG_LEVEL`, `LOG_ENV` |
| API server | `GRPC_HOST/PORT`, `HTTP_HOST/PORT`, `JWT_SECRET`, `DEFAULT_USER_EMAIL/PWD` |
| Fetch/Parser workers | `MINIO_ENDPOINT`, `MINIO_USER`, `MINIO_PWD`, `MINIO_BUCKET_NAME`, `REDIS_ADDRESS`, `REDIS_PWD` |
| RabbitMQ | `RABBITMQ_URL`, `RABBITMQ_CRAWL_QUEUE_NAME`, `RABBITMQ_PARSING_QUEUE_NAME` |
| Kafka | `KAFKA_BROKERS`, `KAFKA_CONSUMER_GROUP`, `KAFKA_CRAWL_TOPIC_NAME`, `KAFKA_PARSING_TOPIC_NAME` |
| gRPC memory broker | `MEMORY_BROKER_ADDR` |

`LIMITER_TYPE=redis|inmemory` controls whether workers use Redis or per-process in-memory rate limiting.

### Database Schema

Key tables: `crawl_jobs`, `crawl_job_configs`, `crawl_tasks`, `page_snapshots`, `extracted_records`, `outbox_events`, `previews`, `users`, `refresh_tokens`, `queue_endpoints`.

All IDs are UUIDs. FKs use `ON DELETE CASCADE`. `outbox_events.processed_at IS NULL` = pending.

### Mocks

Mocks use `github.com/gojuno/minimock/v3`. `//go:generate` directives in `internal/application/service/generate.go` and per-repo `generate.go` files. Run `go generate ./...` to regenerate all mocks.

## Helm Deployment

Chart: `deploy/helm/distributed-crawler/`

```bash
# Dev (subcharts included)
helm upgrade --install crawler ./deploy/helm/distributed-crawler \
  -f values.yaml -f values-dev.yaml

# External infra (infra managed by a separate release)
helm upgrade --install crawler ./deploy/helm/distributed-crawler \
  -f values.yaml -f values-dev.yaml -f values-external-infra.yaml
```

ConfigMap holds non-secret env vars; Secret holds passwords and constructed DSNs/URLs. All pods load both via `envFrom`.

## Code Conventions

- Errors: wrap with context: `fmt.Errorf("doing X: %w", err)`
- Domain errors in `internal/domain/shared/errors.go`
- Tests in `tests/` subdirectory within the package; use `testify/require`
- Commands mutate state; Queries read state (split in `application/service/`)
- All IDs are typed value objects (`CrawlJobID`, `CrawlTaskID`, etc.) — UUID validation enforced at construction

## Known Technical Debt

- `CompletedAt` in `CrawlJob` uses non-nullable time in domain but needs `sql.NullTime` in DB — see `internal/infra/persistence/postgres/converters/crawl_job.go`
- `internal/infra/persistence/dbclient.go` uses `log.Println` instead of zap
