-- +goose Up
-- +goose StatementBegin

ALTER TABLE crawl_job_configs
    ADD COLUMN user_id VARCHAR(36) NULL;

ALTER TABLE crawl_job_configs
    ADD CONSTRAINT fk_crawl_job_configs_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE SET NULL;

CREATE INDEX idx_crawl_job_configs_user_id
    ON crawl_job_configs (user_id);

ALTER TABLE crawl_jobs
    ADD COLUMN user_id VARCHAR(36) NULL;

ALTER TABLE crawl_jobs
    ADD CONSTRAINT fk_crawl_jobs_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE SET NULL;

CREATE INDEX idx_crawl_jobs_user_id_created_at
    ON crawl_jobs (user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_crawl_jobs_user_id_created_at;
ALTER TABLE crawl_jobs DROP CONSTRAINT IF EXISTS fk_crawl_jobs_user_id;
ALTER TABLE crawl_jobs DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS idx_crawl_job_configs_user_id;
ALTER TABLE crawl_job_configs DROP CONSTRAINT IF EXISTS fk_crawl_job_configs_user_id;
ALTER TABLE crawl_job_configs DROP COLUMN IF EXISTS user_id;

-- +goose StatementEnd
