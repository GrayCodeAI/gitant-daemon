package crdt

import (
	"encoding/json"
	"time"
)

// OperationType represents the type of CRDT operation
type OperationType string

const (
	OpCreate      OperationType = "create"
	OpSetTitle    OperationType = "set_title"
	OpSetBody     OperationType = "set_body"
	OpAddComment  OperationType = "add_comment"
	OpSetStatus   OperationType = "set_status"
	OpAddLabel    OperationType = "add_label"
	OpRemoveLabel OperationType = "remove_label"
	OpSetBranch   OperationType = "set_branch"
	OpSetAssignee OperationType = "set_assignee"
)

// Status represents the status of an issue or PR
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
	StatusMerged Status = "merged"
)

// Operation represents a CRDT operation
type Operation struct {
	ID        string                 `json:"id"`
	Type      OperationType          `json:"type"`
	Author    string                 `json:"author"` // DID
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Parent    string                 `json:"parent,omitempty"` // Parent operation ID
}

// LamportClock provides causal ordering
type LamportClock struct {
	counter uint64
}

// NewLamportClock creates a new Lamport clock
func NewLamportClock() *LamportClock {
	return &LamportClock{}
}

// Increment increments the clock
func (c *LamportClock) Increment() uint64 {
	c.counter++
	return c.counter
}

// Value returns the current clock value
func (c *LamportClock) Value() uint64 {
	return c.counter
}

// Merge merges with another clock value
func (c *LamportClock) Merge(other uint64) {
	if other > c.counter {
		c.counter = other
	}
}

// OperationLog stores CRDT operations
type OperationLog struct {
	operations []*Operation
	clock      *LamportClock
}

// NewOperationLog creates a new operation log
func NewOperationLog() *OperationLog {
	return &OperationLog{
		operations: make([]*Operation, 0),
		clock:      NewLamportClock(),
	}
}

// Add adds an operation to the log
func (l *OperationLog) Add(op *Operation) {
	op.Timestamp = time.Now()
	l.clock.Increment()
	l.operations = append(l.operations, op)
}

// Operations returns all operations
func (l *OperationLog) Operations() []*Operation {
	return l.operations
}

// MarshalJSON marshals the operation log to JSON
func (l *OperationLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.operations)
}

// UnmarshalJSON unmarshals the operation log from JSON
func (l *OperationLog) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.operations)
}
