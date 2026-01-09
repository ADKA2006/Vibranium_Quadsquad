// Package main provides the main entry point for the Predictive Liquidity Mesh server.
// Combines all components: WebSocket, API, chaos demo, routing engine, and FX rate worker.
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
	"github.com/plm/predictive-liquidity-mesh/api/middleware"
	"github.com/plm/predictive-liquidity-mesh/auth"
	"github.com/plm/predictive-liquidity-mesh/demo"
	"github.com/plm/predictive-liquidity-mesh/engine/router"
	"github.com/plm/predictive-liquidity-mesh/payments"
	neo4jstore "github.com/plm/predictive-liquidity-mesh/storage/neo4j"
	"github.com/plm/predictive-liquidity-mesh/storage/users"
	"github.com/plm/predictive-liquidity-mesh/websocket"
	"github.com/plm/predictive-liquidity-mesh/workers/fxrates"
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

	// Initialize PASETO token manager
	tokenConfig, err := auth.DefaultTokenConfig()
	if err != nil {
		log.Fatalf("Failed to load token config: %v", err)
	}
	tokenManager, err := auth.NewTokenManager(tokenConfig)
	if err != nil {
		log.Fatalf("Failed to create token manager: %v", err)
	}

	// Initialize user store with default admin/user accounts
	userStore := users.NewStore()
	log.Println("✅ User store initialized with default accounts")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	// Try to connect to Neo4j (non-blocking)
	var neo4jClient *neo4jstore.Client
	var neo4jDriver interface {
		Close(context.Context) error
	}
	neo4jCfg := neo4jstore.DefaultConfig()
	neo4jClient, err = neo4jstore.NewClient(ctx, neo4jCfg)
	if err != nil {
		log.Printf("⚠️  Neo4j not available: %v (continuing without Neo4j)", err)
	} else {
		log.Println("✅ Connected to Neo4j")
		neo4jDriver = neo4jClient

		// Bootstrap countries in Neo4j
		go func() {
			bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer bootstrapCancel()
			if err := neo4jstore.BootstrapCountries(bootstrapCtx, neo4jClient.Driver(), neo4jCfg.Database); err != nil {
				log.Printf("⚠️  Failed to bootstrap countries: %v", err)
			}
		}()

		// Start FX rate worker
		fxConfig := fxrates.DefaultConfig()
		fxConfig.Driver = neo4jClient.Driver()
		fxConfig.Database = neo4jCfg.Database
		fxConfig.Currencies = neo4jstore.GetAllCurrencies()
		fxWorker := fxrates.NewWorker(fxConfig)
		go fxWorker.Start(ctx)
	}

	// Initialize handlers
	chaosHandler := handlers.NewChaosHandler(nil, meshRouter, graph, wsHub)
	chaosDemo := demo.NewChaosDemo(meshRouter, graph, wsHub, func(nodeID string) error {
		graph.SetNodeInactive(nodeID)
		return nil
	})
	authHandler := handlers.NewAuthHandler(tokenManager)
	authHandler.SetUserStore(userStore)
	adminHandler := handlers.NewAdminHandler(graph, neo4jClient, wsHub)
	userHandler := handlers.NewUserHandler(meshRouter, graph)

	// Initialize country handler only if Neo4j is available
	var countryHandler *handlers.CountryHandler
	var countryGraph *router.CountryGraph
	if neo4jClient != nil {
		countryHandler = handlers.NewCountryHandler(neo4jClient.Driver(), neo4jCfg.Database)

		// Build country routing graph from Neo4j
		var err error
		countryGraph, err = router.BuildCountryGraphFromNeo4j(ctx, neo4jClient.Driver(), neo4jCfg.Database)
		if err != nil {
			log.Printf("⚠️  Failed to build country graph from Neo4j: %v", err)
			countryGraph = router.BuildCountryGraphWithDefaults()
			log.Println("📊 Using default country graph")
		} else {
			log.Println("✅ Country routing graph initialized from Neo4j")
		}
	} else {
		// Use defaults if Neo4j not available
		countryGraph = router.BuildCountryGraphWithDefaults()
		log.Println("📊 Country routing graph initialized with defaults")
	}

	// Initialize route handler
	routeHandler := handlers.NewRouteHandler(countryGraph)

	// Initialize payment system
	txnStore := payments.NewTransactionStore()
	
	// Set up credibility callback if Neo4j is available
	if neo4jClient != nil {
		credUpdater := neo4jstore.NewCredibilityUpdater(neo4jClient.Driver(), neo4jCfg.Database)
		txnStore.SetCredibilityCallback(func(countryCode string, success bool) {
			go func() {
				updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				credUpdater.UpdateCredibility(updateCtx, countryCode, success)
			}()
		})
		log.Println("✅ Payment system initialized with credibility tracking")
	} else {
		log.Println("📊 Payment system initialized (no credibility tracking)")
	}
	
	paymentHandler := handlers.NewPaymentHandler(txnStore, countryGraph)
	receiptHandler := handlers.NewReceiptHandler(txnStore)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// CORS middleware for Next.js frontend
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	// Public endpoints
	mux.HandleFunc("/ws", wsHub.ServeWS)
	mux.HandleFunc("/ws/route", routeHandler.HandleRouteWS) // WebSocket for route calculation
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Auth endpoints (public)
	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/v1/auth/register", authHandler.HandleRegister)

	// Protected User endpoints (require auth)
	mux.Handle("/api/v1/settle/preview", authMiddleware.Authenticate(http.HandlerFunc(userHandler.HandleSettlePreview)))
	mux.Handle("/api/v1/route", authMiddleware.Authenticate(http.HandlerFunc(routeHandler.HandleRouteHTTP)))
	
	// Payment endpoints (require auth + regular user only - admins cannot make payments)
	mux.Handle("/api/v1/payments/create", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireUser,
	)(http.HandlerFunc(paymentHandler.HandleCreatePayment)))
	mux.Handle("/api/v1/payments/confirm", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireUser,
	)(http.HandlerFunc(paymentHandler.HandleConfirmPayment)))
	mux.Handle("/api/v1/payments/history", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.HandleGetHistory)))
	mux.Handle("/api/v1/payments/transaction", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.HandleGetTransaction)))
	mux.Handle("/api/v1/payments/charts", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.HandleChartData)))
	mux.HandleFunc("/api/v1/receipts/", receiptHandler.HandleDownloadReceipt) // Public: allow receipt downloads
	
	// Stripe payment endpoints (Endpoint A and B - regular users only)
	mux.Handle("/api/v1/stripe/initiate", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireUser,
	)(http.HandlerFunc(paymentHandler.HandleStripeInitiate)))
	mux.Handle("/api/v1/stripe/complete", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireUser,
	)(http.HandlerFunc(paymentHandler.HandleStripeComplete)))
	mux.HandleFunc("/api/v1/stripe/config", paymentHandler.HandleStripeConfig) // Public: returns publishable key

	// Protected Admin endpoints (require auth + admin role)
	mux.Handle("/api/v1/admin/nodes", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireAdmin,
	)(http.HandlerFunc(adminHandler.HandleCreateNode)))
	mux.Handle("/api/v1/admin/edges", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireAdmin,
	)(http.HandlerFunc(adminHandler.HandleCreateEdge)))

	// Country admin endpoints (if Neo4j available)
	if countryHandler != nil {
		mux.Handle("/api/v1/admin/countries", middleware.Chain(
			authMiddleware.Authenticate,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				countryHandler.HandleListCountries(w, r)
			case http.MethodPost:
				countryHandler.HandleCreateCountry(w, r)
			default:
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			}
		})))
		mux.Handle("/api/v1/admin/countries/", middleware.Chain(
			authMiddleware.Authenticate,
			authMiddleware.RequireAdmin,
		)(http.HandlerFunc(countryHandler.HandleDeleteCountry)))
	}

	// Admin payment stats (admin only)
	mux.Handle("/api/v1/admin/payments/stats", middleware.Chain(
		authMiddleware.Authenticate,
		authMiddleware.RequireAdmin,
	)(http.HandlerFunc(paymentHandler.HandleAdminStats)))

	// Debug/Chaos endpoints
	mux.HandleFunc("/debug/kill/", chaosHandler.HandleKillNode)
	mux.HandleFunc("/debug/revive/", chaosHandler.HandleReviveNode)
	mux.HandleFunc("/debug/killed", chaosHandler.HandleGetKilledNodes)

	// Demo endpoints
	mux.HandleFunc("/demo/attack", chaosDemo.HandleAttackDemo)
	mux.HandleFunc("/demo/reset", chaosDemo.HandleResetDemo)

	// Static files for frontend (now points to Next.js build output)
	fs := http.FileServer(http.Dir("./frontend-next/out"))
	mux.Handle("/", fs)

	// Create server with CORS and security middleware
	// Security middleware chain: InputValidation -> SecurityHeaders -> CSRFMiddleware -> corsHandler
	securityHandler := func(h http.Handler) http.Handler {
		return middleware.InputValidation(
			middleware.SecurityHeaders(
				middleware.CSRFMiddleware(h),
			),
		)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: securityHandler(corsHandler(mux)),
	}

	// Start server in goroutine
	go func() {
		log.Println("📡 HTTP/WebSocket server listening on :8080")
		log.Println("   - Dashboard:    http://localhost:8080/")
		log.Println("   - WebSocket:    ws://localhost:8080/ws")
		log.Println("   - Route WS:     ws://localhost:8080/ws/route")
		log.Println("   - Route API:    POST /api/v1/route")
		log.Println("   - Login:        POST /api/v1/auth/login")
		log.Println("   - Register:     POST /api/v1/auth/register")
		log.Println("   - Countries:    GET /api/v1/admin/countries")
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

	if neo4jDriver != nil {
		neo4jDriver.Close(shutdownCtx)
	}

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
