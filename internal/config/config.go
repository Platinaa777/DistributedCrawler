package config

import "github.com/joho/godotenv"

const (
	// Queue name keys
	CrawlQueueKey   = "crawl_queue"
	ParsingQueueKey = "parsing_queue"
)

func Load(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}

	return nil
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
}

type KafkaConfig interface {
	Brokers() []string
	ConsumerGroup() string
	GetTopicName(key string) string
}

type MinIOConfig interface {
	Endpoint() string
	AccessKeyID() string
	SecretAccessKey() string
	UseSSL() bool
	BucketName() string
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
