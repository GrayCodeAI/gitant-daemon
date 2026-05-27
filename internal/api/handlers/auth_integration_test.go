package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/identity"
)

func setupAuthIntegrationRouter(t *testing.T) (*chi.Mux, *identity.Identity) {
	t.Helper()

	serverID, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	reg := setupTestRegistry(t)
	issueStore := setupTestIssueStore(t)
	wm := setupTestWebhookManager(t)
	revocations := identity.NewRevocationStore("")

	if _, err := reg.Create("public-repo", "public-repo", "public", false); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Create("private-repo", "private-repo", "secret", true); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(authMiddleware.NewHTTPSignatureMiddleware(revocations, nil, serverID.DID))

	r.Route("/api/v1/repos", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(RequireRepoReadAccess(reg, serverID.DID))
			r.Get("/{id}", GetRepo(reg))
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Use(RequireRepoReadAccess(reg, serverID.DID))
			r.Use(authMiddleware.RequireRepoWriteCapability("id"))
			r.Post("/{id}/issues", CreateIssue(issueStore, wm))
		})
	})

	return r, serverID
}

func signUCAN(t *testing.T, issuer *identity.Identity, audience, resource string, actions []string) string {
	t.Helper()
	ucan := identity.NewUCAN(issuer.DID, audience, []identity.Capability{
		{Resource: resource, Actions: actions},
	}, time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAuthIntegration_PrivateRepoReadDenied(t *testing.T) {
	r, _ := setupAuthIntegrationRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/repos/private-repo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_PrivateRepoReadWithUCAN(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	token := signUCAN(t, serverID, serverID.DID, "repo:private-repo", []string{"read"})

	req := httptest.NewRequest("GET", "/api/v1/repos/private-repo", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_WriteDeniedWithReadOnlyUCAN(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	token := signUCAN(t, serverID, serverID.DID, "repo:public-repo", []string{"read"})

	body := bytes.NewBufferString(`{"title":"blocked","body":""}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_WriteAllowedWithWriteUCAN(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	token := signUCAN(t, serverID, serverID.DID, "repo:public-repo", []string{"write"})

	body := bytes.NewBufferString(`{"title":"from agent","body":"hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_ServerOperatorBypassesWriteCapability(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	body := bytes.NewBufferString(`{"title":"operator issue","body":""}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
	req.Header.Set("Content-Type", "application/json")
	req = contextWithIdentity(req, serverID.DID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
