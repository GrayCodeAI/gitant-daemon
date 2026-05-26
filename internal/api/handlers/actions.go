package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/runner"
)

// ActionsHandler handles CI/CD workflow endpoints
type ActionsHandler struct {
	runner *runner.Runner
}

// NewActionsHandler creates a new actions handler
func NewActionsHandler(r *runner.Runner) *ActionsHandler {
	return &ActionsHandler{runner: r}
}

// ListRuns lists all workflow runs
func (h *ActionsHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	runs := h.runner.ListRuns()

	result := make([]map[string]interface{}, len(runs))
	for i, run := range runs {
		result[i] = map[string]interface{}{
			"id":         run.ID,
			"commit_sha": run.CommitSHA,
			"branch":     run.Branch,
			"status":     string(run.Status),
			"started_at": run.StartedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runs": result,
	})
}

// GetRun gets a workflow run by ID
func (h *ActionsHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")

	run, ok := h.runner.GetRun(runID)
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         run.ID,
		"commit_sha": run.CommitSHA,
		"branch":     run.Branch,
		"status":     string(run.Status),
		"started_at": run.StartedAt.Format("2006-01-02T15:04:05Z"),
		"logs":       run.Logs,
	})
}
