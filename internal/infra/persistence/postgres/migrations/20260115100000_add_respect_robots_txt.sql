-- +goose Up
-- +goose StatementBegin
ALTER TABLE crawl_job_configs
ADD COLUMN respect_robots_txt BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN crawl_job_configs.respect_robots_txt IS 'If true, crawler follows robots.txt rules; if false, robots.txt is ignored';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crawl_job_configs
DROP COLUMN respect_robots_txt;
-- +goose StatementEnd
