package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/tvmaly/clikit/pkg/transport"
)

// Context carries per-invocation state passed to command Run functions.
type Context struct {
	Args       []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	IsPiped    bool
	StartTime  time.Time
	GlobalOpts *GlobalOpts
	Transport  transport.Transport
}

// Command represents a CLI command or subcommand.
type Command struct {
	Name        string
	Short       string
	Long        string
	Subcommands []*Command
	Run         func(ctx *Context) error
}

// Execute dispatches to a subcommand if one is named in args, or runs the
// command's own Run function with the remaining args.
func (c *Command) Execute(ctx *Context, args []string) error {
	if len(args) > 0 {
		// Try to match a subcommand.
		for _, sub := range c.Subcommands {
			if sub.Name == args[0] {
				ctx.Args = args[1:]
				return sub.Execute(ctx, args[1:])
			}
		}
		// No subcommand match: if there's a Run func, treat args as its args.
		if c.Run != nil {
			ctx.Args = args
			return c.Run(ctx)
		}
		// Unknown subcommand.
		available := make([]string, len(c.Subcommands))
		for i, s := range c.Subcommands {
			available[i] = s.Name
		}
		return fmt.Errorf("unknown subcommand %q for %q; available: %v", args[0], c.Name, available)
	}

	// No args.
	if c.Run != nil {
		ctx.Args = args
		return c.Run(ctx)
	}
	// No Run func and no args: print help (handled by App layer).
	return nil
}
