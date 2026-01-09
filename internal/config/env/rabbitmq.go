package env

import (
	"distributed-crawler/internal/config"
	"os"
)

const (
	rabbitmqURLEnvName       = "RABBITMQ_URL"
	rabbitmqQueueNameEnvName = "RABBITMQ_QUEUE_NAME"
)

type rabbitmqConfig struct {
	url       string
	queueName string
}

func NewRabbitMQConfig() (config.RabbitMQConfig, error) {
	url := os.Getenv(rabbitmqURLEnvName)
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	queueName := os.Getenv(rabbitmqQueueNameEnvName)
	if queueName == "" {
		queueName = "crawl_tasks"
	}

	return &rabbitmqConfig{
		url:       url,
		queueName: queueName,
	}, nil
}

func (c *rabbitmqConfig) URL() string {
	return c.url
}

func (c *rabbitmqConfig) QueueName() string {
	return c.queueName
}
