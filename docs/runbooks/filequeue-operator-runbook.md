# File-Queue Operator Runbook

This runbook covers `clikit` file-queue transport operations.

## Queue Directory Layout

`QueueRoot` contains these directories:

- `pending`: request metadata waiting for a worker.
- `claimed`: request metadata currently owned by a worker.
- `done`: completed response metadata.
- `dead`: malformed or failed request metadata.
- `payloads/req`: request bodies.
- `payloads/resp`: response bodies.
- `tmp`: temporary files used for write-to-temp-then-rename.

All queue directories must be on the same filesystem or mount. `clikit` relies on same-filesystem atomic rename semantics.

## Lifecycle

Requests move from `pending` to `claimed` to `done` or `dead`. Workers claim by atomic rename. Response files are written to temp files and then renamed so readers never see partial files.

## Worker Placement

Run `toolkit-worker` on the internal-network side that can reach the target APIs. Agent-side machines need only shared filesystem access to `QueueRoot`.

## Secrets

Request metadata and payload files must not contain secrets. The worker resolves bearer tokens from its own environment through handler configuration. Rotate tokens through the worker environment or secret manager used to launch the worker.

## Timeouts and Polling

Configure request timeout, poll interval, stale claim threshold, done retention, dead retention, and cleanup interval from `toolkit-worker` flags. Keep heartbeat frequency lower than the stale claim threshold.

## Stale Claim Recovery

If a worker exits while holding a request, stale claim recovery moves old claimed requests back to `pending` after the heartbeat exceeds the stale claim threshold.

## Cleanup

Cleanup removes expired `done`, `dead`, response payload, request payload, and old temp files. Cleanup is safe to run while workers are active.

## Operational Checks

- Count files in `pending`, `claimed`, `done`, and `dead`.
- Inspect dead-letter metadata for `error`, `suggestion`, and `retry`.
- Check heartbeat age for claimed requests.
- Confirm queue root is writable by agent-side clients and workers.
- Confirm the worker has API network access and token environment variables.

## Startup

```bash
toolkit-worker \
  --queue-root /mnt/shared/clikit \
  --handler-config ./handlers.json \
  --worker-id worker-1
```

## Shutdown

Send SIGINT or SIGTERM. The worker stops polling and exits after the active loop observes cancellation.

## Troubleshooting

Large `pending` backlog usually means no worker is running or the worker cannot access the queue root. Large `claimed` backlog usually means stale claim recovery is misconfigured or workers are blocked on APIs. Large `dead` backlog means malformed requests, handler mismatches, or non-retryable API failures.
