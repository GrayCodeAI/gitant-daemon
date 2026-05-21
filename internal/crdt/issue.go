package crdt

import (
	"fmt"
	"time"
)

// Issue represents a CRDT issue
type Issue struct {
	ID        string
	Title     string
	Body      string
	Status    Status
	Author    string // DID
	Labels    []string
	Assignee  string // DID
	CreatedAt time.Time
	UpdatedAt time.Time
	log       *OperationLog
}

// NewIssue creates a new issue
func NewIssue(id, author, title, body string) *Issue {
	issue := &Issue{
		ID:        id,
		Title:     title,
		Body:      body,
		Status:    StatusOpen,
		Author:    author,
		Labels:    make([]string, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		log:       NewOperationLog(),
	}

	// Add create operation
	issue.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpCreate,
		Author: author,
		Data: map[string]interface{}{
			"title": title,
			"body":  body,
		},
	})

	return issue
}

// SetTitle sets the issue title
func (i *Issue) SetTitle(author, title string) {
	i.Title = title
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetTitle,
		Author: author,
		Data: map[string]interface{}{
			"title": title,
		},
	})
}

// SetBody sets the issue body
func (i *Issue) SetBody(author, body string) {
	i.Body = body
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetBody,
		Author: author,
		Data: map[string]interface{}{
			"body": body,
		},
	})
}

// AddComment adds a comment to the issue
func (i *Issue) AddComment(author, comment string) {
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpAddComment,
		Author: author,
		Data: map[string]interface{}{
			"comment": comment,
		},
	})
}

// SetStatus sets the issue status
func (i *Issue) SetStatus(author string, status Status) {
	i.Status = status
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetStatus,
		Author: author,
		Data: map[string]interface{}{
			"status": string(status),
		},
	})
}

// AddLabel adds a label to the issue
func (i *Issue) AddLabel(author, label string) {
	// Check if label already exists
	for _, l := range i.Labels {
		if l == label {
			return
		}
	}

	i.Labels = append(i.Labels, label)
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpAddLabel,
		Author: author,
		Data: map[string]interface{}{
			"label": label,
		},
	})
}

// RemoveLabel removes a label from the issue
func (i *Issue) RemoveLabel(author, label string) {
	for idx, l := range i.Labels {
		if l == label {
			i.Labels = append(i.Labels[:idx], i.Labels[idx+1:]...)
			break
		}
	}

	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpRemoveLabel,
		Author: author,
		Data: map[string]interface{}{
			"label": label,
		},
	})
}

// SetAssignee sets the issue assignee
func (i *Issue) SetAssignee(author, assignee string) {
	i.Assignee = assignee
	i.UpdatedAt = time.Now()

	i.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetAssignee,
		Author: author,
		Data: map[string]interface{}{
			"assignee": assignee,
		},
	})
}

// Log returns the operation log
func (i *Issue) Log() *OperationLog {
	return i.log
}

// Merge merges another issue's operations into this one
func (i *Issue) Merge(other *Issue) {
	// Collect all operations from both logs
	allOps := append(i.log.Operations(), other.log.Operations()...)

	// Sort by timestamp (simple merge - in production, use Lamport timestamps)
	// For now, just append other's operations
	for _, op := range other.log.Operations() {
		i.log.Add(op)
	}

	// Apply operations to rebuild state
	i.applyOperations(allOps)
}

// applyOperations applies operations to rebuild state
func (i *Issue) applyOperations(ops []*Operation) {
	for _, op := range ops {
		switch op.Type {
		case OpSetTitle:
			if title, ok := op.Data["title"].(string); ok {
				i.Title = title
			}
		case OpSetBody:
			if body, ok := op.Data["body"].(string); ok {
				i.Body = body
			}
		case OpSetStatus:
			if status, ok := op.Data["status"].(string); ok {
				i.Status = Status(status)
			}
		case OpAddLabel:
			if label, ok := op.Data["label"].(string); ok {
				found := false
				for _, l := range i.Labels {
					if l == label {
						found = true
						break
					}
				}
				if !found {
					i.Labels = append(i.Labels, label)
				}
			}
		case OpRemoveLabel:
			if label, ok := op.Data["label"].(string); ok {
				for idx, l := range i.Labels {
					if l == label {
						i.Labels = append(i.Labels[:idx], i.Labels[idx+1:]...)
						break
					}
				}
			}
		case OpSetAssignee:
			if assignee, ok := op.Data["assignee"].(string); ok {
				i.Assignee = assignee
			}
		}
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
