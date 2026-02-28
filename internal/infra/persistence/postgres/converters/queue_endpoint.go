package converters

import (
	"distributed-crawler/internal/domain/queue/models"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

// SaveQueueEndpointToSnapshot converts a domain QueueEndpoint to a DB snapshot.
func SaveQueueEndpointToSnapshot(e models.QueueEndpoint) *snapshots.QueueEndpointSnapshot {
	return &snapshots.QueueEndpointSnapshot{
		ID:          e.ID,
		DisplayName: e.DisplayName,
		BrokerType:  string(e.BrokerType),
		Stage:       string(e.Stage),
		Host:        e.Host,
		QueueName:   e.QueueName,
		SecretKey:   e.SecretKey,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// RestoreQueueEndpointFromSnapshot converts a DB snapshot to a domain QueueEndpoint.
func RestoreQueueEndpointFromSnapshot(s snapshots.QueueEndpointSnapshot) *models.QueueEndpoint {
	return &models.QueueEndpoint{
		ID:          s.ID,
		DisplayName: s.DisplayName,
		BrokerType:  models.BrokerType(s.BrokerType),
		Stage:       models.Stage(s.Stage),
		Host:        s.Host,
		QueueName:   s.QueueName,
		SecretKey:   s.SecretKey,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// RestoreRoutingRuleFromSnapshot converts a rule snapshot to a domain RoutingRule.
func RestoreRoutingRuleFromSnapshot(rule snapshots.QueueRoutingRuleSnapshot) *models.RoutingRule {
	return &models.RoutingRule{
		ID:    rule.ID,
		Stage: models.Stage(rule.Stage),
		Scope: rule.Scope,
	}
}
