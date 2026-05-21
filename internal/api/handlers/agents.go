package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/identity"
)

// ListAgents lists all known agents
func ListAgents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement agent registry
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents": []interface{}{},
			"total":  0,
		})
	}
}

// GetAgent gets an agent by DID
func GetAgent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		did := chi.URLParam(r, "did")

		// TODO: Look up agent in registry
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":         did,
			"trust_score": 0.5,
			"repos":       0,
			"commits":     0,
		})
	}
}

// DelegateCapability delegates a UCAN capability
func DelegateCapability(id *identity.Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Audience string   `json:"audience"`
			Resource string   `json:"resource"`
			Actions  []string `json:"actions"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		caps := []identity.Capability{
			{Resource: req.Resource, Actions: req.Actions},
		}

		ucan := identity.NewUCAN(id.DID, req.Audience, caps, 3600) // 1 hour
		encoded, err := ucan.Encode()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"ucan":     encoded,
			"issuer":   id.DID,
			"audience": req.Audience,
		})
	}
}

// GenerateDID generates a new DID
func GenerateDID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newIdentity, err := identity.NewIdentity()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		doc := newIdentity.DIDDocument()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":      newIdentity.DID,
			"document": doc,
		})
	}
}

// ResolveDID resolves a DID to its document
func ResolveDID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		did := chi.URLParam(r, "did")

		// For did:key, we can parse the public key directly
		if len(did) > 8 && did[:8] == "did:key:" {
			// Return a basic DID document
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"@context": []string{"https://www.w3.org/ns/did/v1"},
				"id":       did,
				"publicKey": []map[string]interface{}{
					{
						"id":              did + "#key-1",
						"type":            "Ed25519VerificationKey2020",
						"controller":      did,
						"publicKeyBase58": did[8:], // Raw key material
					},
				},
			})
			return
		}

		http.Error(w, "Unsupported DID method", http.StatusBadRequest)
	}
}
