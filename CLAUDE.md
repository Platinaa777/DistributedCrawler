# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Distributed Crawling System** - A scalable web crawling platform built with Go that uses RabbitMQ for task distribution, PostgreSQL for data persistence, and implements clean architecture with domain-driven design principles.

**Key Components:**
- **gRPC/HTTP Server**: API endpoints for managing crawl jobs and tasks
- **Angular Frontend (MVP)**: Web UI for job management with visual extraction builder
- **Fetch Worker**: Consumes crawl tasks from RabbitMQ, fetches pages, stores raw content
- **Parser Worker**: Processes fetched pages, extracts structured data
- **RabbitMQ**: Message queue with separate queues for crawling (`crawl_queue`) and parsing (`parsing_queue`)
- **PostgreSQL**: Persistent storage for normalized crawl data
- **MinIO/S3**: Blob storage for raw HTML/JSON pages
- **Redis**: Rate limiting and caching layer

## Development Commands

### Environment Setup

```bash
# Start all infrastructure services (PostgreSQL, RabbitMQ, MinIO, Redis)
docker compose up -d

# Install build dependencies (protoc plugins, goose, statik)
make .bin-deps

# Generate code from protobuf definitions
make generate

# Tidy dependencies
make .tidy
```

**Infrastructure Services:**
- PostgreSQL: `localhost:54321` (UI: N/A)
- RabbitMQ: `localhost:5672` (Management UI: `http://localhost:15672`)
- MinIO: `localhost:9000` (Console UI: `http://localhost:9001`)
- Redis: `localhost:6379` (RedisInsight UI: `http://localhost:5540`)

### Database Migrations

```bash
# Check migration status
make local-migration-status

# Run pending migrations
make local-migration-up

# Rollback last migration
make local-migration-down

# Create new migration (use goose directly)
goose -dir ./internal/infra/persistence/postgres/migrations create migration_name sql
```

**Migration Directory:** `internal/infra/persistence/postgres/migrations/`

### Building and Running

```bash
# Build the gRPC server
make build
# Output: ./bin/distributed-crawler

# Run the gRPC server (with config)
make run-grpc-server
# Runs: go run ./cmd/grpc_server/main.go --config-path=.env

# Run the fetch worker (processes crawl tasks)
make run-fetcher
# Runs: go run ./cmd/fetch_worker/main.go --config-path=.worker.env

# Run the parser worker (processes parsing tasks)
make run-parser
# Runs: go run ./cmd/parser_worker/main.go --config-path=.worker.env

# Run specific commands directly
go run ./cmd/grpc_server/main.go --config-path=.env
go run ./cmd/http_server/main.go
go run ./cmd/fetch_worker/main.go --config-path=.worker.env
go run ./cmd/parser_worker/main.go --config-path=.worker.env
```

**Available Entry Points:**
- `cmd/grpc_server/` - gRPC API server
- `cmd/http_server/` - HTTP REST API server
- `cmd/fetch_worker/` - Worker that fetches web pages
- `cmd/parser_worker/` - Worker that parses fetched pages
- `cmd/grpc_service/` - Alternative gRPC service (check purpose)
- `cmd/http_client/` - HTTP client utility

### Testing

```bash
# Run all tests (5 iterations, with coverage)
make test

# Run tests with coverage report
make test-coverage
# Opens HTML coverage report in browser
# Excludes mocks and config from coverage
```

**Running Individual Tests:**
```bash
# Run tests in specific package
go test ./internal/api/crawl_job/tests/...

# Run single test
go test -run TestCreateCrawlJob ./internal/api/crawl_job/tests/

# With verbose output
go test -v ./...
```

### Frontend Development

The project includes an Angular-based web UI in the `ui/` directory.

**Tech Stack:**
- Angular v19
- Tailwind CSS v4
- PrimeNG (latest compatible)
- Consumes REST API via grpc-gateway

**Running the Frontend:**
```bash
cd ui
npm install
npm start
# UI runs on http://localhost:4200 (default)
```

**Frontend Features (MVP):**
- Jobs list page (`/jobs`)
- Job details page with task inspection (`/jobs/:id`)
- Visual job creation page (`/jobs/create`) with:
  - HTML preview in iframe (using `/api/v1/previews`)
  - DevTools-like element picker
  - Visual builder for ExtractionSpec
  - Job configuration forms (seeds, scope, rate limits)

