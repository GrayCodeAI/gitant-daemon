package crdt

import (
	crypto_rand "crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// issueSnapshot is the JSON-serializable representation of an Issue
type issueSnapshot struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	Status    Status       `json:"status"`
	Author    string       `json:"author"`
	Labels    []string     `json:"labels"`
	Assignee  string       `json:"assignee"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Log       []*Operation `json:"log"`
}

// MarshalJSON serializes an Issue including its operation log
func (i *Issue) MarshalJSON() ([]byte, error) {
	snap := issueSnapshot{
		ID:        i.ID,
		Title:     i.Title,
		Body:      i.Body,
		Status:    i.Status,
		Author:    i.Author,
		Labels:    i.Labels,
		Assignee:  i.Assignee,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		Log:       i.log.Operations(),
	}
	return json.Marshal(snap)
}

// UnmarshalJSON deserializes an Issue and rebuilds its operation log
func (i *Issue) UnmarshalJSON(data []byte) error {
	var snap issueSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	i.ID = snap.ID
	i.Title = snap.Title
	i.Body = snap.Body
	i.Status = snap.Status
	i.Author = snap.Author
	i.Labels = snap.Labels
	if i.Labels == nil {
		i.Labels = make([]string, 0)
	}
	i.Assignee = snap.Assignee
	i.CreatedAt = snap.CreatedAt
	i.UpdatedAt = snap.UpdatedAt
	i.log = NewOperationLog()
	for _, op := range snap.Log {
		i.log.Add(op)
	}
	return nil
}

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
	// Merge Lamport clocks
	i.log.clock.Merge(other.log.clock)

	// Collect existing IDs to deduplicate
	existingIDs := make(map[string]bool)
	for _, op := range i.log.Operations() {
		existingIDs[op.ID] = true
	}

	// Add only new operations from other
	for _, op := range other.log.Operations() {
		if !existingIDs[op.ID] {
			i.log.ImportOperation(op)
		}
	}

	// Collect all operations and sort by Lamport counter, then timestamp
	allOps := make([]*Operation, len(i.log.Operations()))
	copy(allOps, i.log.Operations())
	sort.Slice(allOps, func(a, b int) bool {
		if allOps[a].Lamport != allOps[b].Lamport {
			return allOps[a].Lamport < allOps[b].Lamport
		}
		return allOps[a].Timestamp.Before(allOps[b].Timestamp)
	})

	// Reset state and re-apply
	i.Labels = make([]string, 0)
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
	b := make([]byte, 8)
	_, _ = crypto_rand.Read(b)
	return fmt.Sprintf("%d-%x", time.Now().UnixNano(), b)
}
