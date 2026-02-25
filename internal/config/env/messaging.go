package env

import (
	"os"
	"strings"
)

const (
	messagingBrokerEnvName = "MESSAGING_BROKER"

	BrokerRabbitMQ   = "rabbitmq"
	BrokerKafka      = "kafka"
	BrokerGRPCMemory = "grpc_memory" // remote gRPC memory broker (cmd/memory_broker)
)

// GetBrokerType returns the configured broker type.
// Controlled by the MESSAGING_BROKER environment variable.
// Defaults to "rabbitmq" when not set or unrecognised.
func GetBrokerType() string {
	switch strings.ToLower(os.Getenv(messagingBrokerEnvName)) {
	case BrokerKafka:
		return BrokerKafka
	case BrokerGRPCMemory:
		return BrokerGRPCMemory
	default:
		return BrokerRabbitMQ
	}
}