**API Endpoints Used:**
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/{id}` - Get job details
- `POST /api/v1/jobs` - Create new job
- `GET /api/v1/jobs/{job_id}/tasks` - List job tasks
- `POST /api/v1/previews` - Create page preview
- `GET /api/v1/previews/{id}` - Get preview with download URL

See `docs/create-frontend.txt` for complete frontend specification.

## Architecture

### Clean Architecture Layers

```
cmd/                          # Entry points (main.go files)
  ├── grpc_server/           # gRPC API server (main production API)
  ├── http_server/           # HTTP REST API server
  ├── fetch_worker/          # Worker that fetches web pages from URLs
  ├── parser_worker/         # Worker that parses fetched pages
  ├── grpc_service/          # Alternative gRPC service implementation
  └── http_client/           # HTTP client utility

internal/
  ├── domain/crawl/          # Domain layer (business logic)
  │   ├── models/           # Entities: CrawlJob, CrawlTask, PageSnapshot, ExtractedRecord
  │   ├── valueobjects/     # Type-safe IDs (CrawlJobID, CrawlTaskID, etc.)
  │   ├── events/           # Domain events (TaskEnqueuedEvent)
  │   ├── repos/            # Repository interfaces
  │   ├── services/         # Domain services interfaces (Fetcher, Parser, Queue)
  │   └── domain_services/  # Complex business logic (Pipeline, Scheduler)
  │
  ├── application/service/   # Application/use case layer
  │   ├── crawl_job/        # Job management commands & queries
  │   └── crawl_task/       # Task management commands & queries
  │
  ├── api/                   # API layer (gRPC service implementations)
  │   └── crawl_job/        # gRPC handlers + converters
  │
  ├── interfaces/http/       # HTTP REST interface
  │   ├── handlers/         # HTTP request handlers
  │   ├── dto/              # Data transfer objects
  │   └── server/           # HTTP server + middleware
  │
  └── infra/                 # Infrastructure layer
      ├── persistence/postgres/  # PostgreSQL implementation
      │   ├── repos/            # Repository implementations
      │   ├── converters/       # Domain model ↔ DB snapshot converters
      │   └── migrations/       # SQL schema migrations
      ├── messaging/rabbitmq/   # RabbitMQ client
      ├── services/            # Service implementations (Fetcher, Parser, Queue)
      ├── workers/             # Background workers
      └── logger/              # Zap logger implementation
```

### Domain Model Relationships

**Aggregate Root: CrawlJob**
- A `CrawlJob` represents one crawling operation (e.g., "Crawl e-commerce site")
- Contains multiple `CrawlTask` entities (one per URL)
- Tracks overall job status: InProgress → Completed/Failed

**Entity: CrawlTask**
- Represents a single URL to be crawled within a job
- Referenced by `JobID` (foreign key to CrawlJob)
- Produces one `PageSnapshot` (raw fetch result) and one `ExtractedRecord` (parsed data)

**Entity: PageSnapshot**
- Stores metadata about a fetched page (HTTP status, content type, fetch time)
- Contains `StorageKey` pointing to raw HTML/JSON in blob storage
- Referenced by `TaskID`

**Entity: ExtractedRecord**
- Contains parsed/extracted data from a page as flexible JSON (`map[string]any`)
- Referenced by `TaskID`
- Stores business-relevant information (prices, titles, etc.)

**Entity: OutboxEvent**
- Implements Outbox Pattern for reliable event publishing
- Stores domain events transactionally with business data
- Processed asynchronously by separate outbox publisher

**Entity: Preview**
- Visual representation/screenshot of crawled pages
- Useful for debugging and monitoring crawl results

**Value Objects for Configuration:**
- `Seed` - Initial URLs to start crawling
- `ScopeRules` - Domain/URL filtering rules for crawl scope
- `RetryPolicy` - Configuration for retry logic on failures
- `AuthOptions` - Authentication credentials for protected sites
- `ScheduleOptions` - Recurring crawl schedules
- `RateLimitPolicy` - Rate limiting configuration per domain
- `ExtractionSpec` - Parsing rules and selectors for data extraction
- `CrawlJobConfig` - Complete configuration for a crawl job

### Key Patterns

**Repository Pattern**
- Domain defines interfaces: `CrawlJobRepository`, `CrawlTaskRepository`, etc.
- Infrastructure provides implementations in `internal/infra/persistence/postgres/repos/`
- Repositories work with domain models, not database structs

**Converters Pattern**
- Domain models (e.g., `models.CrawlJob`) are separate from database snapshots (e.g., `snapshots.CrawlJobSnapshot`)
- Converters in `internal/infra/persistence/postgres/converters/` translate between them
- Keeps domain layer independent of persistence details

**Outbox Pattern for Event Reliability**
- Domain events (e.g., `TaskEnqueuedEvent`) stored in `outbox_events` table
- Same transaction as business data ensures consistency
- Separate process publishes events to RabbitMQ and marks as processed
- Prevents lost events and ensures at-least-once delivery

**Transaction Management**
- `TxManager` interface provides `ReadCommitted(ctx, handler)` method
- Transactions passed through `context.Context` using keys
- Repositories automatically detect and use active transaction from context
- Automatic rollback on error or panic

**Value Objects for Type Safety**
- All IDs are wrapped in value objects: `CrawlJobID`, `CrawlTaskID`, etc.
- Prevents mixing different ID types at compile time
- UUID validation enforced in value object constructors

**CQRS-inspired Commands and Queries**
- Commands: `CreateCrawlJobCommand`, `UpdateTaskStatusCommand` (mutate state)
- Queries: `GetCrawlJobQuery`, `ListTasksByJobQuery` (read state)
- Clear separation in application service layer

### Configuration

**Environment Variables** (defined in `.env` and `.worker.env`):
```
# PostgreSQL
PG_DSN=postgres://denis:some-pwd-123@localhost:54321/crawler?sslmode=disable
PG_DATABASE_NAME=crawler
PG_USER=denis
PG_PASSWORD=some-pwd-123
PG_PORT=54321
MIGRATION_DIR=./internal/infra/persistence/postgres/migrations/

