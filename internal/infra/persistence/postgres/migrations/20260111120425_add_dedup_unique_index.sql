-- +goose Up
-- +goose StatementBegin

-- Add unique index to prevent duplicate tasks with same body_hash within a job
-- This enforces deduplication at the database level to avoid race conditions
CREATE UNIQUE INDEX IF NOT EXISTS uq_crawl_tasks_job_id_body_hash
    ON crawl_tasks (job_id, body_hash)
    WHERE body_hash IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove the unique index
DROP INDEX IF EXISTS uq_crawl_tasks_job_id_body_hash;

-- +goose StatementEnd
