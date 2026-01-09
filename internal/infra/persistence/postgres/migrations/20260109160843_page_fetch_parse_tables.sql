-- +goose Up
-- +goose StatementBegin

-- Table: page_fetch
-- Stores fetch metadata and MinIO object reference
CREATE TABLE IF NOT EXISTS page_fetch (
    task_id UUID PRIMARY KEY,
    job_id UUID NOT NULL,
    url TEXT NOT NULL,
    final_url TEXT,
    status_code INT NOT NULL,
    duration_ms INT NOT NULL,
    headers JSONB,
    content_type VARCHAR(255),
    content_length BIGINT,
    body_hash VARCHAR(64) NOT NULL,  -- SHA-256 hash
    minio_object_key VARCHAR(512) NOT NULL,
    fetched_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_page_fetch_job_id ON page_fetch(job_id);
CREATE INDEX idx_page_fetch_fetched_at ON page_fetch(fetched_at);
CREATE INDEX idx_page_fetch_status_code ON page_fetch(status_code);

-- Table: page_extract
-- Stores parsed HTML results and computed features
CREATE TABLE IF NOT EXISTS page_extract (
    task_id UUID PRIMARY KEY,
    title TEXT,
    meta_description TEXT,
    canonical_url TEXT,
    metadata JSONB,  -- Flexible storage for additional extracted data
    link_count INT NOT NULL DEFAULT 0,
    image_count INT NOT NULL DEFAULT 0,
    external_link_count INT NOT NULL DEFAULT 0,
    word_count INT NOT NULL DEFAULT 0,
    parsed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_extract_fetch FOREIGN KEY (task_id) REFERENCES page_fetch(task_id) ON DELETE CASCADE
);

CREATE INDEX idx_page_extract_parsed_at ON page_extract(parsed_at);

-- Table: page_link
-- Stores extracted links (1:N relationship with page_extract)
CREATE TABLE IF NOT EXISTS page_link (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    url TEXT NOT NULL,
    anchor_text TEXT,
    is_external BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_link_extract FOREIGN KEY (task_id) REFERENCES page_extract(task_id) ON DELETE CASCADE,
    CONSTRAINT uq_page_link_task_url UNIQUE (task_id, url)
);

CREATE INDEX idx_page_link_task_id ON page_link(task_id);
CREATE INDEX idx_page_link_is_external ON page_link(is_external);

-- Table: page_image
-- Stores extracted images (1:N relationship with page_extract)
CREATE TABLE IF NOT EXISTS page_image (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    url TEXT NOT NULL,
    alt_text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_page_image_extract FOREIGN KEY (task_id) REFERENCES page_extract(task_id) ON DELETE CASCADE,
    CONSTRAINT uq_page_image_task_url UNIQUE (task_id, url)
);

CREATE INDEX idx_page_image_task_id ON page_image(task_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS page_image;
DROP TABLE IF EXISTS page_link;
DROP TABLE IF EXISTS page_extract;
DROP TABLE IF EXISTS page_fetch;
-- +goose StatementEnd
