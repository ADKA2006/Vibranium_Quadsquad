// Package neo4j provides bootstrap data for top 50 GDP countries with credibility metrics.
package neo4j

import (
	"context"
	"fmt"
	"log"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Country represents a country node with credibility metrics
type Country struct {
	Code           string  `json:"code"`            // ISO 3166-1 alpha-3
	Name           string  `json:"name"`
	Currency       string  `json:"currency"`        // ISO 4217
	BaseCredibility float64 `json:"base_credibility"` // Fixed at 0.85
	SuccessRate    float64 `json:"success_rate"`     // Based on economic stability
	GDPRank        int     `json:"gdp_rank"`
	FXRate         float64 `json:"fx_rate,omitempty"` // Rate to USD, updated by worker
}

// Top50GDPCountries returns hardcoded data for the world's top 50 economies by GDP
// BaseCredibility is fixed at 0.85, SuccessRate based on World Bank/IMF economic stability data
// FXRate is USD exchange rate (amount of currency per 1 USD) - demo values based on early 2026 rates
var Top50GDPCountries = []Country{
	{Code: "USA", Name: "United States", Currency: "USD", BaseCredibility: 0.85, SuccessRate: 0.95, GDPRank: 1, FXRate: 1.0000},
	{Code: "CHN", Name: "China", Currency: "CNY", BaseCredibility: 0.85, SuccessRate: 0.92, GDPRank: 2, FXRate: 7.2450},
	{Code: "DEU", Name: "Germany", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.94, GDPRank: 3, FXRate: 0.9234},
	{Code: "JPN", Name: "Japan", Currency: "JPY", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 4, FXRate: 156.85},
	{Code: "IND", Name: "India", Currency: "INR", BaseCredibility: 0.85, SuccessRate: 0.88, GDPRank: 5, FXRate: 83.42},
	{Code: "GBR", Name: "United Kingdom", Currency: "GBP", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 6, FXRate: 0.7923},
	{Code: "FRA", Name: "France", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.92, GDPRank: 7, FXRate: 0.9234},
	{Code: "ITA", Name: "Italy", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.89, GDPRank: 8, FXRate: 0.9234},
	{Code: "BRA", Name: "Brazil", Currency: "BRL", BaseCredibility: 0.85, SuccessRate: 0.84, GDPRank: 9, FXRate: 4.9867},
	{Code: "CAN", Name: "Canada", Currency: "CAD", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 10, FXRate: 1.3546},
	{Code: "RUS", Name: "Russia", Currency: "RUB", BaseCredibility: 0.85, SuccessRate: 0.78, GDPRank: 11, FXRate: 92.45},
	{Code: "KOR", Name: "South Korea", Currency: "KRW", BaseCredibility: 0.85, SuccessRate: 0.91, GDPRank: 12, FXRate: 1342.50},
	{Code: "AUS", Name: "Australia", Currency: "AUD", BaseCredibility: 0.85, SuccessRate: 0.92, GDPRank: 13, FXRate: 1.5324},
	{Code: "MEX", Name: "Mexico", Currency: "MXN", BaseCredibility: 0.85, SuccessRate: 0.83, GDPRank: 14, FXRate: 17.2340},
	{Code: "ESP", Name: "Spain", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.88, GDPRank: 15, FXRate: 0.9234},
	{Code: "IDN", Name: "Indonesia", Currency: "IDR", BaseCredibility: 0.85, SuccessRate: 0.85, GDPRank: 16, FXRate: 15765.0},
	{Code: "NLD", Name: "Netherlands", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 17, FXRate: 0.9234},
	{Code: "SAU", Name: "Saudi Arabia", Currency: "SAR", BaseCredibility: 0.85, SuccessRate: 0.87, GDPRank: 18, FXRate: 3.7500},
	{Code: "TUR", Name: "Turkey", Currency: "TRY", BaseCredibility: 0.85, SuccessRate: 0.76, GDPRank: 19, FXRate: 32.4560},
	{Code: "CHE", Name: "Switzerland", Currency: "CHF", BaseCredibility: 0.85, SuccessRate: 0.96, GDPRank: 20, FXRate: 0.8765},
	{Code: "POL", Name: "Poland", Currency: "PLN", BaseCredibility: 0.85, SuccessRate: 0.87, GDPRank: 21, FXRate: 4.0234},
	{Code: "TWN", Name: "Taiwan", Currency: "TWD", BaseCredibility: 0.85, SuccessRate: 0.90, GDPRank: 22, FXRate: 31.8540},
	{Code: "BEL", Name: "Belgium", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.91, GDPRank: 23, FXRate: 0.9234},
	{Code: "SWE", Name: "Sweden", Currency: "SEK", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 24, FXRate: 10.6780},
	{Code: "IRL", Name: "Ireland", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.91, GDPRank: 25, FXRate: 0.9234},
	{Code: "AUT", Name: "Austria", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.92, GDPRank: 26, FXRate: 0.9234},
	{Code: "THA", Name: "Thailand", Currency: "THB", BaseCredibility: 0.85, SuccessRate: 0.84, GDPRank: 27, FXRate: 35.4560},
	{Code: "ISR", Name: "Israel", Currency: "ILS", BaseCredibility: 0.85, SuccessRate: 0.89, GDPRank: 28, FXRate: 3.6540},
	{Code: "NGA", Name: "Nigeria", Currency: "NGN", BaseCredibility: 0.85, SuccessRate: 0.72, GDPRank: 29, FXRate: 1456.78},
	{Code: "ARE", Name: "United Arab Emirates", Currency: "AED", BaseCredibility: 0.85, SuccessRate: 0.90, GDPRank: 30, FXRate: 3.6725},
	{Code: "ARG", Name: "Argentina", Currency: "ARS", BaseCredibility: 0.85, SuccessRate: 0.68, GDPRank: 31, FXRate: 867.45},
	{Code: "NOR", Name: "Norway", Currency: "NOK", BaseCredibility: 0.85, SuccessRate: 0.94, GDPRank: 32, FXRate: 10.8934},
	{Code: "EGY", Name: "Egypt", Currency: "EGP", BaseCredibility: 0.85, SuccessRate: 0.74, GDPRank: 33, FXRate: 50.7650},
	{Code: "VNM", Name: "Vietnam", Currency: "VND", BaseCredibility: 0.85, SuccessRate: 0.82, GDPRank: 34, FXRate: 24865.0},
	{Code: "BGD", Name: "Bangladesh", Currency: "BDT", BaseCredibility: 0.85, SuccessRate: 0.79, GDPRank: 35, FXRate: 110.45},
	{Code: "ZAF", Name: "South Africa", Currency: "ZAR", BaseCredibility: 0.85, SuccessRate: 0.77, GDPRank: 36, FXRate: 18.7654},
	{Code: "PHL", Name: "Philippines", Currency: "PHP", BaseCredibility: 0.85, SuccessRate: 0.81, GDPRank: 37, FXRate: 55.8760},
	{Code: "DNK", Name: "Denmark", Currency: "DKK", BaseCredibility: 0.85, SuccessRate: 0.93, GDPRank: 38, FXRate: 6.8976},
	{Code: "MYS", Name: "Malaysia", Currency: "MYR", BaseCredibility: 0.85, SuccessRate: 0.86, GDPRank: 39, FXRate: 4.4567},
	{Code: "SGP", Name: "Singapore", Currency: "SGD", BaseCredibility: 0.85, SuccessRate: 0.95, GDPRank: 40, FXRate: 1.3456},
	{Code: "HKG", Name: "Hong Kong", Currency: "HKD", BaseCredibility: 0.85, SuccessRate: 0.91, GDPRank: 41, FXRate: 7.8123},
	{Code: "PAK", Name: "Pakistan", Currency: "PKR", BaseCredibility: 0.85, SuccessRate: 0.70, GDPRank: 42, FXRate: 278.65},
	{Code: "CHL", Name: "Chile", Currency: "CLP", BaseCredibility: 0.85, SuccessRate: 0.85, GDPRank: 43, FXRate: 934.56},
	{Code: "COL", Name: "Colombia", Currency: "COP", BaseCredibility: 0.85, SuccessRate: 0.80, GDPRank: 44, FXRate: 4023.45},
	{Code: "FIN", Name: "Finland", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.92, GDPRank: 45, FXRate: 0.9234},
	{Code: "CZE", Name: "Czech Republic", Currency: "CZK", BaseCredibility: 0.85, SuccessRate: 0.88, GDPRank: 46, FXRate: 23.4567},
	{Code: "ROU", Name: "Romania", Currency: "RON", BaseCredibility: 0.85, SuccessRate: 0.82, GDPRank: 47, FXRate: 4.5987},
	{Code: "PRT", Name: "Portugal", Currency: "EUR", BaseCredibility: 0.85, SuccessRate: 0.87, GDPRank: 48, FXRate: 0.9234},
	{Code: "NZL", Name: "New Zealand", Currency: "NZD", BaseCredibility: 0.85, SuccessRate: 0.91, GDPRank: 49, FXRate: 1.6234},
	{Code: "PER", Name: "Peru", Currency: "PEN", BaseCredibility: 0.85, SuccessRate: 0.79, GDPRank: 50, FXRate: 3.7654},
}

// BootstrapCountries creates all country nodes in Neo4j if they don't exist
func BootstrapCountries(ctx context.Context, driver neo4jdriver.DriverWithContext, database string) error {
	session := driver.NewSession(ctx, neo4jdriver.SessionConfig{
		DatabaseName: database,
		AccessMode:   neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	log.Println("🌍 Bootstrapping country nodes in Neo4j...")

	for _, country := range Top50GDPCountries {
		query := `
			MERGE (c:Country {code: $code})
			ON CREATE SET
				c.name = $name,
				c.currency = $currency,
				c.base_credibility = $baseCredibility,
				c.success_rate = $successRate,
				c.gdp_rank = $gdpRank,
				c.fx_rate = $fxRate,
				c.created_at = datetime()
			ON MATCH SET
				c.name = $name,
				c.currency = $currency,
				c.base_credibility = $baseCredibility,
				c.success_rate = $successRate,
				c.gdp_rank = $gdpRank,
				c.fx_rate = $fxRate,
				c.updated_at = datetime()
			RETURN c
		`

		_, err := session.Run(ctx, query, map[string]interface{}{
			"code":           country.Code,
			"name":           country.Name,
			"currency":       country.Currency,
			"baseCredibility": country.BaseCredibility,
			"successRate":    country.SuccessRate,
			"gdpRank":        country.GDPRank,
			"fxRate":         country.FXRate,
		})

		if err != nil {
			return fmt.Errorf("failed to bootstrap country %s: %w", country.Code, err)
		}
	}

	log.Printf("✅ Bootstrapped %d country nodes", len(Top50GDPCountries))
	return nil
}

// GetAllCurrencies returns unique currency codes from all countries
func GetAllCurrencies() []string {
	seen := make(map[string]bool)
	currencies := make([]string, 0)

	for _, c := range Top50GDPCountries {
		if !seen[c.Currency] {
			seen[c.Currency] = true
			currencies = append(currencies, c.Currency)
		}
	}

	return currencies
}

// CredibilityUpdater provides credibility update functionality
type CredibilityUpdater struct {
	driver   neo4jdriver.DriverWithContext
	database string
}

// NewCredibilityUpdater creates a new credibility updater
func NewCredibilityUpdater(driver neo4jdriver.DriverWithContext, database string) *CredibilityUpdater {
	return &CredibilityUpdater{
		driver:   driver,
		database: database,
	}
}

// UpdateCredibility updates a country's credibility based on transaction success/failure
// Success: +0.01% (0.0001)
// Failure: -0.0075% (0.000075)
// Credibility is clamped between 0.5 and 1.0
func (u *CredibilityUpdater) UpdateCredibility(ctx context.Context, countryCode string, success bool) error {
	session := u.driver.NewSession(ctx, neo4jdriver.SessionConfig{DatabaseName: u.database})
	defer session.Close(ctx)

	delta := 0.0001 // +0.01% for success
	if !success {
		delta = -0.000075 // -0.0075% for failure
	}

	query := `
		MATCH (c:Country {code: $code})
		SET c.base_credibility = CASE
			WHEN c.base_credibility + $delta > 1.0 THEN 1.0
			WHEN c.base_credibility + $delta < 0.5 THEN 0.5
			ELSE c.base_credibility + $delta
		END,
		c.credibility_updated_at = datetime()
		RETURN c.base_credibility AS new_credibility
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"code":  countryCode,
		"delta": delta,
	})
	if err != nil {
		return fmt.Errorf("failed to update credibility for %s: %w", countryCode, err)
	}

	if result.Next(ctx) {
		newCred, _ := result.Record().Get("new_credibility")
		action := "increased"
		if !success {
			action = "decreased"
		}
		log.Printf("📊 %s credibility %s to %.4f", countryCode, action, newCred)
	}

	return nil
}

