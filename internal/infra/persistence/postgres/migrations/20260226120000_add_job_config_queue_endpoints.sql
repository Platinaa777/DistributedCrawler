-- +goose Up
CREATE TABLE crawl_job_config_queue_endpoints (
    crawl_job_config_id VARCHAR(255) NOT NULL
        REFERENCES crawl_job_configs(id) ON DELETE CASCADE,
    queue_endpoint_id   UUID         NOT NULL
        REFERENCES queue_endpoints(id) ON DELETE CASCADE,
    weight              INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (crawl_job_config_id, queue_endpoint_id)
);
CREATE INDEX idx_cjcqe_job_config_id    ON crawl_job_config_queue_endpoints (crawl_job_config_id);
CREATE INDEX idx_cjcqe_queue_endpoint_id ON crawl_job_config_queue_endpoints (queue_endpoint_id);

-- +goose Down
DROP TABLE IF EXISTS crawl_job_config_queue_endpoints;
