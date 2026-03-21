package queue_admin

import (
	"context"
	"testing"
	"time"

	appqueue "distributed-crawler/internal/application/queue"
	queuemodels "distributed-crawler/internal/domain/queue/models"
	queuerepos "distributed-crawler/internal/domain/queue/repos"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeQueueEndpointRepo struct {
	listFn   func(ctx context.Context) ([]*queuemodels.QueueEndpoint, error)
	createFn func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error)
	updateFn func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error)
	deleteFn func(ctx context.Context, id string) error
}

func (f fakeQueueEndpointRepo) Create(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
	return f.createFn(ctx, endpoint)
}
func (f fakeQueueEndpointRepo) Get(context.Context, string) (*queuemodels.QueueEndpoint, error) { return nil, nil }
func (f fakeQueueEndpointRepo) List(ctx context.Context) ([]*queuemodels.QueueEndpoint, error) {
	return f.listFn(ctx)
}
func (f fakeQueueEndpointRepo) Update(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
	return f.updateFn(ctx, endpoint)
}
func (f fakeQueueEndpointRepo) Delete(ctx context.Context, id string) error {
	return f.deleteFn(ctx, id)
}

type fakeQueueRuleRepo struct {
	listFn   func(ctx context.Context, stage queuemodels.Stage) ([]*queuemodels.RoutingRule, error)
	upsertFn func(ctx context.Context, rule queuemodels.RoutingRule) (*queuemodels.RoutingRule, error)
}

func (f fakeQueueRuleRepo) Upsert(ctx context.Context, rule queuemodels.RoutingRule) (*queuemodels.RoutingRule, error) {
	return f.upsertFn(ctx, rule)
}
func (f fakeQueueRuleRepo) ListByStage(ctx context.Context, stage queuemodels.Stage) ([]*queuemodels.RoutingRule, error) {
	return f.listFn(ctx, stage)
}

var _ queuerepos.QueueEndpointRepository = fakeQueueEndpointRepo{}
var _ queuerepos.QueueRoutingRuleRepository = fakeQueueRuleRepo{}

func TestListAndCreateQueueEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	service := appqueue.NewService(
		fakeQueueEndpointRepo{
			listFn: func(ctx context.Context) ([]*queuemodels.QueueEndpoint, error) {
				return []*queuemodels.QueueEndpoint{{
					ID:          "endpoint-1",
					DisplayName: "Primary",
					BrokerType:  queuemodels.BrokerTypeRabbitMQ,
					Stage:       queuemodels.StageCrawl,
					Host:        "rabbitmq:5672",
					QueueName:   "crawl",
					SecretKey:   "secret",
					CreatedAt:   now,
					UpdatedAt:   now,
				}}, nil
			},
			createFn: func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
				assert.Equal(t, "New", endpoint.DisplayName)
				assert.Equal(t, queuemodels.BrokerTypeKafka, endpoint.BrokerType)
				endpoint.ID = "endpoint-2"
				endpoint.CreatedAt = now
				endpoint.UpdatedAt = now
				return &endpoint, nil
			},
			updateFn: func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
				return &endpoint, nil
			},
			deleteFn: func(ctx context.Context, id string) error { return nil },
		},
		fakeQueueRuleRepo{
			listFn: func(ctx context.Context, stage queuemodels.Stage) ([]*queuemodels.RoutingRule, error) { return nil, nil },
			upsertFn: func(ctx context.Context, rule queuemodels.RoutingRule) (*queuemodels.RoutingRule, error) { return &rule, nil },
		},
	)
	impl := NewImplementation(service)

	listResp, err := impl.ListQueueEndpoints(context.Background(), &crawlergrpc.ListQueueEndpointsRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Endpoints, 1)
	assert.Equal(t, crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_RABBITMQ, listResp.Endpoints[0].BrokerType)

	createResp, err := impl.CreateQueueEndpoint(context.Background(), &crawlergrpc.CreateQueueEndpointRequest{
		Endpoint: &crawlergrpc.QueueEndpoint{
			DisplayName: "New",
			BrokerType:  crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_KAFKA,
			Stage:       crawlergrpc.QueueStage_QUEUE_STAGE_PARSE,
			Host:        "kafka:9092",
			QueueName:   "parse",
			SecretKey:   "secret2",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "endpoint-2", createResp.Endpoint.Id)
	assert.Equal(t, crawlergrpc.QueueStage_QUEUE_STAGE_PARSE, createResp.Endpoint.Stage)
}

func TestRoutingRuleHandlers_MapStageAndScope(t *testing.T) {
	t.Parallel()

	service := appqueue.NewService(
		fakeQueueEndpointRepo{
			listFn: func(ctx context.Context) ([]*queuemodels.QueueEndpoint, error) { return nil, nil },
			createFn: func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) { return &endpoint, nil },
			updateFn: func(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) { return &endpoint, nil },
			deleteFn: func(ctx context.Context, id string) error { return nil },
		},
		fakeQueueRuleRepo{
			listFn: func(ctx context.Context, stage queuemodels.Stage) ([]*queuemodels.RoutingRule, error) {
				assert.Equal(t, queuemodels.StageParse, stage)
				return []*queuemodels.RoutingRule{{ID: "rule-1", Stage: stage, Scope: "eu"}}, nil
			},
			upsertFn: func(ctx context.Context, rule queuemodels.RoutingRule) (*queuemodels.RoutingRule, error) {
				assert.Equal(t, queuemodels.StageCrawl, rule.Stage)
				assert.Equal(t, "global", rule.Scope)
				rule.ID = "rule-2"
				return &rule, nil
			},
		},
	)
	impl := NewImplementation(service)

	listResp, err := impl.ListQueueRoutingRules(context.Background(), &crawlergrpc.ListQueueRoutingRulesRequest{
		Stage: crawlergrpc.QueueStage_QUEUE_STAGE_PARSE,
	})
	require.NoError(t, err)
	require.Len(t, listResp.Rules, 1)
	assert.Equal(t, crawlergrpc.QueueStage_QUEUE_STAGE_PARSE, listResp.Rules[0].Stage)

	upsertResp, err := impl.UpsertQueueRoutingRules(context.Background(), &crawlergrpc.UpsertQueueRoutingRulesRequest{
		Rule: &crawlergrpc.QueueRoutingRule{
			Stage: crawlergrpc.QueueStage_QUEUE_STAGE_CRAWL,
			Scope: "global",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "rule-2", upsertResp.Rule.Id)
	assert.Equal(t, crawlergrpc.QueueStage_QUEUE_STAGE_CRAWL, upsertResp.Rule.Stage)
}

