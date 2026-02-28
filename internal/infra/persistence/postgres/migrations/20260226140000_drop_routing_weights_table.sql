-- +goose Up
DROP TABLE IF EXISTS queue_routing_weights;

-- +goose Down
CREATE TABLE IF NOT EXISTS queue_routing_weights (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id          UUID         NOT NULL REFERENCES queue_routing_rules(id) ON DELETE CASCADE,
    queue_endpoint_id UUID        NOT NULL REFERENCES queue_endpoints(id) ON DELETE CASCADE,
    weight           INT          NOT NULL DEFAULT 1,
    UNIQUE (rule_id, queue_endpoint_id)
);
CREATE INDEX IF NOT EXISTS idx_qrw_rule_id ON queue_routing_weights (rule_id);
