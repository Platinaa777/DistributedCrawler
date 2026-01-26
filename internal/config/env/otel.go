package env

import (
	"distributed-crawler/internal/config"
	"os"
	"strconv"
)

const (
	otelEnabledEnv         = "OTEL_ENABLED"
	otelServiceNameEnv     = "OTEL_SERVICE_NAME"
	otelServiceVersionEnv  = "OTEL_SERVICE_VERSION"
	otelEnvironmentEnv     = "OTEL_ENVIRONMENT"
	otelExporterEndpointEnv = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otelExporterInsecureEnv = "OTEL_EXPORTER_OTLP_INSECURE"
	otelTraceSampleRateEnv  = "OTEL_TRACE_SAMPLE_RATE"
	otelMetricsIntervalEnv  = "OTEL_METRICS_INTERVAL_SECONDS"
)

type otelConfig struct {
	enabled               bool
	serviceName           string
	serviceVersion        string
	environment           string
	otlpEndpoint          string
	otlpInsecure          bool
	traceSampleRate       float64
	metricsIntervalSeconds int
}

func NewOTelConfig() (config.OTelConfig, error) {
	enabled := getEnvBool(otelEnabledEnv, false)
	serviceName := getEnvString(otelServiceNameEnv, "distributed-crawler")
	serviceVersion := getEnvString(otelServiceVersionEnv, "1.0.0")
	environment := getEnvString(otelEnvironmentEnv, "development")
	otlpEndpoint := getEnvString(otelExporterEndpointEnv, "localhost:4317")
	otlpInsecure := getEnvBool(otelExporterInsecureEnv, true)
	traceSampleRate := getEnvFloat64(otelTraceSampleRateEnv, 0.1)
	metricsInterval := getEnvInt(otelMetricsIntervalEnv, 15)

	return &otelConfig{
		enabled:               enabled,
		serviceName:           serviceName,
		serviceVersion:        serviceVersion,
		environment:           environment,
		otlpEndpoint:          otlpEndpoint,
		otlpInsecure:          otlpInsecure,
		traceSampleRate:       traceSampleRate,
		metricsIntervalSeconds: metricsInterval,
	}, nil
}

func (c *otelConfig) Enabled() bool {
	return c.enabled
}

func (c *otelConfig) ServiceName() string {
	return c.serviceName
}

func (c *otelConfig) ServiceVersion() string {
	return c.serviceVersion
}

func (c *otelConfig) Environment() string {
	return c.environment
}

func (c *otelConfig) OTLPEndpoint() string {
	return c.otlpEndpoint
}

func (c *otelConfig) OTLPInsecure() bool {
	return c.otlpInsecure
}

func (c *otelConfig) TraceSampleRate() float64 {
	return c.traceSampleRate
}

func (c *otelConfig) MetricsIntervalSeconds() int {
	return c.metricsIntervalSeconds
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return defaultValue
}
