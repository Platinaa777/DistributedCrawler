-- +goose Up
CREATE TABLE IF NOT EXISTS queue_endpoints (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  display_name VARCHAR(255) NOT NULL,
  broker_type VARCHAR(20) NOT NULL,
  stage VARCHAR(20) NOT NULL,
  region VARCHAR(100) NOT NULL DEFAULT '',
  host TEXT NOT NULL DEFAULT '',
  queue_name VARCHAR(255) NOT NULL,
  secret_key VARCHAR(255) NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS queue_routing_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stage VARCHAR(20) NOT NULL,
  scope VARCHAR(50) NOT NULL DEFAULT 'global',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_rule_stage_scope UNIQUE (stage, scope)
);

CREATE TABLE IF NOT EXISTS queue_routing_weights (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id UUID NOT NULL REFERENCES queue_routing_rules(id) ON DELETE CASCADE,
  queue_endpoint_id UUID NOT NULL REFERENCES queue_endpoints(id) ON DELETE CASCADE,
  weight INT NOT NULL DEFAULT 1,
  CONSTRAINT uq_rule_queue UNIQUE (rule_id, queue_endpoint_id)
);

-- +goose Down
DROP TABLE IF EXISTS queue_routing_weights;
DROP TABLE IF EXISTS queue_routing_rules;
DROP TABLE IF EXISTS queue_endpoints;
