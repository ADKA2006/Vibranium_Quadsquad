// Package postgres provides PostgreSQL client integration for the Predictive Liquidity Mesh.
// Handles ledger operations with hash-chain integrity.
package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection configuration
type Config struct {
	Host              string
	Port              int
	User              string
	Password          string
	Database          string
	SSLMode           string
	MaxOpenConns      int
	MaxIdleConns      int
	SynchronousCommit bool // Set to false for high-throughput
}

// DefaultConfig returns a default configuration for local development
func DefaultConfig() *Config {
	return &Config{
		Host:              "localhost",
		Port:              5432,
		User:              "postgres",
		Password:          "postgres",
		Database:          "plm_ledger",
		SSLMode:           "disable",
		MaxOpenConns:      100,
		MaxIdleConns:      10,
		SynchronousCommit: false, // ACID tuning for throughput
	}
}

// Client wraps PostgreSQL connection with ledger operations
type Client struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewClient creates a new PostgreSQL client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set synchronous_commit based on config (using explicit safe values, not string interpolation)
	// Note: PostgreSQL SET commands don't support parameterized queries for values,
	// so we use a whitelist approach with explicit validation
	var setSyncQuery string
	if cfg.SynchronousCommit {
		setSyncQuery = "SET synchronous_commit = on"
	} else {
		setSyncQuery = "SET synchronous_commit = off"
	}
	_, err = db.ExecContext(ctx, setSyncQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous_commit: %w", err)
	}

	return &Client{db: db}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// DB returns the underlying database connection
func (c *Client) DB() *sql.DB {
	return c.db
}

// LedgerEntry represents a ledger transaction
type LedgerEntry struct {
	ID           string          `json:"id"`
	SequenceNum  int64           `json:"sequence_num"`
	Amount       int64           `json:"amount"`
	Path         json.RawMessage `json:"path"`
	Signature    string          `json:"signature"`
	PreviousHash string          `json:"previous_hash"`
	CurrentHash  string          `json:"current_hash"`
	CreatedAt    string          `json:"created_at"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// InsertLedgerEntry inserts a new entry into the hash-chained ledger
func (c *Client) InsertLedgerEntry(ctx context.Context, amount int64, path []string, signature string, metadata map[string]interface{}) (*LedgerEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get the latest hash for chaining
	var previousHash string
	err := c.db.QueryRowContext(ctx, "SELECT get_latest_ledger_hash()").Scan(&previousHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest hash: %w", err)
	}

	// Marshal path and metadata
	pathJSON, err := json.Marshal(path)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal path: %w", err)
	}

	metadataJSON := []byte("{}")
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Insert the entry
	query := `
		INSERT INTO ledger (amount, path, signature, previous_hash, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, sequence_num, amount, path, signature, previous_hash, current_hash, created_at, metadata
	`

	var entry LedgerEntry
	err = c.db.QueryRowContext(ctx, query, amount, pathJSON, signature, previousHash, metadataJSON).Scan(
		&entry.ID,
		&entry.SequenceNum,
		&entry.Amount,
		&entry.Path,
		&entry.Signature,
		&entry.PreviousHash,
		&entry.CurrentHash,
		&entry.CreatedAt,
		&entry.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert ledger entry: %w", err)
	}

	return &entry, nil
}

// GetLedgerEntry retrieves a ledger entry by ID
func (c *Client) GetLedgerEntry(ctx context.Context, id string) (*LedgerEntry, error) {
	query := `
		SELECT id, sequence_num, amount, path, signature, previous_hash, current_hash, created_at, metadata
		FROM ledger
		WHERE id = $1
	`

	var entry LedgerEntry
	err := c.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.SequenceNum,
		&entry.Amount,
		&entry.Path,
		&entry.Signature,
		&entry.PreviousHash,
		&entry.CurrentHash,
		&entry.CreatedAt,
		&entry.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entry: %w", err)
	}

	return &entry, nil
}

// GetLatestLedgerEntries retrieves the N most recent ledger entries
func (c *Client) GetLatestLedgerEntries(ctx context.Context, limit int) ([]LedgerEntry, error) {
	query := `
		SELECT id, sequence_num, amount, path, signature, previous_hash, current_hash, created_at, metadata
		FROM ledger
		ORDER BY sequence_num DESC
		LIMIT $1
	`

	rows, err := c.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		var entry LedgerEntry
		err := rows.Scan(
			&entry.ID,
			&entry.SequenceNum,
			&entry.Amount,
			&entry.Path,
			&entry.Signature,
			&entry.PreviousHash,
			&entry.CurrentHash,
			&entry.CreatedAt,
			&entry.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// VerifyIntegrity verifies the hash chain integrity of the entire ledger
func (c *Client) VerifyIntegrity(ctx context.Context) ([]IntegrityResult, error) {
	query := `SELECT * FROM verify_ledger_integrity()`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to verify integrity: %w", err)
	}
	defer rows.Close()

	var results []IntegrityResult
	for rows.Next() {
		var result IntegrityResult
		err := rows.Scan(
			&result.EntryID,
			&result.SequenceNum,
			&result.IsValid,
			&result.ExpectedPrevious,
			&result.ActualPrevious,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan integrity result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// IntegrityResult represents the result of an integrity check
type IntegrityResult struct {
	EntryID          string `json:"entry_id"`
	SequenceNum      int64  `json:"sequence_num"`
	IsValid          bool   `json:"is_valid"`
	ExpectedPrevious string `json:"expected_previous"`
	ActualPrevious   string `json:"actual_previous"`
}

// ComputeLocalHash computes a hash locally for verification
func ComputeLocalHash(id string, seqNum, amount int64, path, signature, previousHash string) string {
	hashInput := fmt.Sprintf("%s:%d:%d:%s:%s:%s", id, seqNum, amount, path, signature, previousHash)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// GetSynchronousCommitStatus returns the current synchronous_commit setting
func (c *Client) GetSynchronousCommitStatus(ctx context.Context) (string, error) {
	var status string
	err := c.db.QueryRowContext(ctx, "SHOW synchronous_commit").Scan(&status)
	if err != nil {
		return "", fmt.Errorf("failed to get synchronous_commit status: %w", err)
	}
	return status, nil
}
