-- +goose Up
ALTER TABLE crawl_job_configs
    ADD COLUMN queue_weights JSONB NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE crawl_job_configs
    DROP COLUMN queue_weights;
