package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/application/service"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/infrastructure/adapters"
	"github.com/lakshmanpatel/gitant/internal/store"
)

func setupAuthIntegrationRouter(t *testing.T) (*chi.Mux, *identity.Identity) {
	t.Helper()
	r, serverID, _ := setupAuthIntegrationRouterWithACL(t)
	return r, serverID
}

func setupAuthIntegrationRouterWithACL(t *testing.T) (*chi.Mux, *identity.Identity, store.RepoCollaboratorStore) {
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
	acl := store.NewMemoryRepoCollaboratorStore()

	r := chi.NewRouter()
	r.Use(authMiddleware.NewHTTPSignatureMiddleware(revocations, nil, serverID.DID))

	r.Route("/api/v1/repos", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(RequireRepoReadAccessWithCollaborators(reg, serverID.DID, acl))
			r.Get("/{id}", GetRepo(repoService))
			r.Get("/{id}/info/refs", InfoRefs(reg))
			r.Post("/{id}/git-upload-pack", GitUploadPack(reg))
			r.Get("/{id}/collaborators", ListCollaborators(acl))
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Use(RequireRepoReadAccessWithCollaborators(reg, serverID.DID, acl))
			r.Use(authMiddleware.RequireRepoWrite("id", acl))
			r.Post("/{id}/issues", CreateIssue(issueStore, wm))
			r.Post("/{id}/git-receive-pack", GitReceivePack(reg, setupTestProtectionStore(t), wm))
			r.Post("/{id}/collaborators", AddCollaborator(acl))
			r.Delete("/{id}/collaborators/{user}", RemoveCollaborator(acl))
		})
	})

	return r, serverID, acl
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

func TestAuthIntegration_WriteDeniedWithUCANScopedToDifferentRepo(t *testing.T) {
	r, serverID := setupAuthIntegrationRouter(t)

	token := signUCAN(t, serverID, serverID.DID, "repo:other-repo", []string{"write"})

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

func TestAuthIntegration_SessionOwnerAndCollaboratorCanWrite(t *testing.T) {
	r, _, acl := setupAuthIntegrationRouterWithACL(t)
	owner := &store.User{ID: "owner-user", Username: "owner"}
	collaborator := &store.User{ID: "collab-user", Username: "collab"}
	if err := acl.Add(context.Background(), &store.RepoCollaborator{RepoID: "public-repo", UserID: owner.ID, Role: store.RepoRoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err := acl.Add(context.Background(), &store.RepoCollaborator{RepoID: "public-repo", UserID: collaborator.ID, Role: store.RepoRoleCollaborator}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		user *store.User
	}{
		{name: "owner", user: owner},
		{name: "collaborator", user: collaborator},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"title":"session issue","body":"hello"}`)
			req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
			req.Header.Set("Content-Type", "application/json")
			ctx := authMiddleware.WithUser(req.Context(), tc.user)
			ctx = context.WithValue(ctx, authMiddleware.IdentityKey, tc.user.ID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthIntegration_SessionNonOwnerCannotWrite(t *testing.T) {
	r, _, _ := setupAuthIntegrationRouterWithACL(t)
	user := &store.User{ID: "stranger-user", Username: "stranger"}

	body := bytes.NewBufferString(`{"title":"blocked","body":""}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := authMiddleware.WithUser(req.Context(), user)
	ctx = context.WithValue(ctx, authMiddleware.IdentityKey, user.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_CollaboratorManagementAllowsOwnerToGrantAndRevokeWrite(t *testing.T) {
	r, _, acl := setupAuthIntegrationRouterWithACL(t)
	owner := &store.User{ID: "owner-user", Username: "owner"}
	collaborator := &store.User{ID: "collab-user", Username: "collab"}
	if err := acl.Add(context.Background(), &store.RepoCollaborator{RepoID: "public-repo", UserID: owner.ID, Role: store.RepoRoleOwner}); err != nil {
		t.Fatal(err)
	}

	addBody := bytes.NewBufferString(`{"user_id":"collab-user"}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/collaborators", addBody)
	req.Header.Set("Content-Type", "application/json")
	req = requestWithSessionUser(req, owner)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected owner add collaborator 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/v1/repos/public-repo/collaborators", nil)
	req = requestWithSessionUser(req, owner)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected collaborator list 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("collab-user")) {
		t.Fatalf("expected list to include collab-user, got %s", w.Body.String())
	}

	issueBody := bytes.NewBufferString(`{"title":"from collaborator","body":"hello"}`)
	req = httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", issueBody)
	req.Header.Set("Content-Type", "application/json")
	req = requestWithSessionUser(req, collaborator)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected collaborator session write 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("DELETE", "/api/v1/repos/public-repo/collaborators/collab-user", nil)
	req = requestWithSessionUser(req, owner)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected owner remove collaborator 200, got %d: %s", w.Code, w.Body.String())
	}

	issueBody = bytes.NewBufferString(`{"title":"after removal","body":"blocked"}`)
	req = httptest.NewRequest("POST", "/api/v1/repos/public-repo/issues", issueBody)
	req.Header.Set("Content-Type", "application/json")
	req = requestWithSessionUser(req, collaborator)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected removed collaborator write 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthIntegration_CollaboratorManagementRequiresWriteCapability(t *testing.T) {
	r, _, acl := setupAuthIntegrationRouterWithACL(t)
	owner := &store.User{ID: "owner-user", Username: "owner"}
	stranger := &store.User{ID: "stranger-user", Username: "stranger"}
	if err := acl.Add(context.Background(), &store.RepoCollaborator{RepoID: "public-repo", UserID: owner.ID, Role: store.RepoRoleOwner}); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"user_id":"collab-user"}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/collaborators", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous add 401, got %d: %s", w.Code, w.Body.String())
	}

	body = bytes.NewBufferString(`{"user_id":"collab-user"}`)
	req = httptest.NewRequest("POST", "/api/v1/repos/public-repo/collaborators", body)
	req.Header.Set("Content-Type", "application/json")
	req = requestWithSessionUser(req, stranger)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected stranger add 403, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("DELETE", "/api/v1/repos/public-repo/collaborators/collab-user", nil)
	req = requestWithSessionUser(req, stranger)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected stranger delete 403, got %d: %s", w.Code, w.Body.String())
	}
}

func requestWithSessionUser(req *http.Request, user *store.User) *http.Request {
	ctx := authMiddleware.WithUser(req.Context(), user)
	ctx = context.WithValue(ctx, authMiddleware.IdentityKey, user.ID)
	return req.WithContext(ctx)
}
