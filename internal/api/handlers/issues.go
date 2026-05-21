package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// CreateIssue creates a new issue
func CreateIssue(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		var req struct {
			Title  string   `json:"title"`
			Body   string   `json:"body"`
			Labels []string `json:"labels"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		// Get author from context (set by auth middleware)
		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		issueID := fmt.Sprintf("issue-%d", time.Now().UnixNano())
		issue := store.Create(repoID, issueID, author, req.Title, req.Body)

		for _, label := range req.Labels {
			issue.AddLabel(author, label)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         issue.ID,
			"repo":       repoID,
			"title":      issue.Title,
			"body":       issue.Body,
			"status":     string(issue.Status),
			"author":     issue.Author,
			"labels":     issue.Labels,
			"created_at": issue.CreatedAt.Format(time.RFC3339),
		})
	}
}

// ListIssues lists all issues in a repository
func ListIssues(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		issues := store.List(repoID)

		result := make([]map[string]interface{}, 0, len(issues))
		for _, issue := range issues {
			result = append(result, map[string]interface{}{
				"id":         issue.ID,
				"repo":       repoID,
				"title":      issue.Title,
				"body":       issue.Body,
				"status":     string(issue.Status),
				"author":     issue.Author,
				"labels":     issue.Labels,
				"created_at": issue.CreatedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": result,
			"total":  len(result),
		})
	}
}

// GetIssue gets an issue by ID
func GetIssue(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		issueID := chi.URLParam(r, "issueId")

		issue, err := store.Get(repoID, issueID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         issue.ID,
			"repo":       repoID,
			"title":      issue.Title,
			"body":       issue.Body,
			"status":     string(issue.Status),
			"author":     issue.Author,
			"labels":     issue.Labels,
			"assignee":   issue.Assignee,
			"created_at": issue.CreatedAt.Format(time.RFC3339),
			"updated_at": issue.UpdatedAt.Format(time.RFC3339),
		})
	}
}

// CommentIssue adds a comment to an issue
func CommentIssue(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		issueID := chi.URLParam(r, "issueId")

		var req struct {
			Body string `json:"body"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		issue, err := store.Get(repoID, issueID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		issue.AddComment(author, req.Body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"issue":   issueID,
			"comment": req.Body,
			"author":  author,
		})
	}
}

// CloseIssue closes an issue
func CloseIssue(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		issueID := chi.URLParam(r, "issueId")

		issue, err := store.Get(repoID, issueID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		issue.SetStatus(author, crdt.StatusClosed)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      issueID,
			"status":  string(crdt.StatusClosed),
		})
	}
}
