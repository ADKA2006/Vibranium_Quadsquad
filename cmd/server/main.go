// Package main provides the main entry point for the Predictive Liquidity Mesh server.
// Combines all components: WebSocket, API, chaos demo, and routing engine.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/plm/predictive-liquidity-mesh/api/handlers"
	"github.com/plm/predictive-liquidity-mesh/demo"
	"github.com/plm/predictive-liquidity-mesh/engine/router"
	"github.com/plm/predictive-liquidity-mesh/websocket"
)

func main() {
	log.Println("🚀 Starting Predictive Liquidity Mesh Server...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize the mesh graph with sample topology
	graph := initializeMeshGraph()
	meshRouter := router.NewRouter(graph, 3)

	// Initialize WebSocket hub
	wsServer := websocket.NewServer(":8080")
	wsHub := wsServer.Hub()

	// Start WebSocket hub
	go wsHub.Run(ctx)

	// Initialize chaos handler (without Redis for demo)
	chaosHandler := handlers.NewChaosHandler(nil, meshRouter, graph, wsHub)

	// Initialize demo
	chaosDemo := demo.NewChaosDemo(meshRouter, graph, wsHub, func(nodeID string) error {
		graph.SetNodeInactive(nodeID)
		return nil
	})

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHub.ServeWS)

	// Debug/Chaos endpoints
	mux.HandleFunc("/debug/kill/", chaosHandler.HandleKillNode)
	mux.HandleFunc("/debug/revive/", chaosHandler.HandleReviveNode)
	mux.HandleFunc("/debug/killed", chaosHandler.HandleGetKilledNodes)

	// Demo endpoints
	mux.HandleFunc("/demo/attack", chaosDemo.HandleAttackDemo)
	mux.HandleFunc("/demo/reset", chaosDemo.HandleResetDemo)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static files for frontend
	fs := http.FileServer(http.Dir("./frontend/public"))
	mux.Handle("/", fs)

	// Create server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Println("📡 HTTP/WebSocket server listening on :8080")
		log.Println("   - Dashboard:    http://localhost:8080/")
		log.Println("   - WebSocket:    ws://localhost:8080/ws")
		log.Println("   - Attack Demo:  http://localhost:8080/demo/attack")
		log.Println("   - Kill Node:    POST /debug/kill/{node_id}")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// initializeMeshGraph creates the sample mesh topology
func initializeMeshGraph() *router.Graph {
	graph := router.NewGraph()

	// Add SME nodes
	graph.AddNode(&router.Node{ID: "sme_001", Type: "SME", IsActive: true})
	graph.AddNode(&router.Node{ID: "sme_002", Type: "SME", IsActive: true})
	graph.AddNode(&router.Node{ID: "sme_003", Type: "SME", IsActive: true})
	graph.AddNode(&router.Node{ID: "sme_004", Type: "SME", IsActive: true})
	graph.AddNode(&router.Node{ID: "sme_005", Type: "SME", IsActive: true})

	// Add Liquidity Provider nodes
	graph.AddNode(&router.Node{ID: "lp_alpha", Type: "LiquidityProvider", IsActive: true})
	graph.AddNode(&router.Node{ID: "lp_beta", Type: "LiquidityProvider", IsActive: true})
	graph.AddNode(&router.Node{ID: "lp_gamma", Type: "LiquidityProvider", IsActive: true})

	// Add Hub nodes
	graph.AddNode(&router.Node{ID: "hub_primary", Type: "Hub", IsActive: true})
	graph.AddNode(&router.Node{ID: "hub_secondary", Type: "Hub", IsActive: true})
	graph.AddNode(&router.Node{ID: "hub_backup", Type: "Hub", IsActive: true})

	// Add edges - SME to LP
	graph.AddEdge(&router.Edge{SourceID: "sme_001", TargetID: "lp_alpha", BaseFee: 0.0008, Latency: 5, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_001", TargetID: "lp_beta", BaseFee: 0.0015, Latency: 45, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_002", TargetID: "lp_alpha", BaseFee: 0.0005, Latency: 8, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_002", TargetID: "lp_gamma", BaseFee: 0.0012, Latency: 95, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_003", TargetID: "lp_beta", BaseFee: 0.0007, Latency: 10, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_004", TargetID: "lp_gamma", BaseFee: 0.0010, Latency: 12, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "sme_005", TargetID: "lp_beta", BaseFee: 0.0009, Latency: 18, IsActive: true})

	// Add edges - LP to Hub
	graph.AddEdge(&router.Edge{SourceID: "lp_alpha", TargetID: "hub_primary", BaseFee: 0.0015, Latency: 12, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "lp_beta", TargetID: "hub_primary", BaseFee: 0.0018, Latency: 25, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "lp_beta", TargetID: "hub_secondary", BaseFee: 0.0012, Latency: 8, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "lp_gamma", TargetID: "hub_backup", BaseFee: 0.0010, Latency: 15, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "lp_gamma", TargetID: "hub_primary", BaseFee: 0.0022, Latency: 85, IsActive: true})

	// Add edges - Hub interconnects
	graph.AddEdge(&router.Edge{SourceID: "hub_primary", TargetID: "hub_secondary", BaseFee: 0.0005, Latency: 35, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_secondary", TargetID: "hub_primary", BaseFee: 0.0005, Latency: 35, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_primary", TargetID: "hub_backup", BaseFee: 0.0008, Latency: 75, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_backup", TargetID: "hub_primary", BaseFee: 0.0008, Latency: 75, IsActive: true})

	// Add edges - Hub to destination SMEs
	graph.AddEdge(&router.Edge{SourceID: "hub_primary", TargetID: "sme_003", BaseFee: 0.0006, Latency: 10, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_backup", TargetID: "sme_003", BaseFee: 0.0009, Latency: 20, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_backup", TargetID: "sme_004", BaseFee: 0.0008, Latency: 15, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_secondary", TargetID: "sme_005", BaseFee: 0.0007, Latency: 12, IsActive: true})
	graph.AddEdge(&router.Edge{SourceID: "hub_secondary", TargetID: "sme_003", BaseFee: 0.0008, Latency: 18, IsActive: true})

	log.Println("✅ Mesh graph initialized with topology")
	return graph
}
