-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS crawl_task_outbox (
    id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_crawl_task_outbox_processed ON crawl_task_outbox(processed_at) WHERE processed_at IS NULL;
CREATE INDEX idx_crawl_task_outbox_occurred ON crawl_task_outbox(occurred_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS crawl_task_outbox;
-- +goose StatementEnd
