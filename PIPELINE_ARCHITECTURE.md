# 2-Stage Crawler Pipeline Architecture

## Overview

The distributed crawler has been refactored into a **2-stage pipeline** architecture with MinIO for raw content storage:

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐      ┌──────────────┐
│ Coordinator │─────▶│ crawl_queue  │─────▶│ FetchWorker │─────▶│ parsing_queue│
└─────────────┘      └──────────────┘      └─────────────┘      └──────────────┘
                                                   │                      │
                                                   ▼                      ▼
                                            ┌────────────┐        ┌──────────────┐
                                            │   MinIO    │◀───────│ParserWorker  │
                                            │  (pages)   │        └──────────────┘
                                            └────────────┘               │
                                                   │                     ▼
                                                   └──────────▶┌─────────────────┐
                                                               │   PostgreSQL    │
                                                               │ - page_fetch    │
                                                               │ - page_extract  │
                                                               │ - page_link     │
                                                               │ - page_image    │
                                                               └─────────────────┘
```

## Architecture Components

### Stage 1: FetchWorker

**Responsibilities:**
- Consumes `CrawlTaskMessage` from `crawl_queue`
- Performs HTTP GET request (30s timeout, follows redirects, custom UA)
- Collects fetch metadata:
  - `final_url` (after redirects)
  - `status_code`
  - `duration_ms`
  - `headers` (JSONB)
  - `content_type`
  - `content_length`
  - `body_hash` (SHA-256 of HTML)
- Uploads raw HTML to MinIO:
  - Bucket: `pages`
  - Object key: `pages/{job_id}/{task_id}.html`
- Saves fetch metadata to PostgreSQL (`page_fetch` table)
- Publishes `ParsingTaskMessage` to `parsing_queue`
- **Manual ACK** only after successful publish

**Location:** `internal/worker/fetch_worker.go`

### Stage 2: ParserWorker

**Responsibilities:**
- Consumes `ParsingTaskMessage` from `parsing_queue`
- Loads fetch metadata from PostgreSQL to get MinIO object key
- Retrieves raw HTML from MinIO (no HTTP request)
- Parses HTML using `goquery`:
  - Extracts: `title`, `meta description`, `canonical URL`
  - Collects all links with anchor text and external flag
  - Collects all images with alt text
- Computes features:
  - `link_count`, `image_count`, `external_link_count`, `word_count`
- Saves results to PostgreSQL:
  - `page_extract` table (UPSERT for idempotency)
  - `page_link` table (bulk insert with `ON CONFLICT DO NOTHING`)
  - `page_image` table (bulk insert with `ON CONFLICT DO NOTHING`)
- **Manual ACK** only after successful DB commit

**Location:** `internal/worker/parser_worker.go`

## Database Schema

### page_fetch
Stores fetch metadata and MinIO reference (1:1 with task).

```sql
CREATE TABLE page_fetch (
    task_id UUID PRIMARY KEY,
    job_id UUID NOT NULL,
    url TEXT NOT NULL,
    final_url TEXT,
    status_code INT NOT NULL,
    duration_ms INT NOT NULL,
    headers JSONB,
    content_type VARCHAR(255),
    content_length BIGINT,
    body_hash VARCHAR(64) NOT NULL,  -- SHA-256
    minio_object_key VARCHAR(512) NOT NULL,
    fetched_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### page_extract
Stores parsed results and computed features (1:1 with page_fetch).

```sql
CREATE TABLE page_extract (
    task_id UUID PRIMARY KEY,
    title TEXT,
    meta_description TEXT,
    canonical_url TEXT,
    metadata JSONB,  -- Flexible storage
    link_count INT NOT NULL DEFAULT 0,
    image_count INT NOT NULL DEFAULT 0,
    external_link_count INT NOT NULL DEFAULT 0,
    word_count INT NOT NULL DEFAULT 0,
    parsed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_extract_fetch FOREIGN KEY (task_id) REFERENCES page_fetch(task_id) ON DELETE CASCADE
);
```

### page_link
Stores extracted links (1:N with page_extract).

```sql
CREATE TABLE page_link (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    url TEXT NOT NULL,
    anchor_text TEXT,
    is_external BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_link_extract FOREIGN KEY (task_id) REFERENCES page_extract(task_id) ON DELETE CASCADE,
    CONSTRAINT uq_page_link_task_url UNIQUE (task_id, url)  -- Idempotency
);
```

### page_image
Stores extracted images (1:N with page_extract).

```sql
CREATE TABLE page_image (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    url TEXT NOT NULL,
    alt_text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_image_extract FOREIGN KEY (task_id) REFERENCES page_extract(task_id) ON DELETE CASCADE,
    CONSTRAINT uq_page_image_task_url UNIQUE (task_id, url)  -- Idempotency
);
```

## Message Formats

### crawl_queue (consumed by FetchWorker)

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "job_id": "660e8400-e29b-41d4-a716-446655440001",
  "url": "https://example.com/page.html",
  "enqueued_at": "2026-01-09T10:00:00Z"
}
```

### parsing_queue (consumed by ParserWorker)

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "job_id": "660e8400-e29b-41d4-a716-446655440001",
  "enqueued_at": "2026-01-09T10:01:00Z"
}
```

