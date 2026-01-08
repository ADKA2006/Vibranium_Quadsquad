// Package demo provides the anti-fragility chaos demonstration.
// Simulates a $10,000 transaction with mid-flight node failure and automatic re-routing.
package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plm/predictive-liquidity-mesh/engine/router"
	"github.com/plm/predictive-liquidity-mesh/websocket"
)

// ChaosDemo manages the anti-fragility demonstration
type ChaosDemo struct {
	router    *router.Router
	graph     *router.Graph
	wsHub     *websocket.Hub
	killFunc  func(nodeID string) error
	mu        sync.Mutex
}

// NewChaosDemo creates a new chaos demo manager
func NewChaosDemo(
	routerInstance *router.Router,
	graph *router.Graph,
	wsHub *websocket.Hub,
	killFunc func(nodeID string) error,
) *ChaosDemo {
	return &ChaosDemo{
		router:   routerInstance,
		graph:    graph,
		wsHub:    wsHub,
		killFunc: killFunc,
	}
}

// DemoTransaction represents the demo transaction
type DemoTransaction struct {
	ID           string   `json:"id"`
	Amount       int64    `json:"amount"`
	Source       string   `json:"source"`
	Destination  string   `json:"destination"`
	PrimaryPath  []string `json:"primary_path"`
	ActualPath   []string `json:"actual_path"`
	KilledNode   string   `json:"killed_node"`
	Rerouted     bool     `json:"rerouted"`
	Status       string   `json:"status"`
	StartTime    int64    `json:"start_time"`
	EndTime      int64    `json:"end_time"`
	LatencyMs    int64    `json:"latency_ms"`
}

