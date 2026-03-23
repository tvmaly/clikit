// toolkit is the umbrella registry CLI binary.
// It discovers registered tools and dispatches commands.
package main

import (
	"fmt"
	"os"

	"github.com/tvmaly/clikit/pkg/cli"
	"github.com/tvmaly/clikit/pkg/registry"
)

func main() {
	reg := registry.New()

	// Register built-in tools here. Example tools would call reg.Register().

	app := &cli.App{
		Name:     "toolkit",
		Version:  "1.0.0",
		Commands: reg.Commands(),
	}

	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
