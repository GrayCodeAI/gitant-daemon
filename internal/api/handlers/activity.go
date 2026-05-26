package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// ActivityEvent represents an activity event
type ActivityEvent struct {
	Type      string      `json:"type"`
	RepoID    string      `json:"repo_id"`
	Actor     string      `json:"actor"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ActivityFeed combines all events into a unified feed
type ActivityFeed struct {
	issues   *crdt.IssueStore
	prs      *crdt.PullRequestStore
	tasks    *crdt.TaskStore
	releases *crdt.ReleaseStore
}

// NewActivityFeed creates a new activity feed
func NewActivityFeed(issues *crdt.IssueStore, prs *crdt.PullRequestStore, tasks *crdt.TaskStore, releases *crdt.ReleaseStore) *ActivityFeed {
	return &ActivityFeed{
		issues:   issues,
		prs:      prs,
		tasks:    tasks,
		releases: releases,
	}
}

// GetActivity returns the activity feed
func (f *ActivityFeed) GetActivity(w http.ResponseWriter, r *http.Request) {
	events := make([]ActivityEvent, 0)

	// Collect issue events
	allIssues := f.issues.All()
	for repoID, issues := range allIssues {
		for _, issue := range issues {
			events = append(events, ActivityEvent{
				Type:      "issue." + string(issue.Status),
				RepoID:    repoID,
				Actor:     issue.Author,
				Timestamp: issue.UpdatedAt,
				Data: map[string]interface{}{
					"id":     issue.ID,
					"title":  issue.Title,
					"status": issue.Status,
				},
			})
		}
	}

	// Collect PR events
	allPRs := f.prs.All()
	for repoID, prs := range allPRs {
		for _, pr := range prs {
			events = append(events, ActivityEvent{
				Type:      "pr." + string(pr.Status),
				RepoID:    repoID,
				Actor:     pr.Author,
				Timestamp: pr.UpdatedAt,
				Data: map[string]interface{}{
					"id":     pr.ID,
					"title":  pr.Title,
					"status": pr.Status,
				},
			})
		}
	}

	// Sort by timestamp descending
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	// Apply pagination
	offset, limit := ParsePagination(r)
	if offset >= len(events) {
		events = []ActivityEvent{}
	} else {
		end := offset + limit
		if end > len(events) {
			end = len(events)
		}
		events = events[offset:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}
