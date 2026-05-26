package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// OpenPR opens a new pull request
func OpenPR(store *crdt.PullRequestStore, wm *webhooks.Manager) http.HandlerFunc {
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
		if len(req.Title) > 256 {
			http.Error(w, "Title must be 256 characters or less", http.StatusBadRequest)
			return
		}
		if len(req.Body) > 65536 {
			http.Error(w, "Body must be 65536 characters or less", http.StatusBadRequest)
			return
		}
		if len(req.SourceBranch) > 256 {
			http.Error(w, "Source branch must be 256 characters or less", http.StatusBadRequest)
			return
		}
		if len(req.TargetBranch) > 256 {
			http.Error(w, "Target branch must be 256 characters or less", http.StatusBadRequest)
			return
		}

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		prID := fmt.Sprintf("pr-%d", time.Now().UnixNano())
		pr := store.Create(repoID, prID, author, req.Title, req.Body, req.SourceBranch, req.TargetBranch)
		_ = store.Save()

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventPROpened,
			Repo: repoID,
			Data: map[string]interface{}{
				"pr_id":         prID,
				"title":         req.Title,
				"author":        author,
				"source_branch": req.SourceBranch,
				"target_branch": req.TargetBranch,
			},
		})

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
		offset, limit := ParsePagination(r)
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

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"prs":    paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
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
func ReviewPR(store *crdt.PullRequestStore, wm *webhooks.Manager) http.HandlerFunc {
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

		validVerdicts := map[string]bool{"approve": true, "request_changes": true, "comment": true}
		if !validVerdicts[req.Verdict] {
			http.Error(w, "Verdict must be one of: approve, request_changes, comment", http.StatusBadRequest)
			return
		}
		if len(req.Body) > 65536 {
			http.Error(w, "Body must be 65536 characters or less", http.StatusBadRequest)
			return
		}

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		comment := fmt.Sprintf("Review [%s]: %s", req.Verdict, req.Body)
		err := store.Update(repoID, prID, func(pr *crdt.PullRequest) error {
			pr.AddReviewer(author, author)
			pr.AddComment(author, comment)
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventPRReviewed,
			Repo: repoID,
			Data: map[string]interface{}{
				"pr_id":   prID,
				"author":  author,
				"verdict": req.Verdict,
				"body":    req.Body,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"pr":       prID,
			"reviewer": author,
			"verdict":  req.Verdict,
		})
	}
}

// MergePR merges a pull request into its target branch, then marks it merged.
func MergePR(store *crdt.PullRequestStore, registry *storage.RepositoryRegistry, protections *storage.ProtectionStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		prID := chi.URLParam(r, "prId")

		var req struct {
			MergeMethod string `json:"merge_method"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		mergeMethod := req.MergeMethod
		if mergeMethod == "" {
			mergeMethod = "merge"
		}

		// Get PR to check status and target branch before mutating
		pr, err := store.Get(repoID, prID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if pr.Status != crdt.StatusOpen {
			http.Error(w, "Pull request is not open", http.StatusBadRequest)
			return
		}

		// Check branch protection
		if protections != nil {
			prot := protections.Get(repoID, pr.TargetBranch)
			if prot != nil {
				if prot.RequireApproval {
					hasApproval := false
					for _, op := range pr.Log().Operations() {
						if op.Type == crdt.OpAddComment {
							if comment, ok := op.Data["comment"].(string); ok && strings.Contains(comment, "Review [approve]") {
								hasApproval = true
								break
							}
						}
					}
					if !hasApproval {
						http.Error(w, "Branch protection requires approval before merging", http.StatusForbidden)
						return
					}
				}
					// NoForcePush protection is enforced at push time, not merge time
			}
		}

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}
		mergeHash, err := repo.MergeBranches(pr.TargetBranch, pr.SourceBranch, author, "", mergeMethod)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to merge branch: %v", err), http.StatusInternalServerError)
			return
		}

		err = store.Update(repoID, prID, func(pr *crdt.PullRequest) error {
			pr.SetStatus(author, crdt.StatusMerged)
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventPRMerged,
			Repo: repoID,
			Data: map[string]interface{}{
				"pr_id":  prID,
				"author": author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"id":         prID,
			"status":     string(crdt.StatusMerged),
			"merge_hash": mergeHash.String(),
		})
	}
}

// ListPRComments lists all comments on a pull request
func ListPRComments(store *crdt.PullRequestStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		prID := chi.URLParam(r, "prId")

		pr, err := store.Get(repoID, prID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		comments := make([]map[string]interface{}, 0)
		for _, op := range pr.Log().Operations() {
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