## Domain Models

### PageFetch (`internal/domain/crawl/models/page_fetch.go`)

```go
type PageFetch struct {
    TaskID          valueobjects.CrawlTaskID
    JobID           valueobjects.CrawlJobID
    URL             string
    FinalURL        *string
    StatusCode      int
    DurationMs      int
    Headers         map[string]string
    ContentType     *string
    ContentLength   *int64
    BodyHash        string
    MinioObjectKey  string
    FetchedAt       time.Time
    CreatedAt       time.Time
}
```

### PageExtract (`internal/domain/crawl/models/page_extract.go`)

```go
type PageExtract struct {
    TaskID             valueobjects.CrawlTaskID
    Title              *string
    MetaDescription    *string
    CanonicalURL       *string
    Metadata           map[string]any  // Flexible JSONB
    LinkCount          int
    ImageCount         int
    ExternalLinkCount  int
    WordCount          int
    ParsedAt           time.Time
    CreatedAt          time.Time
}
```

### PageLink, PageImage (`internal/domain/crawl/models/`)

```go
type PageLink struct {
    ID         valueobjects.PageLinkID
    TaskID     valueobjects.CrawlTaskID
    URL        string
    AnchorText *string
    IsExternal bool
    CreatedAt  time.Time
}

type PageImage struct {
    ID        valueobjects.PageImageID
    TaskID    valueobjects.CrawlTaskID
    URL       string
    AltText   *string
    CreatedAt time.Time
}
```

## Infrastructure

### MinIO ContentStore

**Interface:** `internal/domain/crawl/services/content_store.go`

```go
type ContentStore interface {
    Store(ctx context.Context, key string, content []byte, contentType string) error
    Get(ctx context.Context, key string) ([]byte, error)
    GetReader(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

**Implementation:** `internal/infra/services/contentstore/minio_store.go`

Uses `minio-go` SDK to interact with MinIO/S3.

### Repositories

**PageFetchRepository** (`internal/infra/persistence/postgres/repos/page_fetch_repo.go`):
- `Save(ctx, *PageFetch)` - UPSERT by task_id
- `GetByTaskID(ctx, taskID)` - Retrieve fetch metadata
- `GetByJobID(ctx, jobID)` - List all fetches for a job

**PageExtractRepository** (`internal/infra/persistence/postgres/repos/page_extract_repo.go`):
- `Save(ctx, *PageExtract)` - UPSERT by task_id
- `SaveLinks(ctx, []*PageLink)` - Bulk insert with ON CONFLICT DO NOTHING
- `SaveImages(ctx, []*PageImage)` - Bulk insert with ON CONFLICT DO NOTHING
- `GetByTaskID(ctx, taskID)` - Retrieve extract
- `GetLinksByTaskID(ctx, taskID)` - List links
- `GetImagesByTaskID(ctx, taskID)` - List images

## Configuration

Add to `.env`:

```env
# MinIO Configuration
MINIO_ENDPOINT=localhost:9000
MINIO_USER=user
MINIO_PWD=pwd
MINIO_USE_SSL=false
MINIO_BUCKET_NAME=pages

# RabbitMQ (existing)
RABBITMQ_URL=rmq_url
RABBITMQ_QUEUE_NAME=queue_name  # Not used directly, workers specify queue names
```

## Running the System

### 1. Start Infrastructure

```bash
docker compose up -d  # Starts PostgreSQL, RabbitMQ, MinIO
```

### 2. Run Database Migrations

```bash
make local-migration-up
```

This creates the new tables: `page_fetch`, `page_extract`, `page_link`, `page_image`.

### 3. Start Workers

**Terminal 1 - Fetch Worker:**
```bash
go run ./cmd/fetch_worker/main.go
```

**Terminal 2 - Parser Worker:**
```bash
go run ./cmd/parser_worker/main.go
```

You can run multiple instances of each worker for horizontal scaling.

### 4. Publish Tasks to crawl_queue

Use RabbitMQ management UI (http://localhost:15672) or publish via code:

```go
task := rabbitmq.CrawlTaskMessage{
    TaskID:     "550e8400-e29b-41d4-a716-446655440000",
    JobID:      "660e8400-e29b-41d4-a716-446655440001",
    URL:        "https://example.com",
    EnqueuedAt: time.Now(),
}
msgBytes, _ := json.Marshal(task)
rmqClient.Publish(ctx, "crawl_queue", msgBytes)
```

## Key Features

### Idempotency

- **page_fetch**: UPSERT by `task_id` - safe to retry
- **page_extract**: UPSERT by `task_id` - safe to retry
- **page_link/page_image**: `ON CONFLICT (task_id, url) DO NOTHING` - duplicates ignored

### Reliability

- **Manual ACK**: Workers only acknowledge messages after successful processing
- **Transaction Safety**: All DB writes happen before publishing to next queue
- **Separation of Concerns**: Fetch failures don't affect parsing, parsing failures don't require re-fetching

### Scalability

- **Horizontal**: Run multiple instances of each worker
- **Vertical**: Increase worker concurrency (RabbitMQ prefetch)
- **Storage**: MinIO can handle large volumes of HTML
- **Stateless**: Workers can be restarted anytime

## Monitoring

### MinIO UI
http://localhost:9001 (minioadmin / minioadmin)
- View stored pages in `pages` bucket
- Check storage usage

### RabbitMQ UI
http://localhost:15672 (guest / guest)
- Monitor queue depths: `crawl_queue`, `parsing_queue`
- Track message rates and consumer counts

### PostgreSQL Queries

```sql
-- Fetch statistics
SELECT
    status_code,
    COUNT(*) as count,
    AVG(duration_ms) as avg_duration_ms
