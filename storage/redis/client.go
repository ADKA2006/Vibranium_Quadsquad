// Package redis provides Redis Sentinel integration for the Predictive Liquidity Mesh.
// Implements sliding-window rate limiting and circuit breaker patterns.
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection configuration
type Config struct {
	// Sentinel configuration
	MasterName    string
	SentinelAddrs []string

	// Standalone configuration (fallback)
	Addr     string
	Password string
	DB       int

	// Pool configuration
	PoolSize     int
	MinIdleConns int

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns a default configuration for local development
func DefaultConfig() *Config {
	return &Config{
		MasterName:    "mymaster",
		SentinelAddrs: []string{"localhost:26379"},
		Addr:          "localhost:6379",
		Password:      "",
		DB:            0,
		PoolSize:      100,
		MinIdleConns:  10,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
	}
}

// Client wraps Redis client with rate limiting and circuit breaker capabilities
type Client struct {
	rdb          redis.UniversalClient
	rateLimiter  *RateLimiter
	circuitBreaker *CircuitBreaker
	mu           sync.RWMutex
}

// NewClient creates a new Redis client with Sentinel support
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	var rdb redis.UniversalClient

	// Try Sentinel first, fallback to standalone
	if len(cfg.SentinelAddrs) > 0 && cfg.MasterName != "" {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			PoolSize:      cfg.PoolSize,
			MinIdleConns:  cfg.MinIdleConns,
			ReadTimeout:   cfg.ReadTimeout,
			WriteTimeout:  cfg.WriteTimeout,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		})
	}

	// Verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	client := &Client{
		rdb:           rdb,
		rateLimiter:   NewRateLimiter(rdb),
		circuitBreaker: NewCircuitBreaker(rdb),
	}

	return client, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Redis returns the underlying Redis client
func (c *Client) Redis() redis.UniversalClient {
	return c.rdb
}

// RateLimiter returns the rate limiter instance
func (c *Client) RateLimiter() *RateLimiter {
	return c.rateLimiter
}

// CircuitBreaker returns the circuit breaker instance
func (c *Client) CircuitBreaker() *CircuitBreaker {
	return c.circuitBreaker
}