// HandleAttackDemo handles GET /demo/attack
// Runs the full "Waze moment" demonstration
func (d *ChaosDemo) HandleAttackDemo(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	log.Println("🎬 CHAOS DEMO: Starting attack demonstration...")

	// Demo parameters
	source := "sme_001"
	destination := "sme_003"
	amount := int64(1000000) // $10,000 in cents

	tx := &DemoTransaction{
		ID:          uuid.New().String(),
		Amount:      amount,
		Source:      source,
		Destination: destination,
		StartTime:   time.Now().UnixMilli(),
	}

	// Step 1: Find the primary (best) path
	log.Println("📍 Step 1: Finding primary path...")
	paths, err := d.router.FindKShortestPaths(ctx, source, destination)
	if err != nil || len(paths) == 0 {
		http.Error(w, "Failed to find routes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	primaryPath := paths[0]
	tx.PrimaryPath = primaryPath.Nodes
	log.Printf("   Primary path: %v (fee: %.4f%%)", primaryPath.Nodes, primaryPath.TotalFee*100)

	// Step 2: Start the transaction animation
	log.Println("💸 Step 2: Starting transaction animation...")
	d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
		TransactionID: tx.ID,
		Path:          primaryPath.Nodes,
		CurrentHop:    0,
		Amount:        amount,
		Status:        "in_progress",
	})

	// Animate first hop
	time.Sleep(800 * time.Millisecond)
	d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
		TransactionID: tx.ID,
		Path:          primaryPath.Nodes,
		CurrentHop:    1,
		Amount:        amount,
		Status:        "in_progress",
	})

	// Step 3: Kill a node in the primary path mid-flight
	time.Sleep(600 * time.Millisecond)
	
	// Find a node to kill (not source or destination)
	var nodeToKill string
	if len(primaryPath.Nodes) > 2 {
		nodeToKill = primaryPath.Nodes[1] // Kill the first intermediate node
	} else {
		nodeToKill = primaryPath.Nodes[0]
	}
	tx.KilledNode = nodeToKill

	log.Printf("💥 Step 3: KILLING NODE %s mid-flight!", nodeToKill)
	
	// Trigger the kill
	if d.killFunc != nil {
		d.killFunc(nodeToKill)
	}

	// Broadcast the circuit breaker opening
	d.wsHub.BroadcastCircuitBreaker(&websocket.CircuitBreakerEvent{
		NodeID:    nodeToKill,
		State:     "open",
		PrevState: "closed",
	})

	// Show the failure
	time.Sleep(500 * time.Millisecond)
	d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
		TransactionID: tx.ID,
		Path:          primaryPath.Nodes,
		CurrentHop:    1,
		Amount:        amount,
		Status:        "failed",
	})

	// Step 4: Find alternative route
	log.Println("🔄 Step 4: Finding alternative route...")
	time.Sleep(300 * time.Millisecond)

	// Get second best path (or find new paths excluding killed node)
	var alternatePath *router.Path
	if len(paths) > 1 {
		alternatePath = paths[1]
	} else {
		// Find path that doesn't include the killed node
		for _, p := range paths {
			hasKilledNode := false
			for _, n := range p.Nodes {
				if n == nodeToKill {
					hasKilledNode = true
					break
				}
			}
			if !hasKilledNode {
				alternatePath = p
				break
			}
		}
	}

	if alternatePath == nil {
		log.Println("❌ No alternative path found!")
		tx.Status = "failed_no_route"
		tx.EndTime = time.Now().UnixMilli()
		tx.LatencyMs = tx.EndTime - tx.StartTime
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tx)
		return
	}

	tx.ActualPath = alternatePath.Nodes
	tx.Rerouted = true
	log.Printf("   Alternative path: %v (fee: %.4f%%)", alternatePath.Nodes, alternatePath.TotalFee*100)

	// Step 5: Reroute and complete on alternative path
	log.Println("✨ Step 5: REROUTING to alternative path...")
	
	d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
		TransactionID: tx.ID,
		Path:          alternatePath.Nodes,
		OldPath:       primaryPath.Nodes,
		CurrentHop:    0,
		Amount:        amount,
		Status:        "rerouted",
	})

	// Animate the new path
	for i := 1; i <= len(alternatePath.Nodes)-1; i++ {
		time.Sleep(400 * time.Millisecond)
		d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
			TransactionID: tx.ID,
			Path:          alternatePath.Nodes,
			CurrentHop:    i,
			Amount:        amount,
			Status:        "in_progress",
		})
	}

	// Step 6: Complete the transaction
	time.Sleep(300 * time.Millisecond)
	tx.Status = "completed"
	tx.EndTime = time.Now().UnixMilli()
	tx.LatencyMs = tx.EndTime - tx.StartTime

	d.wsHub.BroadcastPathUpdate(&websocket.PathUpdate{
		TransactionID: tx.ID,
		Path:          alternatePath.Nodes,
		CurrentHop:    len(alternatePath.Nodes) - 1,
		Amount:        amount,
		Status:        "completed",
	})

	log.Printf("✅ DEMO COMPLETE: Transaction rerouted successfully!")
	log.Printf("   Original path: %v", primaryPath.Nodes)
	log.Printf("   Killed node:   %s", nodeToKill)
	log.Printf("   Final path:    %v", alternatePath.Nodes)
	log.Printf("   Total time:    %dms", tx.LatencyMs)

	// Send demo results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"transaction": tx,
		"summary": fmt.Sprintf(
			"Transaction rerouted from %v to %v after killing %s in %dms",
			primaryPath.Nodes, alternatePath.Nodes, nodeToKill, tx.LatencyMs,
		),
	})
}

// HandleResetDemo handles POST /demo/reset
// Revives all killed nodes and resets the demo state
func (d *ChaosDemo) HandleResetDemo(w http.ResponseWriter, r *http.Request) {
	log.Println("🔄 Resetting demo state...")

	// Broadcast all nodes as healthy
	nodes := []string{"lp_alpha", "lp_beta", "lp_gamma", "hub_primary", "hub_secondary", "hub_backup"}
	for _, nodeID := range nodes {
		d.wsHub.BroadcastCircuitBreaker(&websocket.CircuitBreakerEvent{
			NodeID:    nodeID,
			State:     "closed",
			PrevState: "open",
		})
		
		// Mark as active in graph
		if d.graph != nil {
			d.graph.SetNodeActive(nodeID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Demo state reset - all nodes revived",
	})
}
