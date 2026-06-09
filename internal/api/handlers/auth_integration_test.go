package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/application/service"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/infrastructure/adapters"
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

	// Create repository service for the handler
	repoAdapter := adapters.NewRepositoryAdapter(reg)
	factory := service.NewServiceFactory(repoAdapter, nil, nil, nil, nil, nil)
	repoService := factory.CreateRepositoryService()

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
			r.Get("/{id}", GetRepo(repoService))
			r.Get("/{id}/info/refs", InfoRefs(reg))
			r.Post("/{id}/git-upload-pack", GitUploadPack(reg))
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Use(RequireRepoReadAccess(reg, serverID.DID))
			r.Use(authMiddleware.RequireRepoWriteCapability("id"))
			r.Post("/{id}/issues", CreateIssue(issueStore, wm))
			r.Post("/{id}/git-receive-pack", GitReceivePack(reg, setupTestProtectionStore(t), wm))
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

func TestAuthIntegration_ServerOperatorRequiresUCANForCapability(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	// HTTP Signature operators must present a UCAN for capability-gated endpoints.
	// Without a UCAN, the request should be rejected with 403.
	body := bytes.NewBufferString(`{"title":"operator issue","body":""}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
	req.Header.Set("Content-Type", "application/json")
	req = contextWithIdentity(req, serverID.DID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (UCAN required), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_PublicUploadPackDoesNotRequireWriteCapability(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	token := signUCAN(t, serverID, serverID.DID, "repo:public-repo", []string{"read"})

	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-upload-pack", bytes.NewBufferString(""))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized {
		t.Fatalf("upload-pack should not require write capability, got %d: %s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusBadRequest || w.Body.String() != "empty request\n" {
		t.Fatalf("expected handler to receive request and reject empty body, got %d: %q", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_PrivateUploadPackRequiresReadAccess(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/repos/private-repo/git-upload-pack", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected private upload-pack without credentials to be hidden, got %d: %s", w.Code, w.Body.String())
	}

	token := signUCAN(t, serverID, serverID.DID, "repo:private-repo", []string{"read"})
	req = httptest.NewRequest("POST", "/api/v1/repos/private-repo/git-upload-pack", bytes.NewBufferString(""))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized || w.Code == http.StatusNotFound {
		t.Fatalf("read-authorized upload-pack should reach handler, got %d: %s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusBadRequest || w.Body.String() != "empty request\n" {
		t.Fatalf("expected handler to receive request and reject empty body, got %d: %q", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_InfoRefsUploadPackPublicAndPrivateReadAccess(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/repos/public-repo/info/refs?service=git-upload-pack", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected public upload-pack info/refs to succeed, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-git-upload-pack-advertisement" {
		t.Fatalf("expected upload-pack advertisement content type, got %q", ct)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("# service=git-upload-pack")) {
		t.Fatalf("expected upload-pack service advertisement, got %q", w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/v1/repos/private-repo/info/refs?service=git-upload-pack", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected private upload-pack info/refs without credentials to be hidden, got %d: %s", w.Code, w.Body.String())
	}

	token := signUCAN(t, serverID, serverID.DID, "repo:private-repo", []string{"read"})
	req = httptest.NewRequest("GET", "/api/v1/repos/private-repo/info/refs?service=git-upload-pack", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected private upload-pack info/refs with read UCAN to succeed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_ReceivePackStillRequiresWriteCapability(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-receive-pack", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous receive-pack to require identity, got %d: %s", w.Code, w.Body.String())
	}

	token := signUCAN(t, serverID, serverID.DID, "repo:public-repo", []string{"read"})
	req = httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-receive-pack", bytes.NewBufferString(""))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected receive-pack with read-only UCAN to require write capability, got %d: %s", w.Code, w.Body.String())
	}
}
