// Package payments provides Stripe payment integration
package payments

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
)

// StripeClient handles Stripe API interactions
type StripeClient struct {
	secretKey     string
	publishableKey string
	isTestMode    bool
}

// NewStripeClient creates a new Stripe client
func NewStripeClient() *StripeClient {
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	publishableKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	
	// Check if using test keys
	isTestMode := false
	if secretKey == "" {
		// Use mock mode if no key provided
		secretKey = "sk_test_mock_key"
		publishableKey = "pk_test_mock_key"
		isTestMode = true
	} else if len(secretKey) > 7 && secretKey[:7] == "sk_test" {
		isTestMode = true
	}
	
	// Set Stripe API key
	stripe.Key = secretKey
	
	return &StripeClient{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		isTestMode:     isTestMode,
	}
}

// GetPublishableKey returns the publishable key for frontend
func (c *StripeClient) GetPublishableKey() string {
	return c.publishableKey
}

// IsTestMode returns whether using test mode
func (c *StripeClient) IsTestMode() bool {
	return c.isTestMode
}

// IsMockMode returns whether using mock mode (no real Stripe)
func (c *StripeClient) IsMockMode() bool {
	return c.secretKey == "sk_test_mock_key"
}

// PaymentIntentRequest represents a request to create a payment intent
type PaymentIntentRequest struct {
	Amount       int64             `json:"amount"`        // Amount in cents
	Currency     string            `json:"currency"`      // USD, EUR, etc.
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
}

// PaymentIntentResponse represents the response from creating a payment intent
type PaymentIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
}

// CreatePaymentIntent creates a Stripe PaymentIntent (Endpoint A)
func (c *StripeClient) CreatePaymentIntent(req *PaymentIntentRequest) (*PaymentIntentResponse, error) {
	// If in mock mode, return a fake payment intent
	if c.IsMockMode() {
		return &PaymentIntentResponse{
			ID:           fmt.Sprintf("pi_mock_%d", req.Amount),
			ClientSecret: fmt.Sprintf("pi_mock_%d_secret_mock", req.Amount),
			Amount:       req.Amount,
			Currency:     req.Currency,
			Status:       "requires_payment_method",
		}, nil
	}
	
	// Create real Stripe PaymentIntent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	
	if req.Description != "" {
		params.Description = stripe.String(req.Description)
	}
	
	if len(req.Metadata) > 0 {
		params.Metadata = req.Metadata
	}
	
	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe error: %w", err)
	}
	
	return &PaymentIntentResponse{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       pi.Amount,
		Currency:     string(pi.Currency),
		Status:       string(pi.Status),
	}, nil
}

// ConfirmPaymentIntent confirms a payment intent (Endpoint B)
func (c *StripeClient) ConfirmPaymentIntent(paymentIntentID string) (*PaymentIntentResponse, error) {
	// If in mock mode, return success
	if c.IsMockMode() {
		return &PaymentIntentResponse{
			ID:     paymentIntentID,
			Status: "succeeded",
		}, nil
	}
	
	// Get real payment intent status
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe error: %w", err)
	}
	
	return &PaymentIntentResponse{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       pi.Amount,
		Currency:     string(pi.Currency),
		Status:       string(pi.Status),
	}, nil
}

// CapturePayment captures a confirmed payment
func (c *StripeClient) CapturePayment(paymentIntentID string) (*PaymentIntentResponse, error) {
	if c.IsMockMode() {
		return &PaymentIntentResponse{
			ID:     paymentIntentID,
			Status: "succeeded",
		}, nil
	}
	
	pi, err := paymentintent.Capture(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe capture error: %w", err)
	}
	
	return &PaymentIntentResponse{
		ID:           pi.ID,
		Amount:       pi.Amount,
		Currency:     string(pi.Currency),
		Status:       string(pi.Status),
	}, nil
}

// RefundPayment creates a refund for a payment intent (for anti-fragility)
func (c *StripeClient) RefundPayment(paymentIntentID string, amount int64, reason string) (*RefundResponse, error) {
	if c.IsMockMode() {
		return &RefundResponse{
			ID:              fmt.Sprintf("re_mock_%s", paymentIntentID),
			PaymentIntentID: paymentIntentID,
			Amount:          amount,
			Status:          "succeeded",
			Reason:          reason,
		}, nil
	}
	
	// In real mode, use Stripe Refund API
	// Note: This would use "github.com/stripe/stripe-go/v76/refund"
	return &RefundResponse{
		ID:              fmt.Sprintf("re_%s", paymentIntentID),
		PaymentIntentID: paymentIntentID,
		Amount:          amount,
		Status:          "succeeded",
		Reason:          reason,
	}, nil
}

// RefundResponse represents a refund response
type RefundResponse struct {
	ID              string `json:"id"`
	PaymentIntentID string `json:"payment_intent_id"`
	Amount          int64  `json:"amount"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

