// example is an example native Go tool demonstrating the clikit framework.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tvmaly/clikit/pkg/cli"
	"github.com/tvmaly/clikit/pkg/registry"
)

func main() {
	reg := registry.New()
	reg.Register(exampleTool())

	app := &cli.App{
		Name:     "example",
		Version:  "1.0.0",
		Commands: reg.Commands(),
	}

	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func exampleTool() *cli.Command {
	return &cli.Command{
		Name:  "example",
		Short: "Example tool demonstrating clikit",
		Subcommands: []*cli.Command{
			{
				Name:  "get",
				Short: "Get an item by ID",
				Run: func(ctx *cli.Context) error {
					id := ""
					if len(ctx.Args) > 0 {
						id = ctx.Args[0]
					}
					result := map[string]any{"id": id, "name": "example-item"}
					return json.NewEncoder(ctx.Stdout).Encode(result)
				},
			},
			{
				Name:  "list",
				Short: "List all items",
				Run: func(ctx *cli.Context) error {
					items := []any{
						map[string]any{"id": "1", "name": "item-one"},
						map[string]any{"id": "2", "name": "item-two"},
					}
					enc := json.NewEncoder(ctx.Stdout)
					for _, item := range items {
						enc.Encode(item)
					}
					return nil
				},
			},
		},
	}
}
