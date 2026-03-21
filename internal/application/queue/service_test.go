package queue

import (
	"context"
	"errors"
	"testing"

	"distributed-crawler/internal/domain/queue/models"
	queuerepos "distributed-crawler/internal/domain/queue/repos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type endpointRepoFake struct {
	listFn   func(ctx context.Context) ([]*models.QueueEndpoint, error)
	createFn func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error)
	updateFn func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error)
	deleteFn func(ctx context.Context, id string) error
}

func (f endpointRepoFake) Create(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
	return f.createFn(ctx, endpoint)
}
func (f endpointRepoFake) Get(context.Context, string) (*models.QueueEndpoint, error) { return nil, nil }
func (f endpointRepoFake) List(ctx context.Context) ([]*models.QueueEndpoint, error) {
	return f.listFn(ctx)
}
func (f endpointRepoFake) Update(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
	return f.updateFn(ctx, endpoint)
}
func (f endpointRepoFake) Delete(ctx context.Context, id string) error {
	return f.deleteFn(ctx, id)
}

type ruleRepoFake struct {
	listFn   func(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error)
	upsertFn func(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error)
}

func (f ruleRepoFake) Upsert(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error) {
	return f.upsertFn(ctx, rule)
}
func (f ruleRepoFake) ListByStage(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error) {
	return f.listFn(ctx, stage)
}

var _ queuerepos.QueueEndpointRepository = endpointRepoFake{}
var _ queuerepos.QueueRoutingRuleRepository = ruleRepoFake{}

func TestService_DelegatesRepositoryCalls(t *testing.T) {
	t.Parallel()

	svc := NewService(
		endpointRepoFake{
			listFn: func(ctx context.Context) ([]*models.QueueEndpoint, error) {
				return []*models.QueueEndpoint{{ID: "1"}}, nil
			},
			createFn: func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
				assert.Equal(t, "endpoint", endpoint.DisplayName)
				return &endpoint, nil
			},
			updateFn: func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
				assert.Equal(t, "1", endpoint.ID)
				return &endpoint, nil
			},
			deleteFn: func(ctx context.Context, id string) error {
				assert.Equal(t, "1", id)
				return nil
			},
		},
		ruleRepoFake{
			listFn: func(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error) {
				assert.Equal(t, models.StageParse, stage)
				return []*models.RoutingRule{{ID: "r1"}}, nil
			},
			upsertFn: func(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error) {
				assert.Equal(t, "scope", rule.Scope)
				return &rule, nil
			},
		},
	)

	endpoints, err := svc.ListEndpoints(context.Background())
	require.NoError(t, err)
	require.Len(t, endpoints, 1)

	created, err := svc.CreateEndpoint(context.Background(), models.QueueEndpoint{DisplayName: "endpoint"})
	require.NoError(t, err)
	assert.Equal(t, "endpoint", created.DisplayName)

	updated, err := svc.UpdateEndpoint(context.Background(), models.QueueEndpoint{ID: "1"})
	require.NoError(t, err)
	assert.Equal(t, "1", updated.ID)

	err = svc.DeleteEndpoint(context.Background(), "1")
	require.NoError(t, err)

	rules, err := svc.ListRoutingRules(context.Background(), models.StageParse)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	upserted, err := svc.UpsertRoutingRule(context.Background(), models.RoutingRule{Scope: "scope"})
	require.NoError(t, err)
	assert.Equal(t, "scope", upserted.Scope)
}

func TestUpdateEndpoint_RequiresID(t *testing.T) {
	t.Parallel()

	svc := NewService(
		endpointRepoFake{
			listFn: func(ctx context.Context) ([]*models.QueueEndpoint, error) { return nil, nil },
			createFn: func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) { return &endpoint, nil },
			updateFn: func(ctx context.Context, endpoint models.QueueEndpoint) (*models.QueueEndpoint, error) {
				return nil, errors.New("should not be called")
			},
			deleteFn: func(ctx context.Context, id string) error { return nil },
		},
		ruleRepoFake{
			listFn: func(ctx context.Context, stage models.Stage) ([]*models.RoutingRule, error) { return nil, nil },
			upsertFn: func(ctx context.Context, rule models.RoutingRule) (*models.RoutingRule, error) { return &rule, nil },
		},
	)

	endpoint, err := svc.UpdateEndpoint(context.Background(), models.QueueEndpoint{})
	require.Error(t, err)
	assert.Nil(t, endpoint)
	assert.Contains(t, err.Error(), "endpoint id is required")
}

