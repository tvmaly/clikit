# nanogo clikit Example

This example shows how `nanogo` can consume `clikit` as a downstream read-only tool surface. It is not a core dependency of `clikit`.

Example commands:

```bash
toolkit nanogo tutor-status --student-id sample-student
toolkit nanogo admin-summary --limit 10
```

The command output should be compact JSON by default so tutors, admins, and agents can parse it without extra formatting steps.

Privacy boundaries:

- Use sample or fixture student identifiers in examples.
- Do not expose real student records through broad list commands.
- Keep tutor, memory, scheduler, and admin data read-only until write operations have explicit authorization and audit rules.
- Avoid leaking local memory, chat history, or scheduling context in generic status commands.

The sample output in `sample-output.json` is a one-line JSON fixture for validation.
