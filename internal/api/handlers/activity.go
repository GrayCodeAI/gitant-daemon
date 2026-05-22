package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// ActivityEvent represents a single activity event
type ActivityEvent struct {
	Type      string    `json:"type"`
	Repo      string    `json:"repo"`
	Actor     string    `json:"actor"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}

// GetActivity returns a unified activity feed across all repos
func GetActivity(issues *crdt.IssueStore, prs *crdt.PullRequestStore, tasks *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}

		events := make([]ActivityEvent, 0)

		// Aggregate from issues
		for repoID, repoIssues := range issues.All() {
			for _, issue := range repoIssues {
				for _, op := range issue.Log().Operations() {
					events = append(events, ActivityEvent{
						Type:      "issue." + string(op.Type),
						Repo:      repoID,
						Actor:     op.Author,
						Summary:   summarizeIssueOp(*op, issue.Title),
						Timestamp: op.Timestamp,
					})
				}
			}
		}

		// Aggregate from PRs
		for repoID, repoPRs := range prs.All() {
			for _, pr := range repoPRs {
				for _, op := range pr.Log().Operations() {
					events = append(events, ActivityEvent{
						Type:      "pr." + string(op.Type),
						Repo:      repoID,
						Actor:     op.Author,
						Summary:   summarizePROp(*op, pr.Title),
						Timestamp: op.Timestamp,
					})
				}
			}
		}

		// Aggregate from tasks
		for repoID, repoTasks := range tasks.All() {
			for _, task := range repoTasks {
				events = append(events, ActivityEvent{
					Type:      "task.created",
					Repo:      repoID,
					Actor:     task.CreatedBy,
					Summary:   "Created task: " + task.Title,
					Timestamp: task.CreatedAt,
				})
				if task.ClaimedAt != nil {
					events = append(events, ActivityEvent{
						Type:      "task.claimed",
						Repo:      repoID,
						Actor:     task.ClaimedBy,
						Summary:   "Claimed task: " + task.Title,
						Timestamp: *task.ClaimedAt,
					})
				}
				if task.CompletedAt != nil {
					events = append(events, ActivityEvent{
						Type:      "task.completed",
						Repo:      repoID,
						Actor:     task.ClaimedBy,
						Summary:   "Completed task: " + task.Title,
						Timestamp: *task.CompletedAt,
					})
				}
			}
		}

		// Sort by timestamp descending
		sort.Slice(events, func(i, j int) bool {
			return events[i].Timestamp.After(events[j].Timestamp)
		})

		// Apply limit
		if len(events) > limit {
			events = events[:limit]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"events": events,
			"total":  len(events),
		})
	}
}

func summarizeIssueOp(op crdt.Operation, title string) string {
	switch op.Type {
	case crdt.OpCreate:
		return "Created issue: " + title
	case crdt.OpSetStatus:
		if status, ok := op.Data["status"]; ok {
			return "Changed status to " + fmt.Sprintf("%v", status) + " on issue: " + title
		}
		return "Updated issue status: " + title
	case crdt.OpAddComment:
		return "Commented on issue: " + title
	case crdt.OpAddLabel:
		if label, ok := op.Data["label"]; ok {
			return "Added label '" + fmt.Sprintf("%v", label) + "' to issue: " + title
		}
		return "Added label to issue: " + title
	case crdt.OpRemoveLabel:
		if label, ok := op.Data["label"]; ok {
			return "Removed label '" + fmt.Sprintf("%v", label) + "' from issue: " + title
		}
		return "Removed label from issue: " + title
	default:
		return "Updated issue: " + title
	}
}

func summarizePROp(op crdt.Operation, title string) string {
	switch op.Type {
	case crdt.OpCreate:
		return "Opened PR: " + title
	case crdt.OpSetStatus:
		if status, ok := op.Data["status"]; ok {
			return "Changed status to " + fmt.Sprintf("%v", status) + " on PR: " + title
		}
		return "Updated PR status: " + title
	case crdt.OpAddComment:
		return "Commented on PR: " + title
	case crdt.OpAddLabel:
		if label, ok := op.Data["label"]; ok {
			return "Added label '" + fmt.Sprintf("%v", label) + "' to PR: " + title
		}
		return "Added label to PR: " + title
	default:
		return "Updated PR: " + title
	}
}
