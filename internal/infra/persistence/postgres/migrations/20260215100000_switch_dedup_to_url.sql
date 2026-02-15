-- +goose Up
-- +goose StatementBegin

-- Drop old body_hash deduplication index
DROP INDEX IF EXISTS uq_crawl_tasks_job_id_body_hash;

-- Drop body_hash index
DROP INDEX IF EXISTS idx_crawl_tasks_body_hash;

-- Remove body_hash column
ALTER TABLE crawl_tasks DROP COLUMN IF EXISTS body_hash;

-- Add unique index for URL-based deduplication within a job
CREATE UNIQUE INDEX IF NOT EXISTS uq_crawl_tasks_job_id_url
    ON crawl_tasks (job_id, url);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove URL dedup index
DROP INDEX IF EXISTS uq_crawl_tasks_job_id_url;

-- Re-add body_hash column
ALTER TABLE crawl_tasks ADD COLUMN body_hash VARCHAR(255) NULL;

-- Re-add body_hash indexes
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_body_hash
    ON crawl_tasks (body_hash);

CREATE UNIQUE INDEX IF NOT EXISTS uq_crawl_tasks_job_id_body_hash
    ON crawl_tasks (job_id, body_hash)
    WHERE body_hash IS NOT NULL;

-- +goose StatementEnd
