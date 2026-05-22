package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

func setupIntegrationRouter(t *testing.T) (*chi.Mux, *storage.RepositoryRegistry) {
	t.Helper()
	reg := setupTestRegistry(t)
	issueStore := setupTestIssueStore(t)
	prStore := setupTestPRStore(t)
	labelStore := setupTestLabelStore(t)
	taskStore := setupTestTaskStore(t)
	protectionStore := setupTestProtectionStore(t)
	blockstore := storage.NewBlockstore("", t.TempDir()+"/blocks")
	wm := setupTestWebhookManager(t)
	id, _ := setupTestIdentity(t)

	r := chi.NewRouter()
	r.Post("/repos", CreateRepo(reg, wm))
	r.Get("/repos", ListRepos(reg))
	r.Get("/repos/{id}", GetRepo(reg))
	r.Delete("/repos/{id}", DeleteRepo(reg, wm))
	r.Post("/repos/{id}/fork", ForkRepo(reg, wm))
	r.Post("/repos/{id}/issues", CreateIssue(issueStore, wm))
	r.Get("/repos/{id}/issues", ListIssues(issueStore))
	r.Post("/repos/{id}/prs", OpenPR(prStore, wm))
	r.Get("/repos/{id}/prs", ListPRs(prStore))
	r.Post("/repos/{id}/push", PushObjects(reg, protectionStore, wm))
	r.Post("/repos/{id}/push-packfile", PushPackfile(reg, protectionStore, wm))
	r.Get("/repos/{id}/files", ListFiles(reg))
	r.Get("/repos/{id}/search", SearchCode(reg))
	r.Post("/repos/{id}/labels", CreateLabel(labelStore))
	r.Get("/repos/{id}/labels", ListLabels(labelStore))
	r.Post("/repos/{id}/tasks", CreateTask(taskStore))
	r.Get("/repos/{id}/tasks", ListTasks(taskStore))
	r.Post("/repos/{id}/protections/{branch}", SetProtection(protectionStore))
	r.Get("/repos/{id}/protections", ListProtections(protectionStore))
	r.Get("/repos/{id}/stars", GetStarCount(reg))
	r.Post("/repos/{id}/star", StarRepo(reg))

	// Silence unused variable warnings
	_ = blockstore
	_ = id

	return r, reg
}

func TestIntegration_CreateAndListRepos(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create a repo
	body := `{"name":"test-repo","description":"Integration test"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create repo: expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	if created["name"] != "test-repo" {
		t.Fatalf("expected name 'test-repo', got %v", created["name"])
	}

	// List repos
	req = httptest.NewRequest("GET", "/repos", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list repos: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&listResult)
	repos := listResult["repos"].([]interface{})
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
}

func TestIntegration_CreateAndListIssues(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create a repo first
	body := `{"name":"issue-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Create an issue
	body = `{"title":"Test issue","body":"Description","labels":["bug"]}`
	req = httptest.NewRequest("POST", "/repos/issue-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create issue: expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	// List issues
	req = httptest.NewRequest("GET", "/repos/issue-repo/issues", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list issues: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&listResult)
	issues := listResult["issues"].([]interface{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
}

func TestIntegration_ForkRepo(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create source repo
	body := `{"name":"source-repo","description":"Original"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Fork it
	body = `{"name":"fork-repo"}`
	req = httptest.NewRequest("POST", "/repos/source-repo/fork", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("fork repo: expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	var forked map[string]interface{}
	json.NewDecoder(w.Body).Decode(&forked)
	if forked["name"] != "fork-repo" {
		t.Fatalf("expected fork name 'fork-repo', got %v", forked["name"])
	}
	if forked["forked_from"] != "source-repo" {
		t.Fatalf("expected forked_from 'source-repo', got %v", forked["forked_from"])
	}
}

func TestIntegration_Pagination(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create 5 repos
	for i := 0; i < 5; i++ {
		body := `{"name":"repo-` + string(rune('a'+i)) + `"}`
		req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// List with limit=2
	req := httptest.NewRequest("GET", "/repos?limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	repos := result["repos"].([]interface{})
	total := int(result["total"].(float64))

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos in page, got %d", len(repos))
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}

	// List page 2
	req = httptest.NewRequest("GET", "/repos?offset=2&limit=2", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&result)
	repos = result["repos"].([]interface{})
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos in page 2, got %d", len(repos))
	}
}

func TestIntegration_LabelsAndTasks(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create repo
	body := `{"name":"label-task-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Create label
	body = `{"name":"bug","color":"#ff0000"}`
	req = httptest.NewRequest("POST", "/repos/label-task-repo/labels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create label: expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	// List labels
	req = httptest.NewRequest("GET", "/repos/label-task-repo/labels", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var labelResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&labelResult)
	labels := labelResult["labels"].([]interface{})
	if len(labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(labels))
	}

	// Create task
	body = `{"title":"Fix the bug","description":"Details here"}`
	req = httptest.NewRequest("POST", "/repos/label-task-repo/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create task: expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	// List tasks
	req = httptest.NewRequest("GET", "/repos/label-task-repo/tasks", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var taskResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResult)
	tasks := taskResult["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestIntegration_BranchProtection(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create repo
	body := `{"name":"protected-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Set protection
	body = `{"require_pr":true,"no_force_push":true}`
	req = httptest.NewRequest("POST", "/repos/protected-repo/protections/main", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("set protection: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List protections
	req = httptest.NewRequest("GET", "/repos/protected-repo/protections", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var protResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&protResult)
	protections := protResult["protections"].([]interface{})
	if len(protections) != 1 {
		t.Fatalf("expected 1 protection, got %d", len(protections))
	}
}

func TestIntegration_Stars(t *testing.T) {
	r, _ := setupIntegrationRouter(t)

	// Create repo
	body := `{"name":"star-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Star it
	req = httptest.NewRequest("POST", "/repos/star-repo/star", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("star repo: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get star count
	req = httptest.NewRequest("GET", "/repos/star-repo/stars", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var starResult map[string]interface{}
	json.NewDecoder(w.Body).Decode(&starResult)
	stars := int(starResult["stars"].(float64))
	if stars != 1 {
		t.Fatalf("expected 1 star, got %d", stars)
	}
}
