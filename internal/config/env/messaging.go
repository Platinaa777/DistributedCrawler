package env

import (
	"os"
	"strings"
)

const (
	messagingBrokerEnvName = "MESSAGING_BROKER"

	BrokerRabbitMQ = "rabbitmq"
	BrokerKafka    = "kafka"
)

// GetBrokerType returns the configured broker type ("rabbitmq" or "kafka").
// Controlled by the MESSAGING_BROKER environment variable.
// Defaults to "rabbitmq" when not set.
func GetBrokerType() string {
	t := strings.ToLower(os.Getenv(messagingBrokerEnvName))
	if t == BrokerKafka {
		return BrokerKafka
	}
	return BrokerRabbitMQ
}
