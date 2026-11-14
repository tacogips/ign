# Implement

You are working in the `ign` repository.

Requested work:

```text
{{workflowInput.requestedWork}}
```

Implement the plan from the prior step. The required end state is:

- `ign init` can receive template variables non-interactively through CLI options.
- `ign checkout` can receive template variables non-interactively through CLI options.
- Interactive prompts remain available when required variables are not supplied.
- Repeated variable options and key-value input should be supported using existing CLI conventions where possible.
- Validation rejects malformed variable assignments.
- Tests cover both commands, repeated values, prompt fallback, and malformed values.
- User-facing documentation explains the new non-interactive usage.

Follow Go conventions, run `gofmt`, and preserve unrelated worktree changes.
