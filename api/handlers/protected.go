// Package handlers provides protected API endpoints for the PLM Dashboard.
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/plm/predictive-liquidity-mesh/api/middleware"
	"github.com/plm/predictive-liquidity-mesh/auth"
	"github.com/plm/predictive-liquidity-mesh/engine/router"
	"github.com/plm/predictive-liquidity-mesh/storage/neo4j"
	"github.com/plm/predictive-liquidity-mesh/storage/users"
	"github.com/plm/predictive-liquidity-mesh/websocket"
)

// AdminHandler handles admin-only API endpoints
type AdminHandler struct {
	graph *router.Graph
	neo4j *neo4j.Client
	wsHub *websocket.Hub
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(graph *router.Graph, neo4jClient *neo4j.Client, wsHub *websocket.Hub) *AdminHandler {
	return &AdminHandler{
		graph: graph,
		neo4j: neo4jClient,
		wsHub: wsHub,
	}
}

// CreateNodeRequest is the request body for creating a node
type CreateNodeRequest struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // "SME", "LiquidityProvider", "Hub"
	Region       string                 `json:"region,omitempty"`
	FullName     string                 `json:"full_name,omitempty"`
	Organization string                 `json:"organization,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NodeResponse is the response for node operations
type NodeResponse struct {
	Success   bool        `json:"success"`
	NodeID    string      `json:"node_id"`
	Node      interface{} `json:"node,omitempty"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
	UpdatedBy string      `json:"updated_by,omitempty"`
}

