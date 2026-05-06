// toolkit is the umbrella registry CLI binary.
// It discovers registered tools and dispatches commands.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tvmaly/clikit/pkg/cli"
	clierrors "github.com/tvmaly/clikit/pkg/errors"
	"github.com/tvmaly/clikit/pkg/filequeue"
	"github.com/tvmaly/clikit/pkg/opcli"
	"github.com/tvmaly/clikit/pkg/registry"
	"github.com/tvmaly/clikit/pkg/tooldefs"
	"github.com/tvmaly/clikit/pkg/worker"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	reg := registry.New()
	reg.Register(opcli.CommandFromOperations("example", "Example read-only tool", tooldefs.ExampleOperations()))
	reg.Register(queueCommand())
	reg.Register(skillGenCommand())

	app := &cli.App{
		Name:     "toolkit",
		Version:  "1.0.0",
		Commands: reg.Commands(),
	}

	if err := app.Run(args, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func skillGenCommand() *cli.Command {
	return &cli.Command{
		Name:  "skill-gen",
		Short: "Generate a toolkit SKILL.md",
		Run: func(ctx *cli.Context) error {
			_, err := io.WriteString(ctx.Stdout, generatedSkillMarkdown())
			return err
		},
	}
}

func generatedSkillMarkdown() string {
	return `---
name: toolkit
description: >
  Internal CLI toolkit for agent-consumable JSON, JSONL, queue inspection,
  structured errors, and bounded internal tool discovery.
allowed-tools: [Bash(toolkit *), Bash(toolkit-fmt *)]
---
# Internal Toolkit

Run ` + "`toolkit --help`" + ` to discover available tools.
Run ` + "`toolkit <tool>`" + ` to see available subcommands.
Run ` + "`toolkit <tool> <command> --help`" + ` for specific parameters.

## Output conventions

- Commands output compact JSON by default.
- Use ` + "`--pretty`" + ` for human-readable JSON.
- Use ` + "`--quiet --field <path>`" + ` for bare values.
- Use ` + "`--output jsonl`" + ` for line-delimited JSON lists.
- Use ` + "`--raw`" + ` to skip presentation metadata for piping or parsing.
- Use ` + "`--from-stdin`" + ` only with text or JSON input.

## Error handling

Read structured errors from stdout. Follow ` + "`suggestion`" + ` before retrying.
Retry the same command only when ` + "`retry`" + ` is true.

## Queue operations

Use ` + "`toolkit queue status --queue-root <dir>`" + ` to count queue states.
Use ` + "`toolkit queue list --queue-root <dir> --state dead`" + ` to find failures.
Use ` + "`toolkit queue inspect --queue-root <dir> <request_id>`" + ` before retrying.
Use ` + "`toolkit queue retry --queue-root <dir> <request_id>`" + ` only for reviewed dead-letter requests.

## Important rules

- Never guess flags. Always check help first.
- Keep write operations behind explicit allowlists.
- Do not put secrets in file-queue request metadata or payloads.
`
}

func queueCommand() *cli.Command {
	return &cli.Command{
		Name:  "queue",
		Short: "Inspect and operate the file-queue transport",
		Run: func(ctx *cli.Context) error {
			cli.RenderCommandHelp(ctx.Stdout, queueCommand())
			return nil
		},
		Subcommands: []*cli.Command{
			{Name: "status", Short: "Count requests by queue state", Run: runQueueStatus},
			{Name: "list", Short: "List request metadata by state", Run: runQueueList},
			{Name: "inspect", Short: "Inspect one queue request", Run: runQueueInspect},
			{Name: "retry", Short: "Move a dead-letter request back to pending", Run: runQueueRetry},
			{Name: "clean", Short: "Run queue cleanup once", Run: runQueueClean},
		},
	}
}

func runQueueStatus(ctx *cli.Context) error {
	fs, queueRoot := newQueueFlagSet("status")
	if err := fs.Parse(ctx.Args); err != nil {
		return err
	}
	if err := requireQueueRoot(*queueRoot); err != nil {
		return err
	}
	p := filequeue.QueuePaths(*queueRoot)
	counts := map[string]int{}
	for state, dir := range map[string]string{
		"pending": p.Pending,
		"claimed": p.Claimed,
		"done":    p.Done,
		"dead":    p.Dead,
	} {
		n, err := countMetaFiles(dir)
		if err != nil {
			return err
		}
		counts[state] = n
	}
	return json.NewEncoder(ctx.Stdout).Encode(counts)
}

func runQueueList(ctx *cli.Context) error {
	fs, queueRoot := newQueueFlagSet("list")
	state := fs.String("state", "pending", "queue state: pending, claimed, done, dead")
	if err := fs.Parse(ctx.Args); err != nil {
		return err
	}
	if err := requireQueueRoot(*queueRoot); err != nil {
		return err
	}
	dir, err := stateDir(*queueRoot, *state)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(ctx.Stdout)
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		payload, err := inspectMetaFile(filepath.Join(dir, e.Name()), *state)
		if err != nil {
			return err
		}
		if err := enc.Encode(payload); err != nil {
			return err
		}
	}
	return nil
}

