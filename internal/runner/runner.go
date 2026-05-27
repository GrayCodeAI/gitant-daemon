package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// maxScriptSize is the maximum allowed size of a workflow script in bytes.
const maxScriptSize = 64 * 1024 // 64 KB

// stepTimeout is the maximum time a single workflow step can run.
const stepTimeout = 10 * time.Minute

// dangerousPatterns matches shell constructs that can be used for command injection
// or privilege escalation. These are blocked in workflow scripts.
var dangerousPatterns = regexp.MustCompile(`(\$\(|` + "`" + `|eval\s|exec\s|source\s|\.\s|nc\s|ncat\s|curl\s.*\|\s*sh|wget\s.*\|\s*sh|python\s*-c|perl\s*-e|ruby\s*-e|node\s*-e|dd\s+if=|mkfs|fdisk|mount\s|umount\s|chmod\s+777|chown\s+root)`)

// WorkflowStatus represents the status of a workflow run
type WorkflowStatus string

const (
	StatusPending WorkflowStatus = "pending"
	StatusRunning WorkflowStatus = "running"
	StatusSuccess WorkflowStatus = "success"
	StatusFailed  WorkflowStatus = "failed"
	StatusCanceled WorkflowStatus = "canceled"
)

// Workflow represents a workflow definition
type Workflow struct {
	Name string              `yaml:"name"`
	On   map[string][]string `yaml:"on"`
	Jobs map[string]Job      `yaml:"jobs"`
}

// Job represents a workflow job
type Job struct {
	Name   string `yaml:"name"`
	RunsOn string `yaml:"runs-on"`
	Steps  []Step `yaml:"steps"`
}

// Step represents a workflow step
type Step struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

// Run represents a running workflow
type Run struct {
	ID        string
	RepoID    string
	CommitSHA string
	Branch    string
	Status    WorkflowStatus
	StartedAt time.Time
	Logs      []string
	mu        sync.Mutex
}

// generateID creates a cryptographically random ID with the given prefix.
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// validateScript checks a workflow script for dangerous patterns.
func validateScript(script string) error {
	if len(script) > maxScriptSize {
		return fmt.Errorf("script exceeds maximum size of %d bytes", maxScriptSize)
	}
	if dangerousPatterns.MatchString(script) {
		return fmt.Errorf("script contains potentially dangerous commands")
	}
	return nil
}

// Runner manages workflow execution
type Runner struct {
	runsDir string
	runs    map[string]*Run
	mu      sync.RWMutex
}

// NewRunner creates a new workflow runner
func NewRunner(dataDir string) *Runner {
	return &Runner{
		runsDir: filepath.Join(dataDir, "workflow-runs"),
		runs:    make(map[string]*Run),
	}
}

// Execute starts a workflow run
func (r *Runner) Execute(ctx context.Context, repoPath string, workflow Workflow, commitSHA, branch string) (*Run, error) {
	run := &Run{
		ID:        generateID("run"),
		CommitSHA: commitSHA,
		Branch:    branch,
		Status:    StatusPending,
		StartedAt: time.Now(),
		Logs:      make([]string, 0),
	}

	r.mu.Lock()
	r.runs[run.ID] = run
	r.mu.Unlock()

	go r.executeWorkflow(ctx, repoPath, workflow, run)

	return run, nil
}

// GetRun gets a workflow run by ID
func (r *Runner) GetRun(id string) (*Run, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	return run, ok
}

// ListRuns lists all workflow runs
func (r *Runner) ListRuns() []*Run {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runs := make([]*Run, 0, len(r.runs))
	for _, run := range r.runs {
		runs = append(runs, run)
	}
	return runs
}

func (r *Runner) executeWorkflow(ctx context.Context, repoPath string, workflow Workflow, run *Run) {
	run.mu.Lock()
	run.Status = StatusRunning
	run.mu.Unlock()

	for jobName, job := range workflow.Jobs {
		run.addLog(fmt.Sprintf("Starting job: %s", jobName))

		for i, step := range job.Steps {
			select {
			case <-ctx.Done():
				run.mu.Lock()
				run.Status = StatusCanceled
				run.mu.Unlock()
				return
			default:
			}

			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("Step %d", i+1)
			}

			run.addLog(fmt.Sprintf("Running: %s", stepName))

			if step.Run != "" {
				if err := r.executeStep(ctx, repoPath, step.Run, run); err != nil {
					run.addLog(fmt.Sprintf("Failed: %s - %v", stepName, err))
					run.mu.Lock()
					run.Status = StatusFailed
					run.mu.Unlock()
					return
				}
			}
		}
	}

	run.mu.Lock()
	run.Status = StatusSuccess
	run.mu.Unlock()

	run.addLog("Workflow completed successfully")
}

func (r *Runner) executeStep(ctx context.Context, repoPath, script string, run *Run) error {
	if err := validateScript(script); err != nil {
		return fmt.Errorf("script validation failed: %w", err)
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("step-%s.sh", generateID("step")))
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nset -e\n"+script), 0700); err != nil {
		return fmt.Errorf("writing script: %w", err)
	}
	defer os.Remove(scriptPath)

	stepCtx, cancel := context.WithTimeout(ctx, stepTimeout)
	defer cancel()

	cmd := exec.CommandContext(stepCtx, "bash", scriptPath)
	cmd.Dir = repoPath
	// Sanitize environment: only pass essential vars, strip sensitive ones
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"LANG=en_US.UTF-8",
		"CI=true",
		"GITANT=true",
	}

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		run.addLog(string(output))
	}

	if err != nil {
		if stepCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("step timed out after %s", stepTimeout)
		}
		return fmt.Errorf("step failed: %w", err)
	}

	return nil
}

func (r *Run) addLog(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Logs = append(r.Logs, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), msg))
	slog.Info("workflow log", "run_id", r.ID, "message", msg)
}
