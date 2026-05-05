package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresQueueRootAndHandlerConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "--queue-root and --handler-config are required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsMalformedHandlerConfig(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "handlers.json")
	if err := os.WriteFile(config, []byte(`not-json`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--queue-root", dir, "--handler-config", config, "--once"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "loading handler config") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsInvalidDurationFlag(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "handlers.json")
	if err := os.WriteFile(config, []byte(`{"handlers":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--queue-root", dir, "--handler-config", config, "--poll-interval", "not-a-duration", "--once"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid value") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
