// Package entropy provides Shannon entropy calculations for liquidity distribution analysis.
// Used for dynamic edge weighting in the routing algorithm.
package entropy

import (
	"math"
)

// Calculate computes the Shannon entropy of a probability distribution.
// Higher entropy indicates more uniform/unpredictable distribution.
// H = -Σ(p_i * log2(p_i))
//
// Input: slice of values representing liquidity volumes or transaction counts.
// Output: entropy in bits (0 = completely concentrated, log2(n) = uniform).
func Calculate(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Calculate total for normalization
	var total float64
	for _, v := range values {
		if v < 0 {
			v = 0 // Treat negative values as 0
		}
		total += v
	}

	if total == 0 {
		return 0
	}

	// Calculate entropy
	var entropy float64
	for _, v := range values {
		if v <= 0 {
			continue
		}
		p := v / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// CalculateNormalized computes normalized entropy (0 to 1 scale).
// Useful for comparing distributions of different sizes.
// Returns H / log2(n), where n is the number of non-zero elements.
func CalculateNormalized(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	// Count non-zero elements
	nonZero := 0
	for _, v := range values {
		if v > 0 {
			nonZero++
		}
	}

	if nonZero <= 1 {
		return 0
	}

	entropy := Calculate(values)
	maxEntropy := math.Log2(float64(nonZero))

	if maxEntropy == 0 {
		return 0
	}

	return entropy / maxEntropy
}

// CalculateFromMap computes entropy from a map of values.
// Convenience method for node liquidity distributions.
func CalculateFromMap(distribution map[string]float64) float64 {
	values := make([]float64, 0, len(distribution))
	for _, v := range distribution {
		values = append(values, v)
	}
	return Calculate(values)
}

// NodeEntropy holds entropy data for a mesh node
type NodeEntropy struct {
	NodeID           string             `json:"node_id"`
	Entropy          float64            `json:"entropy"`
	NormalizedEntropy float64           `json:"normalized_entropy"`
	Distribution     map[string]float64 `json:"distribution"`
	LastUpdated      int64              `json:"last_updated"` // Unix timestamp
}

// CalculateNodeEntropy computes entropy for a node's liquidity distribution.
// Distribution is typically: outgoing edge -> liquidity volume
func CalculateNodeEntropy(nodeID string, distribution map[string]float64) *NodeEntropy {
	values := make([]float64, 0, len(distribution))
	for _, v := range distribution {
		values = append(values, v)
	}

	return &NodeEntropy{
		NodeID:            nodeID,
		Entropy:           Calculate(values),
		NormalizedEntropy: CalculateNormalized(values),
		Distribution:      distribution,
	}
}

// Volatility returns a volatility score based on entropy.
// Higher entropy = higher volatility/unpredictability.
// Returns H capped at a reasonable maximum for weight calculation.
func (n *NodeEntropy) Volatility() float64 {
	// Cap at 3.0 (equivalent to ~8 equal-weight destinations)
	if n.Entropy > 3.0 {
		return 3.0
	}
	return n.Entropy
}
