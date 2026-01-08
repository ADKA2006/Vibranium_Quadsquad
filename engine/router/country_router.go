// Package router implements country-based routing with Yen's K-Shortest Path.
// Uses weighted edges and handles blocked countries.
package router

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"sync"
)

// CountryNode represents a country in the routing graph
type CountryNode struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Currency    string  `json:"currency"`
	Credibility float64 `json:"credibility"` // 0-1, higher is better
	SuccessRate float64 `json:"success_rate"` // 0-1, higher is better
	FXRate      float64 `json:"fx_rate"`      // Exchange rate to USD
	IsActive    bool    `json:"is_active"`
}

// CountryEdge represents a trade connection between countries
type CountryEdge struct {
	SourceCode string  `json:"source_code"`
	TargetCode string  `json:"target_code"`
	BaseCost   float64 `json:"base_cost"` // Base transaction cost (0-1)
	IsActive   bool    `json:"is_active"`
}

// CountryPath represents a calculated route with fees
type CountryPath struct {
	Nodes          []string  `json:"nodes"`           // Country codes in order
	TotalWeight    float64   `json:"total_weight"`    // Sum of edge weights
	TotalFeePercent float64  `json:"total_fee_percent"` // Total fees as percentage
	HopCount       int       `json:"hop_count"`       // Number of hops
	FinalAmount    float64   `json:"final_amount"`    // Amount after fees (per 1.0 input)
}

// CountryGraph holds the routing graph with countries
type CountryGraph struct {
	mu       sync.RWMutex
	nodes    map[string]*CountryNode
	edges    map[string]map[string]*CountryEdge // source -> target -> edge
	blocked  map[string]bool                    // Blocked country codes
}

// NewCountryGraph creates a new country routing graph
func NewCountryGraph() *CountryGraph {
	return &CountryGraph{
		nodes:   make(map[string]*CountryNode),
		edges:   make(map[string]map[string]*CountryEdge),
		blocked: make(map[string]bool),
	}
}

// AddNode adds a country node
func (g *CountryGraph) AddNode(node *CountryNode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.Code] = node
}

// AddEdge adds a trading edge between countries
func (g *CountryGraph) AddEdge(edge *CountryEdge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.edges[edge.SourceCode] == nil {
		g.edges[edge.SourceCode] = make(map[string]*CountryEdge)
	}
	g.edges[edge.SourceCode][edge.TargetCode] = edge
	
	// Also add reverse edge (bidirectional trading)
	if g.edges[edge.TargetCode] == nil {
		g.edges[edge.TargetCode] = make(map[string]*CountryEdge)
	}
	g.edges[edge.TargetCode][edge.SourceCode] = &CountryEdge{
		SourceCode: edge.TargetCode,
		TargetCode: edge.SourceCode,
		BaseCost:   edge.BaseCost,
		IsActive:   edge.IsActive,
	}
}

// SetBlocked updates the set of blocked countries
func (g *CountryGraph) SetBlocked(blockedCodes []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.blocked = make(map[string]bool)
	for _, code := range blockedCodes {
		g.blocked[code] = true
	}
}

// IsBlocked checks if a country is blocked
func (g *CountryGraph) IsBlocked(code string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.blocked[code]
}

// GetEdgeWeight calculates the edge weight using the formula:
// Weight = 0.8 * Cost + 0.1 * (1 - Credibility) + 0.1 * (1 - SuccessRate)
// 
// Where:
// - Cost is the base transaction cost
// - Credibility is the target country's credibility (0-1)
// - SuccessRate is the target country's success rate (0-1)
func (g *CountryGraph) GetEdgeWeight(edge *CountryEdge) float64 {
	targetNode := g.nodes[edge.TargetCode]
	if targetNode == nil {
		return edge.BaseCost // Fallback to just cost
	}
	
	cost := edge.BaseCost
	credibility := targetNode.Credibility
	successRate := targetNode.SuccessRate
	
	// Weight formula: 0.8 * Cost + 0.1 * (1 - Credibility) + 0.1 * (1 - SuccessRate)
	weight := 0.8*cost + 0.1*(1-credibility) + 0.1*(1-successRate)
	
	return weight
}

// CountryRouter provides K-shortest path finding for countries
type CountryRouter struct {
	graph           *CountryGraph
	k               int     // Number of paths to find (default 3)
	hopFeePercent   float64 // Fee per hop (default 0.0002 = 0.02%)
}

