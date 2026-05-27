package crdt

import (
	"encoding/json"
	"sort"
	"time"
)

// prSnapshot is the JSON-serializable representation of a PullRequest
type prSnapshot struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	Status       Status       `json:"status"`
	Author       string       `json:"author"`
	SourceBranch string       `json:"source_branch"`
	TargetBranch string       `json:"target_branch"`
	Labels       []string     `json:"labels"`
	Assignee     string       `json:"assignee"`
	Reviewers    []string     `json:"reviewers"`
	Tombstoned   bool         `json:"tombstoned,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Log          []*Operation `json:"log"`
}

// MarshalJSON serializes a PullRequest including its operation log
func (pr *PullRequest) MarshalJSON() ([]byte, error) {
	snap := prSnapshot{
		ID:           pr.ID,
		Title:        pr.Title,
		Body:         pr.Body,
		Status:       pr.Status,
		Author:       pr.Author,
		SourceBranch: pr.SourceBranch,
		TargetBranch: pr.TargetBranch,
		Labels:       pr.Labels,
		Assignee:     pr.Assignee,
		Reviewers:    pr.Reviewers,
		Tombstoned:   pr.Tombstoned,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
		Log:          pr.log.Operations(),
	}
	return json.Marshal(snap)
}

// UnmarshalJSON deserializes a PullRequest and rebuilds its operation log
func (pr *PullRequest) UnmarshalJSON(data []byte) error {
	var snap prSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	pr.ID = snap.ID
	pr.Title = snap.Title
	pr.Body = snap.Body
	pr.Status = snap.Status
	pr.Author = snap.Author
	pr.SourceBranch = snap.SourceBranch
	pr.TargetBranch = snap.TargetBranch
	pr.Labels = snap.Labels
	if pr.Labels == nil {
		pr.Labels = make([]string, 0)
	}
	pr.Assignee = snap.Assignee
	pr.Reviewers = snap.Reviewers
	if pr.Reviewers == nil {
		pr.Reviewers = make([]string, 0)
	}
	pr.Tombstoned = snap.Tombstoned
	pr.CreatedAt = snap.CreatedAt
	pr.UpdatedAt = snap.UpdatedAt
	pr.log = NewOperationLog()
	for _, op := range snap.Log {
		pr.log.Add(op)
	}
	return nil
}

// PullRequest represents a CRDT pull request
type PullRequest struct {
	ID           string
	Title        string
	Body         string
	Status       Status
	Author       string // DID
	SourceBranch string
	TargetBranch string
	Labels       []string
	Assignee     string // DID
	Reviewers    []string // DIDs
	Tombstoned   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	log          *OperationLog
}

// NewPullRequest creates a new pull request
func NewPullRequest(id, author, title, body, sourceBranch, targetBranch string) *PullRequest {
	pr := &PullRequest{
		ID:           id,
		Title:        title,
		Body:         body,
		Status:       StatusOpen,
		Author:       author,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Labels:       make([]string, 0),
		Reviewers:    make([]string, 0),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		log:          NewOperationLog(),
	}

	// Add create operation
	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpCreate,
		Author: author,
		Data: map[string]interface{}{
			"title":         title,
			"body":          body,
			"source_branch": sourceBranch,
			"target_branch": targetBranch,
		},
	})

	return pr
}

// SetTitle sets the PR title
func (pr *PullRequest) SetTitle(author, title string) {
	pr.Title = title
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetTitle,
		Author: author,
		Data: map[string]interface{}{
			"title": title,
		},
	})
}

// SetBody sets the PR body
func (pr *PullRequest) SetBody(author, body string) {
	pr.Body = body
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetBody,
		Author: author,
		Data: map[string]interface{}{
			"body": body,
		},
	})
}

// AddComment adds a comment to the PR
func (pr *PullRequest) AddComment(author, comment string) {
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpAddComment,
		Author: author,
		Data: map[string]interface{}{
			"comment": comment,
		},
	})
}

// SetStatus sets the PR status
func (pr *PullRequest) SetStatus(author string, status Status) {
	pr.Status = status
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetStatus,
		Author: author,
		Data: map[string]interface{}{
			"status": string(status),
		},
	})
}

// AddLabel adds a label to the PR
func (pr *PullRequest) AddLabel(author, label string) {
	// Check if label already exists
	for _, l := range pr.Labels {
		if l == label {
			return
		}
	}

	pr.Labels = append(pr.Labels, label)
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpAddLabel,
		Author: author,
		Data: map[string]interface{}{
			"label": label,
		},
	})
}

// RemoveLabel removes a label from the PR
func (pr *PullRequest) RemoveLabel(author, label string) {
	for idx, l := range pr.Labels {
		if l == label {
			pr.Labels = append(pr.Labels[:idx], pr.Labels[idx+1:]...)
			break
		}
	}

	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpRemoveLabel,
		Author: author,
		Data: map[string]interface{}{
			"label": label,
		},
	})
}

// SetAssignee sets the PR assignee
func (pr *PullRequest) SetAssignee(author, assignee string) {
	pr.Assignee = assignee
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetAssignee,
		Author: author,
		Data: map[string]interface{}{
			"assignee": assignee,
		},
	})
}

// AddReviewer adds a reviewer to the PR
func (pr *PullRequest) AddReviewer(author, reviewer string) {
	// Check if reviewer already exists
	for _, r := range pr.Reviewers {
		if r == reviewer {
			return
		}
	}

	pr.Reviewers = append(pr.Reviewers, reviewer)
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   "add_reviewer",
		Author: author,
		Data: map[string]interface{}{
			"reviewer": reviewer,
		},
	})
}

// SetBranch sets the source or target branch
func (pr *PullRequest) SetBranch(author, sourceBranch, targetBranch string) {
	pr.SourceBranch = sourceBranch
	pr.TargetBranch = targetBranch
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetBranch,
		Author: author,
		Data: map[string]interface{}{
			"source_branch": sourceBranch,
			"target_branch": targetBranch,
		},
	})
}

// Tombstone marks this PR as deleted
func (pr *PullRequest) Tombstone(author string) {
	pr.Tombstoned = true
	pr.UpdatedAt = time.Now()

	pr.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpTombstone,
		Author: author,
	})
}

// Log returns the operation log
func (pr *PullRequest) Log() *OperationLog {
	return pr.log
}

// Merge merges another PR's operations into this one
func (pr *PullRequest) Merge(other *PullRequest) {
	// Merge Lamport clocks
	pr.log.clock.Merge(other.log.clock)

	// Collect existing IDs to deduplicate
	existingIDs := make(map[string]bool)
	for _, op := range pr.log.Operations() {
		existingIDs[op.ID] = true
	}

	// Add only new operations from other
	for _, op := range other.log.Operations() {
		if !existingIDs[op.ID] {
			pr.log.ImportOperation(op)
		}
	}

	// Collect all operations and sort by Lamport counter, then timestamp
	allOps := make([]*Operation, len(pr.log.Operations()))
	copy(allOps, pr.log.Operations())
	sort.Slice(allOps, func(a, b int) bool {
		if allOps[a].Lamport != allOps[b].Lamport {
			return allOps[a].Lamport < allOps[b].Lamport
		}
		return allOps[a].Timestamp.Before(allOps[b].Timestamp)
	})

	// Reset state and re-apply
	pr.Labels = make([]string, 0)
	pr.Reviewers = make([]string, 0)
	pr.Tombstoned = false
	pr.applyOperations(allOps)
}

// applyOperations applies operations to rebuild state
func (pr *PullRequest) applyOperations(ops []*Operation) {
	for _, op := range ops {
		switch op.Type {
		case OpSetTitle:
			if title, ok := op.Data["title"].(string); ok {
				pr.Title = title
			}
		case OpSetBody:
			if body, ok := op.Data["body"].(string); ok {
				pr.Body = body
			}
		case OpSetStatus:
			if status, ok := op.Data["status"].(string); ok {
				pr.Status = Status(status)
			}
		case OpAddLabel:
			if label, ok := op.Data["label"].(string); ok {
				found := false
				for _, l := range pr.Labels {
					if l == label {
						found = true
						break
					}
				}
				if !found {
					pr.Labels = append(pr.Labels, label)
				}
			}
		case OpRemoveLabel:
			if label, ok := op.Data["label"].(string); ok {
				for idx, l := range pr.Labels {
					if l == label {
						pr.Labels = append(pr.Labels[:idx], pr.Labels[idx+1:]...)
						break
					}
				}
			}
		case OpSetAssignee:
			if assignee, ok := op.Data["assignee"].(string); ok {
				pr.Assignee = assignee
			}
		case "add_reviewer":
			if reviewer, ok := op.Data["reviewer"].(string); ok {
				found := false
				for _, r := range pr.Reviewers {
					if r == reviewer {
						found = true
						break
					}
				}
				if !found {
					pr.Reviewers = append(pr.Reviewers, reviewer)
				}
			}
		case OpSetBranch:
			if sourceBranch, ok := op.Data["source_branch"].(string); ok {
				pr.SourceBranch = sourceBranch
			}
			if targetBranch, ok := op.Data["target_branch"].(string); ok {
				pr.TargetBranch = targetBranch
			}
		case OpTombstone:
			pr.Tombstoned = true
		}
	}
}
