package crdt

import (
	"time"
)

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

// Log returns the operation log
func (pr *PullRequest) Log() *OperationLog {
	return pr.log
}

// Merge merges another PR's operations into this one
func (pr *PullRequest) Merge(other *PullRequest) {
	// Collect all operations from both logs
	allOps := append(pr.log.Operations(), other.log.Operations()...)

	// For now, just append other's operations
	for _, op := range other.log.Operations() {
		pr.log.Add(op)
	}

	// Apply operations to rebuild state
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
		}
	}
}