// NewCountryRouter creates a new country router
func NewCountryRouter(graph *CountryGraph, k int) *CountryRouter {
	if k <= 0 {
		k = 3
	}
	return &CountryRouter{
		graph:         graph,
		k:             k,
		hopFeePercent: 0.0002, // 0.02% per hop
	}
}

// FindKShortestPaths finds the K shortest paths between countries
// blockedCodes are countries to exclude from routing
func (r *CountryRouter) FindKShortestPaths(ctx context.Context, source, target string, blockedCodes []string) ([]*CountryPath, error) {
	r.graph.mu.RLock()
	defer r.graph.mu.RUnlock()
	
	// Build blocked set
	blocked := make(map[string]bool)
	for _, code := range blockedCodes {
		blocked[code] = true
	}
	// Also add graph-level blocked
	for code := range r.graph.blocked {
		blocked[code] = true
	}
	
	// Check source and target aren't blocked
	if blocked[source] {
		return nil, fmt.Errorf("source country %s is blocked", source)
	}
	if blocked[target] {
		return nil, fmt.Errorf("target country %s is blocked", target)
	}
	
	// Verify nodes exist
	if _, ok := r.graph.nodes[source]; !ok {
		return nil, fmt.Errorf("source country not found: %s", source)
	}
	if _, ok := r.graph.nodes[target]; !ok {
		return nil, fmt.Errorf("target country not found: %s", target)
	}
	
	// Find shortest path first using Dijkstra
	shortestPath := r.dijkstra(source, target, nil, blocked)
	if shortestPath == nil {
		return nil, fmt.Errorf("no path found from %s to %s", source, target)
	}
	
	// Calculate fees for the path
	r.calculatePathFees(shortestPath)
	
	A := []*CountryPath{shortestPath}
	
	// Min-heap of candidate paths
	B := &countryPathHeap{}
	heap.Init(B)
	
	// Yen's algorithm
	for k := 1; k < r.k; k++ {
		if ctx.Err() != nil {
			return A, ctx.Err()
		}
		
		prevPath := A[k-1]
		
		for i := 0; i < len(prevPath.Nodes)-1; i++ {
			spurNode := prevPath.Nodes[i]
			rootPath := prevPath.Nodes[:i+1]
			
			excludedEdges := make(map[string]bool)
			excludedNodes := make(map[string]bool)
			
			// Copy blocked nodes
			for code := range blocked {
				excludedNodes[code] = true
			}
			
			// Exclude edges sharing this root
			for _, path := range A {
				if len(path.Nodes) > i && pathsSharePrefixCountry(path.Nodes, rootPath) {
					if i+1 < len(path.Nodes) {
						edgeKey := path.Nodes[i] + "->" + path.Nodes[i+1]
						excludedEdges[edgeKey] = true
					}
				}
			}
			
			// Exclude root nodes except spur
			for j := 0; j < i; j++ {
				excludedNodes[prevPath.Nodes[j]] = true
			}
			
			spurPath := r.dijkstra(spurNode, target, excludedEdges, excludedNodes)
			
			if spurPath != nil {
				totalPath := r.combinePaths(rootPath, spurPath)
				r.calculatePathFees(totalPath)
				
				if !containsCountryPath(A, totalPath) && !heapContainsCountryPath(B, totalPath) {
					heap.Push(B, totalPath)
				}
			}
		}
		
		if B.Len() == 0 {
			break
		}
		
		bestCandidate := heap.Pop(B).(*CountryPath)
		A = append(A, bestCandidate)
	}
	
	return A, nil
}

