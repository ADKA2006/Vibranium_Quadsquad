// Package auth provides PASETO v4.local token management for session authentication.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/o1egl/paseto"
)

// Token errors
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenClaims represents the claims stored in a PASETO token
type TokenClaims struct {
	TokenID   string    `json:"jti"`
	UserID    string    `json:"sub"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Role      Role      `json:"role"`
	IssuedAt  time.Time `json:"iat"`
	NotBefore time.Time `json:"nbf"` // Token not valid before this time
	ExpiresAt time.Time `json:"exp"`
	Issuer    string    `json:"iss"`
}

// Valid checks if the token claims are valid with comprehensive validation
func (c *TokenClaims) Valid() error {
	now := time.Now()
	
	// Check expiry
	if now.After(c.ExpiresAt) {
		return ErrExpiredToken
	}
	
	// Check NotBefore (token not yet valid)
	if !c.NotBefore.IsZero() && now.Before(c.NotBefore) {
		return errors.New("token not yet valid")
	}
	
	// Check IssuedAt is not in the future (with 1 minute clock skew tolerance)
	if c.IssuedAt.After(now.Add(1 * time.Minute)) {
		return errors.New("token issued in the future")
	}
	
	return nil
}

// TokenManager handles PASETO token creation and verification
type TokenManager struct {
	symmetricKey []byte
	issuer       string
	tokenTTL     time.Duration
	v2           *paseto.V2
}

// TokenConfig configures the token manager
type TokenConfig struct {
	// SymmetricKey should be 32 bytes for PASETO v2.local
	SymmetricKey string
	Issuer       string
	TokenTTL     time.Duration
}

// DefaultTokenConfig returns config from environment or development defaults.
// Note: SymmetricKey must be provided via TOKEN_SECRET environment variable.
func DefaultTokenConfig() (*TokenConfig, error) {
	symmetricKey := os.Getenv("TOKEN_SECRET")
	if symmetricKey == "" {
		return nil, errors.New("security error: TOKEN_SECRET environment variable is not set")
	}
	if len(symmetricKey) != 32 {
		return nil, errors.New("security error: TOKEN_SECRET must be exactly 32 bytes long")
	}
	
	issuer := os.Getenv("TOKEN_ISSUER")
	if issuer == "" {
		issuer = "plm-auth"
	}
	
	ttlStr := os.Getenv("TOKEN_TTL")
	ttl := 24 * time.Hour
	if ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			ttl = parsed
		}
	}
	
	return &TokenConfig{
		SymmetricKey: symmetricKey,
		Issuer:       issuer,
		TokenTTL:     ttl,
	}, nil
}

// NewTokenManager creates a new PASETO token manager
func NewTokenManager(cfg *TokenConfig) (*TokenManager, error) {
	if cfg == nil {
		var err error
		cfg, err = DefaultTokenConfig()
		if err != nil {
			return nil, err
		}
	}

	key := []byte(cfg.SymmetricKey)
	if len(key) != 32 {
		return nil, errors.New("symmetric key must be exactly 32 bytes")
	}

	return &TokenManager{
		symmetricKey: key,
		issuer:       cfg.Issuer,
		tokenTTL:     cfg.TokenTTL,
		v2:           paseto.NewV2(),
	}, nil
}

// GenerateToken creates a new PASETO token for the user
func (tm *TokenManager) GenerateToken(user *User) (string, *TokenClaims, error) {
	// Generate unique token ID
	tokenIDBytes := make([]byte, 16)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", nil, err
	}
	tokenID := hex.EncodeToString(tokenIDBytes)

	now := time.Now()
	claims := &TokenClaims{
		TokenID:   tokenID,
		UserID:    user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		IssuedAt:  now,
		NotBefore: now, // Token valid immediately
		ExpiresAt: now.Add(tm.tokenTTL),
		Issuer:    tm.issuer,
	}

	// Create PASETO token
	token, err := tm.v2.Encrypt(tm.symmetricKey, claims, nil)
	if err != nil {
		return "", nil, err
	}

	return token, claims, nil
}

// VerifyToken validates a PASETO token and returns the claims
func (tm *TokenManager) VerifyToken(token string) (*TokenClaims, error) {
	var claims TokenClaims

	err := tm.v2.Decrypt(token, tm.symmetricKey, &claims, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Validate time-based claims
	if err := claims.Valid(); err != nil {
		return nil, err
	}

	// Validate issuer matches our expected issuer
	if claims.Issuer != tm.issuer {
		return nil, errors.New("invalid token issuer")
	}

	return &claims, nil
}

// RefreshToken generates a new token with extended expiry
func (tm *TokenManager) RefreshToken(claims *TokenClaims) (string, *TokenClaims, error) {
	// Create user from claims
	user := &User{
		ID:       claims.UserID,
		Email:    claims.Email,
		Username: claims.Username,
		Role:     claims.Role,
	}

	return tm.GenerateToken(user)
}
