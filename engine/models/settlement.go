// Package models defines core data structures for the Predictive Liquidity Mesh.
// Uses sync.Pool for efficient memory reuse on critical paths.
package models

import (
	"sync"
	"time"
)

// SettlementRequest represents a transaction settlement request.
// Pooled for memory efficiency on high-throughput paths.
type SettlementRequest struct {
	// Unique request identifier (ULID)
	ID string `json:"id"`

	// Source and destination node IDs
	SourceID      string `json:"source_id"`
	DestinationID string `json:"destination_id"`

	// Amount in smallest currency unit
	Amount int64 `json:"amount"`

	// Selected routing path (list of node IDs)
	Path []string `json:"path"`

	// Ed25519 signature
	Signature []byte `json:"signature"`

	// Priority level (1=highest, 5=lowest)
	Priority int `json:"priority"`

	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`

	// Metadata for additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Internal state
	retryCount int
	lastError  error
}

// Reset clears the SettlementRequest for reuse from the pool.
// This is critical for sync.Pool efficiency.
func (r *SettlementRequest) Reset() {
	r.ID = ""
	r.SourceID = ""
	r.DestinationID = ""
	r.Amount = 0
	r.Path = r.Path[:0] // Reuse backing array
	r.Signature = r.Signature[:0]
	r.Priority = 0
	r.CreatedAt = time.Time{}
	r.ProcessedAt = time.Time{}
	r.Metadata = nil
	r.retryCount = 0
	r.lastError = nil
}

// settlementRequestPool is the sync.Pool for SettlementRequest objects.
// Reduces GC pressure by reusing allocated objects.
var settlementRequestPool = sync.Pool{
	New: func() interface{} {
		return &SettlementRequest{
			Path:      make([]string, 0, 10),  // Pre-allocate for typical path length
			Signature: make([]byte, 0, 64),    // Ed25519 signature size
			Metadata:  make(map[string]interface{}),
		}
	},
}

// AcquireSettlementRequest gets a SettlementRequest from the pool.
// Always call ReleaseSettlementRequest when done.
func AcquireSettlementRequest() *SettlementRequest {
	return settlementRequestPool.Get().(*SettlementRequest)
}

// ReleaseSettlementRequest returns a SettlementRequest to the pool.
// The request is reset before being returned to the pool.
func ReleaseSettlementRequest(r *SettlementRequest) {
	if r == nil {
		return
	}
	r.Reset()
	settlementRequestPool.Put(r)
}

// SettlementResponse represents the result of a settlement request.
type SettlementResponse struct {
	// Original request ID
	RequestID string `json:"request_id"`

	// Settlement status
	Status SettlementStatus `json:"status"`

	// Ledger entry ID if successful
	LedgerEntryID string `json:"ledger_entry_id,omitempty"`

	// Actual path used (may differ if rerouted)
	ActualPath []string `json:"actual_path,omitempty"`

	// Total fee charged
	TotalFee int64 `json:"total_fee"`

	// Total latency in milliseconds
	TotalLatencyMs int64 `json:"total_latency_ms"`

	// Error message if failed
	Error string `json:"error,omitempty"`

	// Timestamp
	CompletedAt time.Time `json:"completed_at"`
}

// SettlementStatus represents the status of a settlement.
type SettlementStatus int

const (
	StatusPending SettlementStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
	StatusRerouted
)

func (s SettlementStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusProcessing:
		return "PROCESSING"
	case StatusCompleted:
		return "COMPLETED"
	case StatusFailed:
		return "FAILED"
	case StatusRerouted:
		return "REROUTED"
	default:
		return "UNKNOWN"
	}
}

// settlementResponsePool for response objects
var settlementResponsePool = sync.Pool{
	New: func() interface{} {
		return &SettlementResponse{
			ActualPath: make([]string, 0, 10),
		}
	},
}

// AcquireSettlementResponse gets a SettlementResponse from the pool.
func AcquireSettlementResponse() *SettlementResponse {
	return settlementResponsePool.Get().(*SettlementResponse)
}

// ReleaseSettlementResponse returns a SettlementResponse to the pool.
func ReleaseSettlementResponse(r *SettlementResponse) {
	if r == nil {
		return
	}
	r.RequestID = ""
	r.Status = StatusPending
	r.LedgerEntryID = ""
	r.ActualPath = r.ActualPath[:0]
	r.TotalFee = 0
	r.TotalLatencyMs = 0
	r.Error = ""
	r.CompletedAt = time.Time{}
	settlementResponsePool.Put(r)
}
