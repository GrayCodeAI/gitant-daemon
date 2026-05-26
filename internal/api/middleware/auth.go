package middleware

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/identity"
)

type contextKey string

const (
	IdentityKey contextKey = "identity"
	SignatureKey contextKey = "signature"
	UCANKey      contextKey = "ucan"
)

// NewHTTPSignatureMiddleware creates auth middleware with revocation and audience checking.
func NewHTTPSignatureMiddleware(revocations *identity.RevocationStore, serverDID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Handle UCAN Bearer tokens
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				ucan, err := identity.VerifySignedUCANWithChain(token, revocations)
				if err != nil {
					http.Error(w, "Invalid or expired authentication token", http.StatusUnauthorized)
					return
				}

				// Validate audience matches this server (or wildcard)
				if serverDID != "" && ucan.Audience != serverDID && ucan.Audience != "*" {
					http.Error(w, "UCAN audience does not match this server", http.StatusForbidden)
					return
				}

				ctx := context.WithValue(r.Context(), IdentityKey, ucan.Issuer)
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

			pubKey, err := extractPublicKey(keyId)
			if err != nil {
				http.Error(w, "Invalid authentication key", http.StatusBadRequest)
				return
			}

			if !ed25519.Verify(pubKey, []byte(signingString), sigBytes) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), IdentityKey, keyId)
			ctx = context.WithValue(ctx, SignatureKey, params)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireIdentity is middleware that rejects requests without a valid identity
func RequireIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := r.Context().Value(IdentityKey).(string)
		if !ok || did == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireCapability returns middleware that checks for a specific UCAN capability.
// HTTP-signature authenticated operators (no UCAN) are allowed through.
func RequireCapability(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ucan := GetUCAN(r)
			if ucan == nil {
				if GetIdentity(r) != "" {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Insufficient capabilities", http.StatusForbidden)
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

// GetIdentity extracts the DID from the request context, or empty string
func GetIdentity(r *http.Request) string {
	if did, ok := r.Context().Value(IdentityKey).(string); ok {
		return did
	}
	return ""
}

// GetUCAN extracts the verified UCAN from the request context
func GetUCAN(r *http.Request) *identity.UCAN {
	if ucan, ok := r.Context().Value(UCANKey).(*identity.UCAN); ok {
		return ucan
	}
	return nil
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

// extractPublicKey extracts the public key from a DID:key
func extractPublicKey(did string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(did, "did:key:z") {
		return nil, fmt.Errorf("invalid DID format: %s", did)
	}

	// Remove "did:key:z" prefix
	encoded := strings.TrimPrefix(did, "did:key:z")

	// Decode base64
	pubKey, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}

	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(pubKey))
	}

	return ed25519.PublicKey(pubKey), nil
}
