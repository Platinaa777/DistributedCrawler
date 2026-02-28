-- +goose Up
ALTER TABLE queue_endpoints DROP COLUMN IF EXISTS region;

-- +goose Down
ALTER TABLE queue_endpoints ADD COLUMN region VARCHAR(100) NOT NULL DEFAULT '';
