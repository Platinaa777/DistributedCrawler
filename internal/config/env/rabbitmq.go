package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
)

const (
	rabbitmqURLEnvName = "RABBITMQ_URL"

	// Environment variable names for specific queues
	rabbitmqCrawlQueueEnvName   = "RABBITMQ_CRAWL_QUEUE_NAME"
	rabbitmqParsingQueueEnvName = "RABBITMQ_PARSING_QUEUE_NAME"
)

type rabbitmqConfig struct {
	url        string
	queueNames map[string]string
}

func NewRabbitMQConfig() (config.RabbitMQConfig, error) {
	url := os.Getenv(rabbitmqURLEnvName)
	if url == "" {
		return nil, fmt.Errorf("%s environment variable is required", rabbitmqURLEnvName)
	}

	// Load queue names
	queueNames := make(map[string]string)

	crawlQueue := os.Getenv(rabbitmqCrawlQueueEnvName)
	if crawlQueue == "" {
		return nil, fmt.Errorf("%s environment variable is required", rabbitmqCrawlQueueEnvName)
	}
	queueNames[config.CrawlQueueKey] = crawlQueue

	parsingQueue := os.Getenv(rabbitmqParsingQueueEnvName)
	if parsingQueue == "" {
		return nil, fmt.Errorf("%s environment variable is required", rabbitmqParsingQueueEnvName)
	}
	queueNames[config.ParsingQueueKey] = parsingQueue

	return &rabbitmqConfig{
		url:        url,
		queueNames: queueNames,
	}, nil
}

func (c *rabbitmqConfig) URL() string {
	return c.url
}

func (c *rabbitmqConfig) GetQueueName(key string) string {
	if queueName, exists := c.queueNames[key]; exists {
		return queueName
	}
	// Return the key itself as fallback (for backward compatibility)
	return key
}
