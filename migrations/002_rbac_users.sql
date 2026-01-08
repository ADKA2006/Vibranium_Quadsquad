-- ============================================================================
-- PREDICTIVE LIQUIDITY MESH - IDENTITY & ACCESS LAYER
-- Migration: 002_rbac_users.sql
-- ============================================================================
-- Implements Role-Based Access Control with:
-- - Users table with ADMIN/USER roles
-- - Argon2id password hashing (done in application layer)
-- - Session tracking for token revocation
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ENUM: User Roles
-- ============================================================================
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('ADMIN', 'USER', 'SERVICE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ============================================================================
-- TABLE: users
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    username        VARCHAR(100) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,                    -- Argon2id hash
    role            user_role NOT NULL DEFAULT 'USER',
    
    -- Profile
    full_name       VARCHAR(255),
    organization    VARCHAR(255),
    
    -- Security
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    
    -- Metadata
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ,
    last_login_ip   INET
);

-- ============================================================================
-- TABLE: sessions (for token tracking and revocation)
-- ============================================================================
CREATE TABLE IF NOT EXISTS sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id        UUID NOT NULL UNIQUE,             -- PASETO token ID (jti)
    
    -- Session info
    ip_address      INET,
    user_agent      TEXT,
    
    -- Validity
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    
    -- Index for fast lookups
    CONSTRAINT sessions_not_expired CHECK (expires_at > created_at)
);

-- ============================================================================
-- TABLE: api_keys (for service accounts and integrations)
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash        TEXT NOT NULL,                    -- SHA-256 of the API key
    name            VARCHAR(100) NOT NULL,
    
    -- Permissions
    scopes          TEXT[] NOT NULL DEFAULT '{}',     -- e.g., ['read:nodes', 'write:settle']
    
    -- Usage
    last_used_at    TIMESTAMPTZ,
    usage_count     BIGINT NOT NULL DEFAULT 0,
    
    -- Validity
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    
    UNIQUE(user_id, name)
);

-- ============================================================================
-- TABLE: audit_log (track security-sensitive operations)
-- ============================================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id              BIGSERIAL PRIMARY KEY,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id         UUID REFERENCES users(id),
    action          VARCHAR(50) NOT NULL,             -- 'LOGIN', 'LOGOUT', 'CREATE_NODE', etc.
    resource_type   VARCHAR(50),                      -- 'user', 'node', 'settlement'
    resource_id     TEXT,
    ip_address      INET,
    user_agent      TEXT,
    details         JSONB,
    success         BOOLEAN NOT NULL DEFAULT TRUE
);

-- ============================================================================
-- INDEXES
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token_id ON sessions(token_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);

-- ============================================================================
-- FUNCTIONS: Updated timestamp trigger
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================================================
-- FUNCTION: Check if session is valid
-- ============================================================================
CREATE OR REPLACE FUNCTION is_session_valid(p_token_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_session sessions%ROWTYPE;
BEGIN
    SELECT * INTO v_session
    FROM sessions
    WHERE token_id = p_token_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    IF v_session.revoked_at IS NOT NULL THEN
        RETURN FALSE;
    END IF;
    
    IF v_session.expires_at < NOW() THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- FUNCTION: Revoke all sessions for user
-- ============================================================================
CREATE OR REPLACE FUNCTION revoke_user_sessions(p_user_id UUID)
RETURNS INT AS $$
DECLARE
    v_count INT;
BEGIN
    UPDATE sessions
    SET revoked_at = NOW()
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;
    
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SEED: Create default admin user (password: admin123 - CHANGE IN PRODUCTION!)
-- The hash below is Argon2id for 'admin123' with default parameters
-- ============================================================================
INSERT INTO users (email, username, password_hash, role, full_name, is_verified)
VALUES (
    'admin@plm.local',
    'admin',
    -- Argon2id hash placeholder - will be set by application
    '$argon2id$v=19$m=65536,t=3,p=4$placeholder$placeholder',
    'ADMIN',
    'System Administrator',
    TRUE
) ON CONFLICT (email) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE users IS 'User accounts with RBAC roles for the PLM Dashboard';
COMMENT ON TABLE sessions IS 'Active PASETO token sessions for tracking and revocation';
COMMENT ON TABLE api_keys IS 'API keys for service accounts and external integrations';
COMMENT ON TABLE audit_log IS 'Security audit trail for compliance and debugging';
COMMENT ON COLUMN users.password_hash IS 'Argon2id encoded password hash';
COMMENT ON COLUMN users.role IS 'ADMIN: full access, USER: read + settle preview, SERVICE: API-only';
