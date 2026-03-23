package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON_Compact(t *testing.T) {
	data := map[string]any{"key": "value", "num": 42}
	var buf bytes.Buffer
	if err := FormatJSON(&buf, data, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"key"`) {
		t.Errorf("expected key in output, got %q", out)
	}
	// compact: no leading spaces
	if strings.Contains(out, "\n  ") {
		t.Errorf("expected compact JSON, got indented output")
	}
}

func TestFormatJSON_Pretty(t *testing.T) {
	data := map[string]any{"key": "value"}
	var buf bytes.Buffer
	if err := FormatJSON(&buf, data, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\n") {
		t.Errorf("expected pretty JSON with newlines, got %q", out)
	}
}

func TestFormatJSONL(t *testing.T) {
	items := []any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	}
	var buf bytes.Buffer
	if err := FormatJSONL(&buf, items); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line is not valid JSON: %q", line)
		}
	}
}

func TestFormatQuiet_StringField(t *testing.T) {
	data := map[string]any{"name": "alice", "age": float64(30)}
	var buf bytes.Buffer
	if err := FormatQuiet(&buf, data, "name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "alice" {
		t.Errorf("expected 'alice', got %q", buf.String())
	}
}

func TestFormatQuiet_MissingField(t *testing.T) {
	data := map[string]any{"name": "alice"}
	var buf bytes.Buffer
	err := FormatQuiet(&buf, data, "missing")
	if err == nil {
		t.Error("expected error for missing field")
	}
}

func TestFormatField_NestedPath(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{"name": "bob"},
	}
	var buf bytes.Buffer
	if err := FormatField(&buf, data, "user.name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "bob" {
		t.Errorf("expected 'bob', got %q", buf.String())
	}
}

func TestFormatTable(t *testing.T) {
	rows := []map[string]string{
		{"name": "alice", "role": "admin"},
		{"name": "bob", "role": "user"},
	}
	headers := []string{"name", "role"}
	var buf bytes.Buffer
	if err := FormatTable(&buf, rows, headers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Errorf("expected table with alice and bob, got %q", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header NAME in table output, got %q", out)
	}
}

func TestFormatTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := FormatTable(&buf, []map[string]string{}, []string{"name"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
