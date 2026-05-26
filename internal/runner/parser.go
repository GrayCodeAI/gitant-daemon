package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseWorkflow parses a workflow YAML file
func ParseWorkflow(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading workflow file: %w", err)
	}

	workflow := &Workflow{}
	if err := parseYAML(data, workflow); err != nil {
		return nil, fmt.Errorf("parsing workflow: %w", err)
	}

	return workflow, nil
}

// FindWorkflows finds all workflow files in a repository
func FindWorkflows(repoPath string) ([]*Workflow, error) {
	workflowsDir := filepath.Join(repoPath, ".gitant", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("reading workflows directory: %w", err)
	}

	var workflows []*Workflow
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}

		path := filepath.Join(workflowsDir, name)
		workflow, err := ParseWorkflow(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// Simple YAML parser (no external dependency)
func parseYAML(data []byte, v interface{}) error {
	workflow, ok := v.(*Workflow)
	if !ok {
		return fmt.Errorf("expected *Workflow")
	}

	lines := strings.Split(string(data), "\n")
	workflow.Jobs = make(map[string]Job)
	workflow.On = make(map[string][]string)

	var currentSection string
	var currentJob string
	var currentStep *Step

	for _, line := range lines {
		line = strings.TrimRight(line, " ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		content := strings.TrimSpace(line)

		if indent == 0 {
			if strings.HasPrefix(content, "name:") {
				workflow.Name = strings.TrimSpace(strings.TrimPrefix(content, "name:"))
				workflow.Name = strings.Trim(workflow.Name, "'\"")
			} else if strings.HasPrefix(content, "on:") {
				currentSection = "on"
			} else if strings.HasPrefix(content, "jobs:") {
				currentSection = "jobs"
			}
		} else if currentSection == "on" {
			if indent == 2 {
				trigger := strings.TrimRight(content, ":")
				workflow.On[trigger] = []string{}
			}
		} else if currentSection == "jobs" {
			if indent == 2 {
				currentJob = strings.TrimRight(content, ":")
				workflow.Jobs[currentJob] = Job{
					RunsOn: "ubuntu-latest",
					Steps:  []Step{},
				}
			} else if indent == 4 && currentJob != "" {
				job := workflow.Jobs[currentJob]
				if strings.HasPrefix(content, "runs-on:") {
					job.RunsOn = strings.TrimSpace(strings.TrimPrefix(content, "runs-on:"))
					job.RunsOn = strings.Trim(job.RunsOn, "'\"")
					workflow.Jobs[currentJob] = job
				} else if strings.HasPrefix(content, "name:") {
					job.Name = strings.TrimSpace(strings.TrimPrefix(content, "name:"))
					job.Name = strings.Trim(job.Name, "'\"")
					workflow.Jobs[currentJob] = job
				}
			} else if indent == 6 && currentJob != "" {
				if strings.HasPrefix(content, "- ") {
					content = strings.TrimPrefix(content, "- ")
					if strings.HasPrefix(content, "name:") {
						step := Step{}
						step.Name = strings.TrimSpace(strings.TrimPrefix(content, "name:"))
						step.Name = strings.Trim(step.Name, "'\"")
						currentStep = &step
						job := workflow.Jobs[currentJob]
						job.Steps = append(job.Steps, step)
						workflow.Jobs[currentJob] = job
					} else if strings.HasPrefix(content, "uses:") {
						if currentStep != nil {
							currentStep.Uses = strings.TrimSpace(strings.TrimPrefix(content, "uses:"))
							currentStep.Uses = strings.Trim(currentStep.Uses, "'\"")
						}
					} else if strings.HasPrefix(content, "run:") {
						if currentStep != nil {
							currentStep.Run = strings.TrimSpace(strings.TrimPrefix(content, "run:"))
							currentStep.Run = strings.Trim(currentStep.Run, "'\"")
						}
					}
				}
			}
		}
	}

	return nil
}
