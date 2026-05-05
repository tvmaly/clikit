# clikit JSON Contract v1

This document defines `clikit` contracts that external agents and workers may rely on. The contract version is `clikit.contract/v1`.

## Structured Error Contract

Structured errors are compact JSON objects written to stdout.

Required stable fields:

- `error`: human-readable failure summary.
- `suggestion`: next useful command or corrective action.
- `available`: optional list of valid choices.
- `retry`: boolean retry signal.
- `exit_code`: process exit code.

Consumers may rely on those field names. New optional fields may be added without changing the version.

## Queue Request Metadata

Queue request metadata is written to `pending/<request_id>.meta.json`.

Stable fields include `request_id`, `tool`, `operation`, `method`, `path`, `params`, `has_payload`, `payload_path`, `response_meta_path`, `response_payload_path`, `timeout_seconds`, `created_at`, and `expected_response`.

Request metadata must not contain bearer tokens, API keys, environment values, or other credentials.

## Queue Response Metadata

Queue response metadata is written to `done/<request_id>.meta.json` or `dead/<request_id>.meta.json`.

Stable fields include `request_id`, `status`, `status_code`, `has_payload`, `payload_path`, `error`, `suggestion`, `retry`, `exit_code`, `worker_id`, `claimed_at`, `completed_at`, and `duration_ms`.

The `status` value is `success` or `error`. The `retry` field tells agents whether repeating the same operation is reasonable.

## Command Output Conventions

Default command output is compact JSON. List output uses JSONL, one object per line. Human-readable pretty output is opt-in. Piped output and `--raw` output do not include presentation-only metadata.

Direct terminal or agent output may include a metadata footer in this format:

```text
[exit:N | Xms]
```

For durations of one second or more, the footer uses seconds with one decimal place.

## JSONL List Entries

Each JSONL line must be a complete JSON object. Agents should parse each line independently and should not require a surrounding array.

## Breaking Changes

The contract version must change before any of these changes:

- Removing or renaming structured error fields.
- Changing queue lifecycle directory names.
- Changing `retry` semantics.
- Changing compact JSON as the default output.
- Changing JSONL list output into arrays by default.
- Changing metadata footer format.

Backward-compatible additions may add optional fields, new manifest examples, or new event names.
