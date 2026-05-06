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
- **Write once, expose many ways.** New Go tools can be defined as CLI-independent operations, then exposed through `toolkit`, Nanogo-compatible tools, manifests, and raw in-process calls.
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
│   ├── ops/               # CLI-independent operation metadata + raw invocation
│   ├── opcli/             # Adapter: operations → cli.Command
│   ├── nanogoops/         # Adapter: operations → Nanogo-compatible tools
│   ├── tooldefs/          # Shared operation-backed example definitions
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

### Operation layer

New Go tools should prefer `pkg/ops.Operation`. An operation is independent of CLI parsing and terminal output: it has a tool name, operation name, JSON input schema, read-only/write safety metadata, examples, safety notes, and an `Invoke(ctx,args)` function.

That same operation can be exposed as:

- a `toolkit <tool> <operation>` command through `pkg/opcli`
- a Nanogo-compatible `Name/Schema/Call` tool through `pkg/nanogoops`
- a raw in-process call through `ops.Operation.Call` or `ops.Registry.Invoke`
- manifest permission metadata checked by `ops.ValidateManifest`

Write-capable operations require explicit authorization and audit metadata before they pass validation.

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
    "context"
    "encoding/json"
    "os"

    "github.com/tvmaly/clikit/pkg/cli"
    "github.com/tvmaly/clikit/pkg/opcli"
    "github.com/tvmaly/clikit/pkg/ops"
)

func main() {
    listRepos := ops.Operation{
        Tool:        "repos",
        Name:        "list",
        Description: "List repositories",
        InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
        ReadOnly:    true,
        Examples:    []ops.Example{{Name: "sample", Args: json.RawMessage(`{}`)}},
        SafetyNotes: []string{"Read-only repository listing."},
        Invoke: func(ctx context.Context, args json.RawMessage) (*ops.Result, error) {
            body := []byte(`[{"slug":"my-repo","project":"PLAT"}]`)
            return &ops.Result{Body: body, StatusCode: 200, ExitCode: 0}, nil
        },
    }

    reposCmd := opcli.CommandFromOperations("repos", "Repository tools", []ops.Operation{listRepos})

    app := &cli.App{Name: "toolkit", Commands: []*cli.Command{reposCmd}}
    app.Run(os.Args[1:], os.Stdout, os.Stderr)
}
```

The operation can also be called directly without CLI formatting:

```go
// From the same operation value used to build the CLI command:
result, err := listRepos.Call(context.Background(), json.RawMessage(`{}`))
_ = result.Body // compact JSON, no metadata footer
_ = err
```

See [docs/tool-author-guide.md](docs/tool-author-guide.md) for the checklist covering schemas, read/write safety, Nanogo exposure, and manifest validation.

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
toolkit queue status --queue-root /mnt/shared
toolkit queue inspect --queue-root /mnt/shared <requestID>
toolkit queue retry --queue-root /mnt/shared <requestID>
toolkit queue list --queue-root /mnt/shared --state pending
toolkit queue clean --queue-root /mnt/shared
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

For Claude Code, generate a bounded skill definition directly from the binary:

```bash
toolkit skill-gen > SKILL.md
```

The generated skill tells Claude Code to discover first, parse JSON/JSONL, follow structured `suggestion` fields, and keep queue operations bounded. This is better than giving Claude a generic shell because the allowed command shape is explicit:

```yaml
allowed-tools: [Bash(toolkit *), Bash(toolkit-fmt *)]
```

For agent loops, keep compact JSON or JSONL:

```bash
toolkit example get --id abc
toolkit example list --output jsonl
```

When Claude Code needs to parse output with no footer, use `--raw`:

```bash
toolkit --raw example get --id abc
```

When a structured error is returned, read `retry` and `suggestion`. Retry only when `retry` is `true`; otherwise run the suggested discovery or correction command. The manifest in [manifests/codex.json](manifests/codex.json) is suitable for Codex and Claude Code-style shell-tool allowlists.

### Hermes

Use the Hermes skill example in [examples/agents/hermes-skill.md](examples/agents/hermes-skill.md). The core pattern is:

```bash
toolkit --help
toolkit queue status --help
toolkit queue status --queue-root /mnt/shared/clikit
```

Keep the Hermes permission block bounded to read-only discovery, list, status, and inspect commands until write operations have separate review. The manifest in [manifests/hermes.json](manifests/hermes.json) records the same output and retry contract in machine-readable form.

### OpenClaw

Use the OpenClaw allowlist example in [examples/agents/openclaw-skill.md](examples/agents/openclaw-skill.md):

```bash
toolkit --help
toolkit queue inspect --queue-root /mnt/shared/clikit a1b2c3d4e5f67890
```

Expose `toolkit` as a host tool or workspace command with an allowlist. Avoid unrestricted shell execution for internal systems. In file-queue mode, keep credentials on the `toolkit-worker` side and share only queue metadata and payload files. The machine-readable companion is [manifests/openclaw.json](manifests/openclaw.json).

### nanogo

Nanogo can integrate with `clikit` in two ways.

#### 1. Preferred: direct Go tool adapter

Use `pkg/nanogoops` to adapt operation-backed tools into Nanogo's current `Name/Schema/Call` shape without shelling out or parsing terminal footers:

```go
package nanogotools

import (
    "github.com/tvmaly/clikit/pkg/nanogoops"
    "github.com/tvmaly/clikit/pkg/tooldefs"
)

func ToolkitTools() []nanogoops.Tool {
    ops := tooldefs.ExampleOperations()
    tools := make([]nanogoops.Tool, 0, len(ops))
    for _, op := range ops {
        tools = append(tools, nanogoops.Adapt(op))
    }
    return tools
}
```

Nanogo sees each adapted operation as a schema-bearing tool:

```go
tool := nanogoops.Adapt(tooldefs.ExampleOperations()[0])

name := tool.Name()      // "example_get"
schema := tool.Schema()  // JSON schema for args
result, err := tool.Call(ctx, json.RawMessage(`{"id":"abc"}`))
```

`result` is compact JSON from the operation body. It does not include `[exit:N | Xms]`, truncation messages, or shell stderr.

#### 2. Fallback: call the CLI with `--raw`

For legacy tools that are only exposed through the binary, Nanogo can still call `toolkit` through its shell tool:

```bash
toolkit --raw example get --id abc
```

Use `--raw` so Nanogo receives only the JSON payload. If Nanogo is calling queue operations, pass the queue root explicitly:

```bash
toolkit --raw queue status --queue-root /mnt/shared/clikit
toolkit --raw queue inspect --queue-root /mnt/shared/clikit a1b2c3d4e5f67890
```

#### Tutor/admin reporting pattern

The example in [examples/nanogo/](examples/nanogo/) demonstrates read-only tutor/admin reporting commands:

```bash
toolkit nanogo tutor-status --student-id sample-student
toolkit nanogo admin-summary --limit 10
```

For real Nanogo tutor/admin tools, define operations with:

- `ReadOnly: true` for reporting commands such as tutor status, admin summary, queue status, and cost summary
- explicit `InputSchema` for student IDs, limits, date ranges, or queue request IDs
- sample-only examples and safety notes
- manifest validation through `ops.ValidateManifest`

Write-capable student, tutor memory, scheduler, or admin operations must set `Authorization.Required`, `Authorization.Policy`, and `Authorization.AuditEvent` before they can pass operation metadata validation. Keep those operations out of Nanogo manifests until the authorization and audit policy is reviewed.

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
