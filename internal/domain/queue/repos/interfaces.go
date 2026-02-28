package repos

import (
	"context"

	"distributed-crawler/internal/domain/queue/models"
)

// QueueEndpointRepository manages queue endpoint persistence.
type QueueEndpointRepository interface {
	Create(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error)
	Get(ctx context.Context, id string) (*models.QueueEndpoint, error)
	List(ctx context.Context) ([]*models.QueueEndpoint, error)
	Update(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error)
	Delete(ctx context.Context, id string) error
}

// QueueRoutingRuleRepository manages routing rule persistence.
type QueueRoutingRuleRepository interface {
	Upsert(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error)
	ListByStage(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error)
}
