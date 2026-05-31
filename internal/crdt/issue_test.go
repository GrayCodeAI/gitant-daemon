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

func TestTaskStoreMergeRemote(t *testing.T) {
	store1 := NewTaskStore("")

	// Create task on store1
	task := store1.Create("repo-1", "task-1", "did:key:z123", "Fix bug", "Fix the login bug")

	// Simulate remote task that has been claimed
	remote := &Task{
		ID: "task-1", RepoID: "repo-1", Title: "Fix bug", Description: "Fix the login bug",
		Status: TaskClaimed, ClaimedBy: "did:key:z456", CreatedBy: "did:key:z123",
		log: NewOperationLog(),
	}
	remote.log.Add(&Operation{ID: "op-create", Type: OpCreate, Author: "did:key:z123", Data: map[string]interface{}{"title": "Fix bug"}})
	remote.log.Add(&Operation{ID: "op-claim", Type: OpClaimTask, Author: "did:key:z456", Data: map[string]interface{}{"claimed_by": "did:key:z456"}})

	store1.MergeRemote("repo-1", remote)

	merged, err := store1.Get("repo-1", task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.ClaimedBy != "did:key:z456" {
		t.Fatalf("expected claimed by 'did:key:z456', got '%s'", merged.ClaimedBy)
	}
}

func TestLabelMerge(t *testing.T) {
	label1 := &Label{Name: "bug", Color: "#ff0000", log: NewOperationLog()}
	label2 := &Label{Name: "bug", Color: "#00ff00", log: NewOperationLog()}

	label1.SetColor("did:key:z123", "#ff0000")
	label2.SetColor("did:key:z456", "#00ff00")

	label1.Merge(label2)

	// After merge, one color should win (deterministic)
	if label1.Color == "" {
		t.Fatal("expected label to have a color after merge")
	}
}

func TestLabelMergeTombstone(t *testing.T) {
	label1 := &Label{Name: "bug", Color: "#ff0000", log: NewOperationLog()}
	label2 := &Label{Name: "bug", Color: "#ff0000", log: NewOperationLog()}

	label2.Tombstone("did:key:z456")
	label1.SetColor("did:key:z123", "#00ff00")

	label1.Merge(label2)

	if !label1.Tombstoned {
		t.Fatal("expected label to be tombstoned after merge")
	}
}

func TestReleaseStoreMergeRemote(t *testing.T) {
	store1 := NewReleaseStore("")

	// Create release on store1
	rel1, err := store1.Create("repo-1", "v1.0.0", "Release 1.0", "Initial release", "did:key:z123")
	if err != nil {
		t.Fatal(err)
	}

	// Create same release with tombstone operation
	rel2 := &Release{ID: rel1.ID, RepoID: "repo-1", Tag: "v1.0.0", Title: "Release 1.0", Author: "did:key:z123", log: NewOperationLog()}
	rel2.log.Add(&Operation{ID: "op-create", Type: OpCreate, Author: "did:key:z123"})
	rel2.Tombstone("did:key:z456")

	// Merge tombstone from remote into store1
	store1.MergeRemote("repo-1", rel2)

	// The release should be tombstoned (filtered from List)
	releases := store1.List("repo-1")
	if len(releases) != 0 {
		t.Fatalf("expected 0 releases after tombstone merge, got %d", len(releases))
	}
}

func TestReleaseMergeTitleAndBody(t *testing.T) {
	rel1 := &Release{ID: "rel-1", RepoID: "repo-1", Tag: "v1.0.0", Title: "Release 1.0", Body: "Initial", Author: "did:key:z123", log: NewOperationLog()}
	rel2 := &Release{ID: "rel-1", RepoID: "repo-1", Tag: "v1.0.0", Title: "Release 1.0", Body: "Initial", Author: "did:key:z123", log: NewOperationLog()}

	// Add operations directly (SetTitle/SetBody are applied during merge replay)
	rel1.log.Add(&Operation{ID: "op-title", Type: OpSetTitle, Author: "did:key:z123", Data: map[string]interface{}{"title": "Release 1.0.0"}})
	rel2.log.Add(&Operation{ID: "op-body", Type: OpSetBody, Author: "did:key:z456", Data: map[string]interface{}{"body": "Updated notes"}})

	rel1.Merge(rel2)

	// Both changes should be applied during merge replay
	if rel1.Title != "Release 1.0.0" {
		t.Fatalf("expected title 'Release 1.0.0', got '%s'", rel1.Title)
	}
	if rel1.Body != "Updated notes" {
		t.Fatalf("expected body 'Updated notes', got '%s'", rel1.Body)
	}
}
