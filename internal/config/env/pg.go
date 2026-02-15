package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
	"strings"
)

const (
	dsnEnvName             = "PG_DSN"
	shardingEnabledEnvName = "PG_SHARDING_ENABLED"
	shardDSNsEnvName       = "PG_SHARD_DSNS"
)

type pgConfig struct {
	dsn             string
	shardingEnabled bool
	shardDSNs       []string
}

func NewPGConfig() (config.PGConfig, error) {
	dsn := os.Getenv(dsnEnvName)
	if len(dsn) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", dsnEnvName)
	}

	shardingEnabled := strings.EqualFold(os.Getenv(shardingEnabledEnvName), "true")

	var shardDSNs []string
	if shardingEnabled {
		raw := os.Getenv(shardDSNsEnvName)
		if len(raw) == 0 {
			return nil, fmt.Errorf("%s is required when sharding is enabled", shardDSNsEnvName)
		}
		for _, dsn := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(dsn)
			if trimmed != "" {
				shardDSNs = append(shardDSNs, trimmed)
			}
		}
		if len(shardDSNs) < 2 {
			return nil, fmt.Errorf("%s must contain at least 2 DSNs", shardDSNsEnvName)
		}
	}

	return &pgConfig{
		dsn:             dsn,
		shardingEnabled: shardingEnabled,
		shardDSNs:       shardDSNs,
	}, nil
}

func (cfg *pgConfig) DSN() string {
	return cfg.dsn
}

func (cfg *pgConfig) ShardingEnabled() bool {
	return cfg.shardingEnabled
}

func (cfg *pgConfig) ShardDSNs() []string {
	return cfg.shardDSNs
}