// dijkstra finds shortest path using Dijkstra's algorithm
func (r *CountryRouter) dijkstra(source, target string, excludedEdges, excludedNodes map[string]bool) *CountryPath {
	if excludedNodes[source] || excludedNodes[target] {
		return nil
	}
	
	dist := make(map[string]float64)
	prev := make(map[string]string)
	
	for nodeCode := range r.graph.nodes {
		dist[nodeCode] = math.Inf(1)
	}
	dist[source] = 0
	
	pq := &countryDijkstraHeap{{node: source, dist: 0}}
	heap.Init(pq)
	
	visited := make(map[string]bool)
	
	for pq.Len() > 0 {
		current := heap.Pop(pq).(*countryDijkstraItem)
		
		if visited[current.node] {
			continue
		}
		visited[current.node] = true
		
		if current.node == target {
			break
		}
		
		neighbors := r.graph.edges[current.node]
		for targetCode, edge := range neighbors {
			if !edge.IsActive {
				continue
			}
			if excludedNodes[targetCode] {
				continue
			}
			edgeKey := current.node + "->" + targetCode
			if excludedEdges[edgeKey] {
				continue
			}
			
			weight := r.graph.GetEdgeWeight(edge)
			newDist := dist[current.node] + weight
			
			if newDist < dist[targetCode] {
				dist[targetCode] = newDist
				prev[targetCode] = current.node
				heap.Push(pq, &countryDijkstraItem{node: targetCode, dist: newDist})
			}
		}
	}
	
	if dist[target] == math.Inf(1) {
		return nil
	}
	
	// Reconstruct path
	path := &CountryPath{
		Nodes:       []string{},
		TotalWeight: dist[target],
	}
	
	current := target
	for current != "" {
		path.Nodes = append([]string{current}, path.Nodes...)
		current = prev[current]
	}
	
	return path
}

// combinePaths combines root path with spur path
func (r *CountryRouter) combinePaths(rootNodes []string, spurPath *CountryPath) *CountryPath {
	combined := &CountryPath{
		Nodes: make([]string, 0, len(rootNodes)+len(spurPath.Nodes)-1),
	}
	
	combined.Nodes = append(combined.Nodes, rootNodes...)
	
	// Calculate weight for root edges
	for i := 0; i < len(rootNodes)-1; i++ {
		if edges, ok := r.graph.edges[rootNodes[i]]; ok {
			if edge, ok := edges[rootNodes[i+1]]; ok {
				combined.TotalWeight += r.graph.GetEdgeWeight(edge)
			}
		}
	}
	
	if len(spurPath.Nodes) > 1 {
		combined.Nodes = append(combined.Nodes, spurPath.Nodes[1:]...)
	}
	combined.TotalWeight += spurPath.TotalWeight
	
	return combined
}

// calculatePathFees calculates the transaction fees for a path
// Each hop deducts 0.02% from the amount
func (r *CountryRouter) calculatePathFees(path *CountryPath) {
	path.HopCount = len(path.Nodes) - 1
	
	// Calculate total fee percentage
	// For n hops: finalAmount = (1 - hopFee)^n
	// Fee = 1 - finalAmount
	if path.HopCount > 0 {
		path.FinalAmount = math.Pow(1-r.hopFeePercent, float64(path.HopCount))
		path.TotalFeePercent = (1 - path.FinalAmount) * 100 // As percentage
	} else {
		path.FinalAmount = 1.0
		path.TotalFeePercent = 0
	}
}

// Helper functions
func pathsSharePrefixCountry(path, prefix []string) bool {
	if len(prefix) > len(path) {
		return false
	}
	for i := range prefix {
		if path[i] != prefix[i] {
			return false
		}
	}
	return true
}

func containsCountryPath(paths []*CountryPath, path *CountryPath) bool {
	for _, p := range paths {
		if pathsEqualCountry(p.Nodes, path.Nodes) {
			return true
		}
	}
	return false
}

func heapContainsCountryPath(h *countryPathHeap, path *CountryPath) bool {
	for _, p := range *h {
		if pathsEqualCountry(p.Nodes, path.Nodes) {
			return true
		}
	}
	return false
}

func pathsEqualCountry(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Heap implementations
type countryPathHeap []*CountryPath

func (h countryPathHeap) Len() int           { return len(h) }
func (h countryPathHeap) Less(i, j int) bool { return h[i].TotalWeight < h[j].TotalWeight }
func (h countryPathHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *countryPathHeap) Push(x interface{}) {
	*h = append(*h, x.(*CountryPath))
}

func (h *countryPathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type countryDijkstraItem struct {
	node string
	dist float64
}

type countryDijkstraHeap []*countryDijkstraItem

func (h countryDijkstraHeap) Len() int           { return len(h) }
func (h countryDijkstraHeap) Less(i, j int) bool { return h[i].dist < h[j].dist }
func (h countryDijkstraHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *countryDijkstraHeap) Push(x interface{}) {
	*h = append(*h, x.(*countryDijkstraItem))
}

func (h *countryDijkstraHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
