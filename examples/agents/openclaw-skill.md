# OpenClaw clikit Host Tool

Use this as an allowlist-oriented OpenClaw workspace tool example.

Allowlist:

- `toolkit --help`
- `toolkit * --help`
- `toolkit * list`
- `toolkit * status`
- `toolkit * inspect`

Start with discovery:

```bash
toolkit --help
```

Parse compact JSON by default. Parse JSONL one line at a time for list output. When a structured error appears, inspect `retry` and `suggestion`. Retry only read operations. Follow `suggestion` without leaving the allowlist.

Do not grant unrestricted shell access for internal systems. Keep credentials on the worker side when file-queue mode is used.
