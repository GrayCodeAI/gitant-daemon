package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CSRFProtector provides CSRF protection
type CSRFProtector struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
	ttl    time.Duration
}

// NewCSRFProtector creates a new CSRF protector
func NewCSRFProtector(ttl time.Duration) *CSRFProtector {
	p := &CSRFProtector{
		tokens: make(map[string]time.Time),
		ttl:    ttl,
	}

	// Start cleanup goroutine
	go p.cleanup()

	return p
}

// GenerateToken generates a new CSRF token
func (p *CSRFProtector) GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	p.mu.Lock()
	p.tokens[token] = time.Now().Add(p.ttl)
	p.mu.Unlock()

	return token
}

// ValidateToken validates a CSRF token
func (p *CSRFProtector) ValidateToken(token string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	expires, ok := p.tokens[token]
	if !ok {
		return false
	}

	if time.Now().After(expires) {
		return false
	}

	return true
}

// InvalidateToken invalidates a CSRF token
func (p *CSRFProtector) InvalidateToken(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.tokens, token)
}

func (p *CSRFProtector) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.removeExpired()
	}
}

func (p *CSRFProtector) removeExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for token, expires := range p.tokens {
		if now.After(expires) {
			delete(p.tokens, token)
		}
	}
}

// CSRFMiddleware provides CSRF protection for HTTP requests
func CSRFMiddleware(protector *CSRFProtector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip safe methods
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for CSRF token in header
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				// Check form value
				token = r.FormValue("csrf_token")
			}

			if token == "" || !protector.ValidateToken(token) {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}

			// Invalidate token after use (one-time use)
			protector.InvalidateToken(token)

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFTokenHandler returns a handler that provides CSRF tokens
func CSRFTokenHandler(protector *CSRFProtector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := protector.GenerateToken()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"csrf_token":"` + token + `"}`))
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		
		// Only set HSTS for HTTPS
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware provides CORS protection
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin || allowedOrigin == "*" {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// InputSanitizationMiddleware sanitizes request inputs
func InputSanitizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sanitize query parameters
		query := r.URL.Query()
		for key, values := range query {
			for i, value := range values {
				query[key][i] = SanitizeString(value)
			}
		}
		r.URL.RawQuery = query.Encode()

		// Sanitize path parameters are handled by chi

		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimitMiddleware limits request body size
func RequestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeMiddleware validates content type for POST/PUT requests
func ContentTypeMiddleware(allowedTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				contentType := r.Header.Get("Content-Type")
				
				// Allow empty content type for some requests
				if contentType == "" {
					next.ServeHTTP(w, r)
					return
				}

				allowed := false
				for _, allowedType := range allowedTypes {
					if strings.HasPrefix(contentType, allowedType) {
						allowed = true
						break
					}
				}

				if !allowed {
					http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
