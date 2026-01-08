// Package router provides country graph building utilities
package router

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CountryData represents country data from Neo4j
type CountryData struct {
	Code        string
	Name        string
	Currency    string
	Credibility float64
	SuccessRate float64
	FXRate      float64
}

// TradeConnection represents a trade connection between countries
type TradeConnection struct {
	Source string
	Target string
}

// DefaultTradeConnections returns the standard trade connections
var DefaultTradeConnections = []TradeConnection{
	// USD hub connections
	{"USA", "GBR"}, {"USA", "EUR"}, {"USA", "JPN"}, {"USA", "CHN"}, {"USA", "CAN"},
	{"USA", "MEX"}, {"USA", "AUS"}, {"USA", "CHE"}, {"USA", "KOR"}, {"USA", "IND"},
	{"USA", "BRA"}, {"USA", "SGP"}, {"USA", "HKG"},
	// EUR connections (using DEU as EUR representative)
	{"DEU", "FRA"}, {"DEU", "ITA"}, {"DEU", "ESP"}, {"DEU", "NLD"}, {"DEU", "BEL"},
	{"DEU", "AUT"}, {"DEU", "POL"}, {"DEU", "CHE"}, {"DEU", "GBR"},
	{"FRA", "ITA"}, {"FRA", "ESP"}, {"FRA", "BEL"}, {"FRA", "NLD"},
	// Asian connections
	{"CHN", "JPN"}, {"CHN", "KOR"}, {"CHN", "HKG"}, {"CHN", "TWN"}, {"CHN", "SGP"},
	{"CHN", "THA"}, {"CHN", "VNM"}, {"CHN", "MYS"}, {"CHN", "IDN"}, {"CHN", "IND"},
	{"JPN", "KOR"}, {"JPN", "TWN"}, {"JPN", "SGP"}, {"JPN", "THA"},
	{"SGP", "MYS"}, {"SGP", "HKG"}, {"SGP", "THA"}, {"SGP", "IDN"},
	// Middle East
	{"SAU", "ARE"}, {"SAU", "EGY"}, {"ARE", "IND"},
	// South America
	{"BRA", "ARG"}, {"BRA", "MEX"}, {"BRA", "CHL"}, {"BRA", "COL"},
	{"MEX", "COL"}, {"CHL", "PER"}, {"ARG", "CHL"},
	// Africa
	{"ZAF", "NGA"}, {"ZAF", "EGY"},
	// Oceania
	{"AUS", "NZL"}, {"AUS", "SGP"}, {"AUS", "JPN"}, {"AUS", "CHN"},
	// Nordic
	{"SWE", "NOR"}, {"SWE", "DNK"}, {"SWE", "FIN"}, {"NOR", "DNK"},
	// Eastern Europe
	{"POL", "CZE"}, {"CZE", "AUT"}, {"ROU", "POL"},
	// Other major pairs
	{"GBR", "IRL"}, {"GBR", "CHE"}, {"GBR", "IND"}, {"GBR", "HKG"},
	{"CHE", "AUT"}, {"ISR", "USA"}, {"TUR", "DEU"},
}

// BuildCountryGraphFromNeo4j builds a CountryGraph from Neo4j country data
func BuildCountryGraphFromNeo4j(ctx context.Context, driver neo4j.DriverWithContext, database string) (*CountryGraph, error) {
	graph := NewCountryGraph()

	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: database})
	defer session.Close(ctx)

	// Fetch all countries
	result, err := session.Run(ctx, `
		MATCH (c:Country)
		RETURN c.code AS code, c.name AS name, c.currency AS currency,
		       c.base_credibility AS credibility, c.success_rate AS success_rate,
		       c.fx_rate AS fx_rate
	`, nil)
	if err != nil {
		return nil, err
	}

	countries := make(map[string]*CountryData)

	for result.Next(ctx) {
		record := result.Record()
		code, _ := record.Get("code")
		name, _ := record.Get("name")
		currency, _ := record.Get("currency")
		credibility, _ := record.Get("credibility")
		successRate, _ := record.Get("success_rate")
		fxRate, _ := record.Get("fx_rate")

		data := &CountryData{
			Code:        toString(code),
			Name:        toString(name),
			Currency:    toString(currency),
			Credibility: toFloat(credibility),
			SuccessRate: toFloat(successRate),
			FXRate:      toFloat(fxRate),
		}
		countries[data.Code] = data

		// Add node to graph
		graph.AddNode(&CountryNode{
			Code:        data.Code,
			Name:        data.Name,
			Currency:    data.Currency,
			Credibility: data.Credibility,
			SuccessRate: data.SuccessRate,
			FXRate:      data.FXRate,
			IsActive:    true,
		})
	}

	log.Printf("📊 Loaded %d countries into routing graph", len(countries))

	// Add trade connections
	edgeCount := 0
	for _, conn := range DefaultTradeConnections {
		if _, ok := countries[conn.Source]; !ok {
			continue
		}
		if _, ok := countries[conn.Target]; !ok {
			continue
		}

		// Base cost is derived from FX rates difference (normalized)
		srcRate := countries[conn.Source].FXRate
		tgtRate := countries[conn.Target].FXRate
		if srcRate == 0 {
			srcRate = 1
		}
		if tgtRate == 0 {
			tgtRate = 1
		}

		// Base cost: normalized difference (0-1 range)
		baseCost := 0.01 // Default small cost

		graph.AddEdge(&CountryEdge{
			SourceCode: conn.Source,
			TargetCode: conn.Target,
			BaseCost:   baseCost,
			IsActive:   true,
		})
		edgeCount++
	}

	log.Printf("📊 Added %d trade connections to routing graph", edgeCount)

	return graph, nil
}

