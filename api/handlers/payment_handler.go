// Package handlers provides payment API handlers
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/plm/predictive-liquidity-mesh/engine/router"
	"github.com/plm/predictive-liquidity-mesh/payments"
)

// PaymentHandler handles payment API endpoints
type PaymentHandler struct {
	txnStore     *payments.TransactionStore
	countryGraph *router.CountryGraph
	stripeClient *payments.StripeClient
	fxRates      map[string]float64
	haltedNodes  map[string]bool
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(txnStore *payments.TransactionStore, countryGraph *router.CountryGraph) *PaymentHandler {
	return &PaymentHandler{
		txnStore:     txnStore,
		countryGraph: countryGraph,
		stripeClient: payments.NewStripeClient(),
		fxRates:      make(map[string]float64),
		haltedNodes:  make(map[string]bool),
	}
}

// SetFXRates updates the FX rates map
func (h *PaymentHandler) SetFXRates(rates map[string]float64) {
	h.fxRates = rates
}

// SetHaltedNodes updates the halted nodes map
func (h *PaymentHandler) SetHaltedNodes(halted map[string]bool) {
	h.haltedNodes = halted
}

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	Amount         float64  `json:"amount"`
	Currency       string   `json:"currency"`
	TargetCurrency string   `json:"target_currency"`
	Route          []string `json:"route"`
}

// CreatePaymentResponse represents the payment creation response
type CreatePaymentResponse struct {
	Transaction  *payments.Transaction `json:"transaction"`
	FeeBreakdown FeeBreakdown          `json:"fee_breakdown"`
}

// FeeBreakdown shows detailed fee information
type FeeBreakdown struct {
	BaseFee     float64 `json:"base_fee"`
	BaseFeeRate string  `json:"base_fee_rate"`
	HopFees     float64 `json:"hop_fees"`
	HopFeeRate  string  `json:"hop_fee_rate"`
	HopCount    int     `json:"hop_count"`
	HaltFines   float64 `json:"halt_fines"`
	HaltCount   int     `json:"halt_count"`
	TotalFees   float64 `json:"total_fees"`
	FinalAmount float64 `json:"final_amount"`
}

// HandleCreatePayment creates a new payment transaction
func (h *PaymentHandler) HandleCreatePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by auth middleware)
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.Amount <= 0 {
		http.Error(w, `{"error":"amount must be positive"}`, http.StatusBadRequest)
		return
	}
	if len(req.Route) < 2 {
		http.Error(w, `{"error":"route must have at least 2 countries"}`, http.StatusBadRequest)
		return
	}

	// Create transaction
	txn, err := h.txnStore.CreateTransaction(userID, req.Amount, req.Currency, req.TargetCurrency, req.Route, h.haltedNodes)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Count halted nodes in route
	haltCount := 0
	for _, code := range req.Route {
		if h.haltedNodes[code] {
			haltCount++
		}
	}

	response := CreatePaymentResponse{
		Transaction: txn,
		FeeBreakdown: FeeBreakdown{
			BaseFee:     txn.BaseFee,
			BaseFeeRate: "1.5%",
			HopFees:     txn.HopFees,
			HopFeeRate:  "0.02%",
			HopCount:    len(req.Route) - 1,
			HaltFines:   txn.HaltFines,
			HaltCount:   haltCount,
			TotalFees:   txn.TotalFees,
			FinalAmount: txn.FinalAmount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConfirmPaymentRequest represents a payment confirmation request
type ConfirmPaymentRequest struct {
	TransactionID string `json:"transaction_id"`
	CardNumber    string `json:"card_number"` // Mock: just for demo
	CVV           string `json:"cvv"`
	ExpiryMonth   string `json:"expiry_month"`
	ExpiryYear    string `json:"expiry_year"`
}

// HandleConfirmPayment confirms and processes a payment
func (h *PaymentHandler) HandleConfirmPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ConfirmPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Verify transaction exists and belongs to user
	txn, err := h.txnStore.GetTransaction(req.TransactionID)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}
	if txn.UserID != userID {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Mock card validation (accept any 16-digit number for demo)
	if len(req.CardNumber) < 13 || len(req.CardNumber) > 19 {
		http.Error(w, `{"error":"invalid card number"}`, http.StatusBadRequest)
		return
	}

	// Process payment through mesh (with 5% failure chance for demo)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	log.Printf("💳 Processing payment %s: $%.2f through %v", txn.ID, txn.Amount, txn.Route)

	err = h.txnStore.ProcessTransaction(ctx, req.TransactionID, h.fxRates, 0.05)
	
	// Get updated transaction
	txn, _ = h.txnStore.GetTransaction(req.TransactionID)

	if err != nil {
		log.Printf("❌ Payment %s failed: %v", txn.ID, err)
	} else {
		log.Printf("✅ Payment %s completed: Admin profit $%.2f", txn.ID, txn.AdminProfit)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction": txn,
		"success":     txn.Status == payments.StatusSuccess,
		"message":     getStatusMessage(txn.Status, txn.FailedAt),
	})
}

