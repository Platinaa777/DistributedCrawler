# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Distributed Crawling System** - A scalable web crawling platform built with Go that uses RabbitMQ for task distribution, PostgreSQL for data persistence, and implements clean architecture with domain-driven design principles.

**Key Components:**
- **Coordinator**: Central service that manages crawl jobs and distributes tasks
- **Workers**: Processes that fetch pages, parse content, and store results
- **RabbitMQ**: Message queue for task distribution and worker coordination
- **PostgreSQL**: Persistent storage for normalized crawl data
- **S3/MinIO**: Blob storage for raw HTML/JSON pages (optional)

## Development Commands

### Environment Setup

```bash
# Start infrastructure services (PostgreSQL + RabbitMQ)
docker compose up -d

# Install build dependencies (protoc plugins, goose, statik)
make .bin-deps

# Generate code from protobuf definitions
make generate

# Tidy dependencies
make .tidy
```

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

# Run the HTTP server (development)
make run
# Runs: go run ./cmd/http_server/main.go

# Run specific commands directly
go run ./cmd/coordinator
go run ./cmd/worker
go run ./cmd/grpc_server/main.go
```

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

## Architecture

### Clean Architecture Layers

```
cmd/                          # Entry points (main.go files)
  ├── grpc_server/           # gRPC API server
  ├── http_server/           # HTTP REST API server
  ├── coordinator/           # Coordinator service (planned)
  └── worker/                # Worker process (planned)

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

**Environment Variables** (defined in `.env`):
```
# PostgreSQL
PG_DATABASE_NAME=crawler
PG_USER=denis
PG_PASSWORD=some-pwd-123
PG_PORT=54321
MIGRATION_DIR=./internal/infra/persistence/postgres/migrations/

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_QUEUE_NAME=crawl_tasks
```

**Configuration Interfaces** (`internal/config/config.go`):
- `PGConfig` - PostgreSQL connection settings
- `RabbitMQConfig` - RabbitMQ connection and queue name
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
- `crawl_tasks` - Individual URL tasks (job_id FK, url, status)
- `page_snapshots` - Fetch metadata (task_id FK, http_status, storage_key)
- `extracted_records` - Parsed data (task_id FK, data JSONB)
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

**Queue** (`internal/infra/services/queue/`)
- RabbitMQ client for task distribution
- In-memory queue implementation for testing
- Publishes `TaskEnqueuedEvent` messages to workers

**Workers** (`internal/infra/workers/`)
- `fetch_worker.go` - Consumes tasks, fetches pages, stores snapshots
- `parse_worker.go` - Parses fetched pages, extracts records
- `scheduler_worker.go` - Manages scheduled/recurring crawl jobs

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

2. **Typo in Interface** - `ReadCommited` should be `ReadCommitted` (`internal/infra/persistence/dbtransaction.go:6`)

3. **Logging** - `pgdb.go` uses `log.Println` instead of structured logger
   - Should use Zap logger (`internal/infra/logger/zap_loggger.go`)

4. **Empty Implementations** - Some repository files are stubs:
   - `page_snapshot_repo.go`
   - `extracted_record_repo.go`
   - Domain services in `domain_services/`

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
Tasks are published as JSON messages to `crawl_tasks` queue:
```json
{
  "task_id": "uuid",
  "job_id": "uuid",
  "url": "https://example.com",
  "enqueued_at": "2026-01-09T00:00:00Z"
}
```

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
