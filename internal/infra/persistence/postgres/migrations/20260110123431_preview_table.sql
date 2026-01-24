-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_url TEXT NOT NULL,
    final_url TEXT,
    minio_key TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'text/html; charset=utf-8',
    download_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX idx_previews_created_at ON previews(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS previews;
-- +goose StatementEnd
