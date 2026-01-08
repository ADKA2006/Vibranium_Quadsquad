// Package router implements Yen's K-Shortest Paths algorithm with entropy-based weighting.
// Provides optimal route discovery for the Predictive Liquidity Mesh.
package router

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/plm/predictive-liquidity-mesh/pkg/entropy"
)

// Graph represents the liquidity mesh topology
type Graph struct {
	mu       sync.RWMutex
	nodes    map[string]*Node
	edges    map[string]map[string]*Edge // source -> target -> edge
	entropy  map[string]*entropy.NodeEntropy
}

// Node represents a mesh node (SME, LiquidityProvider, or Hub)
type Node struct {
	ID       string
	Type     string // "SME", "LiquidityProvider", "Hub"
	Region   string
	IsActive bool
	Props    map[string]interface{}
}

// Edge represents a liquidity edge between nodes
type Edge struct {
	SourceID        string
	TargetID        string
	BaseFee         float64 // Base fee percentage (e.g., 0.0015 = 0.15%)
	Latency         int64   // Latency in milliseconds
	LiquidityVolume int64   // Available liquidity
	IsActive        bool
}

// Path represents a route through the mesh
type Path struct {
	Nodes       []string  `json:"nodes"`
	Edges       []*Edge   `json:"edges"`
	TotalWeight float64   `json:"total_weight"`
	TotalFee    float64   `json:"total_fee"`
	TotalLatency int64    `json:"total_latency"`
}

// NewGraph creates a new graph instance
func NewGraph() *Graph {
	return &Graph{
		nodes:   make(map[string]*Node),
		edges:   make(map[string]map[string]*Edge),
		entropy: make(map[string]*entropy.NodeEntropy),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.ID] = node
}

// AddEdge adds an edge to the graph
func (g *Graph) AddEdge(edge *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.edges[edge.SourceID] == nil {
		g.edges[edge.SourceID] = make(map[string]*Edge)
	}
	g.edges[edge.SourceID][edge.TargetID] = edge
}

// UpdateNodeEntropy updates the entropy data for a node
func (g *Graph) UpdateNodeEntropy(nodeID string, distribution map[string]float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entropy[nodeID] = entropy.CalculateNodeEntropy(nodeID, distribution)
}

// SetNodeActive marks a node as active
func (g *Graph) SetNodeActive(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if node, ok := g.nodes[nodeID]; ok {
		node.IsActive = true
	}
}

// SetNodeInactive marks a node as inactive (for chaos testing)
func (g *Graph) SetNodeInactive(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if node, ok := g.nodes[nodeID]; ok {
		node.IsActive = false
	}
}

// GetNode returns a node by ID
func (g *Graph) GetNode(nodeID string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[nodeID]
}

// IsNodeActive checks if a node is active
func (g *Graph) IsNodeActive(nodeID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if node, ok := g.nodes[nodeID]; ok {
		return node.IsActive
	}
	return false
}

// GetEdgeWeight calculates the entropy-weighted edge weight.
// Formula: W = Fee × (1 + H), where H is Shannon entropy.
func (g *Graph) GetEdgeWeight(edge *Edge) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.getEdgeWeightUnlocked(edge)
}

// getEdgeWeightUnlocked calculates edge weight without acquiring lock.
// Caller must hold at least RLock.
func (g *Graph) getEdgeWeightUnlocked(edge *Edge) float64 {
	// Get source node entropy
	H := 0.0
	if nodeEntropy, ok := g.entropy[edge.SourceID]; ok {
		H = nodeEntropy.Volatility()
	}
	
	// W = Fee × (1 + H)
	// Higher entropy = higher weight = less preferred path
	weight := edge.BaseFee * (1.0 + H)
	
	// Add small latency component to break ties
	weight += float64(edge.Latency) * 0.00001
	
	return weight
}

// Router provides path-finding capabilities
type Router struct {
	graph *Graph
	k     int // Number of paths to find
}

// NewRouter creates a new router with the specified K value
func NewRouter(graph *Graph, k int) *Router {
	if k <= 0 {
		k = 3 // Default to 3 shortest paths
	}
	return &Router{graph: graph, k: k}
}

