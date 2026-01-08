// Package middleware provides HTTP middleware for the PLM API.
// Includes authentication and authorization middleware using PASETO tokens.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/plm/predictive-liquidity-mesh/auth"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey ContextKey = "user"
	// ClaimsContextKey is the context key for token claims
	ClaimsContextKey ContextKey = "claims"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tm *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tm}
}

// Authenticate validates the PASETO token and adds user to context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Verify token
		claims, err := m.tokenManager.VerifyToken(token)
		if err != nil {
			if err == auth.ErrExpiredToken {
				http.Error(w, `{"error":"token has expired"}`, http.StatusUnauthorized)
				return
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Create user from claims
		user := &auth.User{
			ID:       claims.UserID,
			Email:    claims.Email,
			Username: claims.Username,
			Role:     claims.Role,
			IsActive: true,
		}

		// Add user and claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole creates middleware that requires a specific role
func (m *AuthMiddleware) RequireRole(role auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !user.HasPermission(role) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin is shorthand for RequireRole(RoleAdmin)
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return m.RequireRole(auth.RoleAdmin)(next)
}

// RequireUser ensures only regular users (not admins) can access
func (m *AuthMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if user.Role == auth.RoleAdmin {
			http.Error(w, `{"error":"admins cannot make payments - use a regular user account"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts the user from request context
func GetUserFromContext(ctx context.Context) *auth.User {
	user, ok := ctx.Value(UserContextKey).(*auth.User)
	if !ok {
		return nil
	}
	return user
}

// GetClaimsFromContext extracts the token claims from request context
func GetClaimsFromContext(ctx context.Context) *auth.TokenClaims {
	claims, ok := ctx.Value(ClaimsContextKey).(*auth.TokenClaims)
	if !ok {
		return nil
	}
	return claims
}

// Chain chains multiple middleware together
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
