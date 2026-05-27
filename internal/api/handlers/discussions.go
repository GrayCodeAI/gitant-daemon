package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/discussions"
)

// DiscussionHandler handles discussion endpoints
type DiscussionHandler struct {
	store *discussions.Store
}

// NewDiscussionHandler creates a new discussion handler
func NewDiscussionHandler(store *discussions.Store) *DiscussionHandler {
	return &DiscussionHandler{store: store}
}

// ListDiscussions lists discussions for a repository
func (h *DiscussionHandler) ListDiscussions(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")

	discussions := h.store.List(repoID, category, status)

	result := make([]map[string]interface{}, len(discussions))
	for i, d := range discussions {
		result[i] = map[string]interface{}{
			"id":         d.ID,
			"title":      d.Title,
			"body":       d.Body,
			"author":     d.Author,
			"category":   d.Category,
			"status":     d.Status,
			"tags":       d.Tags,
			"answers":    len(d.Answers),
			"upvotes":    d.Upvotes,
			"views":      d.Views,
			"created_at": d.CreatedAt,
			"updated_at": d.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"discussions": result,
	})
}

// GetDiscussion gets a discussion by ID
func (h *DiscussionHandler) GetDiscussion(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	discussionID := chi.URLParam(r, "discussionId")

	discussion, err := h.store.Get(repoID, discussionID)
	if err != nil {
		http.Error(w, SanitizeError(err, "discussion not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discussion)
}

// CreateDiscussion creates a new discussion
func (h *DiscussionHandler) CreateDiscussion(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	var req struct {
		Title    string   `json:"title"`
		Body     string   `json:"body"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	author := "anonymous"
	if user := middleware.GetUser(r); user != nil {
		author = user.Username
	}

	discussion := &discussions.Discussion{
		RepoID:   repoID,
		Title:    req.Title,
		Body:     req.Body,
		Author:   author,
		Category: req.Category,
		Tags:     req.Tags,
	}

	if err := h.store.Create(discussion); err != nil {
		http.Error(w, SanitizeError(err, "failed to create discussion"), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(discussion)
}

// AddAnswer adds an answer to a discussion
func (h *DiscussionHandler) AddAnswer(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	discussionID := chi.URLParam(r, "discussionId")

	var req struct {
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	author := "anonymous"
	if user := middleware.GetUser(r); user != nil {
		author = user.Username
	}

	answer := &discussions.Answer{
		Body:   req.Body,
		Author: author,
	}

	if err := h.store.AddAnswer(repoID, discussionID, answer); err != nil {
		http.Error(w, SanitizeError(err, "discussion not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(answer)
}

// AcceptAnswer accepts an answer
func (h *DiscussionHandler) AcceptAnswer(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	discussionID := chi.URLParam(r, "discussionId")
	answerID := chi.URLParam(r, "answerId")

	if err := h.store.AcceptAnswer(repoID, discussionID, answerID); err != nil {
		http.Error(w, SanitizeError(err, "answer not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// UpvoteDiscussion upvotes a discussion
func (h *DiscussionHandler) UpvoteDiscussion(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	discussionID := chi.URLParam(r, "discussionId")

	if err := h.store.Upvote(repoID, discussionID); err != nil {
		http.Error(w, SanitizeError(err, "discussion not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// DeleteDiscussion deletes a discussion
func (h *DiscussionHandler) DeleteDiscussion(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	discussionID := chi.URLParam(r, "discussionId")

	if err := h.store.Delete(repoID, discussionID); err != nil {
		http.Error(w, SanitizeError(err, "discussion not found"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
