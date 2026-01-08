package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Circuit breaker states
type State int

const (
	// StateClosed - circuit is functioning normally, requests flow through
	StateClosed State = iota
	// StateOpen - circuit is broken, requests are rejected immediately
	StateOpen
	// StateHalfOpen - circuit is testing, limited requests allowed
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines the circuit breaker parameters
type CircuitBreakerConfig struct {
	// Name identifies this circuit breaker (e.g., node ID)
	Name string
	// FailureThreshold is the number of failures before opening
	FailureThreshold int64
	// SuccessThreshold is the number of successes in half-open to close
	SuccessThreshold int64
	// Timeout is how long the circuit stays open before trying half-open
	Timeout time.Duration
	// FailureWindow is the time window for counting failures
	FailureWindow time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		FailureWindow:    60 * time.Second,
	}
}

// CircuitState represents the persisted state in Redis
type CircuitState struct {
	State         State     `json:"state"`
	Failures      int64     `json:"failures"`
	Successes     int64     `json:"successes"`
	LastFailure   time.Time `json:"last_failure"`
	LastStateChange time.Time `json:"last_state_change"`
}

// CircuitBreaker implements a distributed circuit breaker using Redis
type CircuitBreaker struct {
	rdb    redis.UniversalClient
	mu     sync.RWMutex
	prefix string
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// ErrCircuitHalfOpen is returned when the circuit is half-open and at capacity
var ErrCircuitHalfOpen = errors.New("circuit breaker is half-open, limited requests allowed")

// NewCircuitBreaker creates a new distributed circuit breaker
func NewCircuitBreaker(rdb redis.UniversalClient) *CircuitBreaker {
	return &CircuitBreaker{
		rdb:    rdb,
		prefix: "plm:circuit:",
	}
}

// key generates the Redis key for a circuit
func (cb *CircuitBreaker) key(name string) string {
	return cb.prefix + name
}

// failuresKey generates the Redis key for failure counts
func (cb *CircuitBreaker) failuresKey(name string) string {
	return cb.prefix + name + ":failures"
}

// GetState retrieves the current state of a circuit
func (cb *CircuitBreaker) GetState(ctx context.Context, cfg *CircuitBreakerConfig) (*CircuitState, error) {
	data, err := cb.rdb.Get(ctx, cb.key(cfg.Name)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Initialize new circuit in closed state
			return &CircuitState{
				State:           StateClosed,
				LastStateChange: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get circuit state: %w", err)
	}

	var state CircuitState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal circuit state: %w", err)
	}

	// Check if open circuit should transition to half-open
	if state.State == StateOpen && time.Since(state.LastStateChange) >= cfg.Timeout {
		state.State = StateHalfOpen
		state.Successes = 0
		state.LastStateChange = time.Now()
		if err := cb.saveState(ctx, cfg.Name, &state); err != nil {
			return nil, err
		}
	}

	return &state, nil
}

// saveState persists the circuit state to Redis
func (cb *CircuitBreaker) saveState(ctx context.Context, name string, state *CircuitState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal circuit state: %w", err)
	}

	return cb.rdb.Set(ctx, cb.key(name), data, 24*time.Hour).Err()
}

// Allow checks if a request should be allowed through the circuit
func (cb *CircuitBreaker) Allow(ctx context.Context, cfg *CircuitBreakerConfig) error {
	state, err := cb.GetState(ctx, cfg)
	if err != nil {
		return err
	}

	switch state.State {
	case StateClosed:
		return nil
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		// In half-open, we allow limited requests to test the circuit
		return nil
	}

	return nil
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(ctx context.Context, cfg *CircuitBreakerConfig) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.GetState(ctx, cfg)
	if err != nil {
		return err
	}

	if state.State == StateHalfOpen {
		state.Successes++
		if state.Successes >= cfg.SuccessThreshold {
			// Transition to closed
			state.State = StateClosed
			state.Failures = 0
			state.Successes = 0
			state.LastStateChange = time.Now()
		}
		return cb.saveState(ctx, cfg.Name, state)
	}

	return nil
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(ctx context.Context, cfg *CircuitBreakerConfig) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.GetState(ctx, cfg)
	if err != nil {
		return err
	}

	now := time.Now()
	state.LastFailure = now
	state.Failures++

	// Use sliding window for failure counting
	failureCount, err := cb.incrementFailureCount(ctx, cfg)
	if err != nil {
		return err
	}

	if state.State == StateHalfOpen {
		// Any failure in half-open reopens the circuit
		state.State = StateOpen
		state.LastStateChange = now
		state.Successes = 0
	} else if state.State == StateClosed && failureCount >= cfg.FailureThreshold {
		// Open the circuit
		state.State = StateOpen
		state.LastStateChange = now
	}

	return cb.saveState(ctx, cfg.Name, state)
}

// incrementFailureCount uses a sliding window to count failures
func (cb *CircuitBreaker) incrementFailureCount(ctx context.Context, cfg *CircuitBreakerConfig) (int64, error) {
	key := cb.failuresKey(cfg.Name)
	now := time.Now()
	windowStart := now.Add(-cfg.FailureWindow).UnixMilli()

	pipe := cb.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	countCmd := pipe.ZCard(ctx, key)
	pipe.PExpire(ctx, key, cfg.FailureWindow)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to record failure: %w", err)
	}

	return countCmd.Val(), nil
}

// ForceOpen forces the circuit to open immediately (for chaos testing)
func (cb *CircuitBreaker) ForceOpen(ctx context.Context, cfg *CircuitBreakerConfig) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := &CircuitState{
		State:           StateOpen,
		LastStateChange: time.Now(),
	}

	return cb.saveState(ctx, cfg.Name, state)
}

// Reset resets the circuit to closed state
func (cb *CircuitBreaker) Reset(ctx context.Context, cfg *CircuitBreakerConfig) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Delete the circuit state and failure counts
	pipe := cb.rdb.Pipeline()
	pipe.Del(ctx, cb.key(cfg.Name))
	pipe.Del(ctx, cb.failuresKey(cfg.Name))
	_, err := pipe.Exec(ctx)

	return err
}

// GetAllCircuits returns the state of all known circuits
func (cb *CircuitBreaker) GetAllCircuits(ctx context.Context) (map[string]*CircuitState, error) {
	pattern := cb.prefix + "*"
	keys, err := cb.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	circuits := make(map[string]*CircuitState)
	for _, key := range keys {
		// Skip failure count keys
		if len(key) > 9 && key[len(key)-9:] == ":failures" {
			continue
		}

		data, err := cb.rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var state CircuitState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		name := key[len(cb.prefix):]
		circuits[name] = &state
	}

	return circuits, nil
}
