package snapshots

import "time"

// QueueEndpointSnapshot is the DB representation of a queue endpoint.
type QueueEndpointSnapshot struct {
	ID          string    `db:"id"`
	DisplayName string    `db:"display_name"`
	BrokerType  string    `db:"broker_type"`
	Stage       string    `db:"stage"`
	Host        string    `db:"host"`
	QueueName   string    `db:"queue_name"`
	SecretKey   string    `db:"secret_key"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// QueueRoutingRuleSnapshot is the DB representation of a routing rule row.
type QueueRoutingRuleSnapshot struct {
	ID    string `db:"id"`
	Stage string `db:"stage"`
	Scope string `db:"scope"`
}

