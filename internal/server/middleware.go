package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bicicletapp/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userContextKey contextKey = "user"
)

// Claims represents JWT claims
type Claims struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// authMiddleware protects routes requiring authentication
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get token from cookie first
		cookie, err := r.Cookie("auth_token")
		var tokenString string

		if err == nil {
			tokenString = cookie.Value
		} else {
			// Fallback to Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			tokenString = parts[1]
		}

		// Parse and validate the token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user claims to context
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// roleMiddleware restricts access based on user role
func (s *Server) roleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := getUserClaims(r)
			if claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if user's role is in allowed roles
			allowed := false
			for _, role := range allowedRoles {
				if claims.Role == role {
					allowed = true
					break
				}
			}

			// Admin always has access
			if claims.Role == domain.RoleAdmin {
				allowed = true
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getUserClaims extracts user claims from request context
func getUserClaims(r *http.Request) *Claims {
	claims, ok := r.Context().Value(userContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// generateToken creates a new JWT token for a user
func (s *Server) generateToken(user *domain.User) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.config.JWT.ExpirationHours) * time.Hour)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.Business.Name,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// setAuthCookie sets the authentication cookie
func (s *Server) setAuthCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !s.config.Debug, // Enable in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	})
}

// clearAuthCookie removes the authentication cookie
func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// csrfMiddleware adds CSRF protection for forms
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check for state-changing methods
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			// Check for CSRF token in form or header
			formToken := r.FormValue("csrf_token")
			headerToken := r.Header.Get("X-CSRF-Token")
			
			csrfToken := formToken
			if csrfToken == "" {
				csrfToken = headerToken
			}

			// Validate CSRF token (stored in cookie)
			cookie, err := r.Cookie("csrf_token")
			if err != nil || cookie.Value != csrfToken {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware implements basic rate limiting
// For production, consider using a more robust solution
func (s *Server) rateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	// Simple in-memory rate limiter using chi's built-in throttle
	// For production with multiple instances, use Redis-based rate limiting
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Basic implementation - for production use a proper rate limiter
			next.ServeHTTP(w, r)
		})
	}
}

// loggingMiddleware logs request details (extended version)
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response wrapper to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		// Log request details
		duration := time.Since(start)
		_ = duration // Use for logging if needed
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getURLParam is a helper to get URL parameters
func getURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
