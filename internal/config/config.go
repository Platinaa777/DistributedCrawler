package config

import (
	"os"

	"github.com/joho/godotenv"
)

const (
	// Queue name keys
	CrawlQueueKey   = "crawl_queue"
	ParsingQueueKey = "parsing_queue"
)

// Load loads configuration from a dotenv file.
// If the CONFIG_SOURCE environment variable is set to "env", the file is
// skipped and the process environment is used as-is (e.g. K8s ConfigMap/Secret).
func Load(path string) error {
	if os.Getenv("CONFIG_SOURCE") == "env" {
		return nil
	}

	return godotenv.Load(path)
}

type GRPCConfig interface {
	Address() string
}

type PGConfig interface {
	DSN() string
	ShardingEnabled() bool
	ShardDSNs() []string
}

type HTTPConfig interface {
	Address() string
}

type LoggerConfig interface {
	Level() string
	Env() string
}

type RabbitMQConfig interface {
	URL() string
	GetQueueName(key string) string
	// GetAllCrawlQueueNames returns all configured crawl queue names (multi-region support).
	// Falls back to a single-item slice containing the primary crawl queue.
	GetAllCrawlQueueNames() []string
}

type KafkaConfig interface {
	Brokers() []string
	ConsumerGroup() string
	GetTopicName(key string) string
	// GetAllCrawlTopicNames returns all configured crawl topic names (multi-region support).
	GetAllCrawlTopicNames() []string
}

type MinIOConfig interface {
	Endpoint() string
	AccessKeyID() string
	SecretAccessKey() string
	UseSSL() bool
	BucketName() string
	PublicBaseURL() string
}

type RedisConfig interface {
	Address() string
	Password() string
	DB() int
}

type AuthConfig interface {
	JWTSecret() string
	AccessTokenTTL() string
	RefreshTokenTTL() string
	Issuer() string
	Audience() string
	DefaultUserEmail() string
	DefaultUserPassword() string
}

type OpenSearchConfig interface {
	Enabled() bool
	Endpoint() string
	Index() string
	FlushInterval() int
	BatchSize() int
}

type OTelConfig interface {
	Enabled() bool
	ServiceName() string
	ServiceVersion() string
	Environment() string
	OTLPEndpoint() string
	OTLPInsecure() bool
	TraceSampleRate() float64
	MetricsIntervalSeconds() int
}

// MemoryBrokerConfig holds connection settings for the remote gRPC memory broker.
type MemoryBrokerConfig interface {
	// Address returns the host:port of the memory_broker gRPC server.
	Address() string
	// QueueCapacity is the per-queue channel buffer size reported by the server.
	QueueCapacity() int
}

// SecretsFileConfig holds settings for the file-based secrets store.
type SecretsFileConfig interface {
	FilePath() string
	WatchEnabled() bool
}

// CORSConfig holds allowed origins for the HTTP server CORS policy.
type CORSConfig interface {
	AllowedOrigins() []string
}
