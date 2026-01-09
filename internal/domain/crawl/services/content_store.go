package services

import (
	"context"
	"io"
)

// ContentStore manages raw page content storage (HTML, JSON, etc.)
type ContentStore interface {
	// Store saves content and returns the storage key
	Store(ctx context.Context, key string, content []byte, contentType string) error

	// Get retrieves content by storage key
	Get(ctx context.Context, key string) ([]byte, error)

	// GetReader retrieves content as a reader (for large files)
	GetReader(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes content by storage key
	Delete(ctx context.Context, key string) error

	// Exists checks if content exists
	Exists(ctx context.Context, key string) (bool, error)
}