// HandleCreateNode handles POST /api/v1/admin/nodes
func (h *AdminHandler) HandleCreateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Type == "" {
		http.Error(w, `{"error":"node id and type are required"}`, http.StatusBadRequest)
		return
	}

	validTypes := map[string]bool{"SME": true, "LiquidityProvider": true, "Hub": true}
	if !validTypes[req.Type] {
		http.Error(w, `{"error":"invalid node type"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	node := &router.Node{
		ID:       req.ID,
		Type:     req.Type,
		Region:   req.Region,
		IsActive: true,
		Props:    req.Properties,
	}
	h.graph.AddNode(node)

	if h.neo4j != nil {
		props := map[string]interface{}{
			"id": req.ID, "type": req.Type, "region": req.Region,
			"is_active": true, "created_by": user.Username,
		}
		h.neo4j.CreateNode(ctx, req.Type, props)
	}

	// Broadcast to all WebSocket clients for UI sync
	if h.wsHub != nil {
		h.wsHub.BroadcastJSON(map[string]interface{}{
			"type": "NODE_CREATED",
			"data": map[string]interface{}{
				"id": req.ID, "type": req.Type, "region": req.Region, "is_active": true,
			},
		})
	}

	log.Printf("✅ Admin %s created node: %s (%s)", user.Username, req.ID, req.Type)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NodeResponse{
		Success: true, NodeID: req.ID, Message: "Node created", Timestamp: time.Now(), UpdatedBy: user.Username,
	})
}

// HandleDeleteNode handles DELETE /api/v1/admin/nodes/{id}
func (h *AdminHandler) HandleDeleteNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	// Extract node ID from path
	nodeID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/nodes/")
	nodeID = strings.TrimSuffix(nodeID, "/delete")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	// Remove from graph
	h.graph.RemoveNode(nodeID)

	// Broadcast deletion
	if h.wsHub != nil {
		h.wsHub.BroadcastJSON(map[string]interface{}{
			"type": "NODE_DELETED",
			"data": map[string]interface{}{"id": nodeID},
		})
	}

	log.Printf("🗑️ Admin %s deleted node: %s", user.Username, nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NodeResponse{
		Success: true, NodeID: nodeID, Message: "Node deleted", Timestamp: time.Now(),
	})
}

// UpdateNodeRequest is the request for updating a node
type UpdateNodeRequest struct {
	Region   string `json:"region,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// HandleUpdateNode handles PUT/PATCH /api/v1/admin/nodes/{id}
func (h *AdminHandler) HandleUpdateNode(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	nodeID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/nodes/")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	var req UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Update in graph
	if req.IsActive != nil {
		if *req.IsActive {
			h.graph.SetNodeActive(nodeID)
		} else {
			h.graph.SetNodeInactive(nodeID)
		}
	}

	// Broadcast update
	if h.wsHub != nil {
		h.wsHub.BroadcastJSON(map[string]interface{}{
			"type": "NODE_UPDATED",
			"data": map[string]interface{}{
				"id": nodeID, "is_active": req.IsActive, "region": req.Region,
			},
		})
	}

	log.Printf("✏️ Admin %s updated node: %s", user.Username, nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NodeResponse{
		Success: true, NodeID: nodeID, Message: "Node updated", Timestamp: time.Now(),
	})
}

// HandleGetNodes handles GET /api/v1/admin/nodes
func (h *AdminHandler) HandleGetNodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	nodes := h.graph.GetAllNodes()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// CreateEdgeRequest is the request for creating an edge
type CreateEdgeRequest struct {
	SourceID        string  `json:"source_id"`
	TargetID        string  `json:"target_id"`
	BaseFee         float64 `json:"base_fee"`
	Latency         int64   `json:"latency_ms"`
	LiquidityVolume int64   `json:"liquidity_volume,omitempty"`
}

// HandleCreateEdge handles POST /api/v1/admin/edges
func (h *AdminHandler) HandleCreateEdge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin() {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.SourceID == "" || req.TargetID == "" {
		http.Error(w, `{"error":"source_id and target_id are required"}`, http.StatusBadRequest)
		return
	}

	// Add edge to graph
	edge := &router.Edge{
		SourceID:        req.SourceID,
		TargetID:        req.TargetID,
		BaseFee:         req.BaseFee,
		Latency:         req.Latency,
		LiquidityVolume: req.LiquidityVolume,
		IsActive:        true,
	}
	h.graph.AddEdge(edge)

	log.Printf("✅ Admin %s created edge: %s -> %s", user.Username, req.SourceID, req.TargetID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"source_id": req.SourceID,
		"target_id": req.TargetID,
		"message":   "Edge created successfully",
	})
}

// UserHandler handles user-level API endpoints
type UserHandler struct {
	router *router.Router
	graph  *router.Graph
}

// NewUserHandler creates a new user handler
func NewUserHandler(routerInstance *router.Router, graph *router.Graph) *UserHandler {
	return &UserHandler{
		router: routerInstance,
		graph:  graph,
	}
}

// SettlePreviewRequest is the request for settle preview
type SettlePreviewRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Amount      int64  `json:"amount,omitempty"`
}

