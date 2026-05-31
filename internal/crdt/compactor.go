package crdt

import (
	"log/slog"
	"time"
)

const (
	compactInterval = 6 * time.Hour
	compactOpLimit  = 1000
)

// LogProvider supplies operation logs for compaction.
type LogProvider interface {
	Logs() []*OperationLog
}

// Compactor periodically compacts operation logs to bound memory usage.
type Compactor struct {
	providers []LogProvider
	stop      chan struct{}
}

// IssueLogProvider adapts IssueStore for the compactor.
type IssueLogProvider struct {
	Store *IssueStore
}

// Logs returns all issue operation logs.
func (p *IssueLogProvider) Logs() []*OperationLog {
	all := p.Store.All()
	var logs []*OperationLog
	for _, repoIssues := range all {
		for _, issue := range repoIssues {
			logs = append(logs, issue.Log())
		}
	}
	return logs
}

// PRLogProvider adapts PullRequestStore for the compactor.
type PRLogProvider struct {
	Store *PullRequestStore
}

// Logs returns all PR operation logs.
func (p *PRLogProvider) Logs() []*OperationLog {
	all := p.Store.All()
	var logs []*OperationLog
	for _, repoPRs := range all {
		for _, pr := range repoPRs {
			logs = append(logs, pr.Log())
		}
	}
	return logs
}

// TaskLogProvider adapts TaskStore for the compactor.
type TaskLogProvider struct {
	Store *TaskStore
}

// Logs returns all task operation logs.
func (p *TaskLogProvider) Logs() []*OperationLog {
	// TaskStore does not expose an All() method yet; return nil for now.
	return nil
}

// LabelLogProvider adapts LabelStore for the compactor.
type LabelLogProvider struct {
	Store *LabelStore
}

// Logs returns all label operation logs.
func (p *LabelLogProvider) Logs() []*OperationLog {
	return nil
}

// ReleaseLogProvider adapts ReleaseStore for the compactor.
type ReleaseLogProvider struct {
	Store *ReleaseStore
}

// Logs returns all release operation logs.
func (p *ReleaseLogProvider) Logs() []*OperationLog {
	return nil
}

// NewCompactor creates a compactor with the given log providers.
func NewCompactor(providers ...LogProvider) *Compactor {
	return &Compactor{
		providers: providers,
		stop:      make(chan struct{}),
	}
}

// Start begins the periodic compaction loop.
func (c *Compactor) Start(interval time.Duration) {
	if interval <= 0 {
		interval = compactInterval
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				c.CompactAll()
			}
		}
	}()
}

// Stop terminates the compaction loop.
func (c *Compactor) Stop() {
	close(c.stop)
}

// CompactAll compacts all operation logs from all registered providers.
func (c *Compactor) CompactAll() {
	totalCompacted := 0
	for _, p := range c.providers {
		for _, log := range p.Logs() {
			totalCompacted += CompactLog(log)
		}
	}
	if totalCompacted > 0 {
		slog.Info("CRDT compaction complete", "operations_removed", totalCompacted)
	}
}

// CompactLog compacts a single operation log if it exceeds the threshold.
// It preserves the Lamport clock value and keeps only the latest operation per type+entity.
func CompactLog(log *OperationLog) int {
	ops := log.Operations()
	if len(ops) <= compactOpLimit {
		return 0
	}

	// Keep the last Lamport value
	maxLamport := uint64(0)
	for _, op := range ops {
		if op.Lamport > maxLamport {
			maxLamport = op.Lamport
		}
	}

	// Deduplicate: keep the latest op per (type, entity key)
	type opKey struct {
		typ  OperationType
		key  string
	}
	latest := make(map[opKey]*Operation)
	order := make([]opKey, 0)
	for _, op := range ops {
		var key string
		switch op.Type {
		case OpSetTitle, OpSetBody, OpSetStatus, OpSetAssignee, OpSetColor, OpTombstone:
			key = string(op.Type)
		case OpAddComment:
			key = op.ID // each comment is unique
		case OpAddLabel, OpRemoveLabel:
			if l, ok := op.Data["label"].(string); ok {
				key = l
			}
		case OpSetBranch:
			key = "branch"
		case OpClaimTask, OpCompleteTask, OpFailTask, OpSetResult:
			key = string(op.Type)
		default:
			key = op.ID
		}
		k := opKey{typ: op.Type, key: key}
		if _, exists := latest[k]; !exists {
			order = append(order, k)
		}
		latest[k] = op
	}

	// Build compacted list preserving order
	compacted := make([]*Operation, 0, len(order))
	for _, k := range order {
		compacted = append(compacted, latest[k])
	}

	// Replace operations in the log
	log.operations = compacted
	log.clock.counter = maxLamport

	slog.Debug("compacted operation log", "before", len(ops), "after", len(compacted))
	return len(ops) - len(compacted)
}
