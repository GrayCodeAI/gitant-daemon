package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/wiki"
)

// WikiHandler handles wiki endpoints
type WikiHandler struct {
	wikis map[string]*wiki.Wiki
}

// NewWikiHandler creates a new wiki handler
func NewWikiHandler(baseDir string) *WikiHandler {
	return &WikiHandler{
		wikis: make(map[string]*wiki.Wiki),
	}
}

// getWiki gets or creates a wiki for a repository
func (h *WikiHandler) getWiki(repoID string) *wiki.Wiki {
	if w, ok := h.wikis[repoID]; ok {
		return w
	}
	w := wiki.NewWiki("", repoID)
	h.wikis[repoID] = w
	return w
}

// ListPages lists all wiki pages
func (h *WikiHandler) ListPages(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	wiki := h.getWiki(repoID)

	pages, err := wiki.ListPages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(pages))
	for i, page := range pages {
		result[i] = map[string]interface{}{
			"title":      page.Title,
			"slug":       page.Slug,
			"updated_at": page.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": result,
	})
}

// GetPage gets a wiki page
func (h *WikiHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")
	wiki := h.getWiki(repoID)

	page, err := wiki.GetPage(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":      page.Title,
		"slug":       page.Slug,
		"content":    page.Content,
		"author":     page.Author,
		"created_at": page.CreatedAt,
		"updated_at": page.UpdatedAt,
	})
}

// CreatePage creates a wiki page
func (h *WikiHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	wiki := h.getWiki(repoID)

	var req struct {
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	author := "anonymous"
	if user := middleware.GetUser(r); user != nil {
		author = user.Username
	}

	page, err := wiki.CreatePage(req.Slug, req.Content, author)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":      page.Title,
		"slug":       page.Slug,
		"content":    page.Content,
		"author":     page.Author,
		"created_at": page.CreatedAt,
	})
}

// UpdatePage updates a wiki page
func (h *WikiHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")
	wiki := h.getWiki(repoID)

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	author := "anonymous"
	if user := middleware.GetUser(r); user != nil {
		author = user.Username
	}

	page, err := wiki.UpdatePage(slug, req.Content, author)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":      page.Title,
		"slug":       page.Slug,
		"content":    page.Content,
		"author":     page.Author,
		"updated_at": page.UpdatedAt,
	})
}

// DeletePage deletes a wiki page
func (h *WikiHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	slug := chi.URLParam(r, "slug")
	wiki := h.getWiki(repoID)

	if err := wiki.DeletePage(slug); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// SearchWiki searches wiki pages
func (h *WikiHandler) SearchWiki(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	query := r.URL.Query().Get("q")
	wiki := h.getWiki(repoID)

	if query == "" {
		http.Error(w, "q parameter is required", http.StatusBadRequest)
		return
	}

	pages, err := wiki.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(pages))
	for i, page := range pages {
		result[i] = map[string]interface{}{
			"title":      page.Title,
			"slug":       page.Slug,
			"updated_at": page.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": result,
	})
}
