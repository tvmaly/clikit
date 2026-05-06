package opcli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/tvmaly/clikit/pkg/cli"
	"github.com/tvmaly/clikit/pkg/ops"
)

// CommandFromOperations adapts operation definitions into a clikit command tree.
func CommandFromOperations(tool, short string, operations []ops.Operation) *cli.Command {
	cmd := &cli.Command{Name: tool, Short: short}
	cmd.Run = func(ctx *cli.Context) error {
		cli.RenderCommandHelp(ctx.Stdout, cmd)
		return nil
	}
	for _, op := range operations {
		operation := op
		cmd.Subcommands = append(cmd.Subcommands, &cli.Command{
			Name:  operation.Name,
			Short: operation.Description,
			Run: func(ctx *cli.Context) error {
				args, err := argsFromFlags(operation, ctx.Args)
				if err != nil {
					return err
				}
				result, err := operation.Call(context.Background(), args)
				if result != nil && len(result.Body) > 0 {
					if _, writeErr := ctx.Stdout.Write(result.Body); writeErr != nil {
						return writeErr
					}
					if result.Body[len(result.Body)-1] != '\n' {
						if _, writeErr := io.WriteString(ctx.Stdout, "\n"); writeErr != nil {
							return writeErr
						}
					}
				}
				return err
			},
		})
	}
	return cmd
}

func argsFromFlags(op ops.Operation, raw []string) (json.RawMessage, error) {
	fs := flag.NewFlagSet(op.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	props, err := schemaProperties(op.InputSchema)
	if err != nil {
		return nil, err
	}
	values := map[string]*string{}
	for _, name := range props {
		key := name
		values[key] = fs.String(key, "", "operation argument")
	}
	if err := fs.Parse(raw); err != nil {
		return nil, err
	}
	required, err := schemaRequired(op.InputSchema)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	for name, ptr := range values {
		if *ptr != "" {
			payload[name] = *ptr
		}
	}
	if fs.NArg() == 1 && len(required) == 1 {
		if _, exists := payload[required[0]]; !exists {
			payload[required[0]] = fs.Arg(0)
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func schemaProperties(schema json.RawMessage) ([]string, error) {
	var doc struct {
		Properties map[string]any `json:"properties"`
	}
	if len(schema) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, fmt.Errorf("invalid input schema: %w", err)
	}
	names := make([]string, 0, len(doc.Properties))
	for name := range doc.Properties {
		names = append(names, name)
	}
	return names, nil
}

func schemaRequired(schema json.RawMessage) ([]string, error) {
	var doc struct {
		Required []string `json:"required"`
	}
	if len(schema) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, fmt.Errorf("invalid input schema: %w", err)
	}
	return doc.Required, nil
}