// PathPreview represents a single path option
type PathPreview struct {
	Rank         int      `json:"rank"`
	Path         []string `json:"path"`
	TotalFee     float64  `json:"total_fee_percent"`
	TotalLatency int64    `json:"total_latency_ms"`
	TotalWeight  float64  `json:"total_weight"`
	HopCount     int      `json:"hop_count"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
}

// SettlePreviewResponse is the response with top paths
type SettlePreviewResponse struct {
	Source      string         `json:"source"`
	Destination string         `json:"destination"`
	Amount      int64          `json:"amount,omitempty"`
	Paths       []*PathPreview `json:"paths"`
	PathCount   int            `json:"path_count"`
	ComputeTime string         `json:"compute_time"`
}

// HandleSettlePreview handles GET/POST /api/v1/settle/preview
// Returns top 3 paths found by Yen's algorithm
func (h *UserHandler) HandleSettlePreview(w http.ResponseWriter, r *http.Request) {
	var source, destination string
	var amount int64

	if r.Method == http.MethodGet {
		// Query params
		source = r.URL.Query().Get("source")
		destination = r.URL.Query().Get("destination")
	} else if r.Method == http.MethodPost {
		var req SettlePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		source = req.Source
		destination = req.Destination
		amount = req.Amount
	} else {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if source == "" || destination == "" {
		http.Error(w, `{"error":"source and destination are required"}`, http.StatusBadRequest)
		return
	}

	// Get authenticated user
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()

	// Find K shortest paths using Yen's algorithm
	paths, err := h.router.FindKShortestPaths(ctx, source, destination)
	if err != nil {
		http.Error(w, `{"error":"failed to find paths: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	computeTime := time.Since(start)

	// Convert to preview format
	previews := make([]*PathPreview, 0, len(paths))
	for i, p := range paths {
		preview := &PathPreview{
			Rank:         i + 1,
			Path:         p.Nodes,
			TotalFee:     p.TotalFee * 100, // Convert to percentage
			TotalLatency: p.TotalLatency,
			TotalWeight:  p.TotalWeight,
			HopCount:     len(p.Nodes) - 1,
		}
		if amount > 0 {
			preview.EstimatedCost = float64(amount) * p.TotalFee
		}
		previews = append(previews, preview)
	}

	resp := SettlePreviewResponse{
		Source:      source,
		Destination: destination,
		Amount:      amount,
		Paths:       previews,
		PathCount:   len(previews),
		ComputeTime: computeTime.String(),
	}

	log.Printf("📊 User %s previewed settlement: %s -> %s (%d paths in %v)",
		user.Username, source, destination, len(paths), computeTime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	tokenManager *auth.TokenManager
	userStore    UserStorer
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(tm *auth.TokenManager) *AuthHandler {
	return &AuthHandler{tokenManager: tm}
}

// LoginRequest is the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the login response
type LoginResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      *auth.User `json:"user"`
}

// UserStorer interface for user operations - implemented by users.Store
type UserStorer interface {
	Authenticate(email, password string) (users.UserWithToUser, error)
	CreateUser(email, password, username string, role auth.Role) (users.UserWithToUser, error)
	GetByEmail(email string) (users.UserWithToUser, error)
}

// SetUserStore sets the user store for authentication
func (h *AuthHandler) SetUserStore(store UserStorer) {
	h.userStore = store
}

// RegisterRequest is the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// HandleLogin handles POST /api/v1/auth/login
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var user *auth.User

	// Use user store if available, otherwise fallback to demo mode
	if h.userStore != nil {
		storedUser, err := h.userStore.Authenticate(req.Email, req.Password)
		if err != nil {
			http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
			return
		}
		user = storedUser.ToUser()
	} else {
		// Demo mode fallback
		var role auth.Role
		if req.Email == "admin@plm.local" {
			role = auth.RoleAdmin
		} else {
			role = auth.RoleUser
		}

		user = &auth.User{
			ID:       "demo-user-id",
			Email:    req.Email,
			Username: req.Email[:strings.Index(req.Email, "@")],
			Role:     role,
			IsActive: true,
		}
	}

	// Generate token
	token, claims, err := h.tokenManager.GenerateToken(user)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("🔐 User logged in: %s (role: %s)", user.Email, user.Role)

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: claims.ExpiresAt,
		User:      user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRegister handles POST /api/v1/auth/register
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if h.userStore == nil {
		http.Error(w, `{"error":"registration not available"}`, http.StatusServiceUnavailable)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Username == "" {
		http.Error(w, `{"error":"email, password, and username are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, `{"error":"password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	// Create user with USER role by default
	storedUser, err := h.userStore.CreateUser(req.Email, req.Password, req.Username, auth.RoleUser)
	if err != nil {
		log.Printf("❌ Registration failed: %v", err)
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
		return
	}

	user := storedUser.ToUser()

	// Generate token
	token, claims, err := h.tokenManager.GenerateToken(user)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("🆕 User registered: %s (%s)", user.Email, user.Username)

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: claims.ExpiresAt,
		User:      user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
