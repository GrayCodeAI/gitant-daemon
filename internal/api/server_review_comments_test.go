package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/store"
	sqlitepkg "github.com/lakshmanpatel/gitant/internal/store/sqlite"
)

func TestServerReviewCommentRoutesLifecycle(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	_, server := setupSmartHTTPRouteServerWithRegistry(t)

	sqliteStore, err := sqlitepkg.NewStore(filepath.Join(dataDir, "data"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer sqliteStore.Close()

	user := &store.User{
		ID:           "user-1",
		Username:     "reviewer",
		Email:        "reviewer@example.test",
		PasswordHash: "unused",
		Role:         "developer",
	}
	if err := sqliteStore.NewUserStore().Create(ctx, user); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	if err := sqliteStore.NewRepoCollaboratorStore().Add(ctx, &store.RepoCollaborator{RepoID: "public-repo", UserID: user.ID, Role: store.RepoRoleOwner}); err != nil {
		t.Fatalf("Add owner: %v", err)
	}
	server.SetRepoCollaboratorStore(sqliteStore.NewRepoCollaboratorStore())
	server.SetReviewStore(sqliteStore.NewReviewCommentStore())

	createBody := bytes.NewBufferString(`{"file_path":"main.go","line_number":7,"body":"nit: rename this"}`)
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/prs/pr-1/review", createBody)
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create review comment: got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created response: %v", err)
	}
	commentID, _ := created["id"].(string)
	if commentID == "" {
		t.Fatalf("create response missing comment id: %s", w.Body.String())
	}

	comments := listReviewComments(t, server, "/api/v1/repos/public-repo/prs/pr-1/review")
	if len(comments) != 1 || comments[0]["id"] != commentID {
		t.Fatalf("expected created comment in list, got %#v", comments)
	}
	if comments[0]["status"] != "open" {
		t.Fatalf("expected open status, got %#v", comments[0]["status"])
	}

	req = httptest.NewRequest("POST", "/api/v1/review-comments/"+commentID+"/resolve", nil)
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resolve review comment: got %d: %s", w.Code, w.Body.String())
	}
	comments = listReviewComments(t, server, "/api/v1/repos/public-repo/prs/pr-1/review")
	if len(comments) != 1 || comments[0]["status"] != "resolved" || comments[0]["resolved"] != true {
		t.Fatalf("expected resolved comment in list, got %#v", comments)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/review-comments/"+commentID, nil)
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete review comment: got %d: %s", w.Code, w.Body.String())
	}
	comments = listReviewComments(t, server, "/api/v1/repos/public-repo/prs/pr-1/review")
	if len(comments) != 0 {
		t.Fatalf("expected comment absent after delete, got %#v", comments)
	}
}

func listReviewComments(t *testing.T, server *Server, path string) []map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list review comments: got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Comments []map[string]interface{} `json:"comments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	return body.Comments
}