# RabbitMQ (two separate queues for crawling and parsing)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_CRAWL_QUEUE_NAME=crawl_queue
RABBITMQ_PARSING_QUEUE_NAME=parsing_queue

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_USER=minioadmin
MINIO_PWD=minioadmin
MINIO_USE_SSL=false
MINIO_BUCKET_NAME=pages

# Redis
REDIS_ADDRESS=localhost:6379
REDIS_PWD=some_redis_pwd_123
REDIS_DB=0

# HTTP Server
HTTP_HOST=localhost
HTTP_PORT=8084

# gRPC Server
GRPC_HOST=localhost
GRPC_PORT=8083

# Logger
LOG_LEVEL=info
LOG_ENV=development
```

**Configuration Files:**
- `.env` - Used by gRPC/HTTP servers
- `.worker.env` - Used by fetch and parser workers (same content as `.env`)

**Configuration Interfaces** (`internal/config/config.go`):
- `PGConfig` - PostgreSQL connection settings
- `RabbitMQConfig` - RabbitMQ connection and queue names (crawl + parsing)
- `MinIOConfig` - MinIO/S3 connection and bucket settings
- `RedisConfig` - Redis connection settings
- `HTTPConfig` - HTTP server address
- `GRPCConfig` - gRPC server address
- `LoggerConfig` - Logging level and environment

**Loading Configuration:**
```go
import "distributed-crawler/internal/config"
import "distributed-crawler/internal/config/env"

// Load .env file
config.Load(".env")

