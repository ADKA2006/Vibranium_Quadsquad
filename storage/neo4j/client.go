// Package neo4j provides Neo4j client integration for the Predictive Liquidity Mesh.
// Handles graph queries for routing path discovery.
package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Config holds Neo4j connection configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

// DefaultConfig returns a default configuration for local development
func DefaultConfig() *Config {
	return &Config{
		URI:      "neo4j://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}
}

// Client wraps Neo4j driver with mesh query capabilities
type Client struct {
	driver   neo4j.DriverWithContext
	database string
}

// NewClient creates a new Neo4j client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &Client{
		driver:   driver,
		database: cfg.Database,
	}, nil
}

// Close closes the Neo4j connection
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// Node represents a mesh node
type Node struct {
	ID       string
	Type     string
	Name     string
	Region   string
	IsActive bool
	Props    map[string]interface{}
}

// Edge represents a liquidity edge
type Edge struct {
	ID              string
	Type            string
	SourceID        string
	TargetID        string
	BaseFee         float64
	Latency         int64
	LiquidityVolume int64
	IsActive        bool
}

// Path represents a routing path through the mesh
type Path struct {
	Nodes      []Node
	Edges      []Edge
	TotalFee   float64
	TotalLatency int64
}

// FindPaths finds paths between two nodes (for Yen's K-shortest paths algorithm input)
func (c *Client) FindPaths(ctx context.Context, sourceID, targetID string, maxHops int) ([]Path, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH path = (source {id: $sourceId})-[*1..` + fmt.Sprintf("%d", maxHops) + `]->(target {id: $targetId})
		WHERE all(r IN relationships(path) WHERE r.is_active = true)
		  AND all(n IN nodes(path) WHERE n.is_active = true)
		RETURN path,
		       reduce(fee = 0.0, r IN relationships(path) | fee + coalesce(r.base_fee, 0)) AS totalFee,
		       reduce(lat = 0, r IN relationships(path) | lat + coalesce(r.latency, 0)) AS totalLatency
		ORDER BY totalFee ASC, totalLatency ASC
		LIMIT 10
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"sourceId": sourceID,
		"targetId": targetID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute path query: %w", err)
	}

	var paths []Path
	for result.Next(ctx) {
		record := result.Record()

		pathValue, _ := record.Get("path")
		totalFee, _ := record.Get("totalFee")
		totalLatency, _ := record.Get("totalLatency")

		neo4jPath, ok := pathValue.(neo4j.Path)
		if !ok {
			continue
		}

		path := Path{
			TotalFee:   totalFee.(float64),
			TotalLatency: totalLatency.(int64),
		}

		// Extract nodes
		for _, node := range neo4jPath.Nodes {
			path.Nodes = append(path.Nodes, Node{
				ID:       node.Props["id"].(string),
				Type:     node.Labels[0],
				Name:     getStringProp(node.Props, "name"),
				Region:   getStringProp(node.Props, "region"),
				IsActive: getBoolProp(node.Props, "is_active"),
				Props:    node.Props,
			})
		}

		// Extract edges
		for _, rel := range neo4jPath.Relationships {
			path.Edges = append(path.Edges, Edge{
				ID:              fmt.Sprintf("%d", rel.Id),
				Type:            rel.Type,
				BaseFee:         getFloatProp(rel.Props, "base_fee"),
				Latency:         getIntProp(rel.Props, "latency"),
				LiquidityVolume: getIntProp(rel.Props, "liquidity_volume"),
				IsActive:        getBoolProp(rel.Props, "is_active"),
			})
		}

		paths = append(paths, path)
	}

	return paths, nil
}

// GetNode retrieves a single node by ID
func (c *Client) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `MATCH (n {id: $nodeId}) RETURN n`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	record := result.Record()
	nodeValue, _ := record.Get("n")
	neo4jNode := nodeValue.(neo4j.Node)

	return &Node{
		ID:       neo4jNode.Props["id"].(string),
		Type:     neo4jNode.Labels[0],
		Name:     getStringProp(neo4jNode.Props, "name"),
		Region:   getStringProp(neo4jNode.Props, "region"),
		IsActive: getBoolProp(neo4jNode.Props, "is_active"),
		Props:    neo4jNode.Props,
	}, nil
}

// UpdateEdge updates edge properties (for real-time mesh updates)
func (c *Client) UpdateEdge(ctx context.Context, sourceID, targetID string, updates map[string]interface{}) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source {id: $sourceId})-[r]->(target {id: $targetId})
		SET r += $updates
		RETURN r
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"sourceId": sourceID,
		"targetId": targetID,
		"updates":  updates,
	})

	return err
}

// SetNodeActive updates the active status of a node (for circuit breaker integration)
func (c *Client) SetNodeActive(ctx context.Context, nodeID string, isActive bool) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n {id: $nodeId})
		SET n.is_active = $isActive
		RETURN n
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"nodeId":   nodeID,
		"isActive": isActive,
	})

	return err
}

// Helper functions for property extraction
func getStringProp(props map[string]interface{}, key string) string {
	if val, ok := props[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolProp(props map[string]interface{}, key string) bool {
	if val, ok := props[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getFloatProp(props map[string]interface{}, key string) float64 {
	if val, ok := props[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getIntProp(props map[string]interface{}, key string) int64 {
	if val, ok := props[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
	}
	return 0
}
