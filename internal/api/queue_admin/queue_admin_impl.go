package queue_admin

import (
	"context"

	appqueue "distributed-crawler/internal/application/queue"
	queuemodels "distributed-crawler/internal/domain/queue/models"
	crawlergrpc "distributed-crawler/pkg/v1"
)

// QueueAdminImplementation implements the QueueAdminService gRPC server.
type QueueAdminImplementation struct {
	crawlergrpc.UnimplementedQueueAdminServiceServer
	service *appqueue.Service
}

// NewImplementation creates a new QueueAdminImplementation.
func NewImplementation(service *appqueue.Service) *QueueAdminImplementation {
	return &QueueAdminImplementation{service: service}
}

func (i *QueueAdminImplementation) ListQueueEndpoints(ctx context.Context, _ *crawlergrpc.ListQueueEndpointsRequest) (*crawlergrpc.ListQueueEndpointsResponse, error) {
	endpoints, err := i.service.ListEndpoints(ctx)
	if err != nil {
		return nil, err
	}

	protoEndpoints := make([]*crawlergrpc.QueueEndpoint, 0, len(endpoints))
	for _, e := range endpoints {
		protoEndpoints = append(protoEndpoints, domainEndpointToProto(e))
	}

	return &crawlergrpc.ListQueueEndpointsResponse{Endpoints: protoEndpoints}, nil
}

func (i *QueueAdminImplementation) CreateQueueEndpoint(ctx context.Context, req *crawlergrpc.CreateQueueEndpointRequest) (*crawlergrpc.CreateQueueEndpointResponse, error) {
	domain := protoEndpointToDomain(req.Endpoint)
	created, err := i.service.CreateEndpoint(ctx, domain)
	if err != nil {
		return nil, err
	}
	return &crawlergrpc.CreateQueueEndpointResponse{Endpoint: domainEndpointToProto(created)}, nil
}

func (i *QueueAdminImplementation) UpdateQueueEndpoint(ctx context.Context, req *crawlergrpc.UpdateQueueEndpointRequest) (*crawlergrpc.UpdateQueueEndpointResponse, error) {
	domain := protoEndpointToDomain(req.Endpoint)
	updated, err := i.service.UpdateEndpoint(ctx, domain)
	if err != nil {
		return nil, err
	}
	return &crawlergrpc.UpdateQueueEndpointResponse{Endpoint: domainEndpointToProto(updated)}, nil
}

func (i *QueueAdminImplementation) DeleteQueueEndpoint(ctx context.Context, req *crawlergrpc.DeleteQueueEndpointRequest) (*crawlergrpc.DeleteQueueEndpointResponse, error) {
	if err := i.service.DeleteEndpoint(ctx, req.Id); err != nil {
		return nil, err
	}
	return &crawlergrpc.DeleteQueueEndpointResponse{}, nil
}

func (i *QueueAdminImplementation) ListQueueRoutingRules(ctx context.Context, req *crawlergrpc.ListQueueRoutingRulesRequest) (*crawlergrpc.ListQueueRoutingRulesResponse, error) {
	rules, err := i.service.ListRoutingRules(ctx, protoToStage(req.Stage))
	if err != nil {
		return nil, err
	}

	protoRules := make([]*crawlergrpc.QueueRoutingRule, 0, len(rules))
	for _, r := range rules {
		protoRules = append(protoRules, domainRuleToProto(r))
	}

	return &crawlergrpc.ListQueueRoutingRulesResponse{Rules: protoRules}, nil
}

func (i *QueueAdminImplementation) UpsertQueueRoutingRules(ctx context.Context, req *crawlergrpc.UpsertQueueRoutingRulesRequest) (*crawlergrpc.UpsertQueueRoutingRulesResponse, error) {
	domain := protoRuleToDomain(req.Rule)
	upserted, err := i.service.UpsertRoutingRule(ctx, domain)
	if err != nil {
		return nil, err
	}
	return &crawlergrpc.UpsertQueueRoutingRulesResponse{Rule: domainRuleToProto(upserted)}, nil
}

// -- converters --

func domainEndpointToProto(e *queuemodels.QueueEndpoint) *crawlergrpc.QueueEndpoint {
	if e == nil {
		return nil
	}
	return &crawlergrpc.QueueEndpoint{
		Id:          e.ID,
		DisplayName: e.DisplayName,
		BrokerType:  brokerTypeToProto(e.BrokerType),
		Stage:       stageToProto(e.Stage),
		Host:        e.Host,
		QueueName:   e.QueueName,
		SecretKey:   e.SecretKey,
		CreatedAt:   e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   e.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func protoEndpointToDomain(p *crawlergrpc.QueueEndpoint) queuemodels.QueueEndpoint {
	if p == nil {
		return queuemodels.QueueEndpoint{}
	}
	return queuemodels.QueueEndpoint{
		ID:          p.Id,
		DisplayName: p.DisplayName,
		BrokerType:  protoToBrokerType(p.BrokerType),
		Stage:       protoToStage(p.Stage),
		Host:        p.Host,
		QueueName:   p.QueueName,
		SecretKey:   p.SecretKey,
	}
}

func domainRuleToProto(r *queuemodels.RoutingRule) *crawlergrpc.QueueRoutingRule {
	if r == nil {
		return nil
	}
	return &crawlergrpc.QueueRoutingRule{
		Id:    r.ID,
		Stage: stageToProto(r.Stage),
		Scope: r.Scope,
	}
}

func protoRuleToDomain(p *crawlergrpc.QueueRoutingRule) queuemodels.RoutingRule {
	if p == nil {
		return queuemodels.RoutingRule{}
	}
	return queuemodels.RoutingRule{
		ID:    p.Id,
		Stage: protoToStage(p.Stage),
		Scope: p.Scope,
	}
}

func protoToBrokerType(bt crawlergrpc.QueueBrokerType) queuemodels.BrokerType {
	switch bt {
	case crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_RABBITMQ:
		return queuemodels.BrokerTypeRabbitMQ
	case crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_KAFKA:
		return queuemodels.BrokerTypeKafka
	default:
		return ""
	}
}

func brokerTypeToProto(bt queuemodels.BrokerType) crawlergrpc.QueueBrokerType {
	switch bt {
	case queuemodels.BrokerTypeRabbitMQ:
		return crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_RABBITMQ
	case queuemodels.BrokerTypeKafka:
		return crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_KAFKA
	default:
		return crawlergrpc.QueueBrokerType_QUEUE_BROKER_TYPE_UNSPECIFIED
	}
}

func protoToStage(s crawlergrpc.QueueStage) queuemodels.Stage {
	switch s {
	case crawlergrpc.QueueStage_QUEUE_STAGE_CRAWL:
		return queuemodels.StageCrawl
	case crawlergrpc.QueueStage_QUEUE_STAGE_PARSE:
		return queuemodels.StageParse
	default:
		return ""
	}
}

func stageToProto(s queuemodels.Stage) crawlergrpc.QueueStage {
	switch s {
	case queuemodels.StageCrawl:
		return crawlergrpc.QueueStage_QUEUE_STAGE_CRAWL
	case queuemodels.StageParse:
		return crawlergrpc.QueueStage_QUEUE_STAGE_PARSE
	default:
		return crawlergrpc.QueueStage_QUEUE_STAGE_UNSPECIFIED
	}
}
