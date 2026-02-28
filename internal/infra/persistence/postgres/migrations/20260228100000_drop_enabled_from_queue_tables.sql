-- +goose Up
ALTER TABLE queue_endpoints DROP COLUMN IF EXISTS enabled;
ALTER TABLE queue_routing_rules DROP COLUMN IF EXISTS enabled;

-- +goose Down
ALTER TABLE queue_endpoints ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE queue_routing_rules ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
