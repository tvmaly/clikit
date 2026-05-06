# Tool Author Guide

Use this guide when adding a toolkit operation that may be exposed through CLI, Nanogo, manifests, or file-queue metadata.

## Choose The Surface

| Need | Use |
|---|---|
| New Go tool used by agents and Nanogo | `pkg/ops.Operation` |
| Existing CLI-only command | `pkg/cli.Command` |
| Existing shell script | `toolkit-fmt` |
| Direct internal API access | `pkg/transport.HTTPTransport` |
| Isolated internal API access | file-queue transport plus `toolkit-worker` |

Prefer `pkg/ops.Operation` for new tools. It keeps invocation independent of stdout, argv, terminal formatting, and presentation footers.

## Operation Checklist

- Set `Tool`, `Name`, and `Description`.
- Define `InputSchema` as JSON object schema.
- Mark `ReadOnly` accurately.
- Add `Examples` with safe sample args.
- Add `SafetyNotes`.
- Implement `Invoke(ctx,args)` and return an `ops.Result`.
- Test raw `Operation.Call`.
- If exposing through CLI, test the `opcli` adapter.
- If exposing through Nanogo, test the `nanogoops` adapter.
- If adding manifest entries, validate them against operation metadata.

## JSON Schema

Use object schemas with explicit `required` fields and `properties`. The current built-in validation checks required fields and leaves deeper type validation to operation code.

## Read And Write Safety

Read-only operations can be exposed first. Write-capable operations require:

- `Authorization.Required = true`
- non-empty authorization policy
- non-empty audit event name
- safety notes
- tests that fail when authorization/audit metadata is absent

Student, tutor, memory, scheduler, and admin writes must remain unavailable to Nanogo until explicitly approved and tested.

## CLI Exposure

Use `opcli.CommandFromOperations` to expose operations through `toolkit <tool> <operation>`. The CLI adapter converts schema properties into flags and writes raw operation bodies to stdout. `cli.App` still handles global output flags and presentation behavior.

## Nanogo Exposure

Use `nanogoops.Adapt` or mirror that adapter in Nanogo. The adapter exposes:

- `Name() string`
- `Schema() json.RawMessage`
- `Call(ctx,args) (string,error)`

Direct Nanogo calls should use the adapter instead of shelling out. Shell execution with `toolkit --raw ...` remains a fallback for legacy tools.

## Manifests

Manifest command examples and permissions should map to operation metadata. Use `ops.ValidateManifest` in tests to catch drift between operation names and manifest command strings.
