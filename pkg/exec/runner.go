package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Result holds the output of a completed command.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Run executes name with args, captures stdout and stderr, and returns a Result.
// It does not return an error for non-zero exit codes — those are reflected in
// Result.ExitCode. It does return an error if the command cannot be started or
// the context is cancelled before the command starts.
func Run(ctx context.Context, name string, args []string) (*Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if command was not found.
		if _, ok := err.(*exec.Error); ok {
			return nil, fmt.Errorf("command not found: %w", err)
		}
		// Context cancellation.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Non-zero exit — reflect in Result.
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &Result{
				Stdout:   stdout.Bytes(),
				Stderr:   stderr.Bytes(),
				ExitCode: exitErr.ExitCode(),
			}, nil
		}
		return nil, fmt.Errorf("running command: %w", err)
	}

	return &Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}, nil
}
