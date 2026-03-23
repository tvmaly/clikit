# clikit

**10–32x fewer tokens per tool call.** A Go framework for building internal CLI tools that AI agents (Claude Code, Codex, Copilot, custom agents) consume efficiently.

---

## Why clikit?

Every token your agent spends interpreting tool output is a token it can't spend on your actual task — and it costs money. Most internal tooling returns noisy, unstructured, or verbose output that was designed for humans, not agents.

clikit gives you a framework for building tools that agents understand immediately:

- **JSON by default.** Every command outputs compact JSON. Agents parse it; humans add `--pretty`.
- **JSONL for lists.** One JSON object per line — bounded, pipeable, composable with `jq`.
- **Structured errors with next steps.** Errors include `suggestion` and `retry` fields. Agents follow the suggestion and move on rather than stalling.
- **Progressive disclosure.** Three levels of help (`registry → tool → subcommand`). Agents discover only what they need, when they need it.
- **Metadata footer.** `[exit:0 | 42ms]` on every response — agents learn latency and exit status without parsing extra fields.
- **Overflow protection.** Large responses are truncated with a spillfile path; agents get structure and can drill in rather than consuming a 200KB blob.
- **Binary guard.** Binary output is rejected before it reaches the agent's token budget.

---

## Quick start for agents

```bash
# Discover available tools
toolkit --help

# Discover subcommands for a tool
toolkit <tool>

# Get full details for a specific subcommand
toolkit <tool> <subcommand> --help
```

**Output flags available on every command:**

| Flag | Effect |
|------|--------|
| *(default)* | Compact JSON |
| `--pretty` | Indented JSON |
| `--output jsonl` | One JSON object per line |
| `--quiet --field name` | Bare value extraction |
| `--limit N` | Cap result count |
| `--raw` | Skip presentation layer (for piping) |

**Error handling:**

Every error is structured JSON:

```json
{
  "error": "project MYPROJ not found",
  "suggestion": "run 'toolkit bitbucket project list' to see available projects",
  "available": ["PLAT", "DATA", "INFRA"],
  "retry": false,
  "exit_code": 1
}
```

When `retry` is `true`, wait briefly and retry. When `retry` is `false`, follow `suggestion` and adjust the command.

---

## Architecture

```
clikit/
├── cmd/
│   ├── toolkit/           # Umbrella registry binary — the agent entry point
│   ├── toolkit-fmt/       # Shell script bridge — pipe any script through it
│   └── toolkit-worker/    # File-queue worker daemon (air-gapped environments)
├── pkg/
│   ├── cli/               # App, Command, Context, flag parsing, help rendering
│   ├── output/            # Binary guard, overflow, JSON/JSONL/table formatting,
│   │                      # pipe detection, LLM presentation layer
│   ├── errors/            # Structured CLIError with suggestion + retry
│   ├── schema/            # JSON field mapping for shell script bridge
│   ├── transport/         # Pluggable transport: HTTP or file-queue
│   ├── filequeue/         # Atomic file ops, queue dirs, lifecycle state machine
│   ├── worker/            # Dispatcher, handler registry, heartbeat, cleanup
│   ├── bridge/            # HTTP client helpers, pagination
│   ├── exec/              # os/exec wrapper with stderr capture
│   └── registry/          # Thread-safe tool registration and discovery
├── tools/example/         # Example native Go tool
└── examples/              # Example shell script tools + schemas
```

### Two-layer output model

```
┌──────────────────────────────────────────────────────────────┐
│  Presentation layer  (terminal / agent direct invocation)    │
│  Binary guard → Overflow → Format → Metadata footer → Stderr │
├──────────────────────────────────────────────────────────────┤
│  Execution layer  (piped output)                             │
│  Raw bytes, no truncation, no metadata — safe for pipes      │
└──────────────────────────────────────────────────────────────┘
```

The presentation layer activates only when stdout is a terminal or an agent is reading directly. When stdout is piped (`toolkit foo | jq`), raw bytes flow through unmodified.

### Pluggable transport

Tools work identically regardless of how they reach the backend:

- **HTTP transport** — direct `net/http` call to internal APIs.
- **File-queue transport** — for air-gapped or network-isolated environments. The agent side writes a request file to a shared drive; a `toolkit-worker` daemon on the internal-network side claims it, calls the API, and writes the response back.

The CLI, progressive disclosure, output formatting, and error handling are identical in both modes.

---

## Building a native Go tool

```go
package main

import (
    "encoding/json"
    "os"

    "github.com/tvmaly/clikit/pkg/cli"
    "github.com/tvmaly/clikit/pkg/registry"
)

func main() {
    reg := registry.New()
    reg.Register(&cli.Command{
        Name:  "repos",
        Short: "List repositories",
        Run: func(ctx *cli.Context) error {
            items := []map[string]any{
                {"slug": "my-repo", "project": "PLAT"},
            }
            enc := json.NewEncoder(ctx.Stdout)
            for _, item := range items {
                enc.Encode(item) // JSONL: one object per line
            }
            return nil
        },
    })

    app := &cli.App{Name: "toolkit", Commands: reg.Commands()}
    app.Run(os.Args[1:], os.Stdout, os.Stderr)
}
```

## Building a shell script tool

Shell scripts pipe through `toolkit-fmt` to get the same output contract:

```bash
#!/usr/bin/env bash
curl -sf "https://api.internal/repos" \
  | toolkit-fmt output --schema ./schemas/repos.json
```

On error:

```bash
toolkit-fmt error \
  --message "repo not found" \
  --suggestion "run 'toolkit repos list' to see available repos" \
  --exit-code 1
```

Schema file (`repos.json`) maps API fields to output fields:

```json
{
  "fields": [
    {"from": "slug",        "to": "repo"},
    {"from": "project.key", "to": "project"}
  ]
}
```

---

## File-queue transport (air-gapped environments)

For environments where the agent cannot reach internal APIs directly:

```
Agent side                         Internal-network side
──────────────────────────────     ──────────────────────────────
toolkit --transport filequeue  →   toolkit-worker --queue-root /mnt/shared
         --queue-root /mnt/shared            --handler-config handlers.json
```

The agent writes a request file; the worker claims it, calls the API, and writes the response. The agent polls for it. No credentials ever leave the internal network.

**Queue inspection commands:**

```bash
toolkit queue status                   # counts per state
toolkit queue inspect <requestID>      # full metadata for one request
toolkit queue retry <requestID>        # move dead-letter back to pending
toolkit queue list --state pending     # list pending requests
toolkit queue clean                    # run cleanup now
```

---

## Development

```bash
go test ./...                          # run all tests
go test -race ./...                    # run with race detector
go test ./pkg/output/ -run TestBinary  # run a single test
go build ./...                         # build all binaries
gofmt -w .                             # format
go vet ./...                           # vet
```

Zero external dependencies — standard library only.

---

## Design principles

- **Noun → verb command hierarchy.** `toolkit <resource> <action>` is more discoverable for agents than verb → noun.
- **Errors are navigation.** A bad command returns structured JSON with `available` options so agents self-correct without a human in the loop.
- **Secrets stay server-side.** In file-queue mode, request files never contain credentials. The worker resolves tokens from its own environment.
- **Atomic file operations.** Write-to-temp-then-rename everywhere — readers never see partial files.
- **Zero speculative architecture.** Standard library only. No frameworks, no ORMs, no generated code.
