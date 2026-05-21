package crdt

import (
	"testing"
)

func TestIssue(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	if issue.ID != "issue-1" {
		t.Fatal("expected ID to match")
	}

	if issue.Title != "Test Issue" {
		t.Fatal("expected title to match")
	}

	if issue.Status != StatusOpen {
		t.Fatal("expected status to be open")
	}

	if issue.Author != "did:key:z123" {
		t.Fatal("expected author to match")
	}
}

func TestIssueSetTitle(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.SetTitle("did:key:z123", "Updated Title")

	if issue.Title != "Updated Title" {
		t.Fatal("expected title to be updated")
	}

	ops := issue.Log().Operations()
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
}

func TestIssueSetBody(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.SetBody("did:key:z123", "Updated body")

	if issue.Body != "Updated body" {
		t.Fatal("expected body to be updated")
	}
}

func TestIssueAddComment(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.AddComment("did:key:z456", "This is a comment")

	ops := issue.Log().Operations()
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
}

func TestIssueSetStatus(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.SetStatus("did:key:z123", StatusClosed)

	if issue.Status != StatusClosed {
		t.Fatal("expected status to be closed")
	}
}

func TestIssueLabels(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.AddLabel("did:key:z123", "bug")
	issue.AddLabel("did:key:z123", "priority:high")

	if len(issue.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(issue.Labels))
	}

	issue.RemoveLabel("did:key:z123", "bug")

	if len(issue.Labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(issue.Labels))
	}

	if issue.Labels[0] != "priority:high" {
		t.Fatal("expected label to be priority:high")
	}
}

func TestIssueAssignee(t *testing.T) {
	issue := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")

	issue.SetAssignee("did:key:z123", "did:key:z456")

	if issue.Assignee != "did:key:z456" {
		t.Fatal("expected assignee to be set")
	}
}

func TestIssueMerge(t *testing.T) {
	issue1 := NewIssue("issue-1", "did:key:z123", "Test Issue", "This is a test issue")
	issue2 := NewIssue("issue-1", "did:key:z456", "Test Issue", "This is a test issue")

	// Make changes on both
	issue1.SetTitle("did:key:z123", "Updated by user 1")
	issue2.SetBody("did:key:z456", "Updated by user 2")

	// Merge
	issue1.Merge(issue2)

	// Both changes should be applied
	if issue1.Title != "Updated by user 1" {
		t.Fatal("expected title to be from issue1")
	}
}

func TestPullRequest(t *testing.T) {
	pr := NewPullRequest("pr-1", "did:key:z123", "Test PR", "This is a test PR", "feature", "main")

	if pr.ID != "pr-1" {
		t.Fatal("expected ID to match")
	}

	if pr.Title != "Test PR" {
		t.Fatal("expected title to match")
	}

	if pr.Status != StatusOpen {
		t.Fatal("expected status to be open")
	}

	if pr.SourceBranch != "feature" {
		t.Fatal("expected source branch to match")
	}

	if pr.TargetBranch != "main" {
		t.Fatal("expected target branch to match")
	}
}

func TestPullRequestReviewers(t *testing.T) {
	pr := NewPullRequest("pr-1", "did:key:z123", "Test PR", "This is a test PR", "feature", "main")

	pr.AddReviewer("did:key:z123", "did:key:z456")
	pr.AddReviewer("did:key:z123", "did:key:z789")

	if len(pr.Reviewers) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(pr.Reviewers))
	}
}

func TestPullRequestMerge(t *testing.T) {
	pr1 := NewPullRequest("pr-1", "did:key:z123", "Test PR", "This is a test PR", "feature", "main")
	pr2 := NewPullRequest("pr-1", "did:key:z456", "Test PR", "This is a test PR", "feature", "main")

	// Make changes on both
	pr1.SetTitle("did:key:z123", "Updated by user 1")
	pr2.AddReviewer("did:key:z456", "did:key:z789")

	// Merge
	pr1.Merge(pr2)

	// Both changes should be applied
	if pr1.Title != "Updated by user 1" {
		t.Fatal("expected title to be from pr1")
	}
}
