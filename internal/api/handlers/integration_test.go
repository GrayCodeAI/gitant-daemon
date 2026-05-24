package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// setupWorkflowRouter creates a full chi mux with all API routes needed for integration testing.
func setupWorkflowRouter(t *testing.T) (*chi.Mux, *storage.RepositoryRegistry, *crdt.IssueStore, *crdt.PullRequestStore, *crdt.TaskStore, *crdt.LabelStore, *storage.ProtectionStore) {
	t.Helper()
	reg := setupTestRegistry(t)
	issueStore := setupTestIssueStore(t)
	prStore := setupTestPRStore(t)
	labelStore := setupTestLabelStore(t)
	taskStore := setupTestTaskStore(t)
	protectionStore := setupTestProtectionStore(t)
	wm := setupTestWebhookManager(t)

	r := chi.NewRouter()

	// Repo CRUD
	r.Post("/repos", CreateRepo(reg, wm))
	r.Get("/repos", ListRepos(reg))
	r.Get("/repos/{id}", GetRepo(reg))
	r.Delete("/repos/{id}", DeleteRepo(reg, wm))

	// Issues
	r.Post("/repos/{id}/issues", CreateIssue(issueStore, wm))
	r.Get("/repos/{id}/issues", ListIssues(issueStore))
	r.Get("/repos/{id}/issues/{issueId}", GetIssue(issueStore))
	r.Post("/repos/{id}/issues/{issueId}/close", CloseIssue(issueStore, wm))

	// Pull requests
	r.Post("/repos/{id}/prs", OpenPR(prStore, wm))
	r.Get("/repos/{id}/prs", ListPRs(prStore))
	r.Get("/repos/{id}/prs/{prId}", GetPR(prStore))
	r.Post("/repos/{id}/prs/{prId}/review", ReviewPR(prStore, wm))
	r.Post("/repos/{id}/prs/{prId}/merge", MergePR(prStore, protectionStore, wm))

	// Labels
	r.Post("/repos/{id}/labels", CreateLabel(labelStore))
	r.Get("/repos/{id}/labels", ListLabels(labelStore))
	r.Delete("/repos/{id}/labels/{name}", DeleteLabel(labelStore))

	// Tasks
	r.Post("/repos/{id}/tasks", CreateTask(taskStore))
	r.Get("/repos/{id}/tasks", ListTasks(taskStore))
	r.Post("/repos/{id}/tasks/{taskId}/claim", ClaimTask(taskStore))
	r.Post("/repos/{id}/tasks/{taskId}/complete", CompleteTask(taskStore))

	// Branch protections
	r.Post("/repos/{id}/protections/{branch}", SetProtection(protectionStore))
	r.Get("/repos/{id}/protections", ListProtections(protectionStore))
	r.Get("/repos/{id}/protections/{branch}", GetProtection(protectionStore))
	r.Delete("/repos/{id}/protections/{branch}", RemoveProtection(protectionStore))

	return r, reg, issueStore, prStore, taskStore, labelStore, protectionStore
}

func TestIntegrationWorkflow_RepoLifecycle(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"lifecycle-repo","description":"End-to-end lifecycle test"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created["name"] != "lifecycle-repo" {
		t.Fatalf("expected name 'lifecycle-repo', got %v", created["name"])
	}
	if created["description"] != "End-to-end lifecycle test" {
		t.Fatalf("expected description 'End-to-end lifecycle test', got %v", created["description"])
	}

	// Step 2: Verify it appears in list
	req = httptest.NewRequest("GET", "/repos", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list repos: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	repos := listResult["repos"].([]interface{})
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo in list, got %d", len(repos))
	}
	total := int(listResult["total"].(float64))
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}

	// Step 3: Get repo details
	req = httptest.NewRequest("GET", "/repos/lifecycle-repo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get repo: expected 200, got %d", w.Code)
	}

	var details map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &details)
	if details["name"] != "lifecycle-repo" {
		t.Fatalf("expected name 'lifecycle-repo', got %v", details["name"])
	}
	if details["private"] != false {
		t.Fatalf("expected private=false, got %v", details["private"])
	}

	// Step 4: Delete the repo
	req = httptest.NewRequest("DELETE", "/repos/lifecycle-repo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("delete repo: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var deleteResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &deleteResult)
	if deleteResult["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", deleteResult["deleted"])
	}

	// Step 5: Verify it's gone from list
	req = httptest.NewRequest("GET", "/repos", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &listResult)
	total = int(listResult["total"].(float64))
	if total != 0 {
		t.Fatalf("expected 0 repos after delete, got %d", total)
	}

	// Step 6: Verify get returns 404
	req = httptest.NewRequest("GET", "/repos/lifecycle-repo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("get deleted repo: expected 404, got %d", w.Code)
	}
}

