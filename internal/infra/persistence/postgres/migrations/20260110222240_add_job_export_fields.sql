-- +goose Up
-- +goose StatementBegin
-- Add export fields to crawl_jobs for Part B (ExportWorker)
ALTER TABLE crawl_jobs
    ADD COLUMN export_json_key TEXT NULL,
    ADD COLUMN export_csv_key TEXT NULL,
    ADD COLUMN exported_at TIMESTAMP NULL,
    ADD COLUMN export_status VARCHAR(50) NULL DEFAULT 'NOT_STARTED';

-- Index for querying jobs eligible for export
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_export_status
    ON crawl_jobs (export_status);

-- Index for finding finished jobs that haven't been exported
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_completed_not_exported
    ON crawl_jobs (completed_at, export_status)
    WHERE completed_at IS NOT NULL AND export_status = 'NOT_STARTED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove export fields from crawl_jobs
DROP INDEX IF EXISTS idx_crawl_jobs_completed_not_exported;
DROP INDEX IF EXISTS idx_crawl_jobs_export_status;

ALTER TABLE crawl_jobs
    DROP COLUMN IF EXISTS export_status,
    DROP COLUMN IF EXISTS exported_at,
    DROP COLUMN IF EXISTS export_csv_key,
    DROP COLUMN IF EXISTS export_json_key;
-- +goose StatementEnd
