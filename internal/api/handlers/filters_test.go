package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/lakshmanpatel/gitant/internal/crdt"
)

func TestFilterIssuesByStatusAndLabels(t *testing.T) {
	openBug := &crdt.Issue{ID: "1", Status: crdt.StatusOpen, Labels: []string{"bug"}}
	closedBug := &crdt.Issue{ID: "2", Status: crdt.StatusClosed, Labels: []string{"bug"}}
	openFeature := &crdt.Issue{ID: "3", Status: crdt.StatusOpen, Labels: []string{"feature"}}
	issues := []*crdt.Issue{openBug, closedBug, openFeature}

	filtered := filterIssues(issues, crdt.StatusOpen, nil)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 open issues, got %d", len(filtered))
	}

	filtered = filterIssues(issues, crdt.StatusOpen, []string{"bug"})
	if len(filtered) != 1 || filtered[0].ID != "1" {
		t.Fatalf("expected open bug issue only, got %+v", filtered)
	}
}

func TestFilterPRsByStatus(t *testing.T) {
	open := &crdt.PullRequest{ID: "1", Status: crdt.StatusOpen}
	merged := &crdt.PullRequest{ID: "2", Status: crdt.StatusMerged}
	prs := []*crdt.PullRequest{open, merged}

	filtered := filterPRs(prs, crdt.StatusMerged)
	if len(filtered) != 1 || filtered[0].ID != "2" {
		t.Fatalf("expected merged PR only, got %+v", filtered)
	}
}

func TestParseIssueListFiltersFromQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/issues?status=closed&labels=bug,critical", nil)
	if got := parseIssueStatusFilter(req); got != crdt.StatusClosed {
		t.Fatalf("expected closed status, got %q", got)
	}
	if got := parseLabelsFilter(req); len(got) != 2 || got[0] != "bug" || got[1] != "critical" {
		t.Fatalf("unexpected labels: %#v", got)
	}

	req = httptest.NewRequest("GET", "/issues?status=all", nil)
	if got := parseIssueStatusFilter(req); got != "" {
		t.Fatalf("expected empty status for all, got %q", got)
	}
}
