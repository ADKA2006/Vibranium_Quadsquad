// Package users provides in-memory user storage with Argon2id password hashing.
// This can be upgraded to PostgreSQL persistence as needed.
package users

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plm/predictive-liquidity-mesh/auth"
)

// Common errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailExists       = errors.New("email already exists")
	ErrUsernameExists    = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// StoredUser represents a user with hashed password
type StoredUser struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose
	Role         auth.Role `json:"role"`
	FullName     string    `json:"full_name,omitempty"`
	Organization string    `json:"organization,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToUser converts StoredUser to auth.User (without password hash)
func (su *StoredUser) ToUser() *auth.User {
	return &auth.User{
		ID:           su.ID,
		Email:        su.Email,
		Username:     su.Username,
		Role:         su.Role,
		FullName:     su.FullName,
		Organization: su.Organization,
		IsActive:     su.IsActive,
		CreatedAt:    su.CreatedAt,
	}
}

// UserWithToUser is an interface for types that can convert to auth.User
type UserWithToUser interface {
	ToUser() *auth.User
}

// Store provides user storage operations
type Store struct {
	mu       sync.RWMutex
	users    map[string]*StoredUser // by ID
	byEmail  map[string]string      // email -> ID
	byName   map[string]string      // username -> ID
}

// generateSecurePassword creates a cryptographically secure random password
func generateSecurePassword(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback should never happen, but handle gracefully
		log.Fatal("CRITICAL: Failed to generate secure random password")
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// getPasswordFromEnv retrieves password from environment variable or generates a secure one
func getPasswordFromEnv(envVar, userType string) string {
	if password := os.Getenv(envVar); password != "" {
		return password
	}
	// Generate a secure random password if not provided
	generatedPassword := generateSecurePassword(32)
	log.Printf("WARNING: %s not set. Generated secure password for %s: %s", envVar, userType, generatedPassword)
	log.Printf("IMPORTANT: Set %s environment variable in production!", envVar)
	return generatedPassword
}

// NewStore creates a new user store with default admin user
func NewStore() *Store {
	store := &Store{
		users:   make(map[string]*StoredUser),
		byEmail: make(map[string]string),
		byName:  make(map[string]string),
	}

	// Get passwords from environment variables (secure by default)
	adminPassword := getPasswordFromEnv("ADMIN_PASSWORD", "admin@plm.local")
	userPassword := getPasswordFromEnv("USER_PASSWORD", "user@plm.local")

	// Create default admin user
	adminHash, _ := auth.HashPassword(adminPassword)
	adminUser := &StoredUser{
		ID:           "admin-default-001",
		Email:        "admin@plm.local",
		Username:     "admin",
		PasswordHash: adminHash,
		Role:         auth.RoleAdmin,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.users[adminUser.ID] = adminUser
	store.byEmail[adminUser.Email] = adminUser.ID
	store.byName[adminUser.Username] = adminUser.ID

	// Create default regular user
	userHash, _ := auth.HashPassword(userPassword)
	regularUser := &StoredUser{
		ID:           "user-default-001",
		Email:        "user@plm.local",
		Username:     "user",
		PasswordHash: userHash,
		Role:         auth.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.users[regularUser.ID] = regularUser
	store.byEmail[regularUser.Email] = regularUser.ID
	store.byName[regularUser.Username] = regularUser.ID

	return store
}

// CreateUser creates a new user with hashed password
func (s *Store) CreateUser(email, password, username string, role auth.Role) (UserWithToUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing email
	if _, exists := s.byEmail[email]; exists {
		return nil, ErrEmailExists
	}

	// Check for existing username
	if _, exists := s.byName[username]; exists {
		return nil, ErrUsernameExists
	}

	// Hash password with Argon2id
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &StoredUser{
		ID:           uuid.New().String(),
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.users[user.ID] = user
	s.byEmail[user.Email] = user.ID
	s.byName[user.Username] = user.ID

	return user, nil
}

// GetByEmail retrieves a user by email
func (s *Store) GetByEmail(email string) (UserWithToUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.byEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	return s.users[id], nil
}

// GetByID retrieves a user by ID
func (s *Store) GetByID(id string) (*StoredUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Authenticate verifies credentials and returns the user
func (s *Store) Authenticate(email, password string) (UserWithToUser, error) {
	s.mu.RLock()
	id, exists := s.byEmail[email]
	if !exists {
		s.mu.RUnlock()
		return nil, ErrInvalidCredentials
	}
	user := s.users[id]
	s.mu.RUnlock()

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	// Verify password with Argon2id
	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// ListUsers returns all users (for admin)
func (s *Store) ListUsers() []*auth.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*auth.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u.ToUser())
	}
	return result
}
