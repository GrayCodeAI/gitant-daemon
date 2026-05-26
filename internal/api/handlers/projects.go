package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/projects"
)

// ProjectHandler handles project endpoints
type ProjectHandler struct {
	store *projects.Store
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(store *projects.Store) *ProjectHandler {
	return &ProjectHandler{store: store}
}

// ListProjects lists projects for a repository
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")

	projects := h.store.List(repoID, status)

	result := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		result[i] = map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"status":      p.Status,
			"columns":     len(p.Columns),
			"created_at":  p.CreatedAt,
			"updated_at":  p.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"projects": result,
	})
}

// GetProject gets a project by ID
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	projectID := chi.URLParam(r, "projectId")

	project, err := h.store.Get(repoID, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	project := &projects.Project{
		RepoID:      repoID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.store.Create(project); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

// AddCard adds a card to a project
func (h *ProjectHandler) AddCard(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	projectID := chi.URLParam(r, "projectId")
	columnID := chi.URLParam(r, "columnId")

	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Assignee    string   `json:"assignee"`
		IssueID     string   `json:"issue_id"`
		PRID        string   `json:"pr_id"`
		Labels      []string `json:"labels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	card := &projects.Card{
		Title:       req.Title,
		Description: req.Description,
		Assignee:    req.Assignee,
		IssueID:     req.IssueID,
		PRID:        req.PRID,
		Labels:      req.Labels,
	}

	if err := h.store.AddCard(repoID, projectID, columnID, card); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(card)
}

// MoveCard moves a card between columns
func (h *ProjectHandler) MoveCard(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	projectID := chi.URLParam(r, "projectId")
	cardID := chi.URLParam(r, "cardId")

	var req struct {
		TargetColumnID string `json:"target_column_id"`
		TargetOrder    int    `json:"target_order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.MoveCard(repoID, projectID, cardID, req.TargetColumnID, req.TargetOrder); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// DeleteCard deletes a card
func (h *ProjectHandler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	projectID := chi.URLParam(r, "projectId")
	cardID := chi.URLParam(r, "cardId")

	if err := h.store.DeleteCard(repoID, projectID, cardID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	projectID := chi.URLParam(r, "projectId")

	if err := h.store.Delete(repoID, projectID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
