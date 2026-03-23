package exec

import (
	"context"
	"strings"
	"testing"
)

func TestRun_Success(t *testing.T) {
	result, err := Run(context.Background(), "echo", []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if !strings.Contains(string(result.Stdout), "hello") {
		t.Errorf("expected 'hello' in stdout, got %q", result.Stdout)
	}
	if len(result.Stderr) != 0 {
		t.Errorf("expected empty stderr, got %q", result.Stderr)
	}
}

func TestRun_ExitCode(t *testing.T) {
	result, err := Run(context.Background(), "sh", []string{"-c", "exit 2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 2 {
		t.Errorf("expected exit 2, got %d", result.ExitCode)
	}
}

func TestRun_StderrCapture(t *testing.T) {
	result, err := Run(context.Background(), "sh", []string{"-c", "echo err >&2; exit 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit 1, got %d", result.ExitCode)
	}
	if !strings.Contains(string(result.Stderr), "err") {
		t.Errorf("expected 'err' in stderr, got %q", result.Stderr)
	}
}

func TestRun_NotFound(t *testing.T) {
	_, err := Run(context.Background(), "this-cmd-does-not-exist-xyz", nil)
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestRun_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Run(ctx, "sleep", []string{"10"})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestRun_StdoutAndStderr(t *testing.T) {
	result, err := Run(context.Background(), "sh", []string{"-c", "echo out; echo err >&2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result.Stdout), "out") {
		t.Errorf("expected 'out' in stdout, got %q", result.Stdout)
	}
	if !strings.Contains(string(result.Stderr), "err") {
		t.Errorf("expected 'err' in stderr, got %q", result.Stderr)
	}
}
