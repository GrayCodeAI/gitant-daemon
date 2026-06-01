package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/store"
)

// OAuthHandler handles OAuth authentication flows
type OAuthHandler struct {
	auth *store.AuthService
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(auth *store.AuthService) *OAuthHandler {
	return &OAuthHandler{auth: auth}
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	Name        string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	ClientID    string
}

// OAuthProviders holds configured OAuth providers
var OAuthProviders = map[string]OAuthProvider{
	"github": {
		Name:        "github",
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
	},
	"gitlab": {
		Name:        "gitlab",
		AuthURL:     "https://gitlab.com/oauth/authorize",
		TokenURL:    "https://gitlab.com/oauth/token",
		UserInfoURL: "https://gitlab.com/api/v4/user",
	},
	"google": {
		Name:        "google",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
	},
}

// InitiateOAuth starts the OAuth flow by redirecting to provider
func (h *OAuthHandler) InitiateOAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Get provider config
	providerConfig, ok := OAuthProviders[provider]
	if !ok {
		http.Error(w, "unsupported OAuth provider", http.StatusBadRequest)
		return
	}

	// Generate state parameter for CSRF protection
	state := generateRandomState()

	// Build authorization URL
	authURL, err := url.Parse(providerConfig.AuthURL)
	if err != nil {
		http.Error(w, "invalid provider URL", http.StatusInternalServerError)
		return
	}

	query := authURL.Query()
	query.Set("client_id", providerConfig.ClientID)
	query.Set("redirect_uri", fmt.Sprintf("http://localhost:3303/api/v1/auth/oauth/%s/callback", provider))
	query.Set("state", state)
	query.Set("response_type", "code")
	authURL.RawQuery = query.Encode()

	// Store state in session or cookie (simplified for now)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
	})

	// Redirect to provider
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

// CallbackOAuth handles the OAuth callback from provider
func (h *OAuthHandler) CallbackOAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Verify state parameter (CSRF protection)
	storedState, err := r.Cookie("oauth_state")
	if err != nil || storedState.Value != state {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	providerConfig, ok := OAuthProviders[provider]
	if !ok {
		http.Error(w, "unsupported OAuth provider", http.StatusBadRequest)
		return
	}

	// Exchange code for access token
	_, err = url.Parse(providerConfig.TokenURL)
	if err != nil {
		http.Error(w, "invalid provider URL", http.StatusInternalServerError)
		return
	}

	// In a real implementation, you would:
	// 1. Exchange code for access token via POST request
	// 2. Fetch user info using the access token
	// 3. Find or create user in database
	// 4. Create session and return token

	// For now, return a placeholder response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "OAuth callback received (implementation pending)",
		"provider":  provider,
		"code":      code,
	})
}

// OAuthUserInfo represents the user info from OAuth provider
type OAuthUserInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Username    string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
}

// generateRandomState generates a random state for CSRF protection
func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
