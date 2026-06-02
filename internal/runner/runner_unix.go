//go:build unix

package runner

import (
	"os/exec"
	"syscall"
)

// applyProcAttr applies platform-specific process attributes for runner execution.
// On Unix systems, this sets the process group ID so we can kill the entire process tree.
func applyProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for cleanup
	}
}

// killProcessGroup kills the entire process group associated with the command.
// This ensures all child processes are terminated on timeout.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil && cmd.Process.Pid > 0 {
		// Kill the entire process group (negative PID)
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
