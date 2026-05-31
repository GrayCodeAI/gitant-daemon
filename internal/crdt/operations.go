package crdt

import (
	"encoding/json"
	"sort"
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

	// Label operations
	OpSetColor  OperationType = "set_color"
	OpDeleteLabel OperationType = "delete_label"

	// Task operations
	OpClaimTask    OperationType = "claim_task"
	OpCompleteTask OperationType = "complete_task"
	OpFailTask     OperationType = "fail_task"
	OpSetResult    OperationType = "set_result"

	// Tombstone
	OpTombstone OperationType = "tombstone"
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
	Lamport   uint64                 `json:"lamport"`
	NodeID    string                 `json:"node_id,omitempty"` // Tiebreaker for concurrent ops
	Data      map[string]interface{} `json:"data"`
}

// nodeID is set once at startup and stamped on every local operation.
// It provides a deterministic tiebreaker when two operations have the same
// Lamport timestamp (concurrent edits from different peers).
var nodeID string

// SetNodeID sets the node identifier used for CRDT tiebreaking.
// Call this once at startup with the server's DID or a unique identifier.
func SetNodeID(id string) {
	nodeID = id
}

// GetNodeID returns the current node identifier.
func GetNodeID() string {
	return nodeID
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

// Merge merges with another clock, taking the max
func (c *LamportClock) Merge(other *LamportClock) {
	if other.counter > c.counter {
		c.counter = other.counter
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

// Add adds an operation to the log, stamping the current node ID if not set.
func (l *OperationLog) Add(op *Operation) {
	if op.Timestamp.IsZero() {
		op.Timestamp = time.Now()
	}
	if op.NodeID == "" && nodeID != "" {
		op.NodeID = nodeID
	}
	op.Lamport = l.clock.Increment()
	l.operations = append(l.operations, op)
}

// Observe updates the clock if n is greater than the current value.
func (c *LamportClock) Observe(n uint64) {
	if n > c.counter {
		c.counter = n
	}
}

// ImportOperation appends an operation without reassigning its Lamport timestamp.
func (l *OperationLog) ImportOperation(op *Operation) {
	for _, existing := range l.operations {
		if existing.ID == op.ID {
			return
		}
	}
	l.operations = append(l.operations, op)
	l.clock.Observe(op.Lamport)
}

// Operations returns all operations
func (l *OperationLog) Operations() []*Operation {
	return l.operations
}

// CompareOps provides a deterministic sort order for operations:
// 1. Lamport timestamp (causal order)
// 2. NodeID (tiebreaker for concurrent ops from different peers)
// 3. Wall-clock timestamp (final tiebreaker)
func CompareOps(a, b *Operation) bool {
	if a.Lamport != b.Lamport {
		return a.Lamport < b.Lamport
	}
	if a.NodeID != b.NodeID {
		return a.NodeID < b.NodeID
	}
	return a.Timestamp.Before(b.Timestamp)
}

// SortOps sorts operations in place using CompareOps.
func SortOps(ops []*Operation) {
	sort.Slice(ops, func(i, j int) bool {
		return CompareOps(ops[i], ops[j])
	})
}

// MarshalJSON marshals the operation log to JSON
func (l *OperationLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.operations)
}

// UnmarshalJSON unmarshals the operation log from JSON
func (l *OperationLog) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.operations)
}