FROM page_fetch
GROUP BY status_code;

-- Parsing statistics
SELECT
    AVG(link_count) as avg_links,
    AVG(image_count) as avg_images,
    AVG(word_count) as avg_words
FROM page_extract;

-- Top external domains linked
SELECT
    SUBSTRING(url FROM 'https?://([^/]+)') as domain,
    COUNT(*) as count
FROM page_link
WHERE is_external = true
GROUP BY domain
ORDER BY count DESC
LIMIT 10;
```

## Migration Notes

### Old scraper_worker.go

The previous `internal/worker/scraper_worker.go` did everything in one stage:
- HTTP fetch
- HTML parsing
- Console output

**Migration path:**
1. The new FetchWorker + ParserWorker architecture replaces it
2. You can delete `scraper_worker.go` or keep it for reference
3. Benefits of new architecture:
   - Raw HTML preserved in MinIO (can re-parse later)
   - Fetch and parse can scale independently
   - Failed parses don't require re-fetching
   - Structured data in PostgreSQL (queryable, relational)

## Next Steps

### Potential Enhancements

1. **Task Status Tracking**: Add `crawl_tasks` table to track task state (Pending → Fetching → Parsing → Done)
2. **Error Handling**: Add retry logic with exponential backoff
3. **Dead Letter Queues**: Route failed tasks to DLQ for manual inspection
4. **Content Deduplication**: Use `body_hash` to detect duplicate content
5. **Incremental Crawling**: Check `body_hash` before re-parsing unchanged pages
6. **Custom Extractors**: Extend `PageExtract.Metadata` with site-specific extraction rules
7. **Link Discovery**: Create new `CrawlTaskMessage` for discovered links (breadth-first crawling)

## Troubleshooting

### Workers not processing messages
- Check RabbitMQ connection: `rabbitmqctl list_connections`
- Verify queue declarations: `rabbitmqctl list_queues`
- Check worker logs for errors

### MinIO upload failures
- Verify MinIO is running: `docker ps | grep minio`
- Check bucket exists: MinIO UI → Buckets
- Verify credentials in `.env`

### Database errors
- Run migrations: `make local-migration-up`
- Check PostgreSQL logs: `docker logs <postgres-container>`
- Verify connection string in `.env`

## Files Created/Modified

### New Files
- `internal/domain/crawl/models/page_fetch.go`
- `internal/domain/crawl/models/page_extract.go`
- `internal/domain/crawl/models/page_link.go`
- `internal/domain/crawl/models/page_image.go`
- `internal/domain/crawl/valueobjects/page_link_id.go`
- `internal/domain/crawl/valueobjects/page_image_id.go`
- `internal/domain/crawl/services/content_store.go`
- `internal/domain/crawl/repos/page_fetch/page_fetch_repository.go`
- `internal/domain/crawl/repos/page_extract/page_extract_repository.go`
- `internal/infra/services/contentstore/minio_store.go`
- `internal/infra/persistence/postgres/snapshots/page_fetch_snapshot.go`
- `internal/infra/persistence/postgres/snapshots/page_extract_snapshot.go`
- `internal/infra/persistence/postgres/converters/page_fetch.go`
- `internal/infra/persistence/postgres/converters/page_extract.go`
- `internal/infra/persistence/postgres/repos/page_fetch_repo.go`
- `internal/infra/persistence/postgres/repos/page_extract_repo.go`
- `internal/infra/persistence/postgres/migrations/20260109160843_page_fetch_parse_tables.sql`
- `internal/config/env/minio.go`
- `internal/worker/fetch_worker.go`
- `internal/worker/parser_worker.go`
- `cmd/fetch_worker/main.go`
- `cmd/parser_worker/main.go`

### Modified Files
- `internal/config/config.go` (added MinIOConfig interface)
- `internal/infra/messaging/rabbitmq/messages.go` (added ParsingTaskMessage)

### Migration File
- `internal/infra/persistence/postgres/migrations/20260109160843_page_fetch_parse_tables.sql`
