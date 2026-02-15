package env

import (
	"distributed-crawler/internal/config"
)

const (
	opensearchEnabledEnv       = "OPENSEARCH_ENABLED"
	opensearchEndpointEnv      = "OPENSEARCH_ENDPOINT"
	opensearchIndexEnv         = "OPENSEARCH_INDEX"
	opensearchFlushIntervalEnv = "OPENSEARCH_FLUSH_INTERVAL_SECONDS"
	opensearchBatchSizeEnv     = "OPENSEARCH_BATCH_SIZE"
)

type opensearchConfig struct {
	enabled       bool
	endpoint      string
	index         string
	flushInterval int
	batchSize     int
}

func NewOpenSearchConfig() (config.OpenSearchConfig, error) {
	return &opensearchConfig{
		enabled:       getEnvBool(opensearchEnabledEnv, false),
		endpoint:      getEnvString(opensearchEndpointEnv, "http://localhost:9200"),
		index:         getEnvString(opensearchIndexEnv, "app-logs"),
		flushInterval: getEnvInt(opensearchFlushIntervalEnv, 5),
		batchSize:     getEnvInt(opensearchBatchSizeEnv, 100),
	}, nil
}

func (c *opensearchConfig) Enabled() bool {
	return c.enabled
}

func (c *opensearchConfig) Endpoint() string {
	return c.endpoint
}

func (c *opensearchConfig) Index() string {
	return c.index
}

func (c *opensearchConfig) FlushInterval() int {
	return c.flushInterval
}

func (c *opensearchConfig) BatchSize() int {
	return c.batchSize
}
