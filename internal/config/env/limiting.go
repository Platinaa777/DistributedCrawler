package env

import (
	"os"
	"strings"
)

const (
	limiterTypeEnvName = "LIMITER_TYPE"

	LimiterRedis    = "redis"
	LimiterInMemory = "inmemory"
)

// GetLimiterType returns the configured rate limiter provider ("redis" or "inmemory").
// Controlled by the LIMITER_TYPE environment variable.
// Defaults to "redis" when not set or invalid.
func GetLimiterType() string {
	t := strings.ToLower(strings.TrimSpace(os.Getenv(limiterTypeEnvName)))

	switch t {
	case LimiterInMemory, "in-memory", "in_memory", "memory":
		return LimiterInMemory
	case LimiterRedis:
		return LimiterRedis
	default:
		return LimiterRedis
	}
}
