// Package fxrates provides a background worker to fetch live exchange rates from ExchangeRate-API.
// Free tier: 1,500 requests/month - perfect for development.
// API endpoint: GET https://v6.exchangerate-api.com/v6/YOUR-API-KEY/latest/USD
package fxrates

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ExchangeRateAPIResponse represents the API response structure
type ExchangeRateAPIResponse struct {
	Result             string             `json:"result"`
	Documentation      string             `json:"documentation"`
	TermsOfUse         string             `json:"terms_of_use"`
	TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
	TimeNextUpdateUnix int64              `json:"time_next_update_unix"`
	BaseCode           string             `json:"base_code"`
	ConversionRates    map[string]float64 `json:"conversion_rates"`
}

// Worker fetches FX rates and updates Neo4j country nodes
type Worker struct {
	apiKey     string
	httpClient *http.Client
	driver     neo4j.DriverWithContext
	database   string
	interval   time.Duration
	currencies []string
}

// Config configures the FX rate worker
type Config struct {
	APIKey     string
	Driver     neo4j.DriverWithContext
	Database   string
	Interval   time.Duration
	Currencies []string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	if apiKey == "" {
		apiKey = "YOUR_KEY_HERE" // Placeholder - user must set in .env
	}

	return &Config{
		APIKey:   apiKey,
		Interval: 1 * time.Hour,
	}
}

// NewWorker creates a new FX rate worker
func NewWorker(cfg *Config) *Worker {
	return &Worker{
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		driver:     cfg.Driver,
		database:   cfg.Database,
		interval:   cfg.Interval,
		currencies: cfg.Currencies,
	}
}

// Start begins the periodic FX rate fetching
func (w *Worker) Start(ctx context.Context) {
	log.Println("💱 Starting FX Rate Worker...")

	if w.apiKey == "" || w.apiKey == "YOUR_KEY_HERE" {
		log.Println("⚠️  EXCHANGE_RATE_API_KEY not set - FX worker running in dry-run mode")
		log.Println("   Get your free API key at: https://app.exchangerate-api.com/dashboard")
		return
	}

	// Initial fetch
	w.fetchAndUpdate(ctx)

	// Periodic updates
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("💱 FX Rate Worker stopped")
			return
		case <-ticker.C:
			w.fetchAndUpdate(ctx)
		}
	}
}

// fetchAndUpdate fetches rates from API and updates Neo4j
func (w *Worker) fetchAndUpdate(ctx context.Context) {
	log.Println("💱 Fetching FX rates from ExchangeRate-API...")

	rates, err := w.fetchRates(ctx)
	if err != nil {
		log.Printf("❌ Failed to fetch FX rates: %v", err)
		return
	}

	log.Printf("✅ Fetched %d exchange rates (base: USD)", len(rates))

	// Update Neo4j if driver is configured
	if w.driver != nil {
		if err := w.updateNeo4j(ctx, rates); err != nil {
			log.Printf("❌ Failed to update Neo4j with FX rates: %v", err)
		}
	}
}

// fetchRates calls the ExchangeRate-API
func (w *Worker) fetchRates(ctx context.Context) (map[string]float64, error) {
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/USD", w.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp ExchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("API error: %s", apiResp.Result)
	}

	return apiResp.ConversionRates, nil
}

// updateNeo4j updates country nodes with current FX rates
func (w *Worker) updateNeo4j(ctx context.Context, rates map[string]float64) error {
	session := w.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: w.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	// Update each country's FX rate based on their currency
	query := `
		MATCH (c:Country)
		WHERE c.currency = $currency
		SET c.fx_rate = $rate, c.fx_updated_at = datetime()
		RETURN count(c) as updated
	`

	updated := 0
	for currency, rate := range rates {
		result, err := session.Run(ctx, query, map[string]interface{}{
			"currency": currency,
			"rate":     rate,
		})
		if err != nil {
			log.Printf("⚠️  Failed to update FX rate for %s: %v", currency, err)
			continue
		}

		if result.Next(ctx) {
			record := result.Record()
			if count, ok := record.Get("updated"); ok {
				if c, ok := count.(int64); ok && c > 0 {
					updated += int(c)
				}
			}
		}
	}

	log.Printf("💱 Updated FX rates for %d countries in Neo4j", updated)
	return nil
}

// FetchOnce performs a single fetch (for testing/manual trigger)
func (w *Worker) FetchOnce(ctx context.Context) (map[string]float64, error) {
	return w.fetchRates(ctx)
}
