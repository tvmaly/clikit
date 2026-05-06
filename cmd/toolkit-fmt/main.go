// toolkit-fmt is the shell script bridge utility.
// Shell scripts pipe JSON through this binary to produce the same output
// contract as native Go tools (structured errors, metadata footer, etc.).
//
// Usage:
//
//	toolkit-fmt output [--schema SCHEMA] [--pretty]
//	toolkit-fmt error --message MSG [--suggestion S] [--retry]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	clierrors "github.com/tvmaly/clikit/pkg/errors"
	"github.com/tvmaly/clikit/pkg/schema"
)

type stringList []string

func (s *stringList) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: toolkit-fmt <output|error> [flags]")
		return 1
	}

	switch args[0] {
	case "output":
		return runOutput(args[1:], stdin, stdout)
	case "error":
		return runError(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		return 1
	}
}

func runOutput(args []string, stdin io.Reader, stdout io.Writer) int {
	fs := flag.NewFlagSet("output", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	schemaPath := fs.String("schema", "", "path to schema JSON file")
	pretty := fs.Bool("pretty", false, "pretty-print output")
	format := fs.String("output", "json", "output format: json")
	if err := fs.Parse(args); err != nil {
		writeError(stdout, "parsing flags: "+err.Error(), "", false)
		return 1
	}
	if *format != "json" {
		writeError(stdout, "unsupported output format: "+*format, "use --output json", false)
		return 1
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		writeError(stdout, "reading stdin: "+err.Error(), "", false)
		return 1
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		writeError(stdout, "invalid JSON input: "+err.Error(), "ensure stdin is valid JSON", false)
		return 1
	}

	if *schemaPath != "" {
		s, err := schema.LoadFile(*schemaPath)
		if err != nil {
			writeError(stdout, "loading schema: "+err.Error(), "", false)
			return 1
		}
		m, ok := result.(map[string]any)
		if !ok {
			writeError(stdout, "schema transform requires a JSON object", "", false)
			return 1
		}
		result, err = schema.Transform(s, m)
		if err != nil {
			writeError(stdout, "transforming output: "+err.Error(), "", false)
			return 1
		}
	}

	enc := json.NewEncoder(stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(result); err != nil {
		return 1
	}
	return 0
}

func runError(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("error", flag.ContinueOnError)
	fs.SetOutput(stderr)
	message := fs.String("message", "", "error message (required)")
	suggestion := fs.String("suggestion", "", "suggested action")
	retry := fs.Bool("retry", false, "whether the operation can be retried")
	exitCode := fs.Int("exit-code", 1, "exit code")
	var available stringList
	fs.Var(&available, "available", "available value; may be repeated")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *message == "" {
		writeError(stdout, "--message is required", "provide --message with a short error summary", false)
		return 1
	}
	e := &clierrors.CLIError{
		Message:    *message,
		Suggestion: *suggestion,
		Available:  []string(available),
		Retry:      *retry,
		ExitCode:   *exitCode,
	}
	if err := json.NewEncoder(stdout).Encode(e); err != nil {
		return 1
	}
	return *exitCode
}

func writeError(stdout io.Writer, message, suggestion string, retry bool) {
	e := &clierrors.CLIError{Message: message, Suggestion: suggestion, Retry: retry, ExitCode: 1}
	json.NewEncoder(stdout).Encode(e)
}
