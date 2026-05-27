package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/identity"
)

func TestRequireIdentity_NoIdentity(t *testing.T) {
	handler := RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireIdentity_WithIdentity(t *testing.T) {
	handler := RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did := GetIdentity(r)
		if did == "" {
			t.Fatal("expected identity in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), IdentityKey, "did:key:ztest123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireCapability_NoUCAN(t *testing.T) {
	handler := RequireCapability("repo:test", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireCapability_WithMatchingUCAN(t *testing.T) {
	ucan := &identity.UCAN{
		Issuer: "did:key:zissuer",
		Caps: []identity.Capability{
			{Resource: "repo:test", Actions: []string{"read", "write"}},
		},
	}

	handler := RequireCapability("repo:test", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), UCANKey, ucan)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireCapability_WithNonMatchingUCAN(t *testing.T) {
	ucan := &identity.UCAN{
		Issuer: "did:key:zissuer",
		Caps: []identity.Capability{
			{Resource: "repo:test", Actions: []string{"read"}},
		},
	}

	handler := RequireCapability("repo:test", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), UCANKey, ucan)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestNewHTTPSignatureMiddleware_NoAuth(t *testing.T) {
	revocations := identity.NewRevocationStore("")
	middleware := NewHTTPSignatureMiddleware(revocations, nil, "did:key:zserver")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did := GetIdentity(r)
		if did != "" {
			t.Fatal("expected no identity for unauthenticated request")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewHTTPSignatureMiddleware_ValidUCAN(t *testing.T) {
	issuer, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	revocations := identity.NewRevocationStore("")
	serverDID := "did:key:zserver"

	middleware := NewHTTPSignatureMiddleware(revocations, nil, serverDID)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did := GetIdentity(r)
		if did != issuer.DID {
			t.Fatalf("expected DID %s, got %s", issuer.DID, did)
		}
		ucan := GetUCAN(r)
		if ucan == nil {
			t.Fatal("expected UCAN in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create a valid UCAN for the server
	ucan := identity.NewUCAN(issuer.DID, serverDID, []identity.Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPSignatureMiddleware_WrongAudience(t *testing.T) {
	issuer, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	revocations := identity.NewRevocationStore("")
	serverDID := "did:key:zserver"

	middleware := NewHTTPSignatureMiddleware(revocations, nil, serverDID)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create UCAN with wrong audience
	ucan := identity.NewUCAN(issuer.DID, "did:key:zwrongaudience", []identity.Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPSignatureMiddleware_WildcardAudience(t *testing.T) {
	issuer, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	revocations := identity.NewRevocationStore("")
	serverDID := "did:key:zserver"

	middleware := NewHTTPSignatureMiddleware(revocations, nil, serverDID)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create UCAN with wildcard audience
	ucan := identity.NewUCAN(issuer.DID, "*", []identity.Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPSignatureMiddleware_RevokedUCAN(t *testing.T) {
	issuer, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	revocations := identity.NewRevocationStore("")
	serverDID := "did:key:zserver"

	middleware := NewHTTPSignatureMiddleware(revocations, nil, serverDID)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create UCAN and revoke it
	ucan := identity.NewUCAN(issuer.DID, serverDID, []identity.Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	// Revoke the UCAN by nonce
	revocations.Revoke(ucan.Nonce)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPSignatureMiddleware_ExpiredUCAN(t *testing.T) {
	issuer, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	revocations := identity.NewRevocationStore("")
	serverDID := "did:key:zserver"

	middleware := NewHTTPSignatureMiddleware(revocations, nil, serverDID)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create expired UCAN
	ucan := identity.NewUCAN(issuer.DID, serverDID, []identity.Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	ucan.Expires = time.Now().Add(-1 * time.Hour).Unix()
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetIdentity_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	did := GetIdentity(req)
	if did != "" {
		t.Fatalf("expected empty string, got %s", did)
	}
}

func TestGetUCAN_Nil(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ucan := GetUCAN(req)
	if ucan != nil {
		t.Fatal("expected nil UCAN")
	}
}

func TestRequireRepoWriteCapability_AllowsOperator(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/repos/{id}", func(r chi.Router) {
		r.Use(RequireRepoWriteCapability("id"))
		r.Post("/issues", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
	})

	req := httptest.NewRequest("POST", "/repos/my-repo/issues", nil)
	ctx := context.WithValue(req.Context(), IdentityKey, "did:key:zoperator")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestRequireRepoWriteCapability_DeniesReadOnlyUCAN(t *testing.T) {
	ucan := &identity.UCAN{
		Issuer: "did:key:zagent",
		Caps: []identity.Capability{
			{Resource: "repo:my-repo", Actions: []string{"read"}},
		},
	}

	r := chi.NewRouter()
	r.Route("/repos/{id}", func(r chi.Router) {
		r.Use(RequireRepoWriteCapability("id"))
		r.Post("/issues", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
	})

	req := httptest.NewRequest("POST", "/repos/my-repo/issues", nil)
	ctx := context.WithValue(req.Context(), UCANKey, ucan)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
