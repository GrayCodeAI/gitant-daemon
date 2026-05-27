package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// BatchHandler handles batch operations
type BatchHandler struct {
	issues   *crdt.IssueStore
	prs      *crdt.PullRequestStore
	webhooks *webhooks.Manager
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(issues *crdt.IssueStore, prs *crdt.PullRequestStore, webhooks *webhooks.Manager) *BatchHandler {
	return &BatchHandler{
		issues:   issues,
		prs:      prs,
		webhooks: webhooks,
	}
}

// BatchRequest represents a batch request
type BatchRequest struct {
	Operations []BatchOperation `json:"operations"`
}

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	Type   string          `json:"type"` // "create_issue", "close_issue", "create_pr"
	Params json.RawMessage `json:"params"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

const maxBatchOperations = 100

// Execute executes a batch of operations
func (h *BatchHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Operations) > maxBatchOperations {
		http.Error(w, fmt.Sprintf("batch exceeds maximum of %d operations", maxBatchOperations), http.StatusBadRequest)
		return
	}

	results := make([]BatchResult, len(req.Operations))
	for i, op := range req.Operations {
		switch op.Type {
		case "create_issue":
			var params struct {
				RepoID string `json:"repo_id"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				Author string `json:"author"`
			}
			if err := json.Unmarshal(op.Params, &params); err != nil {
				results[i] = BatchResult{Success: false, Error: "invalid params"}
				continue
			}
			if params.RepoID == "" || params.Title == "" {
				results[i] = BatchResult{Success: false, Error: "repo_id and title are required"}
				continue
			}
			issue := h.issues.Create(params.RepoID, generateID("batch"), params.Author, params.Title, params.Body)
			results[i] = BatchResult{Success: true, Data: issue}

		case "close_issue":
			var params struct {
				RepoID  string `json:"repo_id"`
				IssueID string `json:"issue_id"`
				Author  string `json:"author"`
			}
			if err := json.Unmarshal(op.Params, &params); err != nil {
				results[i] = BatchResult{Success: false, Error: "invalid params"}
				continue
			}
			if params.RepoID == "" || params.IssueID == "" {
				results[i] = BatchResult{Success: false, Error: "repo_id and issue_id are required"}
				continue
			}
			err := h.issues.Update(params.RepoID, params.IssueID, func(issue *crdt.Issue) error {
				issue.SetStatus(params.Author, crdt.StatusClosed)
				return nil
			})
			if err != nil {
				results[i] = BatchResult{Success: false, Error: SanitizeError(err, "operation failed")}
			} else {
				results[i] = BatchResult{Success: true}
			}

		default:
			results[i] = BatchResult{Success: false, Error: "unknown operation type"}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}
