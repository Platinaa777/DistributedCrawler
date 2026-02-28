package env

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultSecretsReloadInterval = 60 * time.Second
)

// SecretsConfig holds configuration for the secrets file store.
type SecretsConfig struct {
	filePath       string
	watchEnabled   bool
	reloadInterval time.Duration
}

// NewSecretsConfig reads QUEUE_SECRETS_* env vars.
func NewSecretsConfig() (*SecretsConfig, error) {
	path := os.Getenv("QUEUE_SECRETS_FILE_PATH")
	if path == "" {
		return nil, fmt.Errorf("QUEUE_SECRETS_FILE_PATH is not set")
	}

	reloadStr := os.Getenv("QUEUE_SECRETS_RELOAD_INTERVAL")
	reloadInterval := defaultSecretsReloadInterval
	if reloadStr != "" {
		d, err := time.ParseDuration(reloadStr)
		if err == nil {
			reloadInterval = d
		}
	}

	watchEnabled := os.Getenv("QUEUE_SECRETS_WATCH_ENABLED") != "false"

	return &SecretsConfig{
		filePath:       path,
		watchEnabled:   watchEnabled,
		reloadInterval: reloadInterval,
	}, nil
}

// FilePath returns the path to the secrets JSON file.
func (c *SecretsConfig) FilePath() string { return c.filePath }

// WatchEnabled returns true if file-watch polling is enabled.
func (c *SecretsConfig) WatchEnabled() bool { return c.watchEnabled }

// ReloadInterval returns the polling interval for the secrets file.
func (c *SecretsConfig) ReloadInterval() time.Duration { return c.reloadInterval }
