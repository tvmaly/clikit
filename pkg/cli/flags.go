package cli

import (
	"flag"
	"strings"
)

// GlobalOpts holds the global flags available on every command.
type GlobalOpts struct {
	Output  string // json, jsonl, pretty, table, quiet
	Pretty  bool
	Quiet   bool
	Raw     bool
	Field   string
	Limit   int
	Filter  string
}

// DefaultGlobalOpts returns GlobalOpts with default values.
func DefaultGlobalOpts() *GlobalOpts {
	return &GlobalOpts{Output: "json"}
}

// ParseGlobalFlags parses global flags from args, returning the populated opts
// and the remaining (non-global) args. Unknown flags are preserved in remaining.
func ParseGlobalFlags(args []string) (*GlobalOpts, []string, error) {
	opts := DefaultGlobalOpts()
	fs := flag.NewFlagSet("global", flag.ContinueOnError)
	fs.StringVar(&opts.Output, "output", "json", "output format: json, jsonl, pretty, table, quiet")
	fs.BoolVar(&opts.Pretty, "pretty", false, "pretty-print JSON output")
	fs.BoolVar(&opts.Quiet, "quiet", false, "output bare values only")
	fs.BoolVar(&opts.Raw, "raw", false, "raw output, skip presentation layer")
	fs.StringVar(&opts.Field, "field", "", "extract a single field by dot-path")
	fs.IntVar(&opts.Limit, "limit", 0, "limit number of results")
	fs.StringVar(&opts.Filter, "filter", "", "filter results by key=value")

	// Parse only flags that are known; collect unknown args.
	var remaining []string
	i := 0
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			// Non-flag argument: consume it and all following args as remaining.
			remaining = append(remaining, args[i:]...)
			break
		}
		// Try parsing this flag.
		if err := fs.Parse(args[i:]); err != nil {
			// The flag package stops at the first unknown flag. Capture what's left.
			remaining = append(remaining, fs.Args()...)
			break
		}
		remaining = append(remaining, fs.Args()...)
		break
	}
	if i >= len(args) {
		// Consumed everything.
	}

	return opts, remaining, nil
}
