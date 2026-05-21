package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// OpenPR opens a new pull request
func OpenPR(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		var req struct {
			Title        string `json:"title"`
			Body         string `json:"body"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		prID := fmt.Sprintf("pr-%d", time.Now().UnixNano())
		pr := store.Create(repoID, prID, author, req.Title, req.Body, req.SourceBranch, req.TargetBranch)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            pr.ID,
			"repo":          repoID,
			"title":         pr.Title,
			"body":          pr.Body,
			"status":        string(pr.Status),
			"author":        pr.Author,
			"source_branch": pr.SourceBranch,
			"target_branch": pr.TargetBranch,
			"created_at":    pr.CreatedAt.Format(time.RFC3339),
		})
	}
}

// ListPRs lists all pull requests in a repository
func ListPRs(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		prs := store.List(repoID)

		result := make([]map[string]interface{}, 0, len(prs))
		for _, pr := range prs {
			result = append(result, map[string]interface{}{
				"id":            pr.ID,
				"repo":          repoID,
				"title":         pr.Title,
				"body":          pr.Body,
				"status":        string(pr.Status),
				"author":        pr.Author,
				"source_branch": pr.SourceBranch,
				"target_branch": pr.TargetBranch,
				"created_at":    pr.CreatedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"prs":   result,
			"total": len(result),
		})
	}
}

// GetPR gets a pull request by ID
func GetPR(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		prID := chi.URLParam(r, "prId")

		pr, err := store.Get(repoID, prID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            pr.ID,
			"repo":          repoID,
			"title":         pr.Title,
			"body":          pr.Body,
			"status":        string(pr.Status),
			"author":        pr.Author,
			"source_branch": pr.SourceBranch,
			"target_branch": pr.TargetBranch,
			"labels":        pr.Labels,
			"assignee":      pr.Assignee,
			"reviewers":     pr.Reviewers,
			"created_at":    pr.CreatedAt.Format(time.RFC3339),
			"updated_at":    pr.UpdatedAt.Format(time.RFC3339),
		})
	}
}

// ReviewPR adds a review to a pull request
func ReviewPR(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		prID := chi.URLParam(r, "prId")

		var req struct {
			Verdict string `json:"verdict"`
			Body    string `json:"body"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		pr, err := store.Get(repoID, prID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		// Add reviewer and comment
		pr.AddReviewer(author, author)
		comment := fmt.Sprintf("Review [%s]: %s", req.Verdict, req.Body)
		pr.AddComment(author, comment)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"pr":      prID,
			"reviewer": author,
			"verdict":  req.Verdict,
		})
	}
}

// MergePR merges a pull request
func MergePR(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		prID := chi.URLParam(r, "prId")

		pr, err := store.Get(repoID, prID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
		}

		pr.SetStatus(author, crdt.StatusMerged)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      prID,
			"status":  string(crdt.StatusMerged),
		})
	}
}