// FindKShortestPaths implements Yen's algorithm to find K shortest paths.
// Returns up to K alternative routes from source to target.
func (r *Router) FindKShortestPaths(ctx context.Context, source, target string) ([]*Path, error) {
	r.graph.mu.RLock()
	defer r.graph.mu.RUnlock()
	
	// Verify source and target exist
	if _, ok := r.graph.nodes[source]; !ok {
		return nil, fmt.Errorf("source node not found: %s", source)
	}
	if _, ok := r.graph.nodes[target]; !ok {
		return nil, fmt.Errorf("target node not found: %s", target)
	}
	
	// Find the shortest path first using Dijkstra
	shortestPath := r.dijkstra(source, target, nil, nil)
	if shortestPath == nil {
		return nil, fmt.Errorf("no path found from %s to %s", source, target)
	}
	
	// A holds the K shortest paths
	A := []*Path{shortestPath}
	
	// B is a min-heap of candidate paths
	B := &pathHeap{}
	heap.Init(B)
	
	// Yen's algorithm main loop
	for k := 1; k < r.k; k++ {
		// Check context
		if ctx.Err() != nil {
			return A, ctx.Err()
		}
		
		// Get the previous shortest path
		prevPath := A[k-1]
		
		// For each node in the previous path (except the last)
		for i := 0; i < len(prevPath.Nodes)-1; i++ {
			// Spur node is where we diverge from previous path
			spurNode := prevPath.Nodes[i]
			rootPath := prevPath.Nodes[:i+1]
			
			// Track edges and nodes to exclude
			excludedEdges := make(map[string]bool)
			excludedNodes := make(map[string]bool)
			
			// Exclude edges that share this root path
			for _, path := range A {
				if len(path.Nodes) > i && pathsSharePrefix(path.Nodes, rootPath) {
					if i+1 < len(path.Nodes) {
						edgeKey := path.Nodes[i] + "->" + path.Nodes[i+1]
						excludedEdges[edgeKey] = true
					}
				}
			}
			
			// Exclude root path nodes (except spur node)
			for j := 0; j < i; j++ {
				excludedNodes[prevPath.Nodes[j]] = true
			}
			
			// Find shortest path from spur to target, excluding edges/nodes
			spurPath := r.dijkstra(spurNode, target, excludedEdges, excludedNodes)
			
			if spurPath != nil {
				// Combine root path with spur path
				totalPath := r.combinePaths(rootPath, spurPath)
				
				// Add to candidates if not already in A
				if !containsPath(A, totalPath) && !heapContainsPath(B, totalPath) {
					heap.Push(B, totalPath)
				}
			}
		}
		
		// No more candidates
		if B.Len() == 0 {
			break
		}
		
		// Add the best candidate to A
		bestCandidate := heap.Pop(B).(*Path)
		A = append(A, bestCandidate)
	}
	
	return A, nil
}

