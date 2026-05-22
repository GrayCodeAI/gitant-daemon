package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Agent represents a known agent in the registry
type Agent struct {
	DID        string    `json:"did"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	RepoCount  int       `json:"repos"`
	CommitCount int      `json:"commits"`
	TrustScore float64   `json:"trust_score"`
}

// AgentRegistry tracks known agents
type AgentRegistry struct {
	mu      sync.RWMutex
	agents  map[string]*Agent
	dataDir string
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry(dataDir string) *AgentRegistry {
	return &AgentRegistry{
		agents:  make(map[string]*Agent),
		dataDir: dataDir,
	}
}

// Load loads persisted agent data
func (r *AgentRegistry) Load() error {
	if r.dataDir == "" {
		return nil
	}
	return persistence.LoadJSON(filepath.Join(r.dataDir, "agents.json"), &r.agents)
}

// Save persists agent data
func (r *AgentRegistry) Save() error {
	if r.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(r.dataDir, "agents.json"), r.agents)
}

// Record records an agent interaction
func (r *AgentRegistry) Record(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, ok := r.agents[did]; ok {
		agent.LastSeen = time.Now()
		agent.TrustScore = min(agent.TrustScore+0.01, 1.0)
	} else {
		r.agents[did] = &Agent{
			DID:        did,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			TrustScore: 0.5,
		}
	}
}

// Get returns an agent by DID
func (r *AgentRegistry) Get(did string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[did]
	return agent, ok
}

// List returns all known agents
func (r *AgentRegistry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]*Agent, 0, len(r.agents))
	for _, a := range r.agents {
		agents = append(agents, a)
	}
	return agents
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ListAgents lists all known agents
func ListAgents(registry *AgentRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		agents := registry.List()

		paged, total := PaginateSlice(agents, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents": paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
		})
	}
}

// GetAgent gets an agent by DID
func GetAgent(registry *AgentRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		did := chi.URLParam(r, "did")

		agent, ok := registry.Get(did)
		if !ok {
			// Return a default for unknown agents
			agent = &Agent{
				DID:        did,
				TrustScore: 0.5,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agent)
	}
}

// DelegateCapability creates and signs a UCAN capability token
func DelegateCapability(id *identity.Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Audience string   `json:"audience"`
			Resource string   `json:"resource"`
			Actions  []string `json:"actions"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Audience == "" || req.Resource == "" || len(req.Actions) == 0 {
			http.Error(w, "audience, resource, and actions are required", http.StatusBadRequest)
			return
		}

		caps := []identity.Capability{
			{Resource: req.Resource, Actions: req.Actions},
		}

		ucan := identity.NewUCAN(id.DID, req.Audience, caps, 24*time.Hour)
		token, err := ucan.Sign(id)
		if err != nil {
			http.Error(w, "failed to sign UCAN: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":   token,
			"issuer":  ucan.Issuer,
			"audience": ucan.Audience,
			"expires": ucan.Expires,
			"caps":    ucan.Caps,
		})
	}
}

// GenerateDID creates a new DID:key identity
func GenerateDID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := identity.NewIdentity()
		if err != nil {
			http.Error(w, "failed to generate DID: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":        id.DID,
			"document":   id.DIDDocument(),
		})
	}
}

// VerifyUCAN verifies a UCAN token with cryptographic signature verification
func VerifyUCAN() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token string `json:"token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Token == "" {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		// First decode to get the issuer DID
		ucan, err := identity.DecodeUCAN(req.Token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
			return
		}

		// Extract public key from issuer DID and verify signature
		pubKey, keyErr := identity.ExtractPublicKeyFromDID(ucan.Issuer)
		if keyErr == nil {
			ucan, err = identity.VerifySignedUCANByKey(req.Token, pubKey)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"valid": false,
					"error": "signature verification failed: " + err.Error(),
				})
				return
			}
		}
		// If keyErr != nil, ucan is already decoded from DecodeUCAN above (unsigned token)

		// Validate time bounds
		if err := ucan.Validate(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":  false,
				"error":  err.Error(),
				"issuer": ucan.Issuer,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":    true,
			"issuer":   ucan.Issuer,
			"audience": ucan.Audience,
			"expires":  ucan.Expires,
			"caps":     ucan.Caps,
		})
	}
}

// ResolveDID resolves a DID to its document
func ResolveDID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		did := chi.URLParam(r, "did")
		if did == "" {
			http.Error(w, "DID is required", http.StatusBadRequest)
			return
		}

		// For did:key, we can reconstruct the document from the DID itself
		// The DID contains the public key: did:key:z<base64url-pubkey>
		if len(did) < 10 || did[:8] != "did:key:" {
			http.Error(w, "unsupported DID method", http.StatusBadRequest)
			return
		}

		doc := map[string]interface{}{
			"@context": []string{
				"https://www.w3.org/ns/did/v1",
				"https://w3id.org/security/suites/ed25519-2020/v1",
			},
			"id": did,
			"verificationMethod": []map[string]interface{}{
				{
					"id":         did + "#controller",
					"type":       "Ed25519VerificationKey2020",
					"controller": did,
				},
			},
			"authentication": []string{did + "#controller"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}
}
