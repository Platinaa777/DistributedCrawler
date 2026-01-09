-- +goose Up
-- +goose StatementBegin

-- Optional: if you plan to query JSONB fields (e.g. scopes/seeds/spec) by keys/contains,
-- these extensions help with indexing patterns. Safe to keep commented if you don't need them.
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS crawl_job_configs (
    id              VARCHAR(255) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,

    extraction_spec JSONB,
    scopes          JSONB,
    seeds           JSONB,
    rate_limit      JSONB,
    retries         JSONB,
    auth            JSONB,
    schedule        JSONB
);

-- Commonly you'll want config names unique (optional; remove if not needed)
-- ALTER TABLE crawl_job_configs
--     ADD CONSTRAINT uq_crawl_job_configs_name UNIQUE (name);

CREATE INDEX IF NOT EXISTS idx_crawl_job_configs_name
    ON crawl_job_configs (name);

CREATE TABLE IF NOT EXISTS crawl_jobs (
    id             VARCHAR(255) PRIMARY KEY,
    job_config_id  VARCHAR(255) NULL,
    status         VARCHAR(50)  NOT NULL,
    created_at     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at   TIMESTAMP   NULL,
    error          JSONB,

    CONSTRAINT fk_crawl_jobs_job_config
        FOREIGN KEY (job_config_id)
        REFERENCES crawl_job_configs(id)
        ON DELETE SET NULL
);

-- Typical filters: status + time windows, and listing by config
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_job_config_id
    ON crawl_jobs (job_config_id);

CREATE INDEX IF NOT EXISTS idx_crawl_jobs_status_created_at
    ON crawl_jobs (status, created_at DESC);

-- Optional: helps "unfinished jobs" queries (Postgres supports partial indexes)
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_incomplete
    ON crawl_jobs (created_at DESC)
    WHERE completed_at IS NULL;

CREATE TABLE IF NOT EXISTS crawl_tasks (
    id               VARCHAR(255) PRIMARY KEY,
    job_id            VARCHAR(255) NOT NULL,

    url               TEXT NOT NULL,
    final_url         TEXT NULL,

    status            VARCHAR(50) NOT NULL,
    enqueued_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    depth             BIGINT      NOT NULL DEFAULT 1,
    body_hash         VARCHAR(255) NOT NULL,
    minio_object_key  TEXT NOT NULL,

    CONSTRAINT fk_crawl_tasks_job
        FOREIGN KEY (job_id)
        REFERENCES crawl_jobs(id)
        ON DELETE CASCADE
);

-- Typical access patterns: tasks by job, picking next tasks by status/time, dedupe by body_hash, etc.
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_job_id
    ON crawl_tasks (job_id);

CREATE INDEX IF NOT EXISTS idx_crawl_tasks_job_id_status
    ON crawl_tasks (job_id, status);

CREATE INDEX IF NOT EXISTS idx_crawl_tasks_status_enqueued_at
    ON crawl_tasks (status, enqueued_at);

-- Optional: if you often pick "oldest queued per job"
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_job_id_status_enqueued_at
    ON crawl_tasks (job_id, status, enqueued_at);

-- Optional: if you de-duplicate or lookup by body_hash frequently
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_body_hash
    ON crawl_tasks (body_hash);

-- Optional: if you query by final_url (e.g., redirects normalization)
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_final_url
    ON crawl_tasks (final_url);

-- Transactional outbox
CREATE TABLE IF NOT EXISTS outbox_events (
    id           VARCHAR(255) PRIMARY KEY,
    event_type   VARCHAR(255) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL,
    payload      BYTEA        NOT NULL,
    occurred_at  TIMESTAMP    NOT NULL,
    processed_at TIMESTAMP    NULL,
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Typical publisher queries:
-- WHERE processed_at IS NULL ORDER BY occurred_at LIMIT N
CREATE INDEX IF NOT EXISTS idx_outbox_events_unprocessed_occurred_at
    ON outbox_events (occurred_at)
    WHERE processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_events_event_type_processed_at
    ON outbox_events (event_type, processed_at);

CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate_id
    ON outbox_events (aggregate_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS crawl_tasks;
DROP TABLE IF EXISTS crawl_jobs;
DROP TABLE IF EXISTS crawl_job_configs;
-- +goose StatementEnd
