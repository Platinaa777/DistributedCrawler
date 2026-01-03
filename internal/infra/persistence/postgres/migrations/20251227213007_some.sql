-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS crawl_jobs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS crawl_tasks (
    id VARCHAR(255) PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    enqueued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES crawl_jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_crawl_tasks_job_id ON crawl_tasks(job_id);

CREATE TABLE IF NOT EXISTS page_snapshots (
    id VARCHAR(255) PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    http_status INTEGER NOT NULL,
    content_type VARCHAR(255),
    storage_key TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES crawl_tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_page_snapshots_task_id ON page_snapshots(task_id);

CREATE TABLE IF NOT EXISTS extracted_records (
    id VARCHAR(255) PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    source_url TEXT NOT NULL,
    data TEXT,
    parsed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES crawl_tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_extracted_records_task_id ON extracted_records(task_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS extracted_records;
DROP TABLE IF EXISTS page_snapshots;
DROP TABLE IF EXISTS crawl_tasks;
DROP TABLE IF EXISTS crawl_jobs;
-- +goose StatementEnd
