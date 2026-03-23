package schema

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoadSchema_Valid(t *testing.T) {
	s, err := LoadFile("../../testdata/simple.schema.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(s.Fields))
	}
}

func TestLoadSchema_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("not json")
	f.Close()

	_, err = LoadFile(f.Name())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadSchema_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/schema.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestTransformSimple(t *testing.T) {
	s, _ := LoadFile("../../testdata/simple.schema.json")
	input := map[string]any{"id": float64(1), "name": "alice", "slug": "alice-slug", "extra": "ignored"}
	result, err := Transform(s, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != float64(1) {
		t.Errorf("expected id=1, got %v", result["id"])
	}
	if result["name"] != "alice" {
		t.Errorf("expected name='alice', got %v", result["name"])
	}
	if _, ok := result["extra"]; ok {
		t.Error("expected 'extra' to be excluded from result")
	}
}

func TestTransformNested(t *testing.T) {
	s, _ := LoadFile("../../testdata/nested.schema.json")
	input := map[string]any{
		"slug":    "my-repo",
		"project": map[string]any{"key": "PROJ", "name": "My Project"},
	}
	result, err := Transform(s, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["project"] != "PROJ" {
		t.Errorf("expected project='PROJ', got %v", result["project"])
	}
	if result["repo"] != "my-repo" {
		t.Errorf("expected repo='my-repo', got %v", result["repo"])
	}
}

func TestTransformMissingField_Error(t *testing.T) {
	s := &Schema{Fields: []FieldMapping{{From: "nonexistent", To: "out"}}}
	input := map[string]any{"id": 1}
	_, err := Transform(s, input)
	if err == nil {
		t.Error("expected error for missing source field")
	}
}

func TestTransformList(t *testing.T) {
	raw, _ := os.ReadFile("../../testdata/bitbucket_repos.json")
	var doc map[string]any
	json.Unmarshal(raw, &doc)

	values := doc["values"].([]any)
	s, _ := LoadFile("../../testdata/nested.schema.json")

	results := make([]map[string]any, 0, len(values))
	for _, v := range values {
		r, err := Transform(s, v.(map[string]any))
		if err != nil {
			t.Fatalf("transform error: %v", err)
		}
		results = append(results, r)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if results[0]["project"] != "PROJ" {
		t.Errorf("expected project='PROJ', got %v", results[0]["project"])
	}
}
