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
