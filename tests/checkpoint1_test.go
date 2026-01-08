// Package tests contains integration tests for the Predictive Liquidity Mesh.
// Checkpoint 1: Verify hash-chained ledger integrity and ACID tuning.
package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/plm/predictive-liquidity-mesh/storage/postgres"
)

// TestCheckpoint1_LedgerIntegrity verifies:
// 1. Insert 5 rows into the ledger
// 2. Verify row N's previous_hash matches row N-1's cryptographic hash
// 3. Verify synchronous_commit status is 'off'
func TestCheckpoint1_LedgerIntegrity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to PostgreSQL
	cfg := postgres.DefaultConfig()
	client, err := postgres.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer client.Close()

	t.Run("VerifySynchronousCommit", func(t *testing.T) {
		status, err := client.GetSynchronousCommitStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get synchronous_commit status: %v", err)
		}

		if status != "off" {
			t.Errorf("Expected synchronous_commit = 'off', got '%s'", status)
		} else {
			t.Logf("✅ synchronous_commit = %s (ACID tuned for throughput)", status)
		}
	})

	t.Run("InsertAndVerifyHashChain", func(t *testing.T) {
		// Insert 5 test transactions
		testTransactions := []struct {
			amount int64
			path   []string
		}{
			{100000, []string{"sme_001", "lp_alpha", "hub_primary"}},
			{250000, []string{"sme_002", "lp_beta", "hub_secondary"}},
			{75000, []string{"sme_003", "lp_gamma", "hub_backup"}},
			{500000, []string{"sme_004", "lp_alpha", "hub_primary", "hub_secondary"}},
			{150000, []string{"sme_005", "lp_beta", "hub_primary"}},
		}

		var insertedEntries []*postgres.LedgerEntry

		for i, tx := range testTransactions {
			// Mock signature (in production, this would be Ed25519 signed)
			signature := fmt.Sprintf("mock_sig_%d_%d", time.Now().UnixNano(), i)

			entry, err := client.InsertLedgerEntry(ctx, tx.amount, tx.path, signature, map[string]interface{}{
				"test":       true,
				"checkpoint": 1,
				"tx_number":  i + 1,
			})
			if err != nil {
				t.Fatalf("Failed to insert entry %d: %v", i+1, err)
			}

			insertedEntries = append(insertedEntries, entry)
			t.Logf("✅ Inserted entry %d: ID=%s, Amount=%d, Hash=%s...",
				i+1, entry.ID, entry.Amount, entry.CurrentHash[:16])
		}

		// Verify hash chain integrity
		t.Log("\n--- Verifying Hash Chain ---")
		for i := 1; i < len(insertedEntries); i++ {
			prev := insertedEntries[i-1]
			curr := insertedEntries[i]

			// Verify current entry's previous_hash matches previous entry's current_hash
			if curr.PreviousHash != prev.CurrentHash {
				t.Errorf("❌ Hash chain broken at entry %d: expected previous_hash=%s, got=%s",
					i+1, prev.CurrentHash, curr.PreviousHash)
			} else {
				t.Logf("✅ Entry %d -> Entry %d: Hash chain valid", i, i+1)
			}
		}

		// Verify using the database function
		t.Log("\n--- Database Integrity Check ---")
		results, err := client.VerifyIntegrity(ctx)
		if err != nil {
			t.Fatalf("Failed to verify integrity: %v", err)
		}

		allValid := true
		for _, result := range results {
			if !result.IsValid {
				t.Errorf("❌ Entry %s (seq=%d): Invalid - expected=%s, actual=%s",
					result.EntryID, result.SequenceNum, result.ExpectedPrevious, result.ActualPrevious)
				allValid = false
			}
		}

		if allValid {
			t.Logf("✅ All %d ledger entries passed integrity verification", len(results))
		}
	})
}

// TestCheckpoint1_HashComputation verifies local hash computation matches database
func TestCheckpoint1_HashComputation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := postgres.DefaultConfig()
	client, err := postgres.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer client.Close()

	// Get latest entries
	entries, err := client.GetLatestLedgerEntries(ctx, 5)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) == 0 {
		t.Skip("No entries in ledger - run InsertAndVerifyHashChain first")
	}

	for _, entry := range entries {
		// Compute hash locally
		hashInput := fmt.Sprintf("%s:%d:%d:%s:%s:%s",
			entry.ID,
			entry.SequenceNum,
			entry.Amount,
			string(entry.Path),
			entry.Signature,
			entry.PreviousHash,
		)
		hash := sha256.Sum256([]byte(hashInput))
		localHash := hex.EncodeToString(hash[:])

		if localHash != entry.CurrentHash {
			t.Errorf("❌ Hash mismatch for entry %s: local=%s, db=%s",
				entry.ID, localHash, entry.CurrentHash)
		} else {
			t.Logf("✅ Entry %s: Local hash matches database hash", entry.ID)
		}
	}
}
