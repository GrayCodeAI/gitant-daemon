package middleware

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/identity"
)

type contextKey string

const (
	IdentityKey  contextKey = "identity"
	SignatureKey contextKey = "signature"
	UCANKey      contextKey = "ucan"
	// AuthAttemptedKey is set when the auth middleware has run, even if no
	// credentials were provided. Downstream middleware can distinguish
	// "no auth header" from "auth middleware not applied".
	AuthAttemptedKey contextKey = "auth_attempted"
)

// NewHTTPSignatureMiddleware creates auth middleware with revocation, replay protection, and audience checking.
func NewHTTPSignatureMiddleware(revocations *identity.RevocationStore, nonces *identity.NonceCache, serverDID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mark that auth middleware has been applied, even for unauthenticated
			// requests. This lets downstream middleware distinguish "no credentials"
			// from "auth middleware not applied to this route".
			ctx := context.WithValue(r.Context(), AuthAttemptedKey, true)

			auth := r.Header.Get("Authorization")
			if auth == "" {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Handle UCAN Bearer tokens
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")

				// Distinguish a UCAN from an opaque session token by shape.
				// UCANs are structured tokens with dot-separated base64 segments
				// (header.payload.signature). Opaque session tokens are flat
				// strings with no dots. If the token is not UCAN-shaped, fall
				// through so downstream session auth middleware can validate it.
				if !looksLikeUCAN(token) {
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				ucan, err := identity.VerifySignedUCANWithChain(token, revocations, nonces)
				if err != nil {
					// Token is UCAN-shaped but failed verification (bad
					// signature, expired, or revoked). This is a genuine
					// authentication failure — reject.
					http.Error(w, "Invalid or expired authentication token", http.StatusUnauthorized)
					return
				}

				// Validate audience matches this server — wildcard audience is rejected
				// as it weakens the audience-binding security property
				if serverDID != "" && ucan.Audience != serverDID {
					http.Error(w, "UCAN audience does not match this server", http.StatusForbidden)
					return
				}

				ctx = context.WithValue(ctx, IdentityKey, ucan.Issuer)
				ctx = context.WithValue(ctx, UCANKey, ucan)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Handle HTTP Signatures
			if !strings.HasPrefix(auth, "Signature") {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}

			params, err := parseSignatureParams(auth)
			if err != nil {
				http.Error(w, "Invalid signature parameters", http.StatusBadRequest)
				return
			}

			keyId, ok := params["keyId"]
			if !ok {
				http.Error(w, "Missing keyId in signature", http.StatusBadRequest)
				return
			}

			signature, ok := params["signature"]
			if !ok {
				http.Error(w, "Missing signature", http.StatusBadRequest)
				return
			}

			sigBytes, err := base64.StdEncoding.DecodeString(signature)
			if err != nil {
				http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
				return
			}

			signingString, err := buildSigningString(r, params)
			if err != nil {
				http.Error(w, "Invalid signature", http.StatusBadRequest)
				return
			}

			pubKey, err := identity.ExtractPublicKeyFromDID(keyId)
			if err != nil {
				http.Error(w, "Invalid authentication key", http.StatusBadRequest)
				return
			}

			if !ed25519.Verify(pubKey, []byte(signingString), sigBytes) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(ctx, IdentityKey, keyId)
			ctx = context.WithValue(ctx, SignatureKey, params)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireIdentity is middleware that rejects requests without a valid identity
// It checks for UCAN/HTTP Signature identity first, then falls back to session-based authentication
func RequireIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First check for UCAN/HTTP Signature identity
		did, ok := r.Context().Value(IdentityKey).(string)
		if ok && did != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Fall back to checking for session-based authentication
		user := GetUser(r)
		if user != nil {
			// For session-based auth, set the identity to the user ID for compatibility
			ctx := context.WithValue(r.Context(), IdentityKey, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, "Authentication required", http.StatusUnauthorized)
	})
}

// RequireCapability returns middleware that checks for a specific UCAN capability.
// All authenticated requests (both UCAN Bearer and HTTP Signature) must present a
// valid UCAN with the required capability. HTTP Signature alone is not sufficient
// for capability-gated endpoints — the caller must also hold a UCAN delegation.
func RequireCapability(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetIdentity(r) == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			ucan := GetUCAN(r)
			if ucan == nil {
				http.Error(w, "UCAN capability token required for this operation", http.StatusForbidden)
				return
			}
			if !ucan.HasCapability(resource, action) {
				http.Error(w, "Insufficient capabilities", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRepoWriteCapability enforces repo:{id} write for all authenticated requests.
// Both UCAN Bearer and HTTP Signature callers must present a valid UCAN with the
// required repo write capability.
func RequireRepoWriteCapability(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			repoID := chi.URLParam(r, paramName)
			if repoID == "" {
				http.Error(w, "Repository ID required", http.StatusBadRequest)
				return
			}
			RequireCapability("repo:"+repoID, "write")(next).ServeHTTP(w, r)
		})
	}
}

// GetIdentity extracts the DID from the request context, or empty string
func GetIdentity(r *http.Request) string {
	if did, ok := r.Context().Value(IdentityKey).(string); ok {
		return did
	}
	return ""
}

// AuthAttempted returns true if the auth middleware has been applied to this request,
// even if no credentials were provided.
func AuthAttempted(r *http.Request) bool {
	v, _ := r.Context().Value(AuthAttemptedKey).(bool)
	return v
}

// GetUCAN extracts the verified UCAN from the request context
func GetUCAN(r *http.Request) *identity.UCAN {
	if ucan, ok := r.Context().Value(UCANKey).(*identity.UCAN); ok {
		return ucan
	}
	return nil
}

// looksLikeUCAN reports whether a Bearer token has the structural shape of a
// signed UCAN (base64(payload).base64(signature)) rather than an opaque session
// token. Session tokens are flat hex strings with no dots, so the presence of a
// dot-separated two-part structure distinguishes a UCAN. This lets the auth
// middleware cleanly hand opaque session tokens to the session auth layer while
// still rejecting malformed/expired/revoked UCANs.
func looksLikeUCAN(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	return parts[0] != "" && parts[1] != ""
}

// parseSignatureParams parses the Authorization header signature params
func parseSignatureParams(auth string) (map[string]string, error) {
	params := make(map[string]string)

	// Remove "Signature " prefix
	auth = strings.TrimPrefix(auth, "Signature ")

	// Parse key=value pairs
	pairs := strings.Split(auth, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		params[key] = value
	}

	return params, nil
}

// buildSigningString builds the signing string for verification
func buildSigningString(r *http.Request, params map[string]string) (string, error) {
	headers, ok := params["headers"]
	if !ok {
		headers = "(request-target) date host"
	}

	var parts []string
	for _, header := range strings.Split(headers, " ") {
		switch header {
		case "(request-target)":
			parts = append(parts, fmt.Sprintf("(request-target): %s %s", strings.ToLower(r.Method), r.URL.Path))
		case "date":
			parts = append(parts, "date: "+r.Header.Get("Date"))
		case "host":
			parts = append(parts, "host: "+r.Host)
		default:
			parts = append(parts, header+": "+r.Header.Get(header))
		}
	}

	return strings.Join(parts, "\n"), nil
}
