# Intake And Plan

You are working in the `ign` repository.

Requested work:

```text
{{workflowInput.requestedWork}}
```

Acceptance criteria:

```json
{{workflowInput.acceptanceCriteria}}
```

Constraints:

```json
{{workflowInput.constraints}}
```

Inspect the current `ign init` and `ign checkout` behavior. Determine whether non-interactive variable passing already exists. If it does not, produce a concise design and implementation plan that fits the existing Go CLI architecture, test layout, and documentation style.

Return:

- current behavior evidence
- target CLI surface
- files likely to change
- verification commands to run