// Create typed config
pgCfg, _ := env.NewPGConfig()
rmqCfg, _ := env.NewRabbitMQConfig()
```

### Database Schema

**Key Tables:**
- `crawl_jobs` - Job metadata (name, status, timestamps)
- `crawl_job_configs` - Job configuration (seeds, scope rules, extraction specs)
- `crawl_tasks` - Individual URL tasks (job_id FK, url, status, depth)
- `page_snapshots` - Fetch metadata (task_id FK, http_status, storage_key)
- `extracted_records` - Parsed data (task_id FK, data JSONB)
- `previews` - Visual previews/screenshots of crawled pages
- `outbox_events` - Domain events for reliable publishing

**Important Notes:**
- All IDs are UUIDs (stored as `uuid` type in PostgreSQL)
- Foreign keys use `ON DELETE CASCADE` for cleanup
- Indexes on all foreign keys for query performance
- `outbox_events.processed_at` NULL = unprocessed, timestamp = processed

### Infrastructure Services

**Fetcher** (`internal/infra/services/fetcher/`)
- HTTP client for fetching web pages
- Implements retry logic and rate limiting
- Respects robots.txt (recommended in docs)

**Parser** (`internal/infra/services/parser/`)
- HTML parsing using CSS/XPath selectors
- Extractor registry for different content types
- Converts raw HTML to structured data

**ContentStore** (`internal/infra/services/contentstore/`)
- S3/MinIO client for storing raw page content
- Stores HTML/JSON blobs referenced by `storage_key`
- Implementation: `minio_store.go`

**Sanitizer** (`internal/infra/services/sanitizer/`)
- HTML sanitization using bluemonday
- Cleans user-provided HTML for safe preview rendering
- Prevents XSS attacks in preview iframe

**Queue** (`internal/infra/messaging/rabbitmq/`)
- RabbitMQ client for task distribution
- Separate queues for crawl and parsing tasks
- Publishes `TaskEnqueuedEvent` messages to workers

**RateLimiter** (`internal/domain/crawl/services/`)
- Redis-based rate limiting per domain
- Prevents overwhelming target sites
- Configurable via `RateLimitPolicy`

**ScopeValidator** (`internal/domain/crawl/services/`)
- Validates URLs against scope rules
- Enforces domain and pattern restrictions

**Workers** (separate processes in `cmd/`):
- `fetch_worker` - Consumes from `crawl_queue`, fetches pages, stores snapshots, publishes to `parsing_queue`
- `parser_worker` - Consumes from `parsing_queue`, parses pages, extracts records

## Code Style and Conventions

### Naming Conventions
- Domain models: `CrawlJob`, `CrawlTask` (PascalCase nouns)
- Value objects: `CrawlJobID`, `TaskStatus` (PascalCase with ID suffix)
- Repositories: `CrawlJobRepository` (noun + Repository)
- Services: `CrawlJobService` (noun + Service)
- Commands: `CreateCrawlJobCommand` (Verb + Noun + Command)
- Queries: `GetCrawlJobQuery` (Verb + Noun + Query)
- Events: `TaskEnqueuedEvent` (PastTense + Event)

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to create job: %w", err)`
- Return errors, don't panic (except in truly exceptional cases)
- Domain layer returns domain-specific errors (e.g., `ErrJobNotFound`)
- Infrastructure layer may return wrapped errors

### Testing Conventions
- Test files in `tests/` subdirectory within package
- Use `testify` for assertions: `require.NoError(t, err)`
- Use `minimock` for mocking interfaces (see `internal/tools/mocks_helper.go`)
- Generate mocks with: `go generate ./...`

### Known Technical Debt

1. **Nullable Fields** - `CompletedAt` field conversion has TODO comments in converters (`internal/infra/persistence/postgres/converters/crawl_job.go:14,24`)
   - Use `sql.NullTime` for database and `*time.Time` for domain model

2. **Logging** - `pgdb.go` uses `log.Println` instead of structured logger
   - Should use Zap logger (`internal/infra/logger/zap_loggger.go`)

3. **Empty Implementations** - Some repository files may be stubs:
   - Check domain services in `domain_services/` for completeness

## Important Implementation Notes

### When Adding New Domain Events
1. Create event struct in `internal/domain/crawl/events/`
2. Embed `BaseEvent` for ID and timestamp
3. Define event type constant (e.g., `EventTypeTaskEnqueued`)
4. Marshal to JSON and store in `OutboxEvent` table
5. Implement outbox publisher to consume and publish to RabbitMQ

### When Adding New Repositories
1. Define interface in `internal/domain/crawl/repos/<entity>/`
2. Implement in `internal/infra/persistence/postgres/repos/`
3. Create DB snapshot struct in `snapshots/`
4. Create converter functions in `converters/`
5. Use `TxManager` for operations requiring transactions

### When Creating Database Migrations
```bash
# Navigate to project root
goose -dir ./internal/infra/persistence/postgres/migrations create <description> sql
```
- Use descriptive names: `add_crawl_jobs_table`, `add_index_on_tasks_status`
- Include both `-- +goose Up` and `-- +goose Down` sections
- Add indexes for foreign keys and frequently queried columns
- Use `ON DELETE CASCADE` for foreign keys when child records should be deleted

### Message Format for RabbitMQ

**Crawl Queue** (`crawl_queue`) - Tasks for fetching pages:
```json
{
  "task_id": "uuid",
  "job_id": "uuid",
  "url": "https://example.com",
  "depth": 0,
  "enqueued_at": "2026-01-09T00:00:00Z"
}
```

