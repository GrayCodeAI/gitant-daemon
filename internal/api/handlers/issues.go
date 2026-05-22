package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// CreateIssue creates a new issue
func CreateIssue(store *crdt.IssueStore, wm *webhooks.Manager) http.HandlerFunc {
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
		_ = store.Save()

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventIssueCreated,
			Repo: repoID,
			Data: map[string]interface{}{
				"issue_id": issueID,
				"title":    req.Title,
				"author":   author,
			},
		})

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
		offset, limit := ParsePagination(r)
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

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
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
func CommentIssue(store *crdt.IssueStore, wm *webhooks.Manager) http.HandlerFunc {
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
		_ = store.Save()

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventIssueCommented,
			Repo: repoID,
			Data: map[string]interface{}{
				"issue_id": issueID,
				"author":   author,
				"body":     req.Body,
			},
		})

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
func CloseIssue(store *crdt.IssueStore, wm *webhooks.Manager) http.HandlerFunc {
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
		_ = store.Save()

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventIssueClosed,
			Repo: repoID,
			Data: map[string]interface{}{
				"issue_id": issueID,
				"author":   author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      issueID,
			"status":  string(crdt.StatusClosed),
		})
	}
}

// ListIssueComments lists all comments on an issue
func ListIssueComments(store *crdt.IssueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		issueID := chi.URLParam(r, "issueId")

		issue, err := store.Get(repoID, issueID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		comments := make([]map[string]interface{}, 0)
		for _, op := range issue.Log().Operations() {
			if op.Type == crdt.OpAddComment {
				comments = append(comments, map[string]interface{}{
					"id":        op.ID,
					"author":    op.Author,
					"body":      op.Data["comment"],
					"timestamp": op.Timestamp.Format(time.RFC3339),
				})
			}
		}

		paged, total := PaginateSlice(comments, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"comments": paged,
			"total":    total,
			"offset":   offset,
			"limit":    limit,
		})
	}
}
