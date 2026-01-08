// Package payments provides mock payment processing and transaction handling
package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TransactionStatus represents the status of a payment
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusProcessing TransactionStatus = "processing"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
)

// Transaction represents a payment transaction through the mesh
type Transaction struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	Amount        float64           `json:"amount"`          // Original amount
	Currency      string            `json:"currency"`        // Source currency
	TargetCurrency string           `json:"target_currency"` // Target currency
	Route         []string          `json:"route"`           // Country codes in order
	Status        TransactionStatus `json:"status"`
	
	// Fee breakdown
	BaseFee       float64           `json:"base_fee"`        // 1.5% platform fee
	HopFees       float64           `json:"hop_fees"`        // 0.02% per hop
	HaltFines     float64           `json:"halt_fines"`      // 0.1% per halted node
	TotalFees     float64           `json:"total_fees"`
	FinalAmount   float64           `json:"final_amount"`    // Amount after fees
	AdminProfit   float64           `json:"admin_profit"`    // Total fees collected
	
	// Mesh simulation
	HopResults    []HopResult       `json:"hop_results"`     // Result of each hop
	HopsCompleted int               `json:"hops_completed"`
	FailedAt      string            `json:"failed_at,omitempty"` // Country code where failed
	
	// Timestamps
	CreatedAt     time.Time         `json:"created_at"`
	ProcessedAt   *time.Time        `json:"processed_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	
	// Mock payment details
	CardLast4     string            `json:"card_last4,omitempty"`
	PaymentMethod string            `json:"payment_method"`
}

// HopResult represents the result of a single hop in the mesh
type HopResult struct {
	FromCountry   string    `json:"from_country"`
	ToCountry     string    `json:"to_country"`
	Success       bool      `json:"success"`
	Latency       int64     `json:"latency_ms"`      // Simulated latency
	FXRate        float64   `json:"fx_rate"`         // Exchange rate used
	AmountIn      float64   `json:"amount_in"`       // Amount entering this hop
	AmountOut     float64   `json:"amount_out"`      // Amount after hop fee
	HopFee        float64   `json:"hop_fee"`         // Fee for this hop
	Timestamp     time.Time `json:"timestamp"`
	Error         string    `json:"error,omitempty"` // Error message if failed
}

// FeeConfig holds fee configuration
type FeeConfig struct {
	BaseFeePercent    float64 // Default 1.5% (0.015)
	HopFeePercent     float64 // Default 0.02% (0.0002)
	HaltFinePercent   float64 // Default 0.1% (0.001)
}

// DefaultFeeConfig returns the default fee configuration
func DefaultFeeConfig() FeeConfig {
	return FeeConfig{
		BaseFeePercent:  0.015,  // 1.5%
		HopFeePercent:   0.0002, // 0.02%
		HaltFinePercent: 0.001,  // 0.1%
	}
}

// TransactionStore stores transactions in memory (for demo)
type TransactionStore struct {
	mu              sync.RWMutex
	transactions    map[string]*Transaction
	userTxns        map[string][]string // userID -> transaction IDs
	feeConfig       FeeConfig
	processingLocks map[string]*sync.Mutex // Per-transaction locks to prevent concurrent processing
	
	// Callbacks
	onCredibilityUpdate func(countryCode string, success bool)
}

// NewTransactionStore creates a new transaction store
func NewTransactionStore() *TransactionStore {
	return &TransactionStore{
		transactions:    make(map[string]*Transaction),
		userTxns:        make(map[string][]string),
		feeConfig:       DefaultFeeConfig(),
		processingLocks: make(map[string]*sync.Mutex),
	}
}

// SetCredibilityCallback sets the callback for credibility updates
func (s *TransactionStore) SetCredibilityCallback(cb func(countryCode string, success bool)) {
	s.onCredibilityUpdate = cb
}

// GetProcessingLock returns a per-transaction mutex to prevent concurrent processing
// This prevents race conditions during anti-fragility retry logic
func (s *TransactionStore) GetProcessingLock(txnID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.processingLocks[txnID]; !exists {
		s.processingLocks[txnID] = &sync.Mutex{}
	}
	return s.processingLocks[txnID]
}

// generateTxID generates a unique transaction ID
func generateTxID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "txn_" + hex.EncodeToString(bytes)
}

// CreateTransaction creates a new pending transaction
func (s *TransactionStore) CreateTransaction(userID string, amount float64, currency, targetCurrency string, route []string, haltedNodes map[string]bool) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(route) < 2 {
		return nil, fmt.Errorf("route must have at least 2 countries")
	}

	hopCount := len(route) - 1
	
	// Calculate fees
	baseFee := amount * s.feeConfig.BaseFeePercent
	hopFees := amount * s.feeConfig.HopFeePercent * float64(hopCount)
	
	// Count halted nodes in route
	haltFines := 0.0
	for _, code := range route {
		if haltedNodes[code] {
			haltFines += amount * s.feeConfig.HaltFinePercent
		}
	}
	
	totalFees := baseFee + hopFees + haltFines
	finalAmount := amount - totalFees

	// Generate mock card number
	cardLast4 := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)

	txn := &Transaction{
		ID:             generateTxID(),
		UserID:         userID,
		Amount:         amount,
		Currency:       currency,
		TargetCurrency: targetCurrency,
		Route:          route,
		Status:         StatusPending,
		BaseFee:        baseFee,
		HopFees:        hopFees,
		HaltFines:      haltFines,
		TotalFees:      totalFees,
		FinalAmount:    finalAmount,
		AdminProfit:    totalFees,
		HopResults:     make([]HopResult, 0),
		CreatedAt:      time.Now(),
		CardLast4:      cardLast4,
		PaymentMethod:  "mock_card",
	}

	s.transactions[txn.ID] = txn
	s.userTxns[userID] = append(s.userTxns[userID], txn.ID)

	return txn, nil
}

// ProcessTransaction simulates the mesh payment flow
func (s *TransactionStore) ProcessTransaction(ctx context.Context, txnID string, fxRates map[string]float64, failureChance float64) error {
	s.mu.Lock()
	txn, ok := s.transactions[txnID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txnID)
	}
	
	if txn.Status != StatusPending {
		s.mu.Unlock()
		return fmt.Errorf("transaction already processed")
	}
	
	txn.Status = StatusProcessing
	now := time.Now()
	txn.ProcessedAt = &now
	s.mu.Unlock()

	// Simulate mesh hops
	currentAmount := txn.Amount - txn.TotalFees
	hopFeePerHop := txn.Amount * s.feeConfig.HopFeePercent

	for i := 0; i < len(txn.Route)-1; i++ {
		select {
		case <-ctx.Done():
			s.setTransactionFailed(txnID, txn.Route[i], "context cancelled")
			return ctx.Err()
		default:
		}

		fromCountry := txn.Route[i]
		toCountry := txn.Route[i+1]

		// Simulate latency (50-200ms per hop)
		latency := int64(50 + (time.Now().UnixNano() % 150))
		time.Sleep(time.Duration(latency) * time.Millisecond)

		// Get FX rate (default to 1 if not available)
		fxRate := 1.0
		if rate, ok := fxRates[toCountry]; ok {
			fxRate = rate
		}

		// Simulate random failure (for demo purposes)
		failed := false
		errorMsg := ""
		if failureChance > 0 {
			// Simple random failure simulation
			if time.Now().UnixNano()%100 < int64(failureChance*100) {
				failed = true
				errorMsg = "node timeout"
			}
		}

		amountOut := currentAmount - hopFeePerHop
		if failed {
			amountOut = 0
		}

		hopResult := HopResult{
			FromCountry: fromCountry,
			ToCountry:   toCountry,
			Success:     !failed,
			Latency:     latency,
			FXRate:      fxRate,
			AmountIn:    currentAmount,
			AmountOut:   amountOut,
			HopFee:      hopFeePerHop,
			Timestamp:   time.Now(),
			Error:       errorMsg,
		}

		s.mu.Lock()
		txn.HopResults = append(txn.HopResults, hopResult)
		txn.HopsCompleted = i + 1
		s.mu.Unlock()

		// Update credibility
		if s.onCredibilityUpdate != nil {
			s.onCredibilityUpdate(toCountry, !failed)
		}

		if failed {
			s.setTransactionFailed(txnID, toCountry, errorMsg)
			return fmt.Errorf("payment failed at %s: %s", toCountry, errorMsg)
		}

		currentAmount = amountOut
	}

	// Success!
	s.mu.Lock()
	txn.Status = StatusSuccess
	now = time.Now()
	txn.CompletedAt = &now
	txn.FinalAmount = currentAmount
	s.mu.Unlock()

	return nil
}

// setTransactionFailed marks a transaction as failed
func (s *TransactionStore) setTransactionFailed(txnID, failedAt, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if txn, ok := s.transactions[txnID]; ok {
		txn.Status = StatusFailed
		txn.FailedAt = failedAt
		now := time.Now()
		txn.CompletedAt = &now
	}
}

// GetTransaction returns a transaction by ID
func (s *TransactionStore) GetTransaction(txnID string) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	txn, ok := s.transactions[txnID]
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}
	return txn, nil
}

// GetUserTransactions returns all transactions for a user
func (s *TransactionStore) GetUserTransactions(userID string) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	txnIDs := s.userTxns[userID]
	result := make([]*Transaction, 0, len(txnIDs))
	
	for _, id := range txnIDs {
		if txn, ok := s.transactions[id]; ok {
			result = append(result, txn)
		}
	}
	
	return result
}

// GetAdminStats returns admin profit statistics
func (s *TransactionStore) GetAdminStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	totalProfit := 0.0
	successCount := 0
	failedCount := 0
	pendingCount := 0
	totalVolume := 0.0
	
	for _, txn := range s.transactions {
		totalVolume += txn.Amount
		switch txn.Status {
		case StatusSuccess:
			successCount++
			totalProfit += txn.AdminProfit
		case StatusFailed:
			failedCount++
			// Still collect partial fees on failed transactions
			totalProfit += txn.BaseFee
		case StatusPending, StatusProcessing:
			pendingCount++
		}
	}
	
	return map[string]interface{}{
		"total_profit":    totalProfit,
		"total_volume":    totalVolume,
		"success_count":   successCount,
		"failed_count":    failedCount,
		"pending_count":   pendingCount,
		"total_transactions": len(s.transactions),
	}
}

// GetAllTransactions returns all transactions (for admin)
func (s *TransactionStore) GetAllTransactions() []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*Transaction, 0, len(s.transactions))
	for _, txn := range s.transactions {
		result = append(result, txn)
	}
	return result
}

// ProcessTransactionWithRoute processes a transaction using a specific route (for anti-fragility retries)
func (s *TransactionStore) ProcessTransactionWithRoute(ctx context.Context, txnID string, route []string, fxRates map[string]float64, failureChance float64) error {
	s.mu.Lock()
	txn, ok := s.transactions[txnID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("transaction not found: %s", txnID)
	}
	
	if txn.Status != StatusPending {
		s.mu.Unlock()
		return fmt.Errorf("transaction not in pending state")
	}
	
	// Update route for this attempt
	txn.Route = route
	txn.Status = StatusProcessing
	now := time.Now()
	txn.ProcessedAt = &now
	s.mu.Unlock()

	// Simulate mesh hops with the new route
	currentAmount := txn.Amount - txn.TotalFees
	hopFeePerHop := txn.Amount * s.feeConfig.HopFeePercent

	for i := 0; i < len(route)-1; i++ {
		select {
		case <-ctx.Done():
			s.setTransactionFailed(txnID, route[i], "context cancelled")
			return ctx.Err()
		default:
		}

		fromCountry := route[i]
		toCountry := route[i+1]

		// Simulate latency (50-200ms per hop)
		latency := int64(50 + (time.Now().UnixNano() % 150))
		time.Sleep(time.Duration(latency) * time.Millisecond)

		// Get FX rate
		fxRate := 1.0
		if rate, ok := fxRates[toCountry]; ok {
			fxRate = rate
		}

		// Simulate random failure
		failed := false
		errorMsg := ""
		if failureChance > 0 {
			if time.Now().UnixNano()%100 < int64(failureChance*100) {
				failed = true
				errorMsg = "node timeout"
			}
		}

		amountOut := currentAmount - hopFeePerHop
		if failed {
			amountOut = 0
		}

		hopResult := HopResult{
			FromCountry: fromCountry,
			ToCountry:   toCountry,
			Success:     !failed,
			Latency:     latency,
			FXRate:      fxRate,
			AmountIn:    currentAmount,
			AmountOut:   amountOut,
			HopFee:      hopFeePerHop,
			Timestamp:   time.Now(),
			Error:       errorMsg,
		}

		s.mu.Lock()
		txn.HopResults = append(txn.HopResults, hopResult)
		txn.HopsCompleted = i + 1
		s.mu.Unlock()

		if s.onCredibilityUpdate != nil {
			s.onCredibilityUpdate(toCountry, !failed)
		}

		if failed {
			s.setTransactionFailed(txnID, toCountry, errorMsg)
			return fmt.Errorf("payment failed at %s: %s", toCountry, errorMsg)
		}

		currentAmount = amountOut
	}

	// Success!
	s.mu.Lock()
	txn.Status = StatusSuccess
	now = time.Now()
	txn.CompletedAt = &now
	txn.FinalAmount = currentAmount
	s.mu.Unlock()

	return nil
}

// ResetTransactionForRetry resets a transaction to pending for retry attempts
func (s *TransactionStore) ResetTransactionForRetry(txnID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if txn, ok := s.transactions[txnID]; ok {
		txn.Status = StatusPending
		txn.HopResults = make([]HopResult, 0)
		txn.HopsCompleted = 0
		txn.FailedAt = ""
		txn.ProcessedAt = nil
		txn.CompletedAt = nil
	}
}

// MarkAsRefunded marks a transaction as refunded
func (s *TransactionStore) MarkAsRefunded(txnID string, refundID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if txn, ok := s.transactions[txnID]; ok {
		txn.Status = StatusFailed // Keep as failed but mark refund
		txn.PaymentMethod = "refunded:" + refundID
	}
}


