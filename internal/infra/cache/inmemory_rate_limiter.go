package cache

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
	"fmt"
	"math"
	"sync"
	"time"
)

type rateLimitBucket struct {
	tokens       float64
	lastRefillAt time.Time
	lastSeenAt   time.Time
}

// InMemoryRateLimiter stores rate limit buckets in-process (per worker instance).
type InMemoryRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateLimitBucket
	ttl     time.Duration
}

// NewInMemoryRateLimiter creates a local token-bucket rate limiter.
func NewInMemoryRateLimiter(ttl time.Duration) services.RateLimiter {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	return &InMemoryRateLimiter{
		buckets: make(map[string]rateLimitBucket),
		ttl:     ttl,
	}
}

func (r *InMemoryRateLimiter) Allow(
	ctx context.Context,
	jobCfg models.CrawlJobConfig,
	scope, id string,
) (allowed bool, retryAfter time.Duration, err error) {
	if err := ctx.Err(); err != nil {
		return false, 0, err
	}

	rps := jobCfg.RateLimit.Rps
	if rps <= 0 {
		return true, 0, nil
	}

	key := fmt.Sprintf("rl:rps:%s:%s:%s", jobCfg.ID.String(), scope, id)
	now := time.Now()
	capacity := math.Max(1, math.Ceil(rps))
	cost := 1.0

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanupExpiredLocked(now)

	bucket, ok := r.buckets[key]
	if !ok {
		bucket = rateLimitBucket{
			tokens:       capacity,
			lastRefillAt: now,
			lastSeenAt:   now,
		}
	}

	elapsedSec := now.Sub(bucket.lastRefillAt).Seconds()
	if elapsedSec > 0 {
		bucket.tokens = math.Min(capacity, bucket.tokens+(elapsedSec*rps))
		bucket.lastRefillAt = now
	}

	if bucket.tokens >= cost {
		bucket.tokens -= cost
		bucket.lastSeenAt = now
		r.buckets[key] = bucket
		return true, 0, nil
	}

	tokensNeeded := cost - bucket.tokens
	retryAfterMs := math.Ceil((tokensNeeded / rps) * 1000)
	if retryAfterMs < 1 {
		retryAfterMs = 1
	}

	bucket.lastSeenAt = now
	r.buckets[key] = bucket

	return false, time.Duration(retryAfterMs) * time.Millisecond, nil
}

func (r *InMemoryRateLimiter) cleanupExpiredLocked(now time.Time) {
	if r.ttl <= 0 {
		return
	}

	for key, bucket := range r.buckets {
		if now.Sub(bucket.lastSeenAt) > r.ttl {
			delete(r.buckets, key)
		}
	}
}