func TestIntegrationWorkflow_IssueWorkflow(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"issue-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d", w.Code)
	}

	// Step 2: Create an issue with title and labels
	body = `{"title":"Login broken","body":"Cannot log in on Safari","labels":["bug","critical"]}`
	req = httptest.NewRequest("POST", "/repos/issue-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create issue: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdIssue map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createdIssue)
	issueID := createdIssue["id"].(string)
	if issueID == "" {
		t.Fatal("expected non-empty issue id")
	}
	if createdIssue["title"] != "Login broken" {
		t.Fatalf("expected title 'Login broken', got %v", createdIssue["title"])
	}

	// Verify labels were attached
	labels := createdIssue["labels"].([]interface{})
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}

	// Step 3: Get the issue
	req = httptest.NewRequest("GET", "/repos/issue-repo/issues/"+issueID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get issue: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var fetchedIssue map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &fetchedIssue)
	if fetchedIssue["title"] != "Login broken" {
		t.Fatalf("expected title 'Login broken', got %v", fetchedIssue["title"])
	}
	if fetchedIssue["status"] != string(crdt.StatusOpen) {
		t.Fatalf("expected status 'open', got %v", fetchedIssue["status"])
	}

	// Step 4: Create a second issue
	body = `{"title":"Dark mode request","labels":["enhancement"]}`
	req = httptest.NewRequest("POST", "/repos/issue-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create second issue: expected 201, got %d", w.Code)
	}

	// List issues -- should have 2 open issues
	req = httptest.NewRequest("GET", "/repos/issue-repo/issues", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list issues: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	issues := listResult["issues"].([]interface{})
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	// Both should be open
	for _, iss := range issues {
		m := iss.(map[string]interface{})
		if m["status"] != string(crdt.StatusOpen) {
			t.Fatalf("expected all issues open, got %v", m["status"])
		}
	}

	// Step 5: Close the first issue
	req = httptest.NewRequest("POST", "/repos/issue-repo/issues/"+issueID+"/close", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("close issue: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var closeResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &closeResult)
	if closeResult["status"] != string(crdt.StatusClosed) {
		t.Fatalf("expected status 'closed', got %v", closeResult["status"])
	}

	// Step 6: Verify status changed
	req = httptest.NewRequest("GET", "/repos/issue-repo/issues/"+issueID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &fetchedIssue)
	if fetchedIssue["status"] != string(crdt.StatusClosed) {
		t.Fatalf("expected status 'closed' after close, got %v", fetchedIssue["status"])
	}
}

func TestIntegrationWorkflow_PRWorkflow(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"pr-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d", w.Code)
	}

	// Step 2: Open a PR (branch creation not required for the CRDT PR store)
	body = `{"title":"Add dark mode","body":"Implements dark theme","source_branch":"feature/dark-mode","target_branch":"main"}`
	req = httptest.NewRequest("POST", "/repos/pr-repo/prs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("open PR: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdPR map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createdPR)
	prID := createdPR["id"].(string)
	if prID == "" {
		t.Fatal("expected non-empty PR id")
	}
	if createdPR["title"] != "Add dark mode" {
		t.Fatalf("expected title 'Add dark mode', got %v", createdPR["title"])
	}
	if createdPR["source_branch"] != "feature/dark-mode" {
		t.Fatalf("expected source_branch 'feature/dark-mode', got %v", createdPR["source_branch"])
	}
	if createdPR["target_branch"] != "main" {
		t.Fatalf("expected target_branch 'main', got %v", createdPR["target_branch"])
	}
	if createdPR["status"] != string(crdt.StatusOpen) {
		t.Fatalf("expected status 'open', got %v", createdPR["status"])
	}

	// Step 3: Get PR details
	req = httptest.NewRequest("GET", "/repos/pr-repo/prs/"+prID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get PR: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var fetchedPR map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &fetchedPR)
	if fetchedPR["title"] != "Add dark mode" {
		t.Fatalf("expected title 'Add dark mode', got %v", fetchedPR["title"])
	}
	if fetchedPR["status"] != string(crdt.StatusOpen) {
		t.Fatalf("expected status 'open', got %v", fetchedPR["status"])
	}

	// Step 4: Open a second PR and list all PRs
	body = `{"title":"Fix header","source_branch":"fix/header","target_branch":"main"}`
	req = httptest.NewRequest("POST", "/repos/pr-repo/prs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("open second PR: expected 201, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/repos/pr-repo/prs", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list PRs: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	prs := listResult["prs"].([]interface{})
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}

	// Step 5: Review the PR (approve)
	body = `{"verdict":"approve","body":"LGTM, nice work!"}`
	req = httptest.NewRequest("POST", "/repos/pr-repo/prs/"+prID+"/review", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("review PR: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var reviewResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &reviewResult)
	if reviewResult["verdict"] != "approve" {
		t.Fatalf("expected verdict 'approve', got %v", reviewResult["verdict"])
	}

	// Step 6: Merge the PR
	req = httptest.NewRequest("POST", "/repos/pr-repo/prs/"+prID+"/merge", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("merge PR: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var mergeResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &mergeResult)
	if mergeResult["status"] != string(crdt.StatusMerged) {
		t.Fatalf("expected status 'merged', got %v", mergeResult["status"])
	}

	// Verify PR is now merged
	req = httptest.NewRequest("GET", "/repos/pr-repo/prs/"+prID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &fetchedPR)
	if fetchedPR["status"] != string(crdt.StatusMerged) {
		t.Fatalf("expected status 'merged' after merge, got %v", fetchedPR["status"])
	}
}

func TestIntegrationWorkflow_TaskWorkflow(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"task-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d", w.Code)
	}

	// Step 2: Create a task
	body = `{"title":"Implement caching","description":"Add Redis caching layer"}`
	req = httptest.NewRequest("POST", "/repos/task-repo/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdTask map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createdTask)
	taskID := createdTask["id"].(string)
	if taskID == "" {
		t.Fatal("expected non-empty task id")
	}
	if createdTask["title"] != "Implement caching" {
		t.Fatalf("expected title 'Implement caching', got %v", createdTask["title"])
	}
	if createdTask["status"] != string(crdt.TaskOpen) {
		t.Fatalf("expected status 'open', got %v", createdTask["status"])
	}

	// Step 3: List tasks
	req = httptest.NewRequest("GET", "/repos/task-repo/tasks", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list tasks: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	tasks := listResult["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	// Verify filtering by status works
	req = httptest.NewRequest("GET", "/repos/task-repo/tasks?status=open", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	if int(listResult["total"].(float64)) != 1 {
		t.Fatalf("expected 1 open task, got %v", listResult["total"])
	}

	req = httptest.NewRequest("GET", "/repos/task-repo/tasks?status=claimed", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	if int(listResult["total"].(float64)) != 0 {
		t.Fatalf("expected 0 claimed tasks, got %v", listResult["total"])
	}

	// Step 4: Claim the task
	req = httptest.NewRequest("POST", "/repos/task-repo/tasks/"+taskID+"/claim", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("claim task: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var claimResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &claimResult)
	if claimResult["status"] != string(crdt.TaskClaimed) {
		t.Fatalf("expected status 'claimed', got %v", claimResult["status"])
	}

	// Verify filtering: 0 open, 1 claimed
	req = httptest.NewRequest("GET", "/repos/task-repo/tasks?status=open", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	if int(listResult["total"].(float64)) != 0 {
		t.Fatalf("expected 0 open tasks after claim, got %v", listResult["total"])
	}

	req = httptest.NewRequest("GET", "/repos/task-repo/tasks?status=claimed", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	if int(listResult["total"].(float64)) != 1 {
		t.Fatalf("expected 1 claimed task, got %v", listResult["total"])
	}

	// Step 5: Complete the task
	body = `{"result":"Implemented with Redis Cluster"}`
	req = httptest.NewRequest("POST", "/repos/task-repo/tasks/"+taskID+"/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("complete task: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var completeResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &completeResult)
	if completeResult["status"] != string(crdt.TaskCompleted) {
		t.Fatalf("expected status 'completed', got %v", completeResult["status"])
	}

	// Verify filtering: 1 completed
	req = httptest.NewRequest("GET", "/repos/task-repo/tasks?status=completed", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	if int(listResult["total"].(float64)) != 1 {
		t.Fatalf("expected 1 completed task, got %v", listResult["total"])
	}
}

func TestIntegrationWorkflow_LabelWorkflow(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"label-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d", w.Code)
	}

	// Step 2: Create labels
	labels := []struct {
		name  string
		color string
	}{
		{"bug", "#ef4444"},
		{"enhancement", "#10b981"},
		{"documentation", "#3b82f6"},
	}

	for _, lb := range labels {
		labelBody := `{"name":"` + lb.name + `","color":"` + lb.color + `"}`
		req = httptest.NewRequest("POST", "/repos/label-repo/labels", bytes.NewBufferString(labelBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("create label %s: expected 201, got %d: %s", lb.name, w.Code, w.Body.String())
		}

		var result map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		if result["name"] != lb.name {
			t.Fatalf("expected label name '%s', got %v", lb.name, result["name"])
		}
		if result["color"] != lb.color {
			t.Fatalf("expected label color '%s', got %v", lb.color, result["color"])
		}
	}

	// Step 3: List labels
	req = httptest.NewRequest("GET", "/repos/label-repo/labels", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list labels: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	labelList := listResult["labels"].([]interface{})
	if len(labelList) != 3 {
		t.Fatalf("expected 3 labels, got %d", len(labelList))
	}
	total := int(listResult["total"].(float64))
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}

	// Step 4: Delete a label
	req = httptest.NewRequest("DELETE", "/repos/label-repo/labels/bug", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("delete label: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var deleteResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &deleteResult)
	if deleteResult["success"] != true {
		t.Fatalf("expected success=true, got %v", deleteResult["success"])
	}

	// Step 5: Verify label is gone
	req = httptest.NewRequest("GET", "/repos/label-repo/labels", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &listResult)
	labelList = listResult["labels"].([]interface{})
	if len(labelList) != 2 {
		t.Fatalf("expected 2 labels after delete, got %d", len(labelList))
	}

	// Verify the deleted label is not in the list
	for _, lb := range labelList {
		m := lb.(map[string]interface{})
		if m["name"] == "bug" {
			t.Fatal("expected 'bug' label to be deleted, but it still exists")
		}
	}
}

func TestIntegrationWorkflow_BranchProtection(t *testing.T) {
	r, _, _, _, _, _, _ := setupWorkflowRouter(t)

	// Step 1: Create a repo
	body := `{"name":"protected-repo"}`
	req := httptest.NewRequest("POST", "/repos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create repo: expected 201, got %d", w.Code)
	}

	// Step 2: Set branch protection (no force push + require PR)
	body = `{"require_pr":true,"require_approval":true,"no_force_push":true}`
	req = httptest.NewRequest("POST", "/repos/protected-repo/protections/main", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("set protection: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var setResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &setResult)
	if setResult["require_pr"] != true {
		t.Fatalf("expected require_pr=true, got %v", setResult["require_pr"])
	}
	if setResult["no_force_push"] != true {
		t.Fatalf("expected no_force_push=true, got %v", setResult["no_force_push"])
	}

	// Step 3: Get protection rules
	req = httptest.NewRequest("GET", "/repos/protected-repo/protections/main", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get protection: expected 200, got %d", w.Code)
	}

	var getResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResult)
	if getResult["protected"] != true {
		t.Fatalf("expected protected=true, got %v", getResult["protected"])
	}
	if getResult["require_pr"] != true {
		t.Fatalf("expected require_pr=true, got %v", getResult["require_pr"])
	}
	if getResult["require_approval"] != true {
		t.Fatalf("expected require_approval=true, got %v", getResult["require_approval"])
	}
	if getResult["no_force_push"] != true {
		t.Fatalf("expected no_force_push=true, got %v", getResult["no_force_push"])
	}

	// Step 4: Add a second protection and list all
	body = `{"no_force_push":true}`
	req = httptest.NewRequest("POST", "/repos/protected-repo/protections/develop", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("set second protection: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/repos/protected-repo/protections", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list protections: expected 200, got %d", w.Code)
	}

	var listResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResult)
	protections := listResult["protections"].([]interface{})
	if len(protections) != 2 {
		t.Fatalf("expected 2 protections, got %d", len(protections))
	}

	// Step 5: Remove protection on main
	req = httptest.NewRequest("DELETE", "/repos/protected-repo/protections/main", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("remove protection: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var removeResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &removeResult)
	if removeResult["success"] != true {
		t.Fatalf("expected success=true, got %v", removeResult["success"])
	}

	// Step 6: Verify main is now unprotected
	req = httptest.NewRequest("GET", "/repos/protected-repo/protections/main", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &getResult)
	if getResult["protected"] != false {
		t.Fatalf("expected protected=false after removal, got %v", getResult["protected"])
	}

	// Only develop protection should remain
	req = httptest.NewRequest("GET", "/repos/protected-repo/protections", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResult)
	protections = listResult["protections"].([]interface{})
	if len(protections) != 1 {
		t.Fatalf("expected 1 protection after removal, got %d", len(protections))
	}
	remaining := protections[0].(map[string]interface{})
	if remaining["branch"] != "develop" {
		t.Fatalf("expected remaining protection on 'develop', got %v", remaining["branch"])
	}
}

// Silence unused import warnings.
var _ = webhooks.Manager{}