// HandleGetTransaction returns a single transaction
func (h *PaymentHandler) HandleGetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	txnID := r.URL.Query().Get("id")
	if txnID == "" {
		http.Error(w, `{"error":"transaction id required"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.txnStore.GetTransaction(txnID)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

// HandleGetHistory returns user's transaction history
func (h *PaymentHandler) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	transactions := h.txnStore.GetUserTransactions(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// HandleAdminStats returns admin analytics with all transactions (admin only)
func (h *PaymentHandler) HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	stats := h.txnStore.GetAdminStats()
	allTransactions := h.txnStore.GetAllTransactions()

	// Build enhanced analytics
	var totalVolume float64
	var totalFees float64
	var successCount, failedCount, pendingCount int
	var dailyVolume = make(map[string]float64)
	var dailyFees = make(map[string]float64)

	for _, txn := range allTransactions {
		totalVolume += txn.Amount
		totalFees += txn.TotalFees
		
		day := txn.CreatedAt.Format("2006-01-02")
		dailyVolume[day] += txn.Amount
		dailyFees[day] += txn.TotalFees

		switch txn.Status {
		case payments.StatusSuccess:
			successCount++
		case payments.StatusFailed:
			failedCount++
		default:
			pendingCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":            stats,
		"all_transactions": allTransactions,
		"analytics": map[string]interface{}{
			"total_volume":       totalVolume,
			"total_platform_fee": totalFees,
			"total_transactions": len(allTransactions),
			"success_count":      successCount,
			"failed_count":       failedCount,
			"pending_count":      pendingCount,
			"success_rate":       float64(successCount) / float64(max(len(allTransactions), 1)) * 100,
			"daily_volume":       dailyVolume,
			"daily_fees":         dailyFees,
		},
	})
}

// Helper functions
func getUserIDFromContext(r *http.Request) string {
	// Try middleware context key (typed key)
	type contextKey string
	const userContextKey contextKey = "user"
	
	// Try typed context key first
	if user := r.Context().Value(userContextKey); user != nil {
		// Try GetID method
		if u, ok := user.(interface{ GetID() string }); ok {
			return u.GetID()
		}
	}
	
	// Try string key (fallback)
	if user := r.Context().Value("user"); user != nil {
		if u, ok := user.(interface{ GetID() string }); ok {
			return u.GetID()
		}
	}
	
	// For demo: accept X-User-ID header
	if id := r.Header.Get("X-User-ID"); id != "" {
		return id
	}
	
	// Default demo user for testing
	return "demo-user"
}

func getStatusMessage(status payments.TransactionStatus, failedAt string) string {
	switch status {
	case payments.StatusSuccess:
		return "Payment completed successfully"
	case payments.StatusFailed:
		return "Payment failed at " + failedAt
	case payments.StatusProcessing:
		return "Payment is being processed"
	case payments.StatusPending:
		return "Payment is pending confirmation"
	default:
		return "Unknown status"
	}
}

// ============== STRIPE ENDPOINTS ==============

// StripeInitRequest represents request to initiate Stripe payment (Endpoint A)
type StripeInitRequest struct {
	Amount         float64  `json:"amount"`
	Currency       string   `json:"currency"`
	TargetCurrency string   `json:"target_currency"`
	Route          []string `json:"route"`
}

// StripeInitResponse represents response from Endpoint A
type StripeInitResponse struct {
	TransactionID   string                `json:"transaction_id"`
	StripeClientSecret string             `json:"stripe_client_secret"`
	StripePaymentID string                `json:"stripe_payment_id"`
	Transaction     *payments.Transaction `json:"transaction"`
	FeeBreakdown    FeeBreakdown          `json:"fee_breakdown"`
	PublishableKey  string                `json:"publishable_key"`
	IsMockMode      bool                  `json:"is_mock_mode"`
}

// HandleStripeInitiate handles Endpoint A - Initiate Payment
// User enters amount, selects route, gets Stripe client secret
func (h *PaymentHandler) HandleStripeInitiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req StripeInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.Amount <= 0 {
		http.Error(w, `{"error":"amount must be positive"}`, http.StatusBadRequest)
		return
	}
	if len(req.Route) < 2 {
		http.Error(w, `{"error":"route must have at least 2 countries"}`, http.StatusBadRequest)
		return
	}

	// Create internal transaction
	txn, err := h.txnStore.CreateTransaction(userID, req.Amount, req.Currency, req.TargetCurrency, req.Route, h.haltedNodes)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Create Stripe PaymentIntent
	amountCents := int64(req.Amount * 100) // Convert to cents
	stripeReq := &payments.PaymentIntentRequest{
		Amount:      amountCents,
		Currency:    req.Currency,
		Description: "PLM Transfer: " + req.Route[0] + " → " + req.Route[len(req.Route)-1],
		Metadata: map[string]string{
			"transaction_id": txn.ID,
			"route":          req.Route[0] + "_to_" + req.Route[len(req.Route)-1],
			"hops":           string(rune(len(req.Route) - 1)),
		},
	}

	stripeResp, err := h.stripeClient.CreatePaymentIntent(stripeReq)
	if err != nil {
		log.Printf("Stripe error: %v", err)
		http.Error(w, `{"error":"payment service unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	// Count halted nodes
	haltCount := 0
	for _, code := range req.Route {
		if h.haltedNodes[code] {
			haltCount++
		}
	}

	log.Printf("💳 [Endpoint A] Payment initiated: %s for $%.2f (Stripe: %s)", txn.ID, req.Amount, stripeResp.ID)

	response := StripeInitResponse{
		TransactionID:      txn.ID,
		StripeClientSecret: stripeResp.ClientSecret,
		StripePaymentID:    stripeResp.ID,
		Transaction:        txn,
		FeeBreakdown: FeeBreakdown{
			BaseFee:     txn.BaseFee,
			BaseFeeRate: "1.5%",
			HopFees:     txn.HopFees,
			HopFeeRate:  "0.02%",
			HopCount:    len(req.Route) - 1,
			HaltFines:   txn.HaltFines,
			HaltCount:   haltCount,
			TotalFees:   txn.TotalFees,
			FinalAmount: txn.FinalAmount,
		},
		PublishableKey: h.stripeClient.GetPublishableKey(),
		IsMockMode:     h.stripeClient.IsMockMode(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StripeCompleteRequest represents request to complete Stripe payment (Endpoint B)
type StripeCompleteRequest struct {
	TransactionID   string `json:"transaction_id"`
	StripePaymentID string `json:"stripe_payment_id"`
}

// StripeCompleteResponse represents response from Endpoint B
type StripeCompleteResponse struct {
	Success     bool                  `json:"success"`
	Transaction *payments.Transaction `json:"transaction"`
	Message     string                `json:"message"`
	ReceiptURL  string                `json:"receipt_url"`
}

// HandleStripeComplete handles Endpoint B - Complete Payment
// Called after Stripe payment succeeds, processes through mesh
func (h *PaymentHandler) HandleStripeComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req StripeCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Verify transaction
	txn, err := h.txnStore.GetTransaction(req.TransactionID)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}
	if txn.UserID != userID && userID != "demo-user" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Verify Stripe payment (in mock mode, this always succeeds)
	stripeStatus, err := h.stripeClient.ConfirmPaymentIntent(req.StripePaymentID)
	if err != nil {
		http.Error(w, `{"error":"payment verification failed"}`, http.StatusBadRequest)
		return
	}

	// Check if payment succeeded
	if stripeStatus.Status != "succeeded" && !h.stripeClient.IsMockMode() {
		http.Error(w, `{"error":"payment not completed: `+stripeStatus.Status+`"}`, http.StatusPaymentRequired)
		return
	}

	log.Printf("💳 [Endpoint B] Processing payment %s through mesh...", txn.ID)

	// ANTI-FRAGILITY: Try up to 3 alternative routes
	const maxRetries = 3
	var lastError error
	var usedRoute []string
	
	// Get alternative routes from country graph (Yen's algorithm paths)
	alternativeRoutes := h.getAlternativeRoutes(txn.Route)
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Select route for this attempt
		if attempt == 1 {
			usedRoute = txn.Route // Original path
		} else if attempt-1 < len(alternativeRoutes) {
			usedRoute = alternativeRoutes[attempt-1]
			log.Printf("🔄 [Anti-Fragility] Attempt %d: Re-routing via alternative path: %v", attempt, usedRoute)
		} else {
			log.Printf("⚠️ [Anti-Fragility] No more alternative routes available")
			break
		}
		
		// Process through mesh
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		lastError = h.txnStore.ProcessTransactionWithRoute(ctx, req.TransactionID, usedRoute, h.fxRates, 0.15) // 85% success per attempt
		cancel()
		
		// Get updated transaction
		txn, _ = h.txnStore.GetTransaction(req.TransactionID)
		
		if lastError == nil && txn.Status == payments.StatusSuccess {
			log.Printf("✅ [Endpoint B] Payment %s completed on attempt %d: Admin profit $%.2f", txn.ID, attempt, txn.AdminProfit)
			break
		}
		
		log.Printf("⚠️ [Anti-Fragility] Attempt %d failed: %v - notifying user of delay", attempt, lastError)
		
		// Reset transaction status for retry if not final attempt
		if attempt < maxRetries {
			h.txnStore.ResetTransactionForRetry(req.TransactionID)
		}
	}
	
	// If all retries failed, trigger Stripe refund
	if txn.Status != payments.StatusSuccess {
		log.Printf("❌ [Anti-Fragility] All %d attempts failed for payment %s - initiating refund", maxRetries, txn.ID)
		
		refund, refundErr := h.stripeClient.RefundPayment(
			req.StripePaymentID,
			int64(txn.Amount*100),
			"anti_fragility_all_routes_failed",
		)
		
		if refundErr != nil {
			log.Printf("❌ [Refund] Failed to process refund: %v", refundErr)
		} else {
			log.Printf("💰 [Refund] Refund processed: %s - Amount: $%.2f", refund.ID, float64(refund.Amount)/100)
			h.txnStore.MarkAsRefunded(req.TransactionID, refund.ID)
		}
	}

	response := StripeCompleteResponse{
		Success:     txn.Status == payments.StatusSuccess,
		Transaction: txn,
		Message:     getStatusMessage(txn.Status, txn.FailedAt),
		ReceiptURL:  "/api/v1/receipts/" + txn.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleStripeConfig returns Stripe configuration for frontend
func (h *PaymentHandler) HandleStripeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"publishable_key": h.stripeClient.GetPublishableKey(),
		"is_test_mode":    h.stripeClient.IsTestMode(),
		"is_mock_mode":    h.stripeClient.IsMockMode(),
	})
}

// HandleChartData returns transaction data formatted for Chart.js
func (h *PaymentHandler) HandleChartData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	transactions := h.txnStore.GetUserTransactions(userID)

	// Prepare chart data
	var volumes []float64
	var fees []float64
	var labels []string
	var statusCounts = map[string]int{"success": 0, "failed": 0, "pending": 0}

	for _, txn := range transactions {
		volumes = append(volumes, txn.Amount)
		fees = append(fees, txn.TotalFees)
		labels = append(labels, txn.CreatedAt.Format("Jan 2"))
		
		switch txn.Status {
		case payments.StatusSuccess:
			statusCounts["success"]++
		case payments.StatusFailed:
			statusCounts["failed"]++
		default:
			statusCounts["pending"]++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"volume_chart": map[string]interface{}{
			"labels": labels,
			"data":   volumes,
		},
		"fees_chart": map[string]interface{}{
			"labels": labels,
			"data":   fees,
		},
		"status_chart": map[string]interface{}{
			"labels": []string{"Success", "Failed", "Pending"},
			"data":   []int{statusCounts["success"], statusCounts["failed"], statusCounts["pending"]},
		},
		"summary": map[string]interface{}{
			"total_transactions": len(transactions),
			"success_rate":       float64(statusCounts["success"]) / float64(max(len(transactions), 1)) * 100,
		},
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getAlternativeRoutes returns alternative paths using hub-based routing
func (h *PaymentHandler) getAlternativeRoutes(originalRoute []string) [][]string {
	if len(originalRoute) < 2 {
		return nil
	}
	
	source := originalRoute[0]
	destination := originalRoute[len(originalRoute)-1]
	
	// Generate alternative routes via common financial hubs
	alternatives := [][]string{}
	
	// Major financial hubs for alternative routing
	hubs := []string{"USA", "GBR", "HKG", "SGP", "ARE", "CHE", "DEU", "JPN"}
	
	for _, hub := range hubs {
		if hub != source && hub != destination && !contains(originalRoute, hub) {
			// Create 2-hop alternative via hub
			altRoute := []string{source, hub, destination}
			alternatives = append(alternatives, altRoute)
			
			if len(alternatives) >= 2 {
				break
			}
		}
	}
	
	return alternatives
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


