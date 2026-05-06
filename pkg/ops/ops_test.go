package ops

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOperationInvokeValidatesRequiredArgsAndReturnsResult(t *testing.T) {
	op := Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example",
		InputSchema: json.RawMessage(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`),
		ReadOnly:    true,
		Examples:    []Example{{Name: "sample", Args: json.RawMessage(`{"id":"abc"}`)}},
		SafetyNotes: []string{"Read-only."},
		Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			return &Result{Body: []byte(`{"id":"abc"}`), StatusCode: 200, ExitCode: 0}, nil
		},
	}

	result, err := op.Call(context.Background(), json.RawMessage(`{"id":"abc"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Body) != `{"id":"abc"}` || result.StatusCode != 200 || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Duration <= 0 {
		t.Fatalf("expected duration to be set")
	}

	_, err = op.Call(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("expected required-field error, got %v", err)
	}
}

func TestRegistryInvokeByToolAndOperation(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			return &Result{Body: []byte(`{"ok":true}`), StatusCode: 200, ExitCode: 0}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := reg.Invoke(context.Background(), "example", "get", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", result.Body)
	}
}

func TestOperationCallHandlesContextAndRetryableError(t *testing.T) {
	wantErr := errors.New("backend unavailable")
	op := Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly:    true,
		Retryable:   true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			return nil, wantErr
		},
	}
	result, err := op.Call(context.Background(), json.RawMessage(`{}`))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped operation error, got %v", err)
	}
	if result == nil || !result.Retry || result.ExitCode != 1 || result.Duration <= 0 {
		t.Fatalf("expected retryable failure result, got %#v", result)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = op.Call(ctx, json.RawMessage(`{}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if result == nil || !result.Retry || result.ExitCode != 124 {
		t.Fatalf("expected timeout-style result, got %#v", result)
	}
}

func TestOperationMetadataRequiresWriteAuthorization(t *testing.T) {
	op := Operation{Tool: "student", Name: "update", Description: "Update student", InputSchema: json.RawMessage(`{"type":"object"}`), ReadOnly: false}
	if err := op.ValidateMetadata(); err == nil || !strings.Contains(err.Error(), "authorization") {
		t.Fatalf("expected authorization metadata error, got %v", err)
	}
	op.Authorization = &Authorization{Required: true, Policy: "parent approval", AuditEvent: "student.updated"}
	op.SafetyNotes = []string{"Requires parent approval."}
	op.Invoke = func(ctx context.Context, args json.RawMessage) (*Result, error) {
		return &Result{Body: []byte(`{}`)}, nil
	}
	if err := op.ValidateMetadata(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestResultContainsNoPresentationFooter(t *testing.T) {
	result := &Result{Body: []byte(`{"ok":true}`), Duration: time.Millisecond}
	if strings.Contains(string(result.Body), "[exit:") {
		t.Fatalf("raw result must not contain presentation footer")
	}
}
