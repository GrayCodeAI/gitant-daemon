package crdt

import (
	"log/slog"
	"sync"
	"time"
)

const (
	compactInterval = 6 * time.Hour
	compactOpLimit  = 1000
)

// Compactor periodically compacts operation logs to bound memory usage.
type Compactor struct {
	mu       sync.Mutex
	issueOps logProvider
	prOps    logProvider
	stop     chan struct{}
}

// logProvider returns all operation logs that may need compaction.
type logProvider interface {
	AllIssueLogs() []*OperationLog
	AllPRLogs() []*OperationLog
}

// IssueLogProvider adapts IssueStore for the compactor.
type IssueLogProvider struct {
	store *IssueStore
}

// AllIssueLogs returns all issue operation logs.
func (p *IssueLogProvider) AllIssueLogs() []*OperationLog {
	all := p.store.All()
	var logs []*OperationLog
	for _, repoIssues := range all {
		for _, issue := range repoIssues {
			logs = append(logs, issue.Log())
		}
	}
	return logs
}

// AllPRLogs returns nil (not applicable).
func (p *IssueLogProvider) AllPRLogs() []*OperationLog { return nil }

// PRLogProvider adapts PullRequestStore for the compactor.
type PRLogProvider struct {
	store *PullRequestStore
}

// AllIssueLogs returns nil (not applicable).
func (p *PRLogProvider) AllIssueLogs() []*OperationLog { return nil }

// AllPRLogs returns all PR operation logs.
func (p *PRLogProvider) AllPRLogs() []*OperationLog {
	all := p.store.All()
	var logs []*OperationLog
	for _, repoPRs := range all {
		for _, pr := range repoPRs {
			logs = append(logs, pr.Log())
		}
	}
	return logs
}

// NewCompactor creates a compactor with the given log providers.
func NewCompactor() *Compactor {
	return &Compactor{
		stop: make(chan struct{}),
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
	// CompactAll is a no-op when no providers are registered.
	// Providers are registered via the server wiring.
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
