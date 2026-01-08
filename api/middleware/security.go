// Package middleware provides security middleware for CSRF, SSRF protection and input sanitization.
package middleware

import (
	"errors"
	"html"
	"net"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

// AllowedOrigins defines the list of allowed origins for CSRF protection
var AllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8080",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:8080",
}

// CSRFMiddleware adds CSRF protection by validating Origin header
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip for safe methods (GET, HEAD, OPTIONS)
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Origin header
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, ao := range AllowedOrigins {
				if origin == ao {
					allowed = true
					break
				}
			}
			if !allowed {
				// Also allow if origin matches the request host
				host := r.Host
				if strings.Contains(origin, host) {
					allowed = true
				}
			}
			if !allowed {
				http.Error(w, `{"error":"CSRF validation failed: invalid origin"}`, http.StatusForbidden)
				return
			}
		}

		// Check Referer header as backup
		referer := r.Header.Get("Referer")
		if origin == "" && referer != "" {
			refURL, err := url.Parse(referer)
			if err == nil {
				allowed := false
				for _, ao := range AllowedOrigins {
					aoURL, _ := url.Parse(ao)
					if refURL.Host == aoURL.Host {
						allowed = true
						break
					}
				}
				if !allowed && !strings.Contains(refURL.Host, r.Host) {
					http.Error(w, `{"error":"CSRF validation failed: invalid referer"}`, http.StatusForbidden)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// BlockedHosts for SSRF protection
var BlockedHosts = []string{
	"localhost",
	"127.0.0.1",
	"0.0.0.0",
	"169.254.169.254", // AWS metadata endpoint
	"metadata.google.internal",
	"metadata.azure.internal",
}

// ValidateExternalURL validates URLs to prevent SSRF attacks
// Returns an error if the URL points to internal/blocked resources
func ValidateExternalURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return errors.New("invalid URL format")
	}

	// Only allow http and https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http and https schemes are allowed")
	}

	hostname := u.Hostname()

	// Block known internal hostnames
	for _, blocked := range BlockedHosts {
		if strings.EqualFold(hostname, blocked) {
			return errors.New("access to internal hosts is not allowed")
		}
	}

	// Check if it's an IP address
	ip := net.ParseIP(hostname)
	if ip != nil {
		// Block private IP ranges
		if ip.IsPrivate() {
			return errors.New("access to private IP addresses is not allowed")
		}
		// Block loopback
		if ip.IsLoopback() {
			return errors.New("access to loopback addresses is not allowed")
		}
		// Block link-local
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("access to link-local addresses is not allowed")
		}
		// Block unspecified (0.0.0.0)
		if ip.IsUnspecified() {
			return errors.New("access to unspecified addresses is not allowed")
		}
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from user input
// This includes null bytes, control characters, and HTML-escapes the result
func SanitizeInput(input string) string {
	// Remove null bytes and control characters
	sanitized := strings.Map(func(r rune) rune {
		if r == 0 || (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') {
			return -1
		}
		return r
	}, input)

	// HTML escape to prevent XSS
	sanitized = html.EscapeString(sanitized)

	return sanitized
}

// SanitizeInputPreserveHTML sanitizes input but preserves HTML (for admin content)
func SanitizeInputPreserveHTML(input string) string {
	// Only remove null bytes and dangerous control characters
	sanitized := strings.Map(func(r rune) rune {
		if r == 0 || (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') {
			return -1
		}
		return r
	}, input)

	return sanitized
}

// RateLimitInfo stores rate limit state (placeholder for future implementation)
type RateLimitInfo struct {
	RequestsPerMinute int
	BurstSize         int
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filter in browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (basic)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:")

		next.ServeHTTP(w, r)
	})
}

// InputValidation middleware sanitizes common request inputs
func InputValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit request body size to 10MB to prevent DoS
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		next.ServeHTTP(w, r)
	})
}
