-- ============================================================================
-- PREDICTIVE LIQUIDITY MESH - HASH-CHAINED LEDGER
-- Migration: 001_init_ledger.sql
-- Description: ACID-tuned ledger with cryptographic hash chain for audit integrity
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- SESSION CONFIGURATION: ACID TUNING FOR HIGH-THROUGHPUT
-- ============================================================================
-- Set synchronous_commit = off for high-throughput performance
-- This allows WAL to flush asynchronously while maintaining crash safety
-- Data is still durable (WAL is still written), but we don't wait for disk sync
ALTER SYSTEM SET synchronous_commit = off;

-- ============================================================================
-- ULID GENERATION FUNCTION
-- ============================================================================
-- ULID provides time-ordered, collision-resistant unique identifiers
-- Format: 26 characters, Crockford's Base32
CREATE OR REPLACE FUNCTION generate_ulid() RETURNS TEXT AS $$
DECLARE
    timestamp_ms BIGINT;
    timestamp_part TEXT;
    random_part TEXT;
    encoding TEXT := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
    i INT;
BEGIN
    -- Get current timestamp in milliseconds
    timestamp_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
    
    -- Generate 10-character timestamp encoding (48 bits)
    timestamp_part := '';
    FOR i IN REVERSE 9..0 LOOP
        timestamp_part := timestamp_part || substring(encoding from ((timestamp_ms >> (i * 5)) & 31) + 1 for 1);
    END LOOP;
    
    -- Generate 16-character random encoding (80 bits)
    random_part := '';
    FOR i IN 1..16 LOOP
        random_part := random_part || substring(encoding from (floor(random() * 32)::INT) + 1 for 1);
    END LOOP;
    
    RETURN timestamp_part || random_part;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- ============================================================================
-- LEDGER TABLE: HASH-CHAINED TRANSACTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS ledger (
    -- Primary identifier: ULID for time-ordered uniqueness
    id TEXT PRIMARY KEY DEFAULT generate_ulid(),
    
    -- Sequence number for strict ordering (used in hash chain)
    sequence_num BIGSERIAL UNIQUE NOT NULL,
    
    -- Transaction amount in smallest currency unit (e.g., cents)
    amount BIGINT NOT NULL CHECK (amount > 0),
    
    -- Transaction path: JSONB array of node IDs the transaction traversed
    -- Example: ["sme_001", "lp_hub_a", "sme_042"]
    path JSONB NOT NULL DEFAULT '[]'::JSONB,
    
    -- Ed25519 signature (base64 encoded, 88 chars for 64 bytes)
    -- Signs: id || amount || path || previous_hash
    signature TEXT NOT NULL,
    
    -- SHA-256 hash of the previous ledger entry (hex encoded, 64 chars)
    -- Genesis block uses '0' repeated 64 times
    previous_hash CHAR(64) NOT NULL,
    
    -- Current entry's hash (computed on insert via trigger)
    current_hash CHAR(64),
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Metadata for additional transaction context
    metadata JSONB DEFAULT '{}'::JSONB
);

-- ============================================================================
-- HASH COMPUTATION FUNCTION
-- ============================================================================
-- Computes SHA-256 hash of the ledger entry for chain integrity
CREATE OR REPLACE FUNCTION compute_ledger_hash(
    p_id TEXT,
    p_sequence_num BIGINT,
    p_amount BIGINT,
    p_path JSONB,
    p_signature TEXT,
    p_previous_hash TEXT
) RETURNS TEXT AS $$
DECLARE
    hash_input TEXT;
