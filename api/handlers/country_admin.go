// Package handlers provides API endpoints for country node management.
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/plm/predictive-liquidity-mesh/api/middleware"
)

// CountryHandler handles country node API endpoints
type CountryHandler struct {
	driver   neo4j.DriverWithContext
	database string
}

// NewCountryHandler creates a new country handler
func NewCountryHandler(driver neo4j.DriverWithContext, database string) *CountryHandler {
	return &CountryHandler{
		driver:   driver,
		database: database,
	}
}

// Country represents a country node
type Country struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Currency        string  `json:"currency"`
	BaseCredibility float64 `json:"base_credibility"`
	SuccessRate     float64 `json:"success_rate"`
	GDPRank         int     `json:"gdp_rank,omitempty"`
	FXRate          float64 `json:"fx_rate,omitempty"`
}

// CreateCountryRequest is the request body for creating a country
type CreateCountryRequest struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Currency        string  `json:"currency"`
	BaseCredibility float64 `json:"base_credibility"`
	SuccessRate     float64 `json:"success_rate"`
}

// HandleListCountries handles GET /api/v1/admin/countries
func (h *CountryHandler) HandleListCountries(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	session := h.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: h.database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (c:Country)
		RETURN c.code AS code, c.name AS name, c.currency AS currency,
		       c.base_credibility AS base_credibility, c.success_rate AS success_rate,
		       c.gdp_rank AS gdp_rank, c.fx_rate AS fx_rate
		ORDER BY c.gdp_rank ASC
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch countries"}`, http.StatusInternalServerError)
		return
	}

	countries := make([]Country, 0)
	for result.Next(ctx) {
		record := result.Record()
		country := Country{}

		if v, ok := record.Get("code"); ok && v != nil {
			country.Code = v.(string)
		}
		if v, ok := record.Get("name"); ok && v != nil {
			country.Name = v.(string)
		}
		if v, ok := record.Get("currency"); ok && v != nil {
			country.Currency = v.(string)
		}
		if v, ok := record.Get("base_credibility"); ok && v != nil {
			country.BaseCredibility = v.(float64)
		}
		if v, ok := record.Get("success_rate"); ok && v != nil {
			country.SuccessRate = v.(float64)
		}
		if v, ok := record.Get("gdp_rank"); ok && v != nil {
			country.GDPRank = int(v.(int64))
		}
		if v, ok := record.Get("fx_rate"); ok && v != nil {
			country.FXRate = v.(float64)
		}

		countries = append(countries, country)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"countries": countries,
		"count":     len(countries),
	})
}

// HandleCreateCountry handles POST /api/v1/admin/countries
func (h *CountryHandler) HandleCreateCountry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	var req CreateCountryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.Name == "" || req.Currency == "" {
		http.Error(w, `{"error":"code, name, and currency are required"}`, http.StatusBadRequest)
		return
	}

	// Default to 0.85 if not specified
	if req.BaseCredibility == 0 {
		req.BaseCredibility = 0.85
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session := h.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: h.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MERGE (c:Country {code: $code})
		ON CREATE SET
			c.name = $name,
			c.currency = $currency,
			c.base_credibility = $baseCredibility,
			c.success_rate = $successRate,
			c.created_at = datetime(),
			c.created_by = $createdBy
		ON MATCH SET
			c.name = $name,
			c.currency = $currency,
			c.base_credibility = $baseCredibility,
			c.success_rate = $successRate,
			c.updated_at = datetime()
		RETURN c
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"code":           strings.ToUpper(req.Code),
		"name":           req.Name,
		"currency":       strings.ToUpper(req.Currency),
		"baseCredibility": req.BaseCredibility,
		"successRate":    req.SuccessRate,
		"createdBy":      user.Username,
	})

	if err != nil {
		log.Printf("❌ Failed to create country: %v", err)
		http.Error(w, `{"error":"failed to create country"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Admin %s created country: %s (%s)", user.Username, req.Code, req.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"code":    strings.ToUpper(req.Code),
		"message": "Country created successfully",
	})
}

// HandleDeleteCountry handles DELETE /api/v1/admin/countries/{code}
func (h *CountryHandler) HandleDeleteCountry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	// Extract code from path: /api/v1/admin/countries/{code}
	code := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/countries/")
	if code == "" {
		http.Error(w, `{"error":"country code required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session := h.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: h.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (c:Country {code: $code})
		DELETE c
		RETURN count(c) as deleted
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"code": strings.ToUpper(code),
	})

	if err != nil {
		log.Printf("❌ Failed to delete country: %v", err)
		http.Error(w, `{"error":"failed to delete country"}`, http.StatusInternalServerError)
		return
	}

	var deleted int64
	if result.Next(ctx) {
		record := result.Record()
		if v, ok := record.Get("deleted"); ok {
			deleted = v.(int64)
		}
	}

	if deleted == 0 {
		http.Error(w, `{"error":"country not found"}`, http.StatusNotFound)
		return
	}

	log.Printf("🗑️ Admin %s deleted country: %s", user.Username, code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"code":    strings.ToUpper(code),
		"message": "Country deleted successfully",
	})
}
