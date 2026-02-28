package models

import "time"

// BrokerType identifies the message broker technology.
type BrokerType string

const (
	BrokerTypeRabbitMQ BrokerType = "rabbitmq"
	BrokerTypeKafka    BrokerType = "kafka"
)

// Stage identifies the pipeline stage a queue serves.
type Stage string

const (
	StageCrawl Stage = "crawl"
	StageParse Stage = "parse"
)

// QueueEndpoint describes a single queue/topic that workers can connect to.
type QueueEndpoint struct {
	ID          string
	DisplayName string
	BrokerType  BrokerType
	Stage       Stage
	Host        string
	QueueName   string
	SecretKey   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoutingRule defines routing across endpoints for a given stage.
type RoutingRule struct {
	ID    string
	Stage Stage
	Scope string
}
