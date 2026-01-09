// Package handlers provides WebSocket handlers for route calculation
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/plm/predictive-liquidity-mesh/api/middleware"
	"github.com/plm/predictive-liquidity-mesh/engine/router"
)

// RouteRequest represents a routing request from the client
type RouteRequest struct {
	Type         string   `json:"type"`          // "route_request"
	Source       string   `json:"source"`        // Source country code
	Target       string   `json:"target"`        // Target country code
	BlockedCodes []string `json:"blocked_codes"` // Countries to avoid
	Amount       float64  `json:"amount"`        // Optional: amount to transfer
}

// RouteResponse represents the routing response
type RouteResponse struct {
	Type     string                `json:"type"`      // "route_response"
	Success  bool                  `json:"success"`   
	Paths    []*RoutePathInfo      `json:"paths"`     // Top K paths
	Error    string                `json:"error,omitempty"`
	Duration int64                 `json:"duration_ms"` // Processing time
}

// RoutePathInfo contains detailed path information
type RoutePathInfo struct {
	Rank           int      `json:"rank"`
	Nodes          []string `json:"nodes"`
	HopCount       int      `json:"hop_count"`
	TotalWeight    float64  `json:"total_weight"`
	TotalFeePercent float64 `json:"total_fee_percent"` // Fee as percentage
	FinalAmount    float64  `json:"final_amount"`      // Amount after fees (per 1.0)
	CalculatedFee  float64  `json:"calculated_fee,omitempty"` // Actual fee if amount provided
}

// RouteHandler handles WebSocket connections for route calculation
type RouteHandler struct {
	router   *router.CountryRouter
	graph    *router.CountryGraph
	upgrader websocket.Upgrader
}

// NewRouteHandler creates a new route handler
func NewRouteHandler(graph *router.CountryGraph) *RouteHandler {
	countryRouter := router.NewCountryRouter(graph, 3) // Find top 3 paths
	
	return &RouteHandler{
		router: countryRouter,
		graph:  graph,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return middleware.IsOriginAllowed(origin, r.Host)
			},
		},
	}
}

// HandleRouteWS handles WebSocket connections for routing
func (h *RouteHandler) HandleRouteWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("Route WebSocket client connected")

	for {
		// Read request
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Route WebSocket error: %v", err)
			}
			break
		}

		// Parse request
		var req RouteRequest
		if err := json.Unmarshal(message, &req); err != nil {
			h.sendError(conn, "invalid request format")
			continue
		}

		// Handle route request
		if req.Type == "route_request" {
			h.handleRouteRequest(conn, &req)
		}
	}
}

// handleRouteRequest processes a routing request and sends response
func (h *RouteHandler) handleRouteRequest(conn *websocket.Conn, req *RouteRequest) {
	start := time.Now()

	// Validate request
	if req.Source == "" || req.Target == "" {
		h.sendError(conn, "source and target are required")
		return
	}

	if req.Source == req.Target {
		h.sendError(conn, "source and target must be different")
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find paths
	paths, err := h.router.FindKShortestPaths(ctx, req.Source, req.Target, req.BlockedCodes)
	
	response := &RouteResponse{
		Type:     "route_response",
		Duration: time.Since(start).Milliseconds(),
	}

	if err != nil {
		response.Success = false
		response.Error = err.Error()
	} else {
		response.Success = true
		response.Paths = make([]*RoutePathInfo, len(paths))
		
		for i, path := range paths {
			pathInfo := &RoutePathInfo{
				Rank:            i + 1,
				Nodes:           path.Nodes,
				HopCount:        path.HopCount,
				TotalWeight:     path.TotalWeight,
				TotalFeePercent: path.TotalFeePercent,
				FinalAmount:     path.FinalAmount,
			}
			
			// Calculate actual fee if amount provided
			if req.Amount > 0 {
				pathInfo.CalculatedFee = req.Amount * (1 - path.FinalAmount)
			}
			
			response.Paths[i] = pathInfo
		}
	}

	// Send response
	data, _ := json.Marshal(response)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send route response: %v", err)
	}
}

// sendError sends an error response
func (h *RouteHandler) sendError(conn *websocket.Conn, errorMsg string) {
	response := &RouteResponse{
		Type:    "route_response",
		Success: false,
		Error:   errorMsg,
	}
	data, _ := json.Marshal(response)
	conn.WriteMessage(websocket.TextMessage, data)
}

// HandleRouteHTTP handles HTTP POST requests for routing (non-WebSocket)
func (h *RouteHandler) HandleRouteHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.Source == "" || req.Target == "" {
		http.Error(w, `{"error":"source and target are required"}`, http.StatusBadRequest)
		return
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	paths, err := h.router.FindKShortestPaths(ctx, req.Source, req.Target, req.BlockedCodes)

	w.Header().Set("Content-Type", "application/json")

	response := &RouteResponse{
		Type:     "route_response",
		Duration: time.Since(start).Milliseconds(),
	}

	if err != nil {
		response.Success = false
		response.Error = err.Error()
		w.WriteHeader(http.StatusOK) // Still 200, error in response
	} else {
		response.Success = true
		response.Paths = make([]*RoutePathInfo, len(paths))
		
		for i, path := range paths {
			response.Paths[i] = &RoutePathInfo{
				Rank:            i + 1,
				Nodes:           path.Nodes,
				HopCount:        path.HopCount,
				TotalWeight:     path.TotalWeight,
				TotalFeePercent: path.TotalFeePercent,
				FinalAmount:     path.FinalAmount,
			}
			if req.Amount > 0 {
				response.Paths[i].CalculatedFee = req.Amount * (1 - path.FinalAmount)
			}
		}
	}

	json.NewEncoder(w).Encode(response)
}
