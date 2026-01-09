package env

import (
	"fmt"
	"os"

	"distributed-crawler/internal/config"
)

const (
	logLevelEnvName = "LOG_LEVEL"
	logEnvEnvName   = "LOG_ENV"
)

const (
	EnvProduction  = "production"
	EnvDevelopment = "development"
)

type loggerConfig struct {
	level string
	env   string
}

func NewLoggerConfig() (config.LoggerConfig, error) {
	level := os.Getenv(logLevelEnvName)
	if len(level) == 0 {
		level = "info" // default level
	}

	env := os.Getenv(logEnvEnvName)
	if len(env) == 0 {
		env = EnvDevelopment // default env
	}

	if env != EnvProduction && env != EnvDevelopment {
		return nil, fmt.Errorf("%s must be '%s' or '%s'", logEnvEnvName, EnvProduction, EnvDevelopment)
	}

	return &loggerConfig{
		level: level,
		env:   env,
	}, nil
}

func (cfg *loggerConfig) Level() string {
	return cfg.level
}

func (cfg *loggerConfig) Env() string {
	return cfg.env
}
