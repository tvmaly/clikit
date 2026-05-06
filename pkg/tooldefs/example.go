package tooldefs

import (
	"context"
	"encoding/json"

	"github.com/tvmaly/clikit/pkg/ops"
)

func ExampleOperations() []ops.Operation {
	return []ops.Operation{
		{
			Tool:        "example",
			Name:        "get",
			Description: "Get an example item by ID",
			InputSchema: json.RawMessage(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`),
			ReadOnly:    true,
			Examples:    []ops.Example{{Name: "sample", Args: json.RawMessage(`{"id":"abc"}`)}},
			SafetyNotes: []string{"Read-only fixture operation."},
			Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
				var req struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &req); err != nil {
					return &ops.Result{ExitCode: 2}, err
				}
				body, err := json.Marshal(map[string]any{"id": req.ID, "name": "example-item"})
				if err != nil {
					return &ops.Result{ExitCode: 1, Retry: false}, err
				}
				return &ops.Result{Body: body, StatusCode: 200, ExitCode: 0}, nil
			},
		},
		{
			Tool:        "example",
			Name:        "list",
			Description: "List example items",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			ReadOnly:    true,
			Examples:    []ops.Example{{Name: "sample", Args: json.RawMessage(`{}`)}},
			SafetyNotes: []string{"Read-only fixture operation."},
			Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
				body := []byte(`[{"id":"1","name":"item-one"},{"id":"2","name":"item-two"}]`)
				return &ops.Result{Body: body, StatusCode: 200, ExitCode: 0}, nil
			},
		},
	}
}
