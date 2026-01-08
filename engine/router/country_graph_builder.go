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

	// Default countries with sample data
	defaultCountries := []CountryData{
		{Code: "USA", Name: "United States", Currency: "USD", Credibility: 0.98, SuccessRate: 0.99, FXRate: 1.0},
		{Code: "GBR", Name: "United Kingdom", Currency: "GBP", Credibility: 0.95, SuccessRate: 0.97, FXRate: 0.79},
		{Code: "DEU", Name: "Germany", Currency: "EUR", Credibility: 0.96, SuccessRate: 0.98, FXRate: 0.92},
		{Code: "JPN", Name: "Japan", Currency: "JPY", Credibility: 0.94, SuccessRate: 0.96, FXRate: 149.50},
		{Code: "CHN", Name: "China", Currency: "CNY", Credibility: 0.88, SuccessRate: 0.92, FXRate: 7.24},
		{Code: "IND", Name: "India", Currency: "INR", Credibility: 0.85, SuccessRate: 0.90, FXRate: 83.12},
		{Code: "CAN", Name: "Canada", Currency: "CAD", Credibility: 0.93, SuccessRate: 0.96, FXRate: 1.36},
		{Code: "AUS", Name: "Australia", Currency: "AUD", Credibility: 0.92, SuccessRate: 0.95, FXRate: 1.55},
		{Code: "BRA", Name: "Brazil", Currency: "BRL", Credibility: 0.80, SuccessRate: 0.85, FXRate: 4.97},
		{Code: "MEX", Name: "Mexico", Currency: "MXN", Credibility: 0.78, SuccessRate: 0.84, FXRate: 17.15},
		{Code: "SGP", Name: "Singapore", Currency: "SGD", Credibility: 0.97, SuccessRate: 0.98, FXRate: 1.34},
		{Code: "CHE", Name: "Switzerland", Currency: "CHF", Credibility: 0.99, SuccessRate: 0.99, FXRate: 0.88},
		{Code: "KOR", Name: "South Korea", Currency: "KRW", Credibility: 0.91, SuccessRate: 0.94, FXRate: 1320.50},
		{Code: "HKG", Name: "Hong Kong", Currency: "HKD", Credibility: 0.93, SuccessRate: 0.96, FXRate: 7.82},
		{Code: "FRA", Name: "France", Currency: "EUR", Credibility: 0.94, SuccessRate: 0.96, FXRate: 0.92},
		{Code: "ITA", Name: "Italy", Currency: "EUR", Credibility: 0.90, SuccessRate: 0.93, FXRate: 0.92},
		{Code: "ESP", Name: "Spain", Currency: "EUR", Credibility: 0.89, SuccessRate: 0.92, FXRate: 0.92},
		{Code: "NLD", Name: "Netherlands", Currency: "EUR", Credibility: 0.95, SuccessRate: 0.97, FXRate: 0.92},
		{Code: "TUR", Name: "Turkey", Currency: "TRY", Credibility: 0.70, SuccessRate: 0.78, FXRate: 32.15},
		{Code: "ARE", Name: "UAE", Currency: "AED", Credibility: 0.92, SuccessRate: 0.95, FXRate: 3.67},
		{Code: "SAU", Name: "Saudi Arabia", Currency: "SAR", Credibility: 0.90, SuccessRate: 0.93, FXRate: 3.75},
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
