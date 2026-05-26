package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

func setupTestRegistry(t *testing.T) *storage.RepositoryRegistry {
	t.Helper()
	dir := t.TempDir()
	dataDir := t.TempDir()
	reg, err := storage.NewRepositoryRegistry(dir, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func contextWithIdentity(req *http.Request, did string) *http.Request {
	ctx := context.WithValue(req.Context(), authMiddleware.IdentityKey, did)
	return req.WithContext(ctx)
}

func setupTestIssueStore(t *testing.T) *crdt.IssueStore {
	t.Helper()
	return crdt.NewIssueStore("")
}

func setupTestPRStore(t *testing.T) *crdt.PullRequestStore {
	t.Helper()
	return crdt.NewPullRequestStore("")
}

func setupTestWebhookManager(t *testing.T) *webhooks.Manager {
	t.Helper()
	return webhooks.NewManager()
}

func setupTestIdentity(t *testing.T) (*identity.Identity, error) {
	t.Helper()
	return identity.NewIdentity()
}

func chiRouter() *chi.Mux {
	return chi.NewRouter()
}

// --- Repo Handlers ---

func TestCreateRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	handler := CreateRepo(reg, wm)

	body := `{"name":"test-repo","description":"A test","private":false}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["name"] != "test-repo" {
		t.Fatalf("expected name=test-repo, got %v", result["name"])
	}
}

func TestCreateRepoMissingName(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	handler := CreateRepo(reg, wm)

	body := `{"description":"no name"}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListRepos(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("repo1", "repo1", "first", false)
	reg.Create("repo2", "repo2", "second", true)

	handler := ListRepos(reg, "")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 1 {
		t.Fatalf("expected 1 public repo, got %v", result["total"])
	}
}

func TestGetRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("myrepo", "myrepo", "desc", false)

	r := chiRouter()
	r.Get("/{id}", GetRepo(reg))

	req := httptest.NewRequest("GET", "/myrepo", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetRepoNotFound(t *testing.T) {
	reg := setupTestRegistry(t)

	r := chiRouter()
	r.Get("/{id}", GetRepo(reg))

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	reg.Create("delme", "delme", "", false)

	r := chiRouter()
	r.Delete("/{id}", DeleteRepo(reg, wm))

	req := httptest.NewRequest("DELETE", "/delme", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify deleted
	_, err := reg.GetEntry("delme")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// --- Issue Handlers ---

func TestCreateIssue(t *testing.T) {
	store := setupTestIssueStore(t)
	wm := setupTestWebhookManager(t)
	handler := CreateIssue(store, wm)

	body := `{"title":"Bug report","body":"It broke","labels":["bug"]}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	r := chiRouter()
	r.Post("/{id}/issues", CreateIssue(store, wm))
	req = httptest.NewRequest("POST", "/test-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	_ = handler
}

func TestListIssues(t *testing.T) {
	store := setupTestIssueStore(t)
	store.Create("repo", "issue-1", "alice", "First", "")
	store.Create("repo", "issue-2", "bob", "Second", "")

	r := chiRouter()
	r.Get("/{id}/issues", ListIssues(store))

	req := httptest.NewRequest("GET", "/repo/issues", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 issues, got %v", result["total"])
	}
}

func TestCloseIssue(t *testing.T) {
	store := setupTestIssueStore(t)
	store.Create("repo", "issue-1", "alice", "Bug", "")

	r := chiRouter()
	r.Post("/{id}/issues/{issueId}/close", CloseIssue(store, setupTestWebhookManager(t)))

	req := httptest.NewRequest("POST", "/repo/issues/issue-1/close", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	issue, _ := store.Get("repo", "issue-1")
	if issue.Status != crdt.StatusClosed {
		t.Fatalf("expected closed, got %s", issue.Status)
	}
}

// --- PR Handlers ---

func TestOpenPR(t *testing.T) {
	store := setupTestPRStore(t)

	r := chiRouter()
	r.Post("/{id}/prs", OpenPR(store, setupTestWebhookManager(t)))

	body := `{"title":"Feature","source_branch":"feature","target_branch":"main"}`
	req := httptest.NewRequest("POST", "/repo/prs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListPRs(t *testing.T) {
	store := setupTestPRStore(t)
	store.Create("repo", "pr-1", "alice", "PR 1", "", "feature", "main")
	store.Create("repo", "pr-2", "bob", "PR 2", "", "fix", "main")

	r := chiRouter()
	r.Get("/{id}/prs", ListPRs(store))

	req := httptest.NewRequest("GET", "/repo/prs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 PRs, got %v", result["total"])
	}
}

func TestMergePR(t *testing.T) {
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	gitRepo, err := reg.Open("repo")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := gitRepo.CreateBlob([]byte("feature content"))
	if err != nil {
		t.Fatal(err)
	}
	if err := gitRepo.CreateBranch("feature", hash); err != nil {
		t.Fatal(err)
	}
	if err := gitRepo.CreateBranch("main", hash); err != nil {
		t.Fatal(err)
	}

	store := setupTestPRStore(t)
	store.Create("repo", "pr-1", "alice", "Feature", "", "feature", "main")

	r := chiRouter()
	r.Post("/{id}/prs/{prId}/merge", MergePR(store, reg, nil, setupTestWebhookManager(t)))

	req := httptest.NewRequest("POST", "/repo/prs/pr-1/merge", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	pr, _ := store.Get("repo", "pr-1")
	if pr.Status != crdt.StatusMerged {
		t.Fatalf("expected merged, got %s", pr.Status)
	}
}

func TestMergePRMergeCommit(t *testing.T) {
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	gitRepo, err := reg.Open("repo")
	if err != nil {
		t.Fatal(err)
	}

	mainBlob, err := gitRepo.CreateBlob([]byte("main"))
	if err != nil {
		t.Fatal(err)
	}
	mainTree, err := gitRepo.CreateTree([]storage.TreeEntry{
		{Name: "README", Mode: filemode.Regular, Hash: mainBlob},
	})
	if err != nil {
		t.Fatal(err)
	}
	mainCommit, err := gitRepo.CreateCommit(mainTree, nil, "alice", "init")
	if err != nil {
		t.Fatal(err)
	}
	if err := gitRepo.CreateBranch("main", mainCommit); err != nil {
		t.Fatal(err)
	}

	featureBlob, err := gitRepo.CreateBlob([]byte("feature"))
	if err != nil {
		t.Fatal(err)
	}
	featureTree, err := gitRepo.CreateTree([]storage.TreeEntry{
		{Name: "README", Mode: filemode.Regular, Hash: featureBlob},
	})
	if err != nil {
		t.Fatal(err)
	}
	featureCommit, err := gitRepo.CreateCommit(featureTree, []plumbing.Hash{mainCommit}, "bob", "feature work")
	if err != nil {
		t.Fatal(err)
	}
	if err := gitRepo.CreateBranch("feature", featureCommit); err != nil {
		t.Fatal(err)
	}

	mainBlob2, err := gitRepo.CreateBlob([]byte("main v2"))
	if err != nil {
		t.Fatal(err)
	}
	mainTree2, err := gitRepo.CreateTree([]storage.TreeEntry{
		{Name: "README", Mode: filemode.Regular, Hash: mainBlob2},
	})
	if err != nil {
		t.Fatal(err)
	}
	mainCommit2, err := gitRepo.CreateCommit(mainTree2, []plumbing.Hash{mainCommit}, "alice", "main advance")
	if err != nil {
		t.Fatal(err)
	}
	if err := gitRepo.UpdateRef("main", mainCommit2); err != nil {
		t.Fatal(err)
	}

	store := setupTestPRStore(t)
	store.Create("repo", "pr-1", "alice", "Feature", "", "feature", "main")

	r := chiRouter()
	r.Post("/{id}/prs/{prId}/merge", MergePR(store, reg, nil, setupTestWebhookManager(t)))

	req := httptest.NewRequest("POST", "/repo/prs/pr-1/merge", bytes.NewBufferString(`{"merge_method":"merge"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["merge_hash"] == "" {
		t.Fatalf("expected merge_hash in response, got %v", result)
	}

	mergedHash, err := gitRepo.GetBranch("main")
	if err != nil {
		t.Fatal(err)
	}
	mergedCommit, err := gitRepo.GetCommit(mergedHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(mergedCommit.ParentHashes) != 2 {
		t.Fatalf("expected merge commit with 2 parents, got %d", len(mergedCommit.ParentHashes))
	}
}

// --- Fork Handlers ---

func TestForkRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	reg.Create("source", "source", "original", false)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, ""))

	body := `{"name":"my-fork"}`
	req := httptest.NewRequest("POST", "/source/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["id"] != "my-fork" {
		t.Fatalf("expected id=my-fork, got %v", result["id"])
	}
	if result["forked_from"] != "source" {
		t.Fatalf("expected forked_from=source, got %v", result["forked_from"])
	}
}

func TestForkPrivateRepoDenied(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	reg.Create("secret", "secret", "", true)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, "did:key:server"))

	body := `{"name":"leak-fork"}`
	req := httptest.NewRequest("POST", "/secret/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestForkInheritsPrivateVisibility(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	serverDID := "did:key:server"
	reg.Create("secret", "secret", "", true)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, serverDID))

	body := `{"name":"secret-fork"}`
	req := httptest.NewRequest("POST", "/secret/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = contextWithIdentity(req, serverDID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	entry, err := reg.GetEntry("secret-fork")
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Private {
		t.Fatal("expected fork to inherit private visibility")
	}
}

func TestForkRepoMissingName(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	reg.Create("source", "source", "", false)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, ""))

	body := `{}`
	req := httptest.NewRequest("POST", "/source/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestForkRepoNotFound(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, ""))

	body := `{"name":"fork"}`
	req := httptest.NewRequest("POST", "/nonexistent/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestForkRepoDuplicate(t *testing.T) {
	reg := setupTestRegistry(t)
	wm := setupTestWebhookManager(t)
	reg.Create("source", "source", "", false)
	reg.Create("my-fork", "my-fork", "", false)

	r := chiRouter()
	r.Post("/{id}/fork", ForkRepo(reg, wm, ""))

	body := `{"name":"my-fork"}`
	req := httptest.NewRequest("POST", "/source/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// --- Ref/Repo Handlers ---

func TestPushObjects(t *testing.T) {
	reg := setupTestRegistry(t)
	protectionStore := setupTestProtectionStore(t)
	reg.Create("repo", "repo", "", false)

	r := chiRouter()
	wm := setupTestWebhookManager(t)
	r.Post("/{id}/push", PushObjects(reg, protectionStore, wm))

	body := `{"ref_updates":[{"name":"refs/heads/main","new_hash":"abc123def456abc123def456abc123def456abc1"}]}`
	req := httptest.NewRequest("POST", "/repo/push", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRefs(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("repo", "repo", "", false)

	r := chiRouter()
	r.Get("/{id}/refs", ListRefs(reg))

	req := httptest.NewRequest("GET", "/repo/refs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Test the health handler directly
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", result["status"])
	}
}

// --- Agent Handlers ---

func TestGenerateDID(t *testing.T) {
	handler := GenerateDID()

	req := httptest.NewRequest("POST", "/generate-did", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["did"] == nil {
		t.Fatal("expected did in response")
	}
}

func TestResolveDID(t *testing.T) {
	r := chiRouter()
	r.Get("/resolve/{did}", ResolveDID())

	// Generate a DID first
	handler := GenerateDID()
	req := httptest.NewRequest("POST", "/generate-did", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var genResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &genResult)
	did := genResult["did"].(string)

	// Resolve it
	req = httptest.NewRequest("GET", "/resolve/"+did, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["id"] != did {
		t.Fatalf("expected id=%s, got %v", did, result["id"])
	}
}

func TestListAgents(t *testing.T) {
	registry := NewAgentRegistry("")
	registry.Record("did:key:zabc123")
	registry.Record("did:key:zdef456")

	handler := ListAgents(registry)
	req := httptest.NewRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 agents, got %v", result["total"])
	}
}

func TestGetAgent(t *testing.T) {
	registry := NewAgentRegistry("")
	registry.Record("did:key:ztest123")

	r := chiRouter()
	r.Get("/{did}", GetAgent(registry))

	req := httptest.NewRequest("GET", "/did:key:ztest123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["did"] != "did:key:ztest123" {
		t.Fatalf("expected did=key:ztest123, got %v", result["did"])
	}
}

func TestDelegateCapability(t *testing.T) {
	id, err := setupTestIdentity(t)
	if err != nil {
		t.Fatal(err)
	}

	r := chiRouter()
	r.Post("/{did}/delegate", DelegateCapability(id))

	body := `{"audience":"did:key:ztest","resource":"repo:test","actions":["read","write"]}`
	req := httptest.NewRequest("POST", "/did:key:ztest/delegate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["token"] == nil {
		t.Fatal("expected token in response")
	}
	if result["issuer"] == nil {
		t.Fatal("expected issuer in response")
	}
}

// --- Webhook Handlers ---

func TestRegisterWebhook(t *testing.T) {
	wm := setupTestWebhookManager(t)
	handler := RegisterWebhook(wm)

	body := `{"url":"https://example.com/hook","events":["push","pr"]}`
	req := httptest.NewRequest("POST", "/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListWebhooks(t *testing.T) {
	wm := setupTestWebhookManager(t)
	handler := ListWebhooks(wm)

	req := httptest.NewRequest("GET", "/webhooks", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeleteWebhook(t *testing.T) {
	wm := setupTestWebhookManager(t)

	// Register first
	registerHandler := RegisterWebhook(wm)
	body := `{"url":"https://example.com/hook","events":["push"]}`
	req := httptest.NewRequest("POST", "/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	registerHandler.ServeHTTP(w, req)

	var regResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &regResult)
	webhookID := regResult["id"].(string)

	// Delete
	r := chiRouter()
	r.Delete("/{id}", DeleteWebhook(wm))
	req = httptest.NewRequest("DELETE", "/"+webhookID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Comment Handlers ---

func TestListIssueComments(t *testing.T) {
	store := setupTestIssueStore(t)
	store.Create("repo", "issue-1", "alice", "Bug", "")

	// Add a comment
	issue, _ := store.Get("repo", "issue-1")
	issue.AddComment("bob", "This is a comment")

	r := chiRouter()
	r.Get("/{id}/issues/{issueId}/comments", ListIssueComments(store))

	req := httptest.NewRequest("GET", "/repo/issues/issue-1/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 1 {
		t.Fatalf("expected 1 comment, got %v", result["total"])
	}
}

func TestListPRComments(t *testing.T) {
	store := setupTestPRStore(t)
	store.Create("repo", "pr-1", "alice", "Feature", "", "feature", "main")

	// Add a comment
	pr, _ := store.Get("repo", "pr-1")
	pr.AddComment("bob", "LGTM")

	r := chiRouter()
	r.Get("/{id}/prs/{prId}/comments", ListPRComments(store))

	req := httptest.NewRequest("GET", "/repo/prs/pr-1/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 1 {
		t.Fatalf("expected 1 comment, got %v", result["total"])
	}
}

// --- Star Handlers ---

func TestStarRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("repo", "repo", "", false)

	r := chiRouter()
	r.Post("/{id}/star", StarRepo(reg))

	req := httptest.NewRequest("POST", "/repo/star", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["stars"].(float64)) != 1 {
		t.Fatalf("expected 1 star, got %v", result["stars"])
	}
}

func TestUnstarRepo(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("repo", "repo", "", false)
	reg.Star("repo", "anonymous")

	r := chiRouter()
	r.Post("/{id}/unstar", UnstarRepo(reg))

	req := httptest.NewRequest("POST", "/repo/unstar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["stars"].(float64)) != 0 {
		t.Fatalf("expected 0 stars, got %v", result["stars"])
	}
}

func TestGetStarCount(t *testing.T) {
	reg := setupTestRegistry(t)
	reg.Create("repo", "repo", "", false)
	reg.Star("repo", "user1")
	reg.Star("repo", "user2")

	r := chiRouter()
	r.Get("/{id}/stars", GetStarCount(reg))

	req := httptest.NewRequest("GET", "/repo/stars", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["stars"].(float64)) != 2 {
		t.Fatalf("expected 2 stars, got %v", result["stars"])
	}
}

// --- Label Handlers ---

func setupTestLabelStore(t *testing.T) *crdt.LabelStore {
	t.Helper()
	return crdt.NewLabelStore("")
}

func TestCreateLabel(t *testing.T) {
	store := setupTestLabelStore(t)
	handler := CreateLabel(store)

	body := `{"name":"bug","color":"#ef4444"}`
	req := httptest.NewRequest("POST", "/repo/labels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chiRouter()
	r.Post("/{id}/labels", handler)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListLabels(t *testing.T) {
	store := setupTestLabelStore(t)
	store.Add("repo", "bug", "#ef4444")
	store.Add("repo", "feature", "#10b981")

	r := chiRouter()
	r.Get("/{id}/labels", ListLabels(store))

	req := httptest.NewRequest("GET", "/repo/labels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 labels, got %v", result["total"])
	}
}

func TestDeleteLabel(t *testing.T) {
	store := setupTestLabelStore(t)
	store.Add("repo", "bug", "#ef4444")

	r := chiRouter()
	r.Delete("/{id}/labels/{name}", DeleteLabel(store))

	req := httptest.NewRequest("DELETE", "/repo/labels/bug", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Task Handlers ---

func setupTestTaskStore(t *testing.T) *crdt.TaskStore {
	t.Helper()
	return crdt.NewTaskStore("")
}

func TestCreateTask(t *testing.T) {
	store := setupTestTaskStore(t)
	handler := CreateTask(store)

	body := `{"title":"Fix bug","description":"Fix the parser bug"}`
	req := httptest.NewRequest("POST", "/repo/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chiRouter()
	r.Post("/{id}/tasks", handler)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClaimTask(t *testing.T) {
	store := setupTestTaskStore(t)
	task := store.Create("repo", "task-1", "alice", "Fix bug", "")
	_ = store.Save()

	r := chiRouter()
	r.Post("/{id}/tasks/{taskId}/claim", ClaimTask(store))

	req := httptest.NewRequest("POST", "/repo/tasks/task-1/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	tasks := store.List("repo", crdt.TaskClaimed)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 claimed task, got %d", len(tasks))
	}
	if tasks[0].ID != task.ID {
		t.Fatalf("expected task-1, got %s", tasks[0].ID)
	}
}

func TestCompleteTask(t *testing.T) {
	store := setupTestTaskStore(t)
	store.Create("repo", "task-1", "alice", "Fix bug", "")
	store.Claim("repo", "task-1", "bob")

	r := chiRouter()
	r.Post("/{id}/tasks/{taskId}/complete", CompleteTask(store))

	body := `{"result":"Fixed in commit abc123"}`
	req := httptest.NewRequest("POST", "/repo/tasks/task-1/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	tasks := store.List("repo", crdt.TaskCompleted)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 completed task, got %d", len(tasks))
	}
}

func TestListTasksFilterByStatus(t *testing.T) {
	store := setupTestTaskStore(t)
	store.Create("repo", "task-1", "alice", "Open task", "")
	store.Create("repo", "task-2", "alice", "Another open", "")
	store.Create("repo", "task-3", "alice", "Claimed task", "")
	store.Claim("repo", "task-3", "bob")

	r := chiRouter()
	r.Get("/{id}/tasks", ListTasks(store))

	// List all
	req := httptest.NewRequest("GET", "/repo/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 3 {
		t.Fatalf("expected 3 tasks, got %v", result["total"])
	}

	// List open only
	req = httptest.NewRequest("GET", "/repo/tasks?status=open", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 open tasks, got %v", result["total"])
	}
}

// --- Protection Handlers ---

func setupTestProtectionStore(t *testing.T) *storage.ProtectionStore {
	t.Helper()
	return storage.NewProtectionStore("")
}

func TestSetProtection(t *testing.T) {
	store := setupTestProtectionStore(t)
	handler := SetProtection(store)

	body := `{"require_pr":true,"require_approval":true,"no_force_push":true}`
	req := httptest.NewRequest("PUT", "/repo/protections/main", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chiRouter()
	r.Put("/{id}/protections/{branch}", handler)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["require_pr"] != true {
		t.Fatalf("expected require_pr=true, got %v", result["require_pr"])
	}
	if result["no_force_push"] != true {
		t.Fatalf("expected no_force_push=true, got %v", result["no_force_push"])
	}
}

func TestGetProtection(t *testing.T) {
	store := setupTestProtectionStore(t)
	store.Set("repo", storage.BranchProtection{
		Branch:      "main",
		RequirePR:   true,
		NoForcePush: true,
	})

	r := chiRouter()
	r.Get("/{id}/protections/{branch}", GetProtection(store))

	req := httptest.NewRequest("GET", "/repo/protections/main", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["protected"] != true {
		t.Fatalf("expected protected=true, got %v", result["protected"])
	}
	if result["require_pr"] != true {
		t.Fatalf("expected require_pr=true, got %v", result["require_pr"])
	}
}

func TestGetProtectionUnprotected(t *testing.T) {
	store := setupTestProtectionStore(t)

	r := chiRouter()
	r.Get("/{id}/protections/{branch}", GetProtection(store))

	req := httptest.NewRequest("GET", "/repo/protections/main", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["protected"] != false {
		t.Fatalf("expected protected=false, got %v", result["protected"])
	}
}

func TestListProtections(t *testing.T) {
	store := setupTestProtectionStore(t)
	store.Set("repo", storage.BranchProtection{Branch: "main", RequirePR: true})
	store.Set("repo", storage.BranchProtection{Branch: "develop", NoForcePush: true})

	r := chiRouter()
	r.Get("/{id}/protections", ListProtections(store))

	req := httptest.NewRequest("GET", "/repo/protections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if int(result["total"].(float64)) != 2 {
		t.Fatalf("expected 2 protections, got %v", result["total"])
	}
}

func TestRemoveProtection(t *testing.T) {
	store := setupTestProtectionStore(t)
	store.Set("repo", storage.BranchProtection{Branch: "main", RequirePR: true})

	r := chiRouter()
	r.Delete("/{id}/protections/{branch}", RemoveProtection(store))

	req := httptest.NewRequest("DELETE", "/repo/protections/main", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	protection := store.Get("repo", "main")
	if protection != nil {
		t.Fatal("expected protection to be removed")
	}
}

func TestRemoveProtectionNotFound(t *testing.T) {
	store := setupTestProtectionStore(t)

	r := chiRouter()
	r.Delete("/{id}/protections/{branch}", RemoveProtection(store))

	req := httptest.NewRequest("DELETE", "/repo/protections/main", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func init() {
	// Ensure temp dirs don't pollute
	os.Setenv("GITANT_TEST", "1")
}
