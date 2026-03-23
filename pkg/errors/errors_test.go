package errors

import (
	"encoding/json"
	"testing"
)

func TestCLIError_Error(t *testing.T) {
	e := &CLIError{Message: "not found", ExitCode: 1}
	if e.Error() != "not found" {
		t.Errorf("expected 'not found', got %q", e.Error())
	}
}

func TestCLIError_MarshalJSON(t *testing.T) {
	e := &CLIError{
		Message:    "resource not found",
		Suggestion: "run toolkit --list to see available tools",
		Available:  []string{"tool-a", "tool-b"},
		Retry:      false,
		ExitCode:   1,
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if m["error"] != "resource not found" {
		t.Errorf("expected error field 'resource not found', got %v", m["error"])
	}
	if m["suggestion"] != "run toolkit --list to see available tools" {
		t.Errorf("unexpected suggestion: %v", m["suggestion"])
	}
	avail, ok := m["available"].([]any)
	if !ok || len(avail) != 2 {
		t.Errorf("expected 2 available items, got %v", m["available"])
	}
	if m["retry"] != false {
		t.Errorf("expected retry=false, got %v", m["retry"])
	}
}

func TestCLIError_UnmarshalJSON(t *testing.T) {
	raw := `{"error":"timeout","suggestion":"retry","available":[],"retry":true,"exit_code":2}`
	var e CLIError
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if e.Message != "timeout" {
		t.Errorf("expected message 'timeout', got %q", e.Message)
	}
	if !e.Retry {
		t.Error("expected retry=true")
	}
	if e.ExitCode != 2 {
		t.Errorf("expected exit_code 2, got %d", e.ExitCode)
	}
}

func TestNew(t *testing.T) {
	e := New("something failed", 1)
	if e.Message != "something failed" {
		t.Errorf("unexpected message: %q", e.Message)
	}
	if e.ExitCode != 1 {
		t.Errorf("expected exit_code 1, got %d", e.ExitCode)
	}
}

func TestNewWithSuggestion(t *testing.T) {
	e := NewWithSuggestion("bad input", "try --help", 1)
	if e.Suggestion != "try --help" {
		t.Errorf("expected suggestion 'try --help', got %q", e.Suggestion)
	}
}

func TestNewRetryable(t *testing.T) {
	e := NewRetryable("service unavailable")
	if !e.Retry {
		t.Error("expected retry=true")
	}
	if e.ExitCode != 1 {
		t.Errorf("expected exit_code 1, got %d", e.ExitCode)
	}
}

func TestCLIError_WriteJSON(t *testing.T) {
	var buf []byte
	e := &CLIError{Message: "err", ExitCode: 1}
	data, err := e.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	buf = data
	if !json.Valid(buf) {
		t.Errorf("output is not valid JSON: %q", buf)
	}
}
