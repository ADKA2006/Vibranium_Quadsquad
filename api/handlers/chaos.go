// Package handlers provides HTTP handlers for the Predictive Liquidity Mesh API.
// Includes debug endpoints for chaos testing and anti-fragility demos.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/plm/predictive-liquidity-mesh/engine/router"
	redisClient "github.com/plm/predictive-liquidity-mesh/storage/redis"
	"github.com/plm/predictive-liquidity-mesh/websocket"
)

// ChaosHandler handles chaos testing endpoints
type ChaosHandler struct {
	redis     *redisClient.Client
	router    *router.Router
	wsHub     *websocket.Hub
	graph     *router.Graph
	killedNodes map[string]bool
	mu        sync.RWMutex
}

// NewChaosHandler creates a new chaos handler
func NewChaosHandler(
	redis *redisClient.Client,
	routerInstance *router.Router,
	graph *router.Graph,
	wsHub *websocket.Hub,
) *ChaosHandler {
	return &ChaosHandler{
		redis:       redis,
		router:      routerInstance,
		graph:       graph,
		wsHub:       wsHub,
		killedNodes: make(map[string]bool),
	}
}

// KillNodeResponse is the response for the kill endpoint
type KillNodeResponse struct {
	Success   bool   `json:"success"`
	NodeID    string `json:"node_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// HandleKillNode handles POST /debug/kill/{node_id}
// Instantly triggers the Redis circuit breaker for the node
func (h *ChaosHandler) HandleKillNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract node_id from path
	path := strings.TrimPrefix(r.URL.Path, "/debug/kill/")
	nodeID := strings.TrimSuffix(path, "/")

	if nodeID == "" {
		http.Error(w, "Node ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	log.Printf("🔥 CHAOS: Killing node %s", nodeID)

	// 1. Force open the circuit breaker in Redis
	if h.redis != nil {
		cfg := redisClient.DefaultCircuitBreakerConfig(nodeID)
		if err := h.redis.CircuitBreaker().ForceOpen(ctx, cfg); err != nil {
			log.Printf("Failed to force open circuit breaker: %v", err)
		}
	}

	// 2. Mark node as killed
	h.mu.Lock()
	h.killedNodes[nodeID] = true
	h.mu.Unlock()

	// 3. Update graph to mark node as inactive
	if h.graph != nil {
		h.graph.SetNodeInactive(nodeID)
	}

	// 4. Broadcast circuit breaker event to all WebSocket clients
	if h.wsHub != nil {
		h.wsHub.BroadcastCircuitBreaker(&websocket.CircuitBreakerEvent{
			NodeID:    nodeID,
			State:     "open",
			PrevState: "closed",
		})
	}

	// Send response
	resp := KillNodeResponse{
		Success:   true,
		NodeID:    nodeID,
		Message:   fmt.Sprintf("Node %s killed - circuit breaker opened", nodeID),
		Timestamp: time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("✅ Node %s killed successfully", nodeID)
}

// HandleReviveNode handles POST /debug/revive/{node_id}
// Resets the circuit breaker and re-enables the node
func (h *ChaosHandler) HandleReviveNode(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/debug/revive/")
	nodeID := strings.TrimSuffix(path, "/")

	if nodeID == "" {
		http.Error(w, "Node ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	log.Printf("💚 REVIVE: Bringing back node %s", nodeID)

	// 1. Reset circuit breaker
	if h.redis != nil {
		cfg := redisClient.DefaultCircuitBreakerConfig(nodeID)
		h.redis.CircuitBreaker().Reset(ctx, cfg)
	}

	// 2. Remove from killed list
	h.mu.Lock()
	delete(h.killedNodes, nodeID)
	h.mu.Unlock()

	// 3. Mark node as active in graph
	if h.graph != nil {
		h.graph.SetNodeActive(nodeID)
	}

	// 4. Broadcast update
	if h.wsHub != nil {
		h.wsHub.BroadcastCircuitBreaker(&websocket.CircuitBreakerEvent{
			NodeID:    nodeID,
			State:     "closed",
			PrevState: "open",
		})
	}

	resp := KillNodeResponse{
		Success:   true,
		NodeID:    nodeID,
		Message:   fmt.Sprintf("Node %s revived - circuit breaker closed", nodeID),
		Timestamp: time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetKilledNodes returns the list of killed nodes
func (h *ChaosHandler) HandleGetKilledNodes(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	nodes := make([]string, 0, len(h.killedNodes))
	for nodeID := range h.killedNodes {
		nodes = append(nodes, nodeID)
	}
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"killed_nodes": nodes,
		"count":        len(nodes),
	})
}

// IsNodeKilled checks if a node is currently killed
func (h *ChaosHandler) IsNodeKilled(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.killedNodes[nodeID]
}
