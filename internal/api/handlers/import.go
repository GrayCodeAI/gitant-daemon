package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// ImportHandler handles repository import/export
type ImportHandler struct {
	repos    *storage.RepositoryRegistry
	issues   *crdt.IssueStore
	prs      *crdt.PullRequestStore
	webhooks *webhooks.Manager
}

// NewImportHandler creates a new import handler
func NewImportHandler(repos *storage.RepositoryRegistry, issues *crdt.IssueStore, prs *crdt.PullRequestStore, webhooks *webhooks.Manager) *ImportHandler {
	return &ImportHandler{
		repos:    repos,
		issues:   issues,
		prs:      prs,
		webhooks: webhooks,
	}
}

// ExportRequest contains export parameters
type ExportRequest struct {
	RepoID        string `json:"repo_id"`
	Format        string `json:"format"`
	IncludeIssues bool   `json:"include_issues"`
	IncludePRs    bool   `json:"include_prs"`
}

// Export exports a repository
func (h *ImportHandler) Export(w http.ResponseWriter, r *http.Request) {
	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RepoID == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}

	repo, err := h.repos.GetEntry(req.RepoID)
	if err != nil {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	export := map[string]interface{}{
		"repository": map[string]interface{}{
			"id":          repo.ID,
			"name":        repo.Name,
			"description": repo.Description,
			"created_at":  repo.CreatedAt,
		},
	}

	if req.IncludeIssues {
		issues := h.issues.List(req.RepoID)
		export["issues"] = issues
	}

	if req.IncludePRs {
		prs := h.prs.List(req.RepoID)
		export["pull_requests"] = prs
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-export.json", repo.Name))
	json.NewEncoder(w).Encode(export)
}

// ImportRequest contains import parameters
type ImportRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
}

// Import imports a repository
func (h *ImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	var importData struct {
		Repository struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"repository"`
		Issues []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Body   string `json:"body"`
			Status string `json:"status"`
			Author string `json:"author"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(req.Data, &importData); err != nil {
		http.Error(w, "invalid import data", http.StatusBadRequest)
		return
	}

	repo, err := h.repos.Create(req.Name, req.Name, req.Description, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, issue := range importData.Issues {
		h.issues.Create(repo.ID, issue.ID, issue.Author, issue.Title, issue.Body)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"repo_id": repo.ID,
		"name":    repo.Name,
	})
}

// GitHubImport imports from GitHub
func (h *ImportHandler) GitHubImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "not_implemented",
		"message": "GitHub import coming soon",
	})
}

// GitLabImport imports from GitLab
func (h *ImportHandler) GitLabImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "not_implemented",
		"message": "GitLab import coming soon",
	})
}
