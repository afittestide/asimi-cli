package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// HostShellRunner runs commands directly on the host machine without using containers
type HostShellRunner struct{}

// NewHostShellRunner creates a new host shell runner
func NewHostShellRunner() *HostShellRunner {
	return &HostShellRunner{}
}

// Run executes a command directly on the host machine
func (h *HostShellRunner) Run(ctx context.Context, params RunInShellInput) (RunInShellOutput, error) {
	var output RunInShellOutput

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", params.Command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", params.Command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Combine stdout and stderr
	output.Output = stdout.String() + stderr.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			output.ExitCode = fmt.Sprintf("%d", exitErr.ExitCode())
		} else {
			output.ExitCode = "-1"
		}
	} else {
		if cmd.ProcessState != nil {
			output.ExitCode = fmt.Sprintf("%d", cmd.ProcessState.ExitCode())
		} else {
			output.ExitCode = "0"
		}
	}

	return output, nil
}

// Restart is a no-op for host shell runner since it doesn't maintain state
func (h *HostShellRunner) Restart(ctx context.Context) error {
	// Host shell runner doesn't maintain state, nothing to restart
	return nil
}

// Close is a no-op for host shell runner
func (h *HostShellRunner) Close(ctx context.Context) error {
	// Nothing to clean up for host shell runner
	return nil
}
