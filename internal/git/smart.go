// Package git implements git smart-HTTP protocol helpers.
package git

import (
	"fmt"
	"strings"
)

// PktLine formats a string as a git pkt-line: <4-hex-length><data>
func PktLine(data string) string {
	length := len(data) + 4 // +4 for the length prefix itself
	return fmt.Sprintf("%04x%s", length, data)
}

// FlushPacket returns the flush packet "0000"
func FlushPacket() string {
	return "0000"
}

// PktLinef is a convenience for PktLine with formatting
func PktLinef(format string, args ...interface{}) string {
	return PktLine(fmt.Sprintf(format, args...))
}

// ServiceRefResponse generates the info/refs response for a given service
func ServiceRefResponse(service string, refs []RefLine) string {
	var sb strings.Builder

	// Header
	sb.WriteString(PktLinef("# service=%s\n", service))
	sb.WriteString(FlushPacket())

	// Refs
	if len(refs) == 0 {
		sb.WriteString(PktLine("0000000000000000000000000000000000000000 capabilities^{}\000\n"))
	} else {
		for i, ref := range refs {
			if i == 0 {
				// First line includes capabilities
				sb.WriteString(PktLinef("%s %s\000%s\n", ref.Hash, ref.Name, "side-band-64k thin-pack"))
			} else {
				sb.WriteString(PktLinef("%s %s\n", ref.Hash, ref.Name))
			}
		}
	}
	sb.WriteString(FlushPacket())

	return sb.String()
}

// RefLine represents a ref in an info/refs response
type RefLine struct {
	Hash string
	Name string
}

// ParseWantLines parses "want <hash>" lines from a git-upload-pack request
func ParseWantLines(data []string) []string {
	var hashes []string
	for _, line := range data {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "want ") {
			hashes = append(hashes, strings.TrimPrefix(line, "want "))
		}
	}
	return hashes
}

// ParseHaveLines parses "have <hash>" lines
func ParseHaveLines(data []string) []string {
	var hashes []string
	for _, line := range data {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "have ") {
			hashes = append(hashes, strings.TrimPrefix(line, "have "))
		}
	}
	return hashes
}

// ParsePushRefUpdates parses ref update lines from git-receive-pack
// Format: "<old-hash> <new-hash> <refname>"
func ParsePushRefUpdates(data []string) []PushRefUpdate {
	var updates []PushRefUpdate
	for _, line := range data {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, " ", 3)
		if len(parts) == 3 {
			updates = append(updates, PushRefUpdate{
				OldHash: parts[0],
				NewHash: parts[1],
				RefName: parts[2],
			})
		}
	}
	return updates
}

// PushRefUpdate represents a ref update in a push
type PushRefUpdate struct {
	OldHash string
	NewHash string
	RefName string
}
