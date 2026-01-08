// Package auth provides authentication utilities for the Predictive Liquidity Mesh.
// Implements Argon2id password hashing and PASETO v4.local token management.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended)
const (
	Argon2Memory      = 64 * 1024 // 64MB
	Argon2Iterations  = 3
	Argon2Parallelism = 4
	Argon2SaltLength  = 16
	Argon2KeyLength   = 32
)

// ErrInvalidHash is returned when the hash format is invalid
var ErrInvalidHash = errors.New("invalid password hash format")

// ErrMismatchedPassword is returned when password doesn't match
var ErrMismatchedPassword = errors.New("password does not match")

// HashPassword creates an Argon2id hash of the password
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, Argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		Argon2Iterations,
		Argon2Memory,
		Argon2Parallelism,
		Argon2KeyLength,
	)

	// Encode to standard format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		Argon2Memory,
		Argon2Iterations,
		Argon2Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// VerifyPassword checks if a password matches the hash
func VerifyPassword(password, encodedHash string) error {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return ErrInvalidHash
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidHash
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(expectedHash, computedHash) != 1 {
		return ErrMismatchedPassword
	}

	return nil
}

// Role represents a user role for RBAC
type Role string

const (
	RoleAdmin   Role = "ADMIN"
	RoleUser    Role = "USER"
	RoleService Role = "SERVICE"
)

// User represents an authenticated user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	Role         Role      `json:"role"`
	FullName     string    `json:"full_name,omitempty"`
	Organization string    `json:"organization,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// HasPermission checks if user has required role
func (u *User) HasPermission(required Role) bool {
	if u.Role == RoleAdmin {
		return true // Admin has all permissions
	}
	return u.Role == required
}

// IsAdmin returns true if user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
