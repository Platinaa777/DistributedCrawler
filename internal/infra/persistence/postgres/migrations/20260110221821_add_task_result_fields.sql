-- +goose Up
-- +goose StatementBegin
-- Add result fields to crawl_tasks for Part A (ParserWorker result persistence)
ALTER TABLE crawl_tasks
    ADD COLUMN result_object_key TEXT NULL,
    ADD COLUMN result_content_type VARCHAR(100) NULL DEFAULT 'application/json',
    ADD COLUMN result_size_bytes BIGINT NULL,
    ADD COLUMN result_created_at TIMESTAMP NULL;

-- Index for querying tasks with results
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_result_object_key
    ON crawl_tasks (result_object_key) WHERE result_object_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove result fields from crawl_tasks
DROP INDEX IF EXISTS idx_crawl_tasks_result_object_key;

ALTER TABLE crawl_tasks
    DROP COLUMN IF EXISTS result_created_at,
    DROP COLUMN IF EXISTS result_size_bytes,
    DROP COLUMN IF EXISTS result_content_type,
    DROP COLUMN IF EXISTS result_object_key;
-- +goose StatementEnd
