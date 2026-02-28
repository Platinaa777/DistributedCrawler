package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// SecretsStore provides access to external credential key-value maps.
type SecretsStore interface {
	// Get returns the secret map for the given key.
	Get(key string) (map[string]string, error)
}

// fileSecretsStore loads secrets from a JSON file and polls for changes.
//
// File format:
//
//	{
//	  "rmq_default":    { "url": "amqp://user:pass@host/vhost" },
//	  "kafka_default":  { "brokers": "b1:9092,b2:9092" }
//	}
type fileSecretsStore struct {
	path           string
	reloadInterval time.Duration
	mu             sync.RWMutex
	data           map[string]map[string]string
}

// NewFileSecretsStore creates and starts a file-based secrets store.
// It loads the file immediately and starts a background poller.
// Cancel ctx to stop polling.
func NewFileSecretsStore(ctx context.Context, path string, reloadInterval time.Duration) (SecretsStore, error) {
	s := &fileSecretsStore{
		path:           path,
		reloadInterval: reloadInterval,
	}

	if err := s.load(); err != nil {
		return nil, fmt.Errorf("secrets: initial load failed: %w", err)
	}

	go s.watch(ctx)
	return s, nil
}

// Get returns the secret map for key, or an error if not found.
func (s *fileSecretsStore) Get(key string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("secrets: key %q not found", key)
	}
	// Return a copy to prevent mutation
	out := make(map[string]string, len(v))
	for k, val := range v {
		out[k] = val
	}
	return out, nil
}

func (s *fileSecretsStore) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var parsed map[string]map[string]string
	if err := json.Unmarshal(b, &parsed); err != nil {
		return fmt.Errorf("secrets: JSON parse error: %w", err)
	}

	s.mu.Lock()
	s.data = parsed
	s.mu.Unlock()
	return nil
}

func (s *fileSecretsStore) watch(ctx context.Context) {
	ticker := time.NewTicker(s.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.load() // ignore errors on reload; stale data remains
		}
	}
}
