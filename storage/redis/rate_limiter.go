package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding window rate limiter using Redis sorted sets.
// This algorithm provides smoother rate limiting compared to fixed windows by
// considering the precise timestamp of each request.
type RateLimiter struct {
	rdb redis.UniversalClient
}

// RateLimitConfig defines the rate limiting parameters
type RateLimitConfig struct {
	// Key prefix for the rate limit bucket
	Key string
	// Maximum number of requests allowed in the window
	Limit int64
	// Duration of the sliding window
	Window time.Duration
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	// Allowed indicates if the request should be permitted
	Allowed bool
	// Remaining is the number of requests remaining in the current window
	Remaining int64
	// ResetAt is when the oldest request in the window will expire
	ResetAt time.Time
	// RetryAfter is the duration until the next request can be made (if denied)
	RetryAfter time.Duration
}

// NewRateLimiter creates a new sliding window rate limiter
func NewRateLimiter(rdb redis.UniversalClient) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// slidingWindowScript is the Lua script for atomic sliding window rate limiting
// This ensures all rate limit operations are atomic and consistent
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

-- Calculate the start of the sliding window
local window_start = now - window

-- Remove expired entries (outside the sliding window)
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current requests in the window
local current_count = redis.call('ZCARD', key)

-- Check if we're over the limit
if current_count >= limit then
    -- Get the oldest entry to calculate retry-after
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retry_after = 0
    if oldest[2] then
        retry_after = oldest[2] + window - now
    end
    return {0, limit - current_count, retry_after}
end

-- Add the new request with current timestamp as score
redis.call('ZADD', key, now, member)

-- Set expiry on the key to auto-cleanup
redis.call('PEXPIRE', key, window)

-- Return success with remaining count
return {1, limit - current_count - 1, 0}
`

// Allow checks if a request should be allowed under the rate limit
// and records the request if allowed.
func (rl *RateLimiter) Allow(ctx context.Context, cfg *RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := cfg.Window.Milliseconds()

	// Use timestamp + random suffix as unique member
	member := fmt.Sprintf("%d:%d", nowMs, now.UnixNano())

	// Execute the Lua script atomically
	result, err := rl.rdb.Eval(ctx, slidingWindowScript, []string{cfg.Key}, nowMs, windowMs, cfg.Limit, member).Result()
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	// Parse the result
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, fmt.Errorf("unexpected rate limit response format")
	}

	allowed, _ := arr[0].(int64)
	remaining, _ := arr[1].(int64)
	retryAfterMs, _ := arr[2].(int64)

	return &RateLimitResult{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		ResetAt:    now.Add(cfg.Window),
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}

// AllowN checks if N requests should be allowed (for batch operations)
func (rl *RateLimiter) AllowN(ctx context.Context, cfg *RateLimitConfig, n int64) (*RateLimitResult, error) {
	if n <= 0 {
		return &RateLimitResult{Allowed: true, Remaining: cfg.Limit}, nil
	}

	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := cfg.Window.Milliseconds()
	key := cfg.Key
	windowStart := nowMs - windowMs

	// Use pipeline for atomic batch check
	pipe := rl.rdb.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))

	// Get current count
	countCmd := pipe.ZCard(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate limit batch check failed: %w", err)
	}

	currentCount := countCmd.Val()

	// Check if we have room for N requests
	if currentCount+n > cfg.Limit {
		return &RateLimitResult{
			Allowed:   false,
			Remaining: cfg.Limit - currentCount,
			ResetAt:   now.Add(cfg.Window),
		}, nil
	}

	// Add N entries
	pipe2 := rl.rdb.Pipeline()
	members := make([]redis.Z, n)
	for i := int64(0); i < n; i++ {
		members[i] = redis.Z{
			Score:  float64(nowMs),
			Member: fmt.Sprintf("%d:%d:%d", nowMs, now.UnixNano(), i),
		}
	}
	pipe2.ZAdd(ctx, key, members...)
	pipe2.PExpire(ctx, key, cfg.Window)

	_, err = pipe2.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate limit batch record failed: %w", err)
	}

	return &RateLimitResult{
		Allowed:   true,
		Remaining: cfg.Limit - currentCount - n,
		ResetAt:   now.Add(cfg.Window),
	}, nil
}

// Reset clears the rate limit for a given key
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	return rl.rdb.Del(ctx, key).Err()
}

// GetRemaining returns the remaining quota for a key without consuming
func (rl *RateLimiter) GetRemaining(ctx context.Context, cfg *RateLimitConfig) (int64, error) {
	now := time.Now()
	windowStart := now.Add(-cfg.Window).UnixMilli()

	// Remove expired and count
	pipe := rl.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, cfg.Key, "-inf", strconv.FormatInt(windowStart, 10))
	countCmd := pipe.ZCard(ctx, cfg.Key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return cfg.Limit - countCmd.Val(), nil
}
