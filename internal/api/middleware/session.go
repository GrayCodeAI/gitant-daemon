package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/store"
)

const (
	// UserContextKey is the context key for storing authenticated user information
	UserContextKey contextKey = "user"
)

// SessionAuthMiddleware validates session tokens and adds user to context
func SessionAuthMiddleware(authService *store.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractSessionToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authService.ValidateSession(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSessionAuth requires a valid session
func RequireSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole requires a specific role
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			hasRole := false
			for _, role := range roles {
				if user.Role == role || user.Role == "owner" {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUser gets the user from the request context
func GetUser(r *http.Request) *store.User {
	user, ok := r.Context().Value(UserContextKey).(*store.User)
	if !ok {
		return nil
	}
	return user
}

// WithUser returns a context with the user stored under the correct typed key.
// Use this from outside the middleware package since the contextKey type is unexported.
func WithUser(ctx context.Context, user *store.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// extractSessionToken extracts the session token from the request
func extractSessionToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check httpOnly session cookie first
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fall back to legacy gitant_session cookie
	cookie, err = r.Cookie("gitant_session")
	if err == nil {
		return cookie.Value
	}

	return ""
}