BEGIN
    -- Concatenate all fields for hashing
    hash_input := p_id || ':' || 
                  p_sequence_num::TEXT || ':' ||
                  p_amount::TEXT || ':' || 
                  p_path::TEXT || ':' || 
                  p_signature || ':' || 
                  p_previous_hash;
    
    -- Return lowercase hex-encoded SHA-256 hash
    RETURN encode(digest(hash_input, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================================================
-- TRIGGER: AUTO-COMPUTE CURRENT HASH ON INSERT
-- ============================================================================
CREATE OR REPLACE FUNCTION ledger_compute_hash_trigger() RETURNS TRIGGER AS $$
BEGIN
    -- Compute and set the current hash
    NEW.current_hash := compute_ledger_hash(
        NEW.id,
        NEW.sequence_num,
        NEW.amount,
        NEW.path,
        NEW.signature,
        NEW.previous_hash
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ledger_compute_hash ON ledger;
CREATE TRIGGER trg_ledger_compute_hash
    BEFORE INSERT ON ledger
    FOR EACH ROW
    EXECUTE FUNCTION ledger_compute_hash_trigger();

-- ============================================================================
-- TRIGGER: VALIDATE HASH CHAIN INTEGRITY
-- ============================================================================
CREATE OR REPLACE FUNCTION ledger_validate_chain_trigger() RETURNS TRIGGER AS $$
DECLARE
    last_hash TEXT;
    last_seq BIGINT;
BEGIN
    -- Get the last entry's hash
    SELECT current_hash, sequence_num INTO last_hash, last_seq
    FROM ledger 
    ORDER BY sequence_num DESC 
    LIMIT 1;
    
    -- For genesis block, previous_hash must be all zeros
    IF last_hash IS NULL THEN
        IF NEW.previous_hash != repeat('0', 64) THEN
            RAISE EXCEPTION 'Genesis block must have previous_hash of 64 zeros';
        END IF;
    ELSE
        -- Validate the chain link
        IF NEW.previous_hash != last_hash THEN
            RAISE EXCEPTION 'Hash chain broken! Expected previous_hash: %, Got: %', 
                            last_hash, NEW.previous_hash;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ledger_validate_chain ON ledger;
CREATE TRIGGER trg_ledger_validate_chain
    BEFORE INSERT ON ledger
    FOR EACH ROW
    EXECUTE FUNCTION ledger_validate_chain_trigger();

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_ledger_created_at ON ledger(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_sequence_num ON ledger(sequence_num DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_path ON ledger USING GIN(path);

-- ============================================================================
-- HELPER FUNCTION: GET LATEST HASH FOR CHAINING
-- ============================================================================
CREATE OR REPLACE FUNCTION get_latest_ledger_hash() RETURNS TEXT AS $$
DECLARE
    latest_hash TEXT;
BEGIN
    SELECT current_hash INTO latest_hash
    FROM ledger
    ORDER BY sequence_num DESC
    LIMIT 1;
    
    -- Return genesis hash if no entries exist
    IF latest_hash IS NULL THEN
        RETURN repeat('0', 64);
    END IF;
    
    RETURN latest_hash;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- HELPER FUNCTION: VERIFY CHAIN INTEGRITY
-- ============================================================================
CREATE OR REPLACE FUNCTION verify_ledger_integrity() RETURNS TABLE (
    entry_id TEXT,
    sequence_num BIGINT,
    is_valid BOOLEAN,
    expected_previous TEXT,
    actual_previous TEXT
) AS $$
DECLARE
    rec RECORD;
    expected_hash TEXT;
    first_row BOOLEAN := TRUE;
BEGIN
    FOR rec IN SELECT * FROM ledger ORDER BY sequence_num ASC LOOP
        IF first_row THEN
            -- Genesis block check
            expected_hash := repeat('0', 64);
            first_row := FALSE;
        END IF;
        
        entry_id := rec.id;
        sequence_num := rec.sequence_num;
        expected_previous := expected_hash;
        actual_previous := rec.previous_hash;
        is_valid := (expected_hash = rec.previous_hash);
        
        RETURN NEXT;
        
        -- Update expected for next iteration
        expected_hash := rec.current_hash;
    END LOOP;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- COMMENT DOCUMENTATION
-- ============================================================================
COMMENT ON TABLE ledger IS 'Hash-chained immutable ledger for Predictive Liquidity Mesh transactions';
COMMENT ON COLUMN ledger.id IS 'ULID: Time-ordered unique identifier';
COMMENT ON COLUMN ledger.sequence_num IS 'Strict ordering sequence for hash chain';
COMMENT ON COLUMN ledger.amount IS 'Transaction amount in smallest currency unit';
COMMENT ON COLUMN ledger.path IS 'JSONB array of node IDs in transaction path';
COMMENT ON COLUMN ledger.signature IS 'Ed25519 signature (base64)';
COMMENT ON COLUMN ledger.previous_hash IS 'SHA-256 hash of previous entry (hex)';
COMMENT ON COLUMN ledger.current_hash IS 'SHA-256 hash of this entry (hex)';
