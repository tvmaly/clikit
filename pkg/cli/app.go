package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tvmaly/clikit/pkg/transport"
)

// App is the top-level CLI application.
type App struct {
	Name      string
	Version   string
	Commands  []*Command
	Transport transport.Transport
}

// Run parses global flags, resolves the command, builds a Context, and runs it.
// stdout and stderr are injected for testability.
func (a *App) Run(args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	// Handle --help / no args early.
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		RenderRegistryHelp(stdout, a.Name, a.Commands)
		return nil
	}

	// Parse global flags.
	globalOpts, remaining, err := ParseGlobalFlags(args)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	if len(remaining) == 0 {
		RenderRegistryHelp(stdout, a.Name, a.Commands)
		return nil
	}

	// Find command.
	cmdName := remaining[0]
	var cmd *Command
	for _, c := range a.Commands {
		if c.Name == cmdName {
			cmd = c
			break
		}
	}
	if cmd == nil {
		return fmt.Errorf("unknown command %q; run '%s' for available commands", cmdName, a.Name)
	}

	ctx := &Context{
		Stdin:      os.Stdin,
		Stdout:     stdout,
		Stderr:     stderr,
		IsPiped:    false,
		StartTime:  time.Now(),
		GlobalOpts: globalOpts,
		Transport:  a.Transport,
	}

	return cmd.Execute(ctx, remaining[1:])
}
