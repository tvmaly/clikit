package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FieldMapping defines a single field extraction: From is a dot-path into the
// source JSON object, To is the output key name.
type FieldMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Schema describes how to transform a raw JSON object into a target shape.
type Schema struct {
	Fields []FieldMapping `json:"fields"`
}

// LoadFile reads and parses a schema JSON file.
func LoadFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema %q: %w", path, err)
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing schema %q: %w", path, err)
	}
	return &s, nil
}

// Transform applies s to src, returning a new map containing only the fields
// declared in s.Fields, extracted by dot-path from src.
func Transform(s *Schema, src map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(s.Fields))
	for _, f := range s.Fields {
		val, err := extractPath(src, f.From)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", f.From, err)
		}
		result[f.To] = val
	}
	return result, nil
}

// extractPath traverses a dot-delimited path into obj and returns the value.
func extractPath(obj map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = obj
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse path %q: not an object at segment %q", path, part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", part)
		}
		current = val
	}
	return current, nil
}