func runQueueInspect(ctx *cli.Context) error {
	fs, queueRoot := newQueueFlagSet("inspect")
	if err := fs.Parse(ctx.Args); err != nil {
		return err
	}
	if err := requireQueueRoot(*queueRoot); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return clierrors.NewWithSuggestion("request id is required", "run 'toolkit queue list --state pending' to find request IDs", 1)
	}
	id := fs.Arg(0)
	if !filequeue.SafeRequestID(id) {
		return clierrors.NewWithSuggestion("invalid request id", "use the 16-character lowercase hex request_id from queue list", 1)
	}
	for _, state := range []string{"pending", "claimed", "done", "dead"} {
		dir, _ := stateDir(*queueRoot, state)
		path := filepath.Join(dir, id+".meta.json")
		if _, err := os.Stat(path); err == nil {
			payload, err := inspectMetaFile(path, state)
			if err != nil {
				return err
			}
			return json.NewEncoder(ctx.Stdout).Encode(payload)
		}
	}
	return clierrors.NewWithSuggestion("request not found", "run 'toolkit queue list --state pending' or inspect done/dead states", 1)
}

func runQueueRetry(ctx *cli.Context) error {
	fs, queueRoot := newQueueFlagSet("retry")
	if err := fs.Parse(ctx.Args); err != nil {
		return err
	}
	if err := requireQueueRoot(*queueRoot); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return clierrors.NewWithSuggestion("request id is required", "run 'toolkit queue list --state dead' to find retryable request IDs", 1)
	}
	id := fs.Arg(0)
	if !filequeue.SafeRequestID(id) {
		return clierrors.NewWithSuggestion("invalid request id", "use the 16-character lowercase hex request_id from queue list", 1)
	}
	p := filequeue.QueuePaths(*queueRoot)
	if err := filequeue.AtomicMove(p.Dead, p.Pending, id+".meta.json"); err != nil {
		return fmt.Errorf("retrying request: %w", err)
	}
	return json.NewEncoder(ctx.Stdout).Encode(map[string]any{
		"request_id": id,
		"state":      "pending",
		"retry":      true,
	})
}

func runQueueClean(ctx *cli.Context) error {
	fs, queueRoot := newQueueFlagSet("clean")
	doneTTL := fs.Duration("done-ttl", 24*time.Hour, "done retention")
	deadTTL := fs.Duration("dead-ttl", 72*time.Hour, "dead retention")
	if err := fs.Parse(ctx.Args); err != nil {
		return err
	}
	if err := requireQueueRoot(*queueRoot); err != nil {
		return err
	}
	cleaned, err := worker.RunCleanup(*queueRoot, *doneTTL, *deadTTL)
	if err != nil {
		return err
	}
	return json.NewEncoder(ctx.Stdout).Encode(map[string]any{"cleaned": cleaned})
}

func newQueueFlagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	queueRoot := fs.String("queue-root", "", "queue root")
	return fs, queueRoot
}

func requireQueueRoot(queueRoot string) error {
	if queueRoot == "" {
		return clierrors.NewWithSuggestion("--queue-root is required", "pass --queue-root with the shared file-queue root", 1)
	}
	return nil
}

func countMetaFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			n++
		}
	}
	return n, nil
}

func stateDir(queueRoot, state string) (string, error) {
	p := filequeue.QueuePaths(queueRoot)
	switch state {
	case "pending":
		return p.Pending, nil
	case "claimed":
		return p.Claimed, nil
	case "done":
		return p.Done, nil
	case "dead":
		return p.Dead, nil
	default:
		return "", clierrors.NewWithSuggestion("unknown queue state", "use one of: pending, claimed, done, dead", 1)
	}
}

func inspectMetaFile(path, state string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	payload["state"] = state
	return payload, nil
}
