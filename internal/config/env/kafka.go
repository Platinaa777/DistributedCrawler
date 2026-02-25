package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
	"strings"
)

const (
	kafkaBrokersEnvName       = "KAFKA_BROKERS"
	kafkaConsumerGroupEnvName = "KAFKA_CONSUMER_GROUP"
	kafkaCrawlTopicEnvName    = "KAFKA_CRAWL_TOPIC_NAME"
	kafkaParsingTopicEnvName  = "KAFKA_PARSING_TOPIC_NAME"
)

type kafkaConfig struct {
	brokers       []string
	consumerGroup string
	topicNames    map[string]string
}

func NewKafkaConfig() (config.KafkaConfig, error) {
	brokersRaw := os.Getenv(kafkaBrokersEnvName)
	if brokersRaw == "" {
		return nil, fmt.Errorf("%s environment variable is required", kafkaBrokersEnvName)
	}
	brokers := strings.Split(brokersRaw, ",")
	for i, b := range brokers {
		brokers[i] = strings.TrimSpace(b)
	}

	consumerGroup := os.Getenv(kafkaConsumerGroupEnvName)
	if consumerGroup == "" {
		return nil, fmt.Errorf("%s environment variable is required", kafkaConsumerGroupEnvName)
	}

	topicNames := make(map[string]string)

	crawlTopic := os.Getenv(kafkaCrawlTopicEnvName)
	if crawlTopic == "" {
		return nil, fmt.Errorf("%s environment variable is required", kafkaCrawlTopicEnvName)
	}
	topicNames[config.CrawlQueueKey] = crawlTopic

	parsingTopic := os.Getenv(kafkaParsingTopicEnvName)
	if parsingTopic == "" {
		return nil, fmt.Errorf("%s environment variable is required", kafkaParsingTopicEnvName)
	}
	topicNames[config.ParsingQueueKey] = parsingTopic

	return &kafkaConfig{
		brokers:       brokers,
		consumerGroup: consumerGroup,
		topicNames:    topicNames,
	}, nil
}

func (c *kafkaConfig) Brokers() []string {
	return c.brokers
}

func (c *kafkaConfig) ConsumerGroup() string {
	return c.consumerGroup
}

func (c *kafkaConfig) GetTopicName(key string) string {
	if name, ok := c.topicNames[key]; ok {
		return name
	}
	return key
}
