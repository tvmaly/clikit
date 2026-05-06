package opcli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tvmaly/clikit/pkg/cli"
	"github.com/tvmaly/clikit/pkg/ops"
)

func TestCommandFromOperationsInvokesOperation(t *testing.T) {
	op := ops.Operation{
		Tool:        "example",
		Name:        "get",
		Description: "Get example item",
		InputSchema: json.RawMessage(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
			var got map[string]string
			if err := json.Unmarshal(args, &got); err != nil {
				t.Fatal(err)
			}
			return &ops.Result{Body: []byte(`{"id":"` + got["id"] + `","name":"example-item"}`), StatusCode: 200}, nil
		},
	}
	cmd := CommandFromOperations("example", "Example tools", []ops.Operation{op})
	app := &cli.App{Name: "toolkit", Commands: []*cli.Command{cmd}}
	var out bytes.Buffer
	if err := app.Run([]string{"--raw", "example", "get", "--id", "abc"}, &out, &out); err != nil {
		t.Fatalf("unexpected error: %v output=%q", err, out.String())
	}
	if strings.TrimSpace(out.String()) != `{"id":"abc","name":"example-item"}` {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestCommandFromOperationsHelpUsesMetadata(t *testing.T) {
	cmd := CommandFromOperations("example", "Example tools", []ops.Operation{{
		Tool:        "example",
		Name:        "get",
		Description: "Get example item",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		ReadOnly:    true,
		Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
			return &ops.Result{}, nil
		},
	}})
	var out bytes.Buffer
	if err := cmd.Execute(&cli.Context{Stdout: &out}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "get") || !strings.Contains(out.String(), "Get example item") {
		t.Fatalf("expected operation metadata in help, got %q", out.String())
	}
}
