package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	clierrors "github.com/tvmaly/clikit/pkg/errors"
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
			e := clierrors.NewWithSuggestion(
				"binary stdin detected: cannot process binary content with --from-stdin",
				"pass text or JSON input, or write binary content to a file and pass its path",
				1,
			)
			_ = json.NewEncoder(stdout).Encode(e)
			return e
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
	if present && err == nil {
		if formatted, formatErr := applyGlobalOutputOptions(outBuf.Bytes(), globalOpts); formatErr != nil {
			err = formatErr
			outBuf.Reset()
			_ = json.NewEncoder(&outBuf).Encode(clierrors.NewWithSuggestion(formatErr.Error(), "check output flags and command output shape", 1))
		} else {
			outBuf.Reset()
			_, _ = outBuf.Write(formatted)
		}
	}
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

func applyGlobalOutputOptions(data []byte, opts *GlobalOpts) ([]byte, error) {
	if opts == nil || len(bytes.TrimSpace(data)) == 0 {
		return data, nil
	}

	if opts.Field != "" {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("extracting field: invalid JSON output: %w", err)
		}
		var buf bytes.Buffer
		if err := output.FormatField(&buf, v, opts.Field); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	if opts.Quiet || opts.Output == "quiet" {
		return nil, fmt.Errorf("--quiet requires --field")
	}

	if opts.Pretty || opts.Output == "pretty" {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("pretty output requires JSON: %w", err)
		}
		var buf bytes.Buffer
		if err := output.FormatJSON(&buf, v, true); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	switch opts.Output {
	case "", "json":
		return data, nil
	case "jsonl":
		var items []any
		if err := json.Unmarshal(data, &items); err != nil {
			return data, nil
		}
		var buf bytes.Buffer
		if err := output.FormatJSONL(&buf, items); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "table":
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", opts.Output)
	}
}