// BuildCountryGraphWithDefaults builds a graph with default country data
func BuildCountryGraphWithDefaults() *CountryGraph {
	graph := NewCountryGraph()

	// Default countries with sample data - Complete list matching frontend
	defaultCountries := []CountryData{
		// Major economies
		{Code: "USA", Name: "United States", Currency: "USD", Credibility: 0.98, SuccessRate: 0.99, FXRate: 1.0},
		{Code: "CHN", Name: "China", Currency: "CNY", Credibility: 0.88, SuccessRate: 0.92, FXRate: 7.24},
		{Code: "DEU", Name: "Germany", Currency: "EUR", Credibility: 0.96, SuccessRate: 0.98, FXRate: 0.92},
		{Code: "JPN", Name: "Japan", Currency: "JPY", Credibility: 0.94, SuccessRate: 0.96, FXRate: 149.50},
		{Code: "IND", Name: "India", Currency: "INR", Credibility: 0.85, SuccessRate: 0.90, FXRate: 83.12},
		{Code: "GBR", Name: "United Kingdom", Currency: "GBP", Credibility: 0.95, SuccessRate: 0.97, FXRate: 0.79},
		{Code: "FRA", Name: "France", Currency: "EUR", Credibility: 0.94, SuccessRate: 0.96, FXRate: 0.92},
		{Code: "ITA", Name: "Italy", Currency: "EUR", Credibility: 0.90, SuccessRate: 0.93, FXRate: 0.92},
		{Code: "BRA", Name: "Brazil", Currency: "BRL", Credibility: 0.80, SuccessRate: 0.85, FXRate: 4.97},
		{Code: "CAN", Name: "Canada", Currency: "CAD", Credibility: 0.93, SuccessRate: 0.96, FXRate: 1.36},
		// 11-20
		{Code: "RUS", Name: "Russia", Currency: "RUB", Credibility: 0.72, SuccessRate: 0.80, FXRate: 90.50},
		{Code: "KOR", Name: "South Korea", Currency: "KRW", Credibility: 0.91, SuccessRate: 0.94, FXRate: 1320.50},
		{Code: "AUS", Name: "Australia", Currency: "AUD", Credibility: 0.92, SuccessRate: 0.95, FXRate: 1.55},
		{Code: "MEX", Name: "Mexico", Currency: "MXN", Credibility: 0.78, SuccessRate: 0.84, FXRate: 17.15},
		{Code: "ESP", Name: "Spain", Currency: "EUR", Credibility: 0.89, SuccessRate: 0.92, FXRate: 0.92},
		{Code: "IDN", Name: "Indonesia", Currency: "IDR", Credibility: 0.76, SuccessRate: 0.82, FXRate: 15750.0},
		{Code: "NLD", Name: "Netherlands", Currency: "EUR", Credibility: 0.95, SuccessRate: 0.97, FXRate: 0.92},
		{Code: "SAU", Name: "Saudi Arabia", Currency: "SAR", Credibility: 0.90, SuccessRate: 0.93, FXRate: 3.75},
		{Code: "TUR", Name: "Turkey", Currency: "TRY", Credibility: 0.70, SuccessRate: 0.78, FXRate: 32.15},
		{Code: "CHE", Name: "Switzerland", Currency: "CHF", Credibility: 0.99, SuccessRate: 0.99, FXRate: 0.88},
		// 21-30
		{Code: "POL", Name: "Poland", Currency: "PLN", Credibility: 0.86, SuccessRate: 0.90, FXRate: 3.95},
		{Code: "TWN", Name: "Taiwan", Currency: "TWD", Credibility: 0.89, SuccessRate: 0.93, FXRate: 31.50},
		{Code: "BEL", Name: "Belgium", Currency: "EUR", Credibility: 0.93, SuccessRate: 0.96, FXRate: 0.92},
		{Code: "SWE", Name: "Sweden", Currency: "SEK", Credibility: 0.94, SuccessRate: 0.96, FXRate: 10.45},
		{Code: "IRL", Name: "Ireland", Currency: "EUR", Credibility: 0.93, SuccessRate: 0.96, FXRate: 0.92},
		{Code: "AUT", Name: "Austria", Currency: "EUR", Credibility: 0.94, SuccessRate: 0.96, FXRate: 0.92},
		{Code: "THA", Name: "Thailand", Currency: "THB", Credibility: 0.82, SuccessRate: 0.87, FXRate: 35.20},
		{Code: "ISR", Name: "Israel", Currency: "ILS", Credibility: 0.88, SuccessRate: 0.92, FXRate: 3.70},
		{Code: "NGA", Name: "Nigeria", Currency: "NGN", Credibility: 0.65, SuccessRate: 0.72, FXRate: 1550.0},
		{Code: "ARE", Name: "UAE", Currency: "AED", Credibility: 0.92, SuccessRate: 0.95, FXRate: 3.67},
		// 31-40
		{Code: "ARG", Name: "Argentina", Currency: "ARS", Credibility: 0.60, SuccessRate: 0.68, FXRate: 875.0},
		{Code: "NOR", Name: "Norway", Currency: "NOK", Credibility: 0.95, SuccessRate: 0.97, FXRate: 10.65},
		{Code: "EGY", Name: "Egypt", Currency: "EGP", Credibility: 0.68, SuccessRate: 0.75, FXRate: 30.90},
		{Code: "VNM", Name: "Vietnam", Currency: "VND", Credibility: 0.75, SuccessRate: 0.81, FXRate: 24500.0},
		{Code: "BGD", Name: "Bangladesh", Currency: "BDT", Credibility: 0.70, SuccessRate: 0.77, FXRate: 110.50},
		{Code: "ZAF", Name: "South Africa", Currency: "ZAR", Credibility: 0.74, SuccessRate: 0.80, FXRate: 18.75},
		{Code: "PHL", Name: "Philippines", Currency: "PHP", Credibility: 0.77, SuccessRate: 0.83, FXRate: 56.25},
		{Code: "DNK", Name: "Denmark", Currency: "DKK", Credibility: 0.94, SuccessRate: 0.96, FXRate: 6.85},
		{Code: "MYS", Name: "Malaysia", Currency: "MYR", Credibility: 0.84, SuccessRate: 0.89, FXRate: 4.70},
		{Code: "SGP", Name: "Singapore", Currency: "SGD", Credibility: 0.97, SuccessRate: 0.98, FXRate: 1.34},
		// 41-51
		{Code: "HKG", Name: "Hong Kong", Currency: "HKD", Credibility: 0.93, SuccessRate: 0.96, FXRate: 7.82},
		{Code: "PAK", Name: "Pakistan", Currency: "PKR", Credibility: 0.62, SuccessRate: 0.70, FXRate: 285.0},
		{Code: "CHL", Name: "Chile", Currency: "CLP", Credibility: 0.82, SuccessRate: 0.87, FXRate: 885.0},
		{Code: "COL", Name: "Colombia", Currency: "COP", Credibility: 0.78, SuccessRate: 0.84, FXRate: 4050.0},
		{Code: "FIN", Name: "Finland", Currency: "EUR", Credibility: 0.95, SuccessRate: 0.97, FXRate: 0.92},
		{Code: "CZE", Name: "Czech Republic", Currency: "CZK", Credibility: 0.88, SuccessRate: 0.92, FXRate: 22.75},
		{Code: "ROU", Name: "Romania", Currency: "RON", Credibility: 0.80, SuccessRate: 0.86, FXRate: 4.56},
		{Code: "PRT", Name: "Portugal", Currency: "EUR", Credibility: 0.88, SuccessRate: 0.92, FXRate: 0.92},
		{Code: "NZL", Name: "New Zealand", Currency: "NZD", Credibility: 0.91, SuccessRate: 0.94, FXRate: 1.68},
		{Code: "PER", Name: "Peru", Currency: "PEN", Credibility: 0.76, SuccessRate: 0.82, FXRate: 3.75},
	}

	for _, c := range defaultCountries {
		graph.AddNode(&CountryNode{
			Code:        c.Code,
			Name:        c.Name,
			Currency:    c.Currency,
			Credibility: c.Credibility,
			SuccessRate: c.SuccessRate,
			FXRate:      c.FXRate,
			IsActive:    true,
		})
	}

	// Add edges
	for _, conn := range DefaultTradeConnections {
		graph.AddEdge(&CountryEdge{
			SourceCode: conn.Source,
			TargetCode: conn.Target,
			BaseCost:   0.01,
			IsActive:   true,
		})
	}

	return graph
}

// Helper functions
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	}
	return 0
}
