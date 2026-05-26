package changelog

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Entry represents a changelog entry
type Entry struct {
	Version   string    `json:"version"`
	Date      time.Time `json:"date"`
	Changes   []Change  `json:"changes"`
	CommitHash string   `json:"commit_hash,omitempty"`
}

// Change represents a single change
type Change struct {
	Type        string `json:"type"` // "added", "changed", "deprecated", "removed", "fixed", "security"
	Description string `json:"description"`
	PR          string `json:"pr,omitempty"`
	Issue       string `json:"issue,omitempty"`
}

// Generator generates changelogs
type Generator struct {
	entries []Entry
}

// NewGenerator creates a new changelog generator
func NewGenerator() *Generator {
	return &Generator{
		entries: []Entry{},
	}
}

// Add adds an entry
func (g *Generator) Add(entry Entry) {
	g.entries = append(g.entries, entry)
}

// Generate generates the changelog
func (g *Generator) Generate() string {
	sort.Slice(g.entries, func(i, j int) bool {
		return g.entries[i].Date.After(g.entries[j].Date)
	})

	var sb strings.Builder
	sb.WriteString("# Changelog\n\n")

	for _, entry := range g.entries {
		sb.WriteString(fmt.Sprintf("## [%s] - %s\n\n", entry.Version, entry.Date.Format("2006-01-02")))

		changesByType := make(map[string][]Change)
		for _, change := range entry.Changes {
			changesByType[change.Type] = append(changesByType[change.Type], change)
		}

		typeOrder := []string{"added", "changed", "deprecated", "removed", "fixed", "security"}
		for _, t := range typeOrder {
			changes, ok := changesByType[t]
			if !ok {
				continue
			}

			sb.WriteString(fmt.Sprintf("### %s%s\n\n", strings.ToUpper(t[:1]), t[1:]))
			for _, change := range changes {
				line := fmt.Sprintf("- %s", change.Description)
				if change.PR != "" {
					line += fmt.Sprintf(" ([#%s])", change.PR)
				}
				if change.Issue != "" {
					line += fmt.Sprintf(" (fixes #%s)", change.Issue)
				}
				sb.WriteString(line + "\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GenerateFromCommits generates changelog from commit messages
func GenerateFromCommits(commits []struct {
	Hash    string
	Message string
	Date    time.Time
	Author  string
}) []Change {
	var changes []Change

	for _, commit := range commits {
		msg := commit.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}

		change := Change{Description: msg}

		if strings.HasPrefix(msg, "feat:") || strings.HasPrefix(msg, "feat(") {
			change.Type = "added"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "fix:") || strings.HasPrefix(msg, "fix(") {
			change.Type = "fixed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "docs:") {
			change.Type = "changed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "refactor:") {
			change.Type = "changed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "perf:") {
			change.Type = "changed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "test:") {
			change.Type = "changed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else if strings.HasPrefix(msg, "chore:") {
			change.Type = "changed"
			change.Description = strings.TrimSpace(strings.SplitN(msg, ":", 2)[1])
		} else {
			change.Type = "changed"
		}

		changes = append(changes, change)
	}

	return changes
}
