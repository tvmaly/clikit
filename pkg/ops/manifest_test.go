package ops

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateManifestAgainstOperations(t *testing.T) {
	manifest := []byte(`{
	  "examples": [
	    {"command": "toolkit example get --id abc", "safety": "Read-only example."}
	  ],
	  "permissions": ["toolkit example get"],
	  "safety_notes": ["Read-only only."]
	}`)
	err := ValidateManifest(manifest, []Operation{{
		Tool:        "example",
		Name:        "get",
		Description: "Get",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			return &Result{}, nil
		},
	}})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateManifestDetectsDrift(t *testing.T) {
	manifest := []byte(`{"examples":[{"command":"toolkit example missing","safety":"Read-only."}],"permissions":["toolkit example missing"],"safety_notes":["Read-only."]}`)
	err := ValidateManifest(manifest, []Operation{{Tool: "example", Name: "get", Description: "Get", InputSchema: json.RawMessage(`{"type":"object"}`), ReadOnly: true, Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
		return &Result{}, nil
	}}})
	if err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Fatalf("expected drift error, got %v", err)
	}
}

func TestValidateManifestRequiresWriteSafety(t *testing.T) {
	manifest := []byte(`{"examples":[{"command":"toolkit student update","safety":"Write."}],"permissions":["toolkit student update"],"safety_notes":[]}`)
	err := ValidateManifest(manifest, []Operation{{
		Tool:          "student",
		Name:          "update",
		Description:   "Update",
		InputSchema:   json.RawMessage(`{"type":"object"}`),
		ReadOnly:      false,
		Authorization: &Authorization{Required: true, Policy: "parent approval", AuditEvent: "student.updated"},
		SafetyNotes:   []string{"Requires approval."},
		Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			return &Result{}, nil
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "write-capable") {
		t.Fatalf("expected write safety error, got %v", err)
	}
}