**Parsing Queue** (`parsing_queue`) - Tasks for parsing fetched pages:
```json
{
  "task_id": "uuid",
  "job_id": "uuid",
  "storage_key": "s3://bucket/key",
  "enqueued_at": "2026-01-09T00:00:00Z"
}
```

**Queue Flow:**
1. API creates `CrawlTask` and publishes to `crawl_queue`
2. Fetch worker consumes from `crawl_queue`, downloads page, stores in MinIO
3. Fetch worker creates `PageSnapshot` and publishes to `parsing_queue`
4. Parser worker consumes from `parsing_queue`, extracts data, stores `ExtractedRecord`

## gRPC and Protobuf

**Proto Definitions:** Located in `api/` directory (exact structure TBD)
**Generated Code:** Managed by `buf generate` (see `vendor.proto.mk`)

**Regenerate Protobuf Code:**
```bash
make generate
```
This runs:
- `buf generate` - generates Go code from `.proto` files
- `statik` - embeds Swagger UI files for API documentation

**gRPC Middleware:**
- Validation: `internal/interceptor/validate.go`
- Logging: `internal/interceptor/logger.go`

## Project Context

This is a **learning/portfolio project** demonstrating:
- Clean Architecture / Hexagonal Architecture
- Domain-Driven Design (aggregates, value objects, domain events)
- Event sourcing with Outbox pattern
- CQRS-inspired command/query separation
- Repository pattern
- Message-driven worker architecture

**Language:** Primarily Russian comments/docs, Go code follows English conventions.

**Go Version:** 1.25.4 (see `go.mod`)

## Running a Complete Crawl Job

To test the full system end-to-end:

1. **Start infrastructure**:
   ```bash
   docker compose up -d
   ```

2. **Run migrations**:
   ```bash
   make local-migration-up
   ```

3. **Start the gRPC server** (in terminal 1):
   ```bash
   make run-grpc-server
   ```

4. **Start the fetch worker** (in terminal 2):
   ```bash
   make run-fetcher
   ```

5. **Start the parser worker** (in terminal 3):
   ```bash
   make run-parser
   ```

6. **Create a crawl job** via gRPC/HTTP API or use `cmd/http_client` or `cmd/grpc_service`

The system will:
- Accept the job via API
- Create tasks in the database
- Publish tasks to `crawl_queue`
- Fetch worker downloads pages and publishes to `parsing_queue`
- Parser worker extracts data and stores results

7. **Access the UI** (optional, in terminal 4):
   ```bash
   cd ui
   npm install
   npm start
   # Navigate to http://localhost:4200
   ```

## Project Structure Summary

```
distributed-crawler/
├── cmd/                    # Entry points (binaries)
├── internal/              # Application code
│   ├── domain/           # Domain layer (business logic)
│   ├── application/      # Use cases (commands & queries)
│   ├── api/              # gRPC service implementations
│   ├── interfaces/       # HTTP handlers
│   ├── infra/            # Infrastructure implementations
│   └── config/           # Configuration management
├── ui/                    # Angular frontend (MVP)
├── docs/                  # Documentation
├── api/                   # Protobuf definitions
├── Makefile              # Build and task automation
├── docker-compose.yaml   # Infrastructure services
├── .env                  # Environment configuration
└── .worker.env           # Worker-specific configuration
```

## Key Dependencies

**Core Libraries:**
- `github.com/jackc/pgx/v4` - PostgreSQL driver
- `github.com/rabbitmq/amqp091-go` - RabbitMQ client
- `github.com/minio/minio-go/v7` - MinIO/S3 client
- `github.com/redis/go-redis/v9` - Redis client
- `google.golang.org/grpc` - gRPC framework
- `github.com/grpc-ecosystem/grpc-gateway/v2` - REST gateway for gRPC

**Parsing & Processing:**
- `github.com/PuerkitoBio/goquery` - HTML parsing (jQuery-like)
- `github.com/microcosm-cc/bluemonday` - HTML sanitization

**Testing:**
- `github.com/stretchr/testify` - Assertions and test utilities
- `github.com/gojuno/minimock/v3` - Mock generation

**Utilities:**
- `go.uber.org/zap` - Structured logging
- `github.com/google/uuid` - UUID generation
- `github.com/Masterminds/squirrel` - SQL query builder
