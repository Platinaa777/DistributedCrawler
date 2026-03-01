package env

import (
	"distributed-crawler/internal/config"
	"os"
	"strings"
)

const corsAllowedOriginsEnvName = "HTTP_CORS_ALLOWED_ORIGINS"

type corsConfig struct {
	allowedOrigins []string
}

func NewCORSConfig() (config.CORSConfig, error) {
	raw := os.Getenv(corsAllowedOriginsEnvName)
	if raw == "" {
		raw = "http://localhost:4200"
	}

	origins := strings.Split(raw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}

	return &corsConfig{allowedOrigins: origins}, nil
}

func (c *corsConfig) AllowedOrigins() []string {
	return c.allowedOrigins
}
