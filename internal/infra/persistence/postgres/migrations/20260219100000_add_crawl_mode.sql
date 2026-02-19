-- +goose Up
-- +goose StatementBegin
ALTER TABLE crawl_job_configs ADD COLUMN crawl_mode VARCHAR(30) NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crawl_job_configs DROP COLUMN crawl_mode;
-- +goose StatementEnd
