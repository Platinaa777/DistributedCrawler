# Export System Documentation

This document describes the result persistence and export pipeline for the Distributed Web Crawler.

## Overview

The export system consists of two parts:
- **Part A**: ParserWorker now persists extraction results to S3 and stores references in the database
- **Part B**: ExportWorker aggregates results from completed crawl jobs into JSON and CSV reports

## Part A: ParserWorker Result Persistence

### Changes

The ParserWorker no longer prints results to stdout. Instead, it:

1. **Marshals extraction output to JSON** with the same structure:
   ```json
   {
     "task_id": "...",
     "url": "...",
     "fields": {...},
     "metrics": {...}
   }
   ```

2. **Uploads to S3** under deterministic path:
   ```
   results/tasks/{task_id}.json
   ```

3. **Stores reference in database** on the `crawl_tasks` table:
   - `result_object_key` - S3 object key (e.g., `results/tasks/{task_id}.json`)
   - `result_content_type` - Content type (`application/json`)
   - `result_size_bytes` - File size in bytes
   - `result_created_at` - Timestamp when result was created

### Database Migration

Migration file: `20260110221821_add_task_result_fields.sql`

Adds columns to `crawl_tasks`:
```sql
ALTER TABLE crawl_tasks
    ADD COLUMN result_object_key TEXT NULL,
    ADD COLUMN result_content_type VARCHAR(100) NULL DEFAULT 'application/json',
    ADD COLUMN result_size_bytes BIGINT NULL,
    ADD COLUMN result_created_at TIMESTAMP NULL;
```

### Idempotency

ParserWorker skips uploading if `result_object_key` is already set (no force flag for MVP).

## Part B: ExportWorker

### Purpose

Aggregates crawl results into job-level export files (JSON and CSV) after all tasks are completed or failed.

### Job Eligibility

A job qualifies for export when:
- `completed_at IS NOT NULL` (job is finished)
- `export_status = 'NOT_STARTED'` (not yet exported)

The worker does NOT check individual task statuses beyond what's stored in the `result_object_key` field.

### Export Process

1. **Poll for eligible jobs** (default: every 30 seconds, batch size: 10)
2. **Atomically transition** `export_status` from `NOT_STARTED` to `IN_PROGRESS`
3. **Load all tasks** for the job from database
4. **Load result JSON** for each completed task from S3 (using `result_object_key`)
5. **Generate reports**:
   - **JSON report**: Array of all task results
   - **CSV report**: Flattened fields with dynamic columns
6. **Upload to S3**:
   ```
   exports/jobs/{job_id}/report.json
   exports/jobs/{job_id}/report.csv
   ```
7. **Mark export as completed** in database

### Database Fields (crawl_jobs)

Migration file: `20260110222240_add_job_export_fields.sql`

Added columns:
```sql
ALTER TABLE crawl_jobs
    ADD COLUMN export_json_key TEXT NULL,
    ADD COLUMN export_csv_key TEXT NULL,
    ADD COLUMN exported_at TIMESTAMP NULL,
    ADD COLUMN export_status VARCHAR(50) NULL DEFAULT 'NOT_STARTED';
```

**Export Status Values:**
- `NOT_STARTED` - Job is eligible for export
- `IN_PROGRESS` - Export is currently running
- `COMPLETED` - Export finished successfully
- `FAILED` - Export failed (error stored in `error` column)

### S3 Paths

**Task Results** (Part A - ParserWorker):
```
results/tasks/{task_id}.json
```

**Job Exports** (Part B - ExportWorker):
```
exports/jobs/{job_id}/report.json
exports/jobs/{job_id}/report.csv
```

### JSON Report Format

```json
{
  "job_id": "...",
  "exported_at": "2026-01-10T22:30:00Z",
  "total_tasks": 150,
  "results": [
    {
      "task_id": "...",
      "url": "https://example.com/page1",
      "status": "Completed",
      "fields": {
        "title": "Page Title",
        "price": 19.99
      },
      "metrics": {
        "word_count": 500
      }
    },
    {
      "task_id": "...",
      "url": "https://example.com/page2",
      "status": "Failed",
      "error": "task failed during crawl/parse"
    }
  ]
}
```

### CSV Report Format

The CSV report flattens the results:

