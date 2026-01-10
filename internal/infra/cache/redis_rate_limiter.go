package cache

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements distributed rate limiting using Redis and Lua scripts
type RedisRateLimiter struct {
	client *redis.Client
	script *redis.Script
	ttl    time.Duration
}

// Lua script for token bucket rate limiting
// Returns: {allowed (0/1), tokens_left, retry_after_ms}
const rateLimitLuaScript = `
-- KEYS[1] = rate limit key (e.g., "rl:rps:jobid:scope:id")
-- ARGV[1] = refill_per_sec (rate limit RPS)
-- ARGV[2] = cost (default 1)
-- ARGV[3] = ttl_ms (TTL in milliseconds)

local key = KEYS[1]
local refill_per_sec = tonumber(ARGV[1])
local cost = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

-- Get current time from Redis (shared clock across instances)
local time_result = redis.call("TIME")
local now_sec = tonumber(time_result[1])
local now_usec = tonumber(time_result[2])
local now_ms = (now_sec * 1000) + math.floor(now_usec / 1000)

-- Calculate capacity (burst = approximately 1 second worth of requests)
local capacity = math.max(1, math.ceil(refill_per_sec))

-- Get current state from Redis HASH
local current_tokens = tonumber(redis.call("HGET", key, "tokens"))
local last_refill_ms = tonumber(redis.call("HGET", key, "ts"))

-- Initialize if not exists
if not current_tokens or not last_refill_ms then
    current_tokens = capacity
    last_refill_ms = now_ms
end

-- Calculate elapsed time and refill tokens
local elapsed_sec = (now_ms - last_refill_ms) / 1000.0
local new_tokens = math.min(capacity, current_tokens + (elapsed_sec * refill_per_sec))

-- Check if we have enough tokens
local allowed = 0
local retry_after_ms = 0

if new_tokens >= cost then
    -- Allow request and deduct tokens
    allowed = 1
    new_tokens = new_tokens - cost
else
    -- Deny request and calculate retry time
    local tokens_needed = cost - new_tokens
    retry_after_ms = math.ceil((tokens_needed / refill_per_sec) * 1000)
end

-- Update Redis HASH with new state
redis.call("HSET", key, "tokens", new_tokens)
redis.call("HSET", key, "ts", now_ms)

-- Set TTL on the key
redis.call("PEXPIRE", key, ttl_ms)

return {allowed, new_tokens, retry_after_ms}
`

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, ttl time.Duration) services.RateLimiter {
	if ttl == 0 {
		ttl = 5 * time.Minute // Default TTL: 5 minutes
	}

	return &RedisRateLimiter{
		client: client,
		script: redis.NewScript(rateLimitLuaScript),
		ttl:    ttl,
	}
}

// Allow checks if a request is allowed under the rate limit
func (r *RedisRateLimiter) Allow(
	ctx context.Context,
	jobCfg models.CrawlJobConfig,
	scope, id string,
) (allowed bool, retryAfter time.Duration, err error) {
	// Extract RPS from job config
	rps := jobCfg.RateLimit.Rps
	if rps <= 0 {
		// No rate limit configured, allow all requests
		return true, 0, nil
	}

	// Build Redis key: rl:rps:<jobID>:<scope>:<id>
	key := fmt.Sprintf("rl:rps:%s:%s:%s", jobCfg.ID.String(), scope, id)

	// Execute Lua script
	// KEYS: [key]
	// ARGV: [refill_per_sec, cost, ttl_ms]
	cost := 1.0
	ttlMs := r.ttl.Milliseconds()

	result, err := r.script.Run(ctx, r.client, []string{key}, rps, cost, ttlMs).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	// Parse result: {allowed, tokens_left, retry_after_ms}
	resultSlice, ok := result.([]any)
	if !ok || len(resultSlice) != 3 {
		return false, 0, fmt.Errorf("unexpected script result format: %v", result)
	}

	allowedInt, ok := resultSlice[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid allowed value: %v", resultSlice[0])
	}

	retryAfterMs, ok := resultSlice[2].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid retry_after_ms value: %v", resultSlice[2])
	}

	return allowedInt == 1, time.Duration(retryAfterMs) * time.Millisecond, nil
}
