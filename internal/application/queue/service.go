package queue

import (
	"context"
	"fmt"

	"distributed-crawler/internal/domain/queue/models"
	"distributed-crawler/internal/domain/queue/repos"
)

// Service provides application-level queue management operations.
type Service struct {
	endpointRepo repos.QueueEndpointRepository
	ruleRepo     repos.QueueRoutingRuleRepository
}

// NewService creates a new queue application service.
func NewService(
	endpointRepo repos.QueueEndpointRepository,
	ruleRepo repos.QueueRoutingRuleRepository,
) *Service {
	return &Service{
		endpointRepo: endpointRepo,
		ruleRepo:     ruleRepo,
	}
}

func (s *Service) ListEndpoints(ctx context.Context) ([]*models.QueueEndpoint, error) {
	return s.endpointRepo.List(ctx)
}

func (s *Service) CreateEndpoint(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
	return s.endpointRepo.Create(ctx, endpoint)
}

func (s *Service) UpdateEndpoint(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
	if endpoint.ID == "" {
		return nil, fmt.Errorf("endpoint id is required")
	}
	return s.endpointRepo.Update(ctx, endpoint)
}

func (s *Service) DeleteEndpoint(ctx context.Context, id string) error {
	return s.endpointRepo.Delete(ctx, id)
}

func (s *Service) ListRoutingRules(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error) {
	return s.ruleRepo.ListByStage(ctx, stage)
}

func (s *Service) UpsertRoutingRule(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error) {
	return s.ruleRepo.Upsert(ctx, rule)
}
