-- +goose NO TRANSACTION
-- +goose Up
-- Add composite index for cursor-based pagination on crawl_jobs
-- Optimizes queries with ORDER BY created_at DESC, id DESC and cursor conditions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_crawl_jobs_cursor_pagination
    ON crawl_jobs (created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_crawl_jobs_cursor_pagination;
