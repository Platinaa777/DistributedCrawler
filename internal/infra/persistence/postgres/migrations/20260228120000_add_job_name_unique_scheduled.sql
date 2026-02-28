-- +goose Up
-- Add unique constraint on crawl_job_configs.name
ALTER TABLE crawl_job_configs ADD CONSTRAINT uq_crawl_job_configs_name UNIQUE (name);

-- Add name column to crawl_jobs for scheduled job run identification
-- For scheduled jobs this is set to "{config_name}_{scheduled_at}" (RFC3339)
ALTER TABLE crawl_jobs ADD COLUMN name VARCHAR(500) NULL;

-- +goose Down
ALTER TABLE crawl_jobs DROP COLUMN name;
ALTER TABLE crawl_job_configs DROP CONSTRAINT uq_crawl_job_configs_name;
