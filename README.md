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
- **Agent manifests.** Versioned manifests describe permissions, output contracts, retry behavior, examples, and safety notes for Codex, Hermes, OpenClaw, and nanogo.
- **Operational contracts.** Versioned JSON contracts and a file-queue operator runbook make integrations auditable instead of prose-only.

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
| `--from-stdin` | Read stdin after binary validation |

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
├── manifests/             # Agent manifests for Codex, Hermes, OpenClaw, nanogo
├── docs/contracts/        # Versioned JSON contracts
├── docs/runbooks/         # Operator runbooks
└── examples/              # Example shell script tools, schemas, skills, integrations
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

The presentation layer activates only when stdout is a terminal or an agent is reading directly. When stdout is piped (`toolkit foo | jq`), raw bytes flow through unmodified. `cli.App` wires command output through this presentation layer for direct invocation. Use `--raw` for pipe-safe output without footers, truncation messages, or presentation-only formatting.

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

See [docs/runbooks/filequeue-operator-runbook.md](docs/runbooks/filequeue-operator-runbook.md) for queue layout, lifecycle, stale claim recovery, cleanup, secrets handling, and troubleshooting.

**Queue inspection commands:**

```bash
toolkit queue status                   # counts per state
toolkit queue inspect <requestID>      # full metadata for one request
toolkit queue retry <requestID>        # move dead-letter back to pending
toolkit queue list --state pending     # list pending requests
toolkit queue clean                    # run cleanup now
```

---

## Agent manifests and contracts

Machine-readable manifests live in [manifests/](manifests/):

| Manifest | Use |
|----------|-----|
| [codex.json](manifests/codex.json) | Codex or Claude Code-style shell tool discovery and allowlisting |
| [hermes.json](manifests/hermes.json) | Hermes skill permissions and retry behavior |
| [openclaw.json](manifests/openclaw.json) | OpenClaw workspace or host-tool allowlists |
| [nanogo.json](manifests/nanogo.json) | Read-only tutor/admin reporting examples for nanogo |

The stable JSON contract is documented in [docs/contracts/json-contract-v1.md](docs/contracts/json-contract-v1.md). External consumers may rely on the v1 field names for structured errors, queue request metadata, queue response metadata, JSON/JSONL output conventions, and metadata footer format.

Worker observability is available through an optional structured event hook. Worker events include request claimed, handler selected, HTTP request started, HTTP response received, request completed, request dead-lettered, and cleanup start/completion. Events intentionally exclude bearer tokens, environment secrets, request payloads, and full response bodies.

---

## Agent examples

### Claude Code and Codex

Use `toolkit` as a shell command with discovery-first behavior:

```bash
toolkit --help
toolkit <tool> --help
toolkit <tool> <operation> --help
```

For agent loops, keep compact JSON:

```bash
toolkit repos list --output jsonl --limit 20
```

When a structured error is returned, read `retry` and `suggestion`. Retry only when `retry` is `true`; otherwise run the suggested discovery or correction command. The manifest in [manifests/codex.json](manifests/codex.json) is suitable for Codex and Claude Code-style shell-tool allowlists.

### Hermes

Use the Hermes skill example in [examples/agents/hermes-skill.md](examples/agents/hermes-skill.md). The core pattern is:

```bash
toolkit --help
toolkit queue status --help
toolkit queue status
```

Keep the Hermes permission block bounded to read-only discovery, list, status, and inspect commands until write operations have separate review. The manifest in [manifests/hermes.json](manifests/hermes.json) records the same output and retry contract in machine-readable form.

### OpenClaw

Use the OpenClaw allowlist example in [examples/agents/openclaw-skill.md](examples/agents/openclaw-skill.md):

```bash
toolkit --help
toolkit queue inspect a1b2c3d4e5f67890
```

Expose `toolkit` as a host tool or workspace command with an allowlist. Avoid unrestricted shell execution for internal systems. In file-queue mode, keep credentials on the `toolkit-worker` side and share only queue metadata and payload files. The machine-readable companion is [manifests/openclaw.json](manifests/openclaw.json).

### nanogo

`nanogo` should consume `clikit` as a downstream read-only integration, not as a core dependency. The example in [examples/nanogo/](examples/nanogo/) demonstrates tutor/admin reporting commands:

```bash
toolkit nanogo tutor-status --student-id sample-student
toolkit nanogo admin-summary --limit 10
```

Examples must use sample student identifiers and compact JSON fixtures. Keep student records, tutor memory, scheduler state, and admin data read-only until write operations have explicit authorization and audit rules. The manifest in [manifests/nanogo.json](manifests/nanogo.json) encodes those boundaries.

---

## Development

```bash
GOCACHE=/tmp/go-cache go test ./...                          # run all tests
GOCACHE=/tmp/go-cache go test -race ./...                    # run with race detector
GOCACHE=/tmp/go-cache go test ./pkg/output/ -run TestBinary  # run a single test
go build ./...                                                # build all binaries
gofmt -w .                                                    # format
go vet ./...                                                  # vet
```

Zero external dependencies — standard library only.

---

## Design principles

- **Noun → verb command hierarchy.** `toolkit <resource> <action>` is more discoverable for agents than verb → noun.
- **Errors are navigation.** A bad command returns structured JSON with `available` options so agents self-correct without a human in the loop.
- **Secrets stay server-side.** In file-queue mode, request files never contain credentials. The worker resolves tokens from its own environment.
- **Atomic file operations.** Write-to-temp-then-rename everywhere — readers never see partial files.
- **Zero speculative architecture.** Standard library only. No frameworks, no ORMs, no generated code.
