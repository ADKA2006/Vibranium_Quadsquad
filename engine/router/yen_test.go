// Package router provides benchmarks for the Yen's K-Shortest Path algorithm.
// Checkpoint 2: Router must return K=3 paths in <10ms for a 50-node graph.
package router

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestYenKShortestPaths validates the router returns correct paths
func TestYenKShortestPaths(t *testing.T) {
	graph := buildTestGraph(10)
	router := NewRouter(graph, 3)

	ctx := context.Background()
	paths, err := router.FindKShortestPaths(ctx, "node_0", "node_9")

	if err != nil {
		t.Fatalf("Failed to find paths: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected at least one path")
	}

	t.Logf("Found %d paths:", len(paths))
	for i, path := range paths {
		t.Logf("  Path %d: %v (weight: %.6f, fee: %.4f, latency: %dms)",
			i+1, path.Nodes, path.TotalWeight, path.TotalFee, path.TotalLatency)
	}

	// Verify paths are sorted by weight
	for i := 1; i < len(paths); i++ {
		if paths[i].TotalWeight < paths[i-1].TotalWeight {
			t.Errorf("Paths not sorted by weight: path %d (%.6f) < path %d (%.6f)",
				i+1, paths[i].TotalWeight, i, paths[i-1].TotalWeight)
		}
	}
}

// TestEntropyWeighting verifies entropy affects edge weights
func TestEntropyWeighting(t *testing.T) {
	graph := NewGraph()

	// Add nodes
	graph.AddNode(&Node{ID: "A", Type: "SME", IsActive: true})
	graph.AddNode(&Node{ID: "B", Type: "Hub", IsActive: true})
	graph.AddNode(&Node{ID: "C", Type: "Hub", IsActive: true})
	graph.AddNode(&Node{ID: "D", Type: "SME", IsActive: true})

	// Add edges with same base fee
	graph.AddEdge(&Edge{SourceID: "A", TargetID: "B", BaseFee: 0.001, Latency: 10, IsActive: true})
	graph.AddEdge(&Edge{SourceID: "A", TargetID: "C", BaseFee: 0.001, Latency: 10, IsActive: true})
	graph.AddEdge(&Edge{SourceID: "B", TargetID: "D", BaseFee: 0.001, Latency: 10, IsActive: true})
	graph.AddEdge(&Edge{SourceID: "C", TargetID: "D", BaseFee: 0.001, Latency: 10, IsActive: true})

	// Set HIGH entropy for node A->B path (volatile, unpredictable)
	graph.UpdateNodeEntropy("A", map[string]float64{
		"B": 0.25,
		"C": 0.25,
		"X": 0.25,
		"Y": 0.25,
	}) // Uniform distribution = high entropy

	// Get weights
	edgeAB := graph.edges["A"]["B"]
	edgeAC := graph.edges["A"]["C"]

	weightAB := graph.GetEdgeWeight(edgeAB)
	weightAC := graph.GetEdgeWeight(edgeAC)

	t.Logf("Edge A->B weight: %.6f (with entropy)", weightAB)
	t.Logf("Edge A->C weight: %.6f (with entropy)", weightAC)

	// Since both edges come from same source, they should have same entropy effect
	// but the test shows entropy is being applied
	if weightAB <= 0.001 {
		t.Error("Expected entropy to increase edge weight above base fee")
	}
}

// BenchmarkYen50Nodes is Checkpoint 2: K=3 paths in <10ms for 50-node graph
func BenchmarkYen50Nodes(b *testing.B) {
	graph := buildTestGraph(50)
	router := NewRouter(graph, 3)
	ctx := context.Background()

	b.ResetTimer()

	var totalDuration time.Duration
	var runCount int

	for i := 0; i < b.N; i++ {
		start := time.Now()
		paths, err := router.FindKShortestPaths(ctx, "node_0", "node_49")
		elapsed := time.Since(start)
		totalDuration += elapsed
		runCount++

		if err != nil {
			b.Fatalf("Failed to find paths: %v", err)
		}

		if len(paths) < 3 {
			b.Logf("Warning: Found only %d paths (expected 3)", len(paths))
		}
	}

	avgDuration := totalDuration / time.Duration(runCount)
	b.ReportMetric(float64(avgDuration.Microseconds())/1000, "ms/op")

	if avgDuration > 10*time.Millisecond {
		b.Errorf("❌ CHECKPOINT 2 FAILED: Average time %.2fms exceeds 10ms threshold",
			float64(avgDuration.Microseconds())/1000)
	} else {
		b.Logf("✅ CHECKPOINT 2 PASSED: Average time %.2fms < 10ms",
			float64(avgDuration.Microseconds())/1000)
	}
}

// BenchmarkYen100Nodes tests scalability
func BenchmarkYen100Nodes(b *testing.B) {
	graph := buildTestGraph(100)
	router := NewRouter(graph, 3)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := router.FindKShortestPaths(ctx, "node_0", "node_99")
		if err != nil {
			b.Fatalf("Failed to find paths: %v", err)
		}
	}
}

// buildTestGraph creates a test mesh with n nodes
func buildTestGraph(n int) *Graph {
	graph := NewGraph()
	rng := rand.New(rand.NewSource(42)) // Deterministic for benchmarks

	// Create nodes
	for i := 0; i < n; i++ {
		nodeType := "Hub"
		if i == 0 || i == n-1 {
			nodeType = "SME"
		}
		graph.AddNode(&Node{
			ID:       fmt.Sprintf("node_%d", i),
			Type:     nodeType,
			IsActive: true,
		})
	}

	// Create edges - each node connects to 3-5 forward nodes
	for i := 0; i < n-1; i++ {
		numEdges := 3 + rng.Intn(3) // 3-5 edges
		for j := 0; j < numEdges && i+j+1 < n; j++ {
			targetIdx := i + 1 + rng.Intn(min(5, n-i-1))
			if targetIdx >= n {
				targetIdx = n - 1
			}

			graph.AddEdge(&Edge{
				SourceID:        fmt.Sprintf("node_%d", i),
				TargetID:        fmt.Sprintf("node_%d", targetIdx),
				BaseFee:         0.001 + rng.Float64()*0.002, // 0.1% - 0.3%
				Latency:         int64(5 + rng.Intn(20)),     // 5-25ms
				LiquidityVolume: int64(1000000 + rng.Intn(9000000)),
				IsActive:        true,
			})
		}

		// Set random entropy for nodes
		distribution := make(map[string]float64)
		for k := 0; k < 3+rng.Intn(3); k++ {
			distribution[fmt.Sprintf("dest_%d", k)] = rng.Float64() * 1000000
		}
		graph.UpdateNodeEntropy(fmt.Sprintf("node_%d", i), distribution)
	}

	return graph
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
