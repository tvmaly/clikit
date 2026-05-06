package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunOutputWithSchema(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"output", "--schema", filepath.Join("..", "..", "testdata", "simple.schema.json")}, strings.NewReader(`{"id":7,"name":"repo","slug":"repo-slug"}`), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid output JSON: %v", err)
	}
	if got["name"] != "repo" || got["slug"] != "repo-slug" {
		t.Fatalf("unexpected transformed output: %v", got)
	}
}

func TestRunOutputInvalidJSONWritesStructuredError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"output"}, strings.NewReader(`not-json`), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"error"`) || !strings.Contains(stdout.String(), `"suggestion"`) {
		t.Fatalf("expected structured error on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunOutputMissingSchemaWritesStructuredError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"output", "--schema", "missing.schema.json"}, strings.NewReader(`{"ok":true}`), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"error"`) || !strings.Contains(stdout.String(), "loading schema") {
		t.Fatalf("expected schema error on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunOutputUnsupportedFormatWritesStructuredError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"output", "--output", "xml"}, strings.NewReader(`{"ok":true}`), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "unsupported output format") {
		t.Fatalf("expected unsupported format error, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunErrorWritesStructuredErrorAndExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"error", "--message", "repo not found", "--suggestion", "list repos", "--retry", "--exit-code", "7"}, strings.NewReader(""), &stdout, &stderr)
	if code != 7 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"error":"repo not found"`) || !strings.Contains(stdout.String(), `"retry":true`) {
		t.Fatalf("expected structured error on stdout, got %q", stdout.String())
	}
}

func TestRunErrorMissingMessageWritesStructuredError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"error"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"error"`) || !strings.Contains(stdout.String(), "--message is required") {
		t.Fatalf("expected structured missing-message error on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunErrorAvailableValues(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"error", "--message", "unknown project", "--available", "PLAT", "--available", "DATA"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"available":["PLAT","DATA"]`) {
		t.Fatalf("expected available values in structured error, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr for structured error, got %q", stderr.String())
	}
}
