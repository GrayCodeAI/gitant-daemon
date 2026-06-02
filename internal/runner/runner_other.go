//go:build !unix

package runner

import (
	"os/exec"
)

// applyProcAttr applies platform-specific process attributes for runner execution.
// On non-Unix systems, no special attributes are set.
func applyProcAttr(cmd *exec.Cmd) {
	// No-op on non-Unix platforms
}

// killProcessGroup attempts to kill the process.
// On non-Unix systems, we can only kill the main process.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
