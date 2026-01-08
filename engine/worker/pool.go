// Package worker provides a bounded worker pool for controlled concurrency.
// Uses github.com/gammazero/workerpool to prevent goroutine explosion.
package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/plm/predictive-liquidity-mesh/engine/models"
)

// Pool manages a bounded pool of workers for settlement processing.
type Pool struct {
	wp          *workerpool.WorkerPool
	maxWorkers  int
	
	// Metrics
	submitted   atomic.Int64
	completed   atomic.Int64
	failed      atomic.Int64
	
	// Shutdown coordination
	mu          sync.RWMutex
	stopped     bool
}

// Config holds worker pool configuration
type Config struct {
	// MaxWorkers is the maximum number of concurrent workers
	MaxWorkers int
	// QueueSize is the maximum number of pending tasks (0 = unbounded)
	QueueSize int
}

// DefaultConfig returns sensible defaults for production
func DefaultConfig() *Config {
	return &Config{
		MaxWorkers: 100,
		QueueSize:  10000,
	}
}

// NewPool creates a new bounded worker pool.
func NewPool(cfg *Config) *Pool {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	
	return &Pool{
		wp:         workerpool.New(cfg.MaxWorkers),
		maxWorkers: cfg.MaxWorkers,
	}
}

// SettlementHandler is the function signature for processing settlements
type SettlementHandler func(ctx context.Context, req *models.SettlementRequest) (*models.SettlementResponse, error)

// Submit submits a settlement request for async processing.
// Returns immediately; results are delivered via the callback.
func (p *Pool) Submit(
	ctx context.Context,
	req *models.SettlementRequest,
	handler SettlementHandler,
	callback func(*models.SettlementResponse, error),
) error {
	p.mu.RLock()
	if p.stopped {
		p.mu.RUnlock()
		return ErrPoolStopped
	}
	p.mu.RUnlock()

	p.submitted.Add(1)

	p.wp.Submit(func() {
		// Check context before processing
		if ctx.Err() != nil {
			p.failed.Add(1)
			if callback != nil {
				resp := models.AcquireSettlementResponse()
				resp.RequestID = req.ID
				resp.Status = models.StatusFailed
				resp.Error = ctx.Err().Error()
				resp.CompletedAt = time.Now()
				callback(resp, ctx.Err())
			}
			return
		}

		// Process the settlement
		resp, err := handler(ctx, req)
		
		if err != nil {
			p.failed.Add(1)
		} else {
			p.completed.Add(1)
		}

		if callback != nil {
			callback(resp, err)
		}
	})

	return nil
}

// SubmitWait submits a request and waits for completion.
// Use for synchronous processing when needed.
func (p *Pool) SubmitWait(
	ctx context.Context,
	req *models.SettlementRequest,
	handler SettlementHandler,
) (*models.SettlementResponse, error) {
	var resp *models.SettlementResponse
	var err error
	done := make(chan struct{})

	submitErr := p.Submit(ctx, req, handler, func(r *models.SettlementResponse, e error) {
		resp = r
		err = e
		close(done)
	})

	if submitErr != nil {
		return nil, submitErr
	}

	select {
	case <-done:
		return resp, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubmitBatch submits multiple requests and waits for all to complete.
func (p *Pool) SubmitBatch(
	ctx context.Context,
	requests []*models.SettlementRequest,
	handler SettlementHandler,
) ([]*models.SettlementResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	responses := make([]*models.SettlementResponse, len(requests))
	errors := make([]error, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, req := range requests {
		wg.Add(1)
		idx := i
		r := req

		err := p.Submit(ctx, r, handler, func(resp *models.SettlementResponse, err error) {
			mu.Lock()
			responses[idx] = resp
			errors[idx] = err
			mu.Unlock()
			wg.Done()
		})

		if err != nil {
			wg.Done()
			return nil, err
		}
	}

	wg.Wait()

	// Check for any errors
	for _, err := range errors {
		if err != nil {
			return responses, err
		}
	}

	return responses, nil
}

// Stop gracefully shuts down the worker pool.
// Waits for all pending tasks to complete.
func (p *Pool) Stop() {
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	
	p.wp.StopWait()
}

// StopNow immediately stops the pool without waiting.
func (p *Pool) StopNow() {
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	
	p.wp.Stop()
}

// Stats returns current pool statistics
type Stats struct {
	MaxWorkers int   `json:"max_workers"`
	Submitted  int64 `json:"submitted"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Pending    int64 `json:"pending"`
}

// Stats returns current pool statistics
func (p *Pool) Stats() Stats {
	submitted := p.submitted.Load()
	completed := p.completed.Load()
	failed := p.failed.Load()
	
	return Stats{
		MaxWorkers: p.maxWorkers,
		Submitted:  submitted,
		Completed:  completed,
		Failed:     failed,
		Pending:    submitted - completed - failed,
	}
}

// Errors
var (
	ErrPoolStopped = &PoolError{msg: "worker pool is stopped"}
)

type PoolError struct {
	msg string
}

func (e *PoolError) Error() string {
	return e.msg
}
