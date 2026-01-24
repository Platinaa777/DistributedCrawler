package services

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"time"
)

// RateLimiter provides distributed rate limiting for crawler requests
type RateLimiter interface {
	// Allow checks if a request is allowed under the rate limit
	// scope and id are used to create unique rate limit buckets (e.g., scope="domain", id="example.com")
	// Returns:
	//   - allowed: true if request can proceed
	//   - retryAfter: duration to wait before retrying if denied
	//   - err: any error that occurred
	Allow(ctx context.Context, jobCfg models.CrawlJobConfig, scope, id string) (allowed bool, retryAfter time.Duration, err error)
}
