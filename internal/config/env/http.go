package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"net"
	"os"
)

const (
	httpHostEnvName = "HTTP_HOST"
	httpPortEnvName = "HTTP_PORT"
)

type httpConfig struct {
	host string
	port string
}

func NewHTTPConfig() (config.HTTPConfig, error) {
	host := os.Getenv(httpHostEnvName)
	if len(host) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", httpHostEnvName)
	}

	port := os.Getenv(httpPortEnvName)
	if len(port) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", httpPortEnvName)
	}

	return &httpConfig{
		host: host,
		port: port,
	}, nil
}

func (cfg *httpConfig) Address() string {
	return net.JoinHostPort(cfg.host, cfg.port)
}
