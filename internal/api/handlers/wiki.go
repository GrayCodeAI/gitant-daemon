package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/wiki"
)

// WikiHandler handles wiki page endpoints
type WikiHandler struct {
	store *wiki.Store
}

// NewWikiHandler creates a new wiki handler
func NewWikiHandler(store *wiki.Store) *WikiHandler {
	return &WikiHandler{store: store}
}

// ListPages lists all wiki pages for a repository
func (h *WikiHandler) ListPages(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	pages := h.store.List(repoID)

	result := make([]map[string]interface{}, len(pages))
	for i, p := range pages {
		result[i] = map[string]interface{}{
			"id":         p.ID,
			"slug":       p.Slug,
			"title":      p.Title,
			"author":     p.Author,
			"created_at": p.CreatedAt,
			"updated_at": p.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": result,
	})
}

// GetPage gets a wiki page by slug
func (h *WikiHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")

	page, err := h.store.Get(repoID, slug)
	if err != nil {
		http.Error(w, SanitizeError(err, "page not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         page.ID,
		"slug":       page.Slug,
		"title":      page.Title,
		"content":    page.Content,
		"author":     page.Author,
		"created_at": page.CreatedAt,
		"updated_at": page.UpdatedAt,
	})
}

// CreatePage creates a new wiki page
func (h *WikiHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	var req struct {
		Slug    string `json:"slug"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Slug == "" || req.Title == "" {
		http.Error(w, "slug and title are required", http.StatusBadRequest)
		return
	}

	author := "anonymous"
	if user := middleware.GetUser(r); user != nil {
		author = user.Username
	}

	page := &wiki.Page{
		RepoID:  repoID,
		Slug:    req.Slug,
		Title:   req.Title,
		Content: req.Content,
		Author:  author,
	}

	if err := h.store.Create(page); err != nil {
		http.Error(w, SanitizeError(err, "failed to create page"), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(page)
}

// UpdatePage updates an existing wiki page
func (h *WikiHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.store.Update(repoID, slug, func(page *wiki.Page) error {
		if req.Title != "" {
			page.Title = req.Title
		}
		if req.Content != "" {
			page.Content = req.Content
		}
		return nil
	})

	if err != nil {
		http.Error(w, SanitizeError(err, "page not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// DeletePage deletes a wiki page
func (h *WikiHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")

	if err := h.store.Delete(repoID, slug); err != nil {
		http.Error(w, SanitizeError(err, "page not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
