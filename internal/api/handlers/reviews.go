package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/store"
)

// ReviewHandler handles PR review comment endpoints
type ReviewHandler struct {
	comments store.ReviewCommentStore
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(comments store.ReviewCommentStore) *ReviewHandler {
	return &ReviewHandler{comments: comments}
}

// CreateComment creates a new review comment
func (h *ReviewHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	prID := chi.URLParam(r, "prId")

	var req struct {
		FilePath   string `json:"file_path"`
		LineNumber int    `json:"line_number"`
		Body       string `json:"body"`
		ParentID   string `json:"parent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		http.Error(w, "file_path is required", http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	user := middleware.GetUser(r)
	authorID := "anonymous"
	if user != nil {
		authorID = user.ID
	}

	comment := &store.ReviewComment{
		ID:         generateID("rc"),
		PRID:       prID,
		FilePath:   req.FilePath,
		LineNumber: req.LineNumber,
		AuthorID:   authorID,
		Body:       req.Body,
		ParentID:   req.ParentID,
		Status:     "open",
	}

	if err := h.comments.Create(r.Context(), comment); err != nil {
		http.Error(w, "failed to create comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          comment.ID,
		"pr_id":       comment.PRID,
		"file_path":   comment.FilePath,
		"line_number": comment.LineNumber,
		"author_id":   comment.AuthorID,
		"body":        comment.Body,
		"parent_id":   comment.ParentID,
		"status":      comment.Status,
		"created_at":  comment.CreatedAt.Format(time.RFC3339),
	})
}

// ListComments lists all review comments for a PR
func (h *ReviewHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	prID := chi.URLParam(r, "prId")

	comments, err := h.comments.ListByPR(r.Context(), prID)
	if err != nil {
		http.Error(w, "failed to list comments", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(comments))
	for i, c := range comments {
		result[i] = map[string]interface{}{
			"id":          c.ID,
			"pr_id":       c.PRID,
			"file_path":   c.FilePath,
			"line_number": c.LineNumber,
			"author_id":   c.AuthorID,
			"body":        c.Body,
			"parent_id":   c.ParentID,
			"status":      c.Status,
			"created_at":  c.CreatedAt.Format(time.RFC3339),
			"updated_at":  c.UpdatedAt.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"comments": result,
	})
}

// ResolveComment resolves a review comment
func (h *ReviewHandler) ResolveComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "commentId")

	if err := h.comments.Resolve(r.Context(), commentID); err != nil {
		http.Error(w, "failed to resolve comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      commentID,
		"status":  "resolved",
	})
}

// DeleteComment deletes a review comment
func (h *ReviewHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "commentId")

	if err := h.comments.Delete(r.Context(), commentID); err != nil {
		http.Error(w, "failed to delete comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
