---
name: clikit-toolkit
description: Hermes skill for bounded read-only toolkit access.
allowed-tools: [Bash(toolkit --help), Bash(toolkit * --help), Bash(toolkit * list), Bash(toolkit * status), Bash(toolkit * inspect)]
---

# Hermes clikit Skill

Start with discovery:

```bash
toolkit --help
```

Then inspect a specific command before execution:

```bash
toolkit queue status --help
```

Parse compact JSON by default and JSONL line by line for lists. On errors, read `retry` and `suggestion`. If `retry` is true, retry the same read operation after a short delay. If `retry` is false, follow `suggestion` and stay inside the allowed-tools list.

This skill intentionally avoids unrestricted shell execution and production write commands.
