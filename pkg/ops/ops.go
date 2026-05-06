package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Example documents one safe invocation for manifests, docs, or adapters.
type Example struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// Authorization documents the required gate for write-capable operations.
type Authorization struct {
	Required   bool   `json:"required"`
	Policy     string `json:"policy"`
	AuditEvent string `json:"audit_event"`
}

// Result is the raw in-process operation result. It intentionally contains no
// terminal presentation footer, truncation message, or argv-specific state.
type Result struct {
	Body       []byte
	StatusCode int
	ExitCode   int
	Stderr     string
	Retry      bool
	Duration   time.Duration
}

// Operation is the CLI-independent tool operation definition.
type Operation struct {
	Tool          string
	Name          string
	Description   string
	InputSchema   json.RawMessage
	ReadOnly      bool
	Retryable     bool
	Examples      []Example
	SafetyNotes   []string
	Authorization *Authorization
	Invoke        func(context.Context, json.RawMessage) (*Result, error)
}

// FullName returns a stable tool_operation name for function-call adapters.
func (o Operation) FullName() string {
	if o.Tool == "" {
		return o.Name
	}
	return o.Tool + "_" + o.Name
}

// ValidateMetadata checks metadata required for safe publication.
func (o Operation) ValidateMetadata() error {
	if o.Tool == "" || o.Name == "" || o.Description == "" {
		return fmt.Errorf("operation requires tool, name, and description")
	}
	if len(o.InputSchema) == 0 {
		return fmt.Errorf("operation %s.%s requires input schema", o.Tool, o.Name)
	}
	if !o.ReadOnly {
		if o.Authorization == nil || !o.Authorization.Required || o.Authorization.Policy == "" || o.Authorization.AuditEvent == "" {
			return fmt.Errorf("write-capable operation %s.%s requires authorization and audit metadata", o.Tool, o.Name)
		}
		if len(o.SafetyNotes) == 0 {
			return fmt.Errorf("write-capable operation %s.%s requires safety notes", o.Tool, o.Name)
		}
	}
	if o.Invoke == nil {
		return fmt.Errorf("operation %s.%s requires invoke function", o.Tool, o.Name)
	}
	return nil
}

// Call validates args, invokes the operation, and fills timing/status defaults.
func (o Operation) Call(ctx context.Context, args json.RawMessage) (*Result, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return &Result{ExitCode: 124, Retry: true, Duration: time.Since(start)}, err
	}
	if err := validateRequiredArgs(o.InputSchema, args); err != nil {
		return &Result{ExitCode: 2, Retry: false, Duration: time.Since(start)}, err
	}
	result, err := o.Invoke(ctx, args)
	if result == nil {
		result = &Result{}
	}
	result.Duration = time.Since(start)
	if err != nil {
		if result.ExitCode == 0 {
			result.ExitCode = 1
		}
		result.Retry = result.Retry || o.Retryable
		return result, err
	}
	if result.ExitCode == 0 && result.StatusCode >= 400 {
		result.ExitCode = 1
	}
	return result, nil
}

func validateRequiredArgs(schema json.RawMessage, args json.RawMessage) error {
	var doc struct {
		Required []string `json:"required"`
	}
	if len(schema) > 0 {
		if err := json.Unmarshal(schema, &doc); err != nil {
			return fmt.Errorf("invalid input schema: %w", err)
		}
	}
	if len(doc.Required) == 0 {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(args, &values); err != nil {
		return fmt.Errorf("invalid JSON args: %w", err)
	}
	for _, key := range doc.Required {
		if _, ok := values[key]; !ok {
			return fmt.Errorf("missing required field %q", key)
		}
	}
	return nil
}

// Registry holds operation definitions for raw in-process invocation.
type Registry struct {
	mu  sync.RWMutex
	ops map[string]Operation
}

func NewRegistry() *Registry {
	return &Registry{ops: map[string]Operation{}}
}

func (r *Registry) Register(op Operation) error {
	if err := op.ValidateMetadata(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := op.Tool + "." + op.Name
	if _, exists := r.ops[key]; exists {
		return fmt.Errorf("operation %s already registered", key)
	}
	r.ops[key] = op
	return nil
}

func (r *Registry) Invoke(ctx context.Context, tool, operation string, args json.RawMessage) (*Result, error) {
	r.mu.RLock()
	op, ok := r.ops[tool+"."+operation]
	r.mu.RUnlock()
	if !ok {
		return &Result{ExitCode: 127, Retry: false}, fmt.Errorf("operation %s.%s not found", tool, operation)
	}
	return op.Call(ctx, args)
}
