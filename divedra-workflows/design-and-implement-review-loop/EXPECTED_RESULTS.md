# Expected Results

This workflow is adapted for this repository as a live Codex-agent loop.

## Validate

```bash
divedra workflow validate design-and-implement-review-loop --workflow-definition-dir ./divedra-workflows
```

Expected result: the workflow definition is valid.

## Run

```bash
divedra workflow run design-and-implement-review-loop --workflow-definition-dir ./divedra-workflows
```

Expected behavior:

- Step 1 normalizes the request into either `issue-resolution` or `design-plan-only` mode.
- Step 2 updates design documentation under `design-docs/`.
- Step 3 reviews the design and routes back to Step 2 when `needs_revision` is true.
- Step 4 creates or revises implementation plans under `impl-plans/`.
- Step 5 reviews the implementation plan and routes back to Step 2 or Step 4 as needed.
- Step 6 implements only in `issue-resolution` mode.
- Step 7 reviews the implementation and routes back to Step 6 when high or mid findings exist.
- Step 8 refreshes user-facing documentation after implementation acceptance.
- `workflow-output` emits the accepted planning handoff or final issue-resolution handoff.

The workflow does not commit or push changes automatically.
