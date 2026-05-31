package handlers

import (
	"encoding/json"
	"net/http"

	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/identity"
)

// RevokeUCAN creates a handler for revoking a UCAN by nonce.
// Only the UCAN issuer (authenticated via their DID) can revoke their own tokens.
func RevokeUCAN(revocations *identity.RevocationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Require authenticated identity — only the UCAN issuer can revoke
		callerDID := authMiddleware.GetIdentity(r)
		if callerDID == "" {
			http.Error(w, "authentication required to revoke UCANs", http.StatusUnauthorized)
			return
		}

		var req struct {
			Nonce string `json:"nonce"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Nonce == "" {
			http.Error(w, "nonce is required", http.StatusBadRequest)
			return
		}

		revocations.Revoke(req.Nonce, callerDID)
		if err := revocations.Save(); err != nil {
			http.Error(w, "failed to persist revocation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"revoked":    true,
			"nonce":      req.Nonce,
			"revoked_by": callerDID,
		})
	}
}

// ListRevocations creates a handler for listing all revoked UCANs.
func ListRevocations(revocations *identity.RevocationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		revoked := revocations.List()

		entries := make([]map[string]interface{}, 0, len(revoked))
		for nonce, revokedAt := range revoked {
			entries = append(entries, map[string]interface{}{
				"nonce":      nonce,
				"revoked_at": revokedAt.Unix(),
			})
		}

		paged, total := PaginateSlice(entries, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"revocations": paged,
			"total":       total,
			"offset":      offset,
			"limit":       limit,
		})
	}
}
