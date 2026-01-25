-- +goose Up
-- +goose StatementBegin
ALTER TABLE crawl_job_configs ADD COLUMN job_type VARCHAR(20) NOT NULL DEFAULT 'ONCE';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crawl_job_configs DROP COLUMN job_type;
-- +goose StatementEnd
