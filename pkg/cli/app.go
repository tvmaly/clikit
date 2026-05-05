package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tvmaly/clikit/pkg/output"
	"github.com/tvmaly/clikit/pkg/transport"
)

// App is the top-level CLI application.
type App struct {
	Name              string
	Version           string
	Commands          []*Command
	Transport         transport.Transport
	Stdin             io.Reader
	OverflowThreshold int
	SpillDir          string
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
	stdin := a.Stdin
	if stdin == nil {
		stdin = os.Stdin
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

	if globalOpts.FromStdin {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if output.IsBinary(data) {
			return fmt.Errorf("binary stdin detected: cannot process binary content with --from-stdin")
		}
		stdin = bytes.NewReader(data)
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

	present := shouldPresent(stdout, globalOpts)
	commandStdout := stdout
	commandStderr := stderr
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	if present {
		commandStdout = &outBuf
		commandStderr = &errBuf
	}

	ctx := &Context{
		Stdin:      stdin,
		Stdout:     commandStdout,
		Stderr:     commandStderr,
		IsPiped:    false,
		StartTime:  time.Now(),
		GlobalOpts: globalOpts,
		Transport:  a.Transport,
	}

	err = cmd.Execute(ctx, remaining[1:])
	if !present {
		return err
	}

	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	spillDir := a.SpillDir
	if spillDir == "" {
		spillDir = os.TempDir()
	}
	threshold := a.OverflowThreshold
	if threshold == 0 {
		threshold = 200 * 1024
	}
	p := output.NewPresenter(stdout, output.Options{
		OverflowThreshold: threshold,
		SpillDir:          spillDir,
	})
	if presentErr := p.Present(outBuf.Bytes(), exitCode, errBuf.Bytes(), time.Since(ctx.StartTime)); presentErr != nil {
		return presentErr
	}
	return err
}

func shouldPresent(stdout io.Writer, opts *GlobalOpts) bool {
	if opts != nil && opts.Raw {
		return false
	}
	if _, ok := stdout.(*os.File); !ok {
		return true
	}
	return !output.IsPipedWriter(stdout)
}
