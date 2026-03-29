package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
	"strings"
)

const (
	rabbitmqURLEnvName = "RABBITMQ_URL"

	// Environment variable names for specific queues
	rabbitmqCrawlQueueEnvName    = "RABBITMQ_CRAWL_QUEUE_NAME"
	rabbitmqCrawlQueuesEnvName   = "RABBITMQ_CRAWL_QUEUE_NAMES" // comma-separated, multi-region
	rabbitmqParsingQueueEnvName  = "RABBITMQ_PARSING_QUEUE_NAME"
)

type rabbitmqConfig struct {
	url              string
	queueNames       map[string]string
	allCrawlQueues   []string
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

	// Parse multi-region crawl queue names. Falls back to single queue if not set.
	var allCrawlQueues []string
	if namesRaw := os.Getenv(rabbitmqCrawlQueuesEnvName); namesRaw != "" {
		allCrawlQueues = splitTrimComma(namesRaw)
	}
	if len(allCrawlQueues) == 0 {
		allCrawlQueues = []string{crawlQueue}
	}

	return &rabbitmqConfig{
		url:            url,
		queueNames:     queueNames,
		allCrawlQueues: allCrawlQueues,
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

func (c *rabbitmqConfig) GetAllCrawlQueueNames() []string {
	return c.allCrawlQueues
}

// splitTrimComma splits s on commas and trims whitespace from each element.
func splitTrimComma(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
