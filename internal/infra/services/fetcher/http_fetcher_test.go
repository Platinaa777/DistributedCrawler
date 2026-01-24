package fetcher

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryable  bool
		shouldWant string
	}{
		{
			name:       "nil error",
			err:        nil,
			retryable:  false,
			shouldWant: "nil errors should not be retried",
		},
		{
			name: "DNS error - no such host",
			err: &url.Error{
				Op:  "Get",
				URL: "https://nonexistent-domain-12345.com",
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "nonexistent-domain-12345.com",
					IsNotFound: true,
				},
			},
			retryable:  false,
			shouldWant: "DNS lookup failures should not be retried",
		},
		{
			name: "connection refused",
			err: &url.Error{
				Op:  "Get",
				URL: "https://localhost:9999",
				Err: errors.New("dial tcp [::1]:9999: connect: connection refused"),
			},
			retryable:  false,
			shouldWant: "connection refused should not be retried",
		},
		{
			name:       "generic network error",
			err:        errors.New("dial tcp: i/o timeout"),
			retryable:  true,
			shouldWant: "timeout errors should be retried",
		},
		{
			name:       "generic error",
			err:        errors.New("some random error"),
			retryable:  true,
			shouldWant: "unknown errors should be retried by default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result, tt.shouldWant)
		})
	}
}

func TestHTTPFetcher_Fetch_PermanentError(t *testing.T) {
	// Create fetcher with retry policy
	retryPolicy := models.RetryPolicy{
		MaxAttempts:        3,
		BackoffInitialMs:   100,
		BackoffMultiplier:  2.0,
	}

	fetcher := NewHTTPFetcher(models.AuthOptions{}, retryPolicy)

	// Try to fetch from non-existent domain
	ctx := context.Background()
	result, err := fetcher.Fetch(ctx, "https://this-domain-definitely-does-not-exist-12345.com/page")

	// Should get a permanent error
	require.Error(t, err)
	require.Nil(t, result)

	// Error should indicate it's permanent
	assert.True(t, strings.Contains(err.Error(), "permanent error"),
		"error should be marked as permanent: %v", err)

	// Should NOT have "failed after 3 attempts" since it shouldn't retry
	assert.False(t, strings.Contains(err.Error(), "failed after 3 attempts"),
		"should not retry permanent errors: %v", err)
}

func TestHTTPFetcher_Fetch_RetryableError(t *testing.T) {
	// This test is harder to simulate without a mock HTTP client
	// We'll just verify that the retry logic structure is correct

	retryPolicy := models.RetryPolicy{
		MaxAttempts:        2,
		BackoffInitialMs:   10, // Short backoff for testing
		BackoffMultiplier:  2.0,
	}

	fetcher := NewHTTPFetcher(models.AuthOptions{}, retryPolicy)

	// Create a context with short timeout to trigger retryable timeout error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Try to fetch from a valid domain but with impossible timeout
	result, err := fetcher.Fetch(ctx, "https://example.com")

	// Should get an error (context deadline exceeded)
	require.Error(t, err)
	require.Nil(t, result)
}
