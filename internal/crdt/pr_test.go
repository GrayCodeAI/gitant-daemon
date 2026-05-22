package crdt

import (
	"encoding/json"
	"testing"
)

func TestPullRequestJSONRoundTrip(t *testing.T) {
	pr := NewPullRequest("pr-1", "alice", "Feature", "A feature", "feature", "main")
	pr.AddLabel("alice", "enhancement")
	pr.AddReviewer("alice", "bob")

	// Marshal
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// Unmarshal
	var restored PullRequest
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if restored.ID != pr.ID {
		t.Fatalf("ID mismatch: %s vs %s", restored.ID, pr.ID)
	}
	if restored.Title != pr.Title {
		t.Fatalf("Title mismatch: %s vs %s", restored.Title, pr.Title)
	}
	if restored.SourceBranch != pr.SourceBranch {
		t.Fatalf("SourceBranch mismatch: %s vs %s", restored.SourceBranch, pr.SourceBranch)
	}
	if restored.TargetBranch != pr.TargetBranch {
		t.Fatalf("TargetBranch mismatch: %s vs %s", restored.TargetBranch, pr.TargetBranch)
	}
	if len(restored.Labels) != len(pr.Labels) {
		t.Fatalf("Labels count mismatch: %d vs %d", len(restored.Labels), len(pr.Labels))
	}
	if len(restored.Reviewers) != len(pr.Reviewers) {
		t.Fatalf("Reviewers count mismatch: %d vs %d", len(restored.Reviewers), len(pr.Reviewers))
	}
}

func TestPullRequestStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/prs.json"

	// Create and save
	store1 := NewPullRequestStore(path)
	store1.Create("repo", "pr-1", "alice", "Feature", "", "feature", "main")
	store1.Create("repo", "pr-2", "bob", "Fix", "", "fix", "main")
	if err := store1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load in new store
	store2 := NewPullRequestStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	prs := store2.List("repo")
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
}

func TestIssueStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/issues.json"

	// Create and save
	store1 := NewIssueStore(path)
	store1.Create("repo", "issue-1", "alice", "Bug", "It broke")
	store1.Create("repo", "issue-2", "bob", "Feature", "New thing")
	if err := store1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load in new store
	store2 := NewIssueStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	issues := store2.List("repo")
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
}

func TestIssueJSONRoundTrip(t *testing.T) {
	issue := NewIssue("issue-1", "alice", "Bug report", "Something broke")
	issue.AddLabel("alice", "bug")
	issue.AddComment("bob", "Confirmed")
	issue.SetStatus("alice", StatusClosed)

	// Marshal
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// Unmarshal
	var restored Issue
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if restored.ID != issue.ID {
		t.Fatalf("ID mismatch: %s vs %s", restored.ID, issue.ID)
	}
	if restored.Status != StatusClosed {
		t.Fatalf("Status mismatch: %s vs %s", restored.Status, StatusClosed)
	}
	if len(restored.Labels) != 1 {
		t.Fatalf("Labels count mismatch: %d vs 1", len(restored.Labels))
	}
}
