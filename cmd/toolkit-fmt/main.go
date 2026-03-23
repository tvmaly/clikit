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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: toolkit-fmt <output|error> [flags]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "output":
		runOutput(os.Args[2:])
	case "error":
		runError(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(1)
	}
}

func runOutput(args []string) {
	fs := flag.NewFlagSet("output", flag.ExitOnError)
	schemaPath := fs.String("schema", "", "path to schema JSON file")
	pretty := fs.Bool("pretty", false, "pretty-print output")
	fs.Parse(args)

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeError("reading stdin: "+err.Error(), "", false)
		os.Exit(1)
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		writeError("invalid JSON input: "+err.Error(), "ensure stdin is valid JSON", false)
		os.Exit(1)
	}

	if *schemaPath != "" {
		s, err := schema.LoadFile(*schemaPath)
		if err != nil {
			writeError("loading schema: "+err.Error(), "", false)
			os.Exit(1)
		}
		m, ok := result.(map[string]any)
		if !ok {
			writeError("schema transform requires a JSON object", "", false)
			os.Exit(1)
		}
		result, err = schema.Transform(s, m)
		if err != nil {
			writeError("transforming output: "+err.Error(), "", false)
			os.Exit(1)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	enc.Encode(result)
}

func runError(args []string) {
	fs := flag.NewFlagSet("error", flag.ExitOnError)
	message := fs.String("message", "", "error message (required)")
	suggestion := fs.String("suggestion", "", "suggested action")
	retry := fs.Bool("retry", false, "whether the operation can be retried")
	exitCode := fs.Int("exit-code", 1, "exit code")
	fs.Parse(args)

	if *message == "" {
		fmt.Fprintln(os.Stderr, "--message is required")
		os.Exit(1)
	}
	e := &clierrors.CLIError{
		Message:    *message,
		Suggestion: *suggestion,
		Retry:      *retry,
		ExitCode:   *exitCode,
	}
	json.NewEncoder(os.Stdout).Encode(e)
	os.Exit(*exitCode)
}

func writeError(message, suggestion string, retry bool) {
	e := &clierrors.CLIError{Message: message, Suggestion: suggestion, Retry: retry, ExitCode: 1}
	json.NewEncoder(os.Stdout).Encode(e)
}
