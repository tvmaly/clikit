package nanogoops

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tvmaly/clikit/pkg/ops"
)

func TestAdaptOperationToNanogoCompatibleTool(t *testing.T) {
	tool := Adapt(ops.Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example",
		InputSchema: json.RawMessage(`{"type":"object","required":["id"]}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
			return &ops.Result{Body: []byte(`{"ok":true}`), StatusCode: 200}, nil
		},
	})
	if tool.Name() != "example_get" {
		t.Fatalf("unexpected name: %q", tool.Name())
	}
	if string(tool.Schema()) != `{"type":"object","required":["id"]}` {
		t.Fatalf("unexpected schema: %s", tool.Schema())
	}
	result, err := tool.Call(context.Background(), json.RawMessage(`{"id":"abc"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `{"ok":true}` {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestAdaptOperationPreservesStructuredFailureShape(t *testing.T) {
	tool := Adapt(ops.Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example",
		InputSchema: json.RawMessage(`{"type":"object","required":["id"]}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
			return &ops.Result{Body: []byte(`{"error":"missing"}`), ExitCode: 1, Retry: false}, nil
		},
	})
	result, err := tool.Call(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("expected validation error, got result=%q err=%v", result, err)
	}
	if !strings.Contains(result, `"retry":false`) || strings.Contains(result, "[exit:") {
		t.Fatalf("expected structured error without footer, got %q", result)
	}
}
