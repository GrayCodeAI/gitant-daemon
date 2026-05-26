package handlers

import (
	"net/http"
	"strings"

	"github.com/lakshmanpatel/gitant/internal/crdt"
)

func parseIssueStatusFilter(r *http.Request) crdt.Status {
	raw := strings.TrimSpace(r.URL.Query().Get("status"))
	if raw == "" || raw == "all" {
		return ""
	}
	return crdt.Status(raw)
}

func parsePRStatusFilter(r *http.Request) crdt.Status {
	raw := strings.TrimSpace(r.URL.Query().Get("status"))
	if raw == "" || raw == "all" {
		return ""
	}
	return crdt.Status(raw)
}

func parseLabelsFilter(r *http.Request) []string {
	raw := strings.TrimSpace(r.URL.Query().Get("labels"))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		if label := strings.TrimSpace(part); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

func issueHasLabels(issue *crdt.Issue, required []string) bool {
	if len(required) == 0 {
		return true
	}

	owned := make(map[string]struct{}, len(issue.Labels))
	for _, label := range issue.Labels {
		owned[label] = struct{}{}
	}
	for _, label := range required {
		if _, ok := owned[label]; !ok {
			return false
		}
	}
	return true
}

func filterIssues(issues []*crdt.Issue, status crdt.Status, labels []string) []*crdt.Issue {
	if status == "" && len(labels) == 0 {
		return issues
	}

	filtered := make([]*crdt.Issue, 0, len(issues))
	for _, issue := range issues {
		if status != "" && issue.Status != status {
			continue
		}
		if !issueHasLabels(issue, labels) {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
}

func filterPRs(prs []*crdt.PullRequest, status crdt.Status) []*crdt.PullRequest {
	if status == "" {
		return prs
	}

	filtered := make([]*crdt.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr.Status == status {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}
