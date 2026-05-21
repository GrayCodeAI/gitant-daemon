package middleware

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const (
	IdentityKey contextKey = "identity"
	SignatureKey contextKey = "signature"
)

// HTTPSignatureMiddleware verifies RFC 9421 HTTP signatures
func HTTPSignatureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(auth, "Signature") {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		// Parse signature params
		params, err := parseSignatureParams(auth)
		if err != nil {
			http.Error(w, "Invalid signature params: "+err.Error(), http.StatusBadRequest)
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

		// Decode signature
		sigBytes, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
			return
		}

		// Build signing string
		signingString, err := buildSigningString(r, params)
		if err != nil {
			http.Error(w, "Failed to build signing string: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Extract public key from DID
		pubKey, err := extractPublicKey(keyId)
		if err != nil {
			http.Error(w, "Failed to extract public key: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Verify signature
		if !ed25519.Verify(pubKey, []byte(signingString), sigBytes) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Add identity to context
		ctx := context.WithValue(r.Context(), IdentityKey, keyId)
		ctx = context.WithValue(ctx, SignatureKey, params)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
