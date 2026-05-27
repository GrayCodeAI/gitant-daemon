package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodUCAN    AuthMethod = "ucan"
	AuthMethodAPIKey  AuthMethod = "api_key"
	AuthMethodOAuth2  AuthMethod = "oauth2"
	AuthMethodSession AuthMethod = "session"
)

// OAuth2Provider represents an OAuth2 provider
type OAuth2Provider struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"user_info_url"`
	Scopes       []string `json:"scopes"`
	Enabled      bool     `json:"enabled"`
}

// APIKey represents an API key
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	KeyHash     string    `json:"key_hash"`
	Prefix      string    `json:"prefix"`
	DID         string    `json:"did"`
	Scopes      []string  `json:"scopes"`
	RateLimit   int       `json:"rate_limit"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Active      bool       `json:"active"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	DID       string    `json:"did"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// OAuth2State represents OAuth2 flow state
type OAuth2State struct {
	State     string    `json:"state"`
	Provider  string    `json:"provider"`
	DID       string    `json:"did"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthManager manages authentication
type AuthManager struct {
	mu         sync.RWMutex
	apiKeys    map[string]*APIKey
	sessions   map[string]*Session
	oauthState map[string]*OAuth2State
	providers  map[string]*OAuth2Provider
}

// NewAuthManager creates a new auth manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		apiKeys:    make(map[string]*APIKey),
		sessions:   make(map[string]*Session),
		oauthState: make(map[string]*OAuth2State),
		providers:  make(map[string]*OAuth2Provider),
	}
}

// RegisterOAuth2Provider registers an OAuth2 provider
func (m *AuthManager) RegisterOAuth2Provider(provider *OAuth2Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.Name] = provider
}

// GenerateAPIKey generates a new API key
func (m *AuthManager) GenerateAPIKey(did, name string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("generating key: %w", err)
	}

	rawKey := hex.EncodeToString(keyBytes)
	prefix := rawKey[:8]

	apiKey := &APIKey{
		ID:        fmt.Sprintf("key-%x", generateRandomID()),
		Name:      name,
		KeyHash:   hashKey(rawKey),
		Prefix:    prefix,
		DID:       did,
		Scopes:    scopes,
		RateLimit: 1000,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		Active:    true,
	}

	m.apiKeys[apiKey.ID] = apiKey

	return apiKey, fmt.Sprintf("gt_%s_%s", prefix, rawKey), nil
}

// ValidateAPIKey validates an API key and returns the associated DID
func (m *AuthManager) ValidateAPIKey(rawKey string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keyHash := hashKey(rawKey)

	for _, key := range m.apiKeys {
		if key.KeyHash == keyHash && key.Active {
			if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
				return nil, fmt.Errorf("key expired")
			}
			now := time.Now()
			key.LastUsedAt = &now
			return key, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// RevokeAPIKey revokes an API key
func (m *AuthManager) RevokeAPIKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := m.apiKeys[keyID]
	if !ok {
		return fmt.Errorf("key not found")
	}

	key.Active = false
	return nil
}

// ListAPIKeys lists API keys for a DID
func (m *AuthManager) ListAPIKeys(did string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []*APIKey
	for _, key := range m.apiKeys {
		if key.DID == did {
			keys = append(keys, key)
		}
	}
	return keys
}

// CreateOAuth2State creates an OAuth2 flow state
func (m *AuthManager) CreateOAuth2State(provider, did string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	m.oauthState[state] = &OAuth2State{
		State:     state,
		Provider:  provider,
		DID:       did,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	return state
}

// ValidateOAuth2State validates and consumes an OAuth2 state
func (m *AuthManager) ValidateOAuth2State(state string) (*OAuth2State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.oauthState[state]
	if !ok {
		return nil, fmt.Errorf("invalid state")
	}

	if s.ExpiresAt.Before(time.Now()) {
		delete(m.oauthState, state)
		return nil, fmt.Errorf("state expired")
	}

	delete(m.oauthState, state)
	return s, nil
}

// CreateSession creates a new session
func (m *AuthManager) CreateSession(did, ip, userAgent string, duration time.Duration) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:        fmt.Sprintf("sess-%x", generateRandomID()),
		DID:       did,
		IP:        ip,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	m.sessions[session.ID] = session
	return session
}

// ValidateSession validates a session
func (m *AuthManager) ValidateSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// GetOAuth2Providers returns all registered OAuth2 providers
func (m *AuthManager) GetOAuth2Providers() []*OAuth2Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var providers []*OAuth2Provider
	for _, p := range m.providers {
		if p.Enabled {
			providers = append(providers, p)
		}
	}
	return providers
}

func hashKey(key string) string {
	// Use SHA-256 for API key hashing
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func generateRandomID() [16]byte {
	var id [16]byte
	rand.Read(id[:])
	return id
}