// dijkstra finds the shortest path using Dijkstra's algorithm
func (r *Router) dijkstra(source, target string, excludedEdges, excludedNodes map[string]bool) *Path {
	if excludedNodes[source] || excludedNodes[target] {
		return nil
	}
	
	// Distance and predecessor maps
	dist := make(map[string]float64)
	prev := make(map[string]string)
	prevEdge := make(map[string]*Edge)
	
	for nodeID := range r.graph.nodes {
		dist[nodeID] = math.Inf(1)
	}
	dist[source] = 0
	
	// Priority queue
	pq := &dijkstraHeap{{node: source, dist: 0}}
	heap.Init(pq)
	
	visited := make(map[string]bool)
	
	for pq.Len() > 0 {
		current := heap.Pop(pq).(*dijkstraItem)
		
		if visited[current.node] {
			continue
		}
		visited[current.node] = true
		
		if current.node == target {
			break
		}
		
		// Explore neighbors
		neighbors := r.graph.edges[current.node]
		for targetID, edge := range neighbors {
			if !edge.IsActive {
				continue
			}
			// Skip inactive nodes
			if targetNode, ok := r.graph.nodes[targetID]; ok && !targetNode.IsActive {
				continue
			}
			if excludedNodes[targetID] {
				continue
			}
			edgeKey := current.node + "->" + targetID
			if excludedEdges[edgeKey] {
				continue
			}
			
			weight := r.graph.getEdgeWeightUnlocked(edge)
			newDist := dist[current.node] + weight
			
			if newDist < dist[targetID] {
				dist[targetID] = newDist
				prev[targetID] = current.node
				prevEdge[targetID] = edge
				heap.Push(pq, &dijkstraItem{node: targetID, dist: newDist})
			}
		}
	}
	
	// Reconstruct path
	if dist[target] == math.Inf(1) {
		return nil
	}
	
	path := &Path{
		Nodes:       []string{},
		Edges:       []*Edge{},
		TotalWeight: dist[target],
	}
	
	// Build path backwards
	current := target
	for current != "" {
		path.Nodes = append([]string{current}, path.Nodes...)
		if edge, ok := prevEdge[current]; ok {
			path.Edges = append([]*Edge{edge}, path.Edges...)
			path.TotalFee += edge.BaseFee
			path.TotalLatency += edge.Latency
		}
		current = prev[current]
	}
	
	return path
}

// combinePaths combines a root path with a spur path
func (r *Router) combinePaths(rootNodes []string, spurPath *Path) *Path {
	combined := &Path{
		Nodes: make([]string, 0, len(rootNodes)+len(spurPath.Nodes)-1),
		Edges: make([]*Edge, 0),
	}
	
	// Add root nodes
	combined.Nodes = append(combined.Nodes, rootNodes...)
	
	// Add root edges
	for i := 0; i < len(rootNodes)-1; i++ {
		if edges, ok := r.graph.edges[rootNodes[i]]; ok {
			if edge, ok := edges[rootNodes[i+1]]; ok {
				combined.Edges = append(combined.Edges, edge)
				combined.TotalFee += edge.BaseFee
				combined.TotalLatency += edge.Latency
				combined.TotalWeight += r.graph.getEdgeWeightUnlocked(edge)
			}
		}
	}
	
	// Add spur path (skip first node as it's the spur node already in root)
	if len(spurPath.Nodes) > 1 {
		combined.Nodes = append(combined.Nodes, spurPath.Nodes[1:]...)
	}
	combined.Edges = append(combined.Edges, spurPath.Edges...)
	combined.TotalFee += spurPath.TotalFee
	combined.TotalLatency += spurPath.TotalLatency
	combined.TotalWeight += spurPath.TotalWeight
	
	return combined
}

// Helper functions
func pathsSharePrefix(path, prefix []string) bool {
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

func containsPath(paths []*Path, path *Path) bool {
	for _, p := range paths {
		if pathsEqual(p.Nodes, path.Nodes) {
			return true
		}
	}
	return false
}

func heapContainsPath(h *pathHeap, path *Path) bool {
	for _, p := range *h {
		if pathsEqual(p.Nodes, path.Nodes) {
			return true
		}
	}
	return false
}

func pathsEqual(a, b []string) bool {
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

// pathHeap is a min-heap of paths ordered by total weight
type pathHeap []*Path

func (h pathHeap) Len() int           { return len(h) }
func (h pathHeap) Less(i, j int) bool { return h[i].TotalWeight < h[j].TotalWeight }
func (h pathHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *pathHeap) Push(x interface{}) {
	*h = append(*h, x.(*Path))
}

func (h *pathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// dijkstraItem represents a node in the priority queue
type dijkstraItem struct {
	node string
	dist float64
}

type dijkstraHeap []*dijkstraItem

func (h dijkstraHeap) Len() int           { return len(h) }
func (h dijkstraHeap) Less(i, j int) bool { return h[i].dist < h[j].dist }
func (h dijkstraHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *dijkstraHeap) Push(x interface{}) {
	*h = append(*h, x.(*dijkstraItem))
}

func (h *dijkstraHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