```csv
task_id,url,status,title,price,error
abc-123,https://example.com/page1,Completed,Page Title,19.99,
def-456,https://example.com/page2,Failed,,,task failed during crawl/parse
```

**Dynamic Columns:**
- Header includes `task_id`, `url`, `status`
- Followed by union of all field names across completed tasks (sorted alphabetically)
- Ends with `error` column

**Missing Fields:**
- If a task doesn't have a field, CSV cell is empty

**Metrics:**
- Not included in CSV (only in JSON report)
- Could be added as separate columns or JSON string if needed

### Configuration

**ExportWorker Settings** (in `internal/app/worker_app.go`):
```go
pollInterval := 30 * time.Second // Poll every 30 seconds
batchSize := 10                   // Process up to 10 jobs per batch
```

These can be made configurable via environment variables if needed.

### Running the Workers

**Start the ExportWorker:**
```bash
make run-export
```

Or directly:
```bash
go run ./cmd/export_worker/main.go --config-path=.worker.env
```

**Complete System:**
```bash
# Terminal 1: gRPC Server
make run-grpc-server

# Terminal 2: Fetch Worker
make run-fetcher

# Terminal 3: Parser Worker (now persists to S3)
make run-parser

# Terminal 4: Export Worker (aggregates completed jobs)
make run-export
```

### Idempotency & Locking

**Job-level Export:**
- `TryStartExport()` uses compare-and-swap (UPDATE WHERE export_status = 'NOT_STARTED')
- Only one worker instance can start export for a job
- If `export_status = 'COMPLETED'`, job is skipped
- If `export_status = 'IN_PROGRESS'` (stale), worker skips for MVP

**Task-level Results:**
- ParserWorker checks if `result_object_key` is set before uploading
- Prevents duplicate uploads on retries

### Error Handling

**ParserWorker:**
- If S3 upload fails: returns error, task can be retried
- If DB update fails after upload: logs error and returns error for retry

**ExportWorker:**
- If export fails at any step: marks job as `FAILED`
- Error message stored in `crawl_jobs.error` column
- Failed jobs can be manually reset to `NOT_STARTED` for retry

### Repository Methods

**CrawlTaskRepository:**
```go
SetTaskResult(ctx, taskID, objectKey, contentType, sizeBytes) error
```

**CrawlJobRepository:**
```go
ListEligibleForExport(ctx, limit) ([]*CrawlJob, error)
TryStartExport(ctx, jobID) (bool, error)
CompleteExport(ctx, jobID, jsonKey, csvKey) error
FailExport(ctx, jobID, errorMsg) error
```

### Migrations

Run migrations to add the new fields:
```bash
make local-migration-up
```

This applies:
1. `20260110221821_add_task_result_fields.sql` - Task result fields
2. `20260110222240_add_job_export_fields.sql` - Job export fields

### Testing

**Manual Test Flow:**

1. Create a crawl job via API
2. Let FetchWorker and ParserWorker process tasks
3. Check S3 bucket for `results/tasks/*.json` files
4. Check database `crawl_tasks.result_object_key` is populated
5. Mark job as completed (or wait for all tasks to finish)
6. ExportWorker will detect eligible job and generate exports
7. Check S3 bucket for `exports/jobs/{job_id}/report.json` and `report.csv`
8. Check database `crawl_jobs.export_status = 'COMPLETED'`

### Future Enhancements

**Not implemented (out of scope for MVP):**
- UI for downloading exports
- Export to external systems (webhooks, databases, etc.)
- Stale `IN_PROGRESS` cleanup (currently ignored)
- Multiple export formats (Excel, Parquet, etc.)
- Export filtering/pagination (currently exports all tasks)
- Incremental exports for very large jobs
- Export progress tracking
- Email notifications when export completes

### Architecture Notes

**Clean Architecture Maintained:**
- Domain models updated with export fields (`CrawlJob`, `CrawlTask`)
- Repository interfaces extended with export methods
- Infrastructure layer implements persistence
- Worker layer handles background processing
- No changes to API layer (exports accessible via S3)

**S3-First Approach:**
- Results stored in S3, DB only holds references
- Scales to large crawl jobs without DB bloat
- Export files also in S3 for durability and access control

**Transactional Outbox Pattern:**
- Not used for export status transitions (no domain events)
- Direct DB updates for simplicity
- Atomic CAS ensures no duplicate exports
