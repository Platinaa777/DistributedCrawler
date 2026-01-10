package fetcher

import (
	"context"
	"crypto/sha256"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPFetcher implements the Fetcher interface using HTTP client
type HTTPFetcher struct {
	httpClient  *http.Client
	authOptions models.AuthOptions
	retryPolicy models.RetryPolicy
}

// NewHTTPFetcher creates a new HTTP fetcher with specified options
func NewHTTPFetcher(auth models.AuthOptions, retry models.RetryPolicy) *HTTPFetcher {
	// Default to at least 1 attempt if MaxAttempts is 0
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 1
	}

	return &HTTPFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
		authOptions: auth,
		retryPolicy: retry,
	}
}

// Fetch performs an HTTP GET request with retry logic and returns the result
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (*services.FetchResult, error) {
	var lastErr error
	backoff := time.Duration(f.retryPolicy.BackoffInitialMs) * time.Millisecond

	for attempt := uint64(0); attempt < f.retryPolicy.MaxAttempts; attempt++ {
		result, err := f.doFetch(ctx, url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < f.retryPolicy.MaxAttempts-1 {
			// Check if context is cancelled before sleeping
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Continue to next attempt
			}

			// Calculate next backoff duration
			backoff = time.Duration(float64(backoff) * f.retryPolicy.BackoffMultiplier)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", f.retryPolicy.MaxAttempts, lastErr)
}

// doFetch performs a single HTTP fetch attempt
func (f *HTTPFetcher) doFetch(ctx context.Context, url string) (*services.FetchResult, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "DistributedCrawler/1.0")

	// Apply authentication options
	if f.authOptions.Cookie != "" {
		req.Header.Set("Cookie", f.authOptions.Cookie)
	}
	if f.authOptions.BasicUser != "" {
		req.SetBasicAuth(f.authOptions.BasicUser, f.authOptions.BasicPassword)
	}
	if f.authOptions.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+f.authOptions.BearerToken)
	}

	// Execute request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(hash[:])

	// Get content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/html"
	}

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	return &services.FetchResult{
		Body:        body,
		BodyHash:    bodyHash,
		FinalURL:    finalURL,
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
	}, nil
}
