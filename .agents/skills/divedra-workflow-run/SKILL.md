---
name: divedra-workflow-run
description: Use when helping an end user run, inspect, monitor, resume, continue, rerun, or troubleshoot existing divedra workflows. Applies to workflow list/status/validate/inspect/run, session progress/status/resume/continue/rerun, GraphQL inspection, mock scenarios, workflow roots, runtime artifacts, local serve, GraphQL endpoint execution, and event-triggered workflow usage.
metadata:
  short-description: Run divedra workflows
---

# Divedra Workflow Run

Use this skill for operating existing divedra workflows. For creating or editing workflow bundles, use `divedra-workflow` instead.

## First Decision

Identify what the user wants:

- Find workflows for human overview: use `workflow list` or `workflow status`.
- Choose a workflow for AI/tool use: use `workflow usage --output json` first.
- Check a workflow before running: use `workflow validate` and optionally `workflow inspect`.
- Run locally: use `workflow run`; for important or long-running work, prefer supervised execution with `--auto-improve`.
- Run deterministically without real agents: add `--mock-scenario`.
- Monitor an existing run: use `session progress`, `session status`, or GraphQL detail queries.
- Continue a run: use `session resume`.
- Start from a specific step in an existing session: use `session rerun <session-id> <step-id>`.
- Continue from a concrete prior step-run artifact boundary: use `session continue <session-id> --start-step <step-id> --after-step-run <step-run-id>`.
- Inspect merged execution history, logs, and export-shaped diagnostic payloads through GraphQL.
- Use a remote control plane: start or target `serve` and pass `--endpoint`.
- Debug one step locally: use `call-step`.

Read `references/runbook.md` when the task involves sessions, remote endpoints, mock scenarios, event dispatch, auto-improve, or troubleshooting.

## Command Forms

Inside the divedra source repo, prefer:

```bash
bun run src/main.ts <command>
```

When divedra is installed, prefer:

```bash
divedra <command>
```

Use `--workflow-definition-dir <path>` when the workflow definition bundles are not coming from scoped project/user lookup. This option points at a directory containing `<workflow-name>/workflow.json` bundles; it does not control logs, sessions, or artifacts. In this repository, examples use `--workflow-definition-dir ./examples`.

## Standard Run Sequence

For LLM-driven workflow selection, start here:

```bash
bun run src/main.ts workflow usage --workflow-definition-dir ./examples --output json
```

Read each workflow's:

- `description`
- `callable.stepId`
- `callable.role`
- `callable.input`
- `callable.output`
- `steps`

Choose the workflow whose purpose and callable contract match the user's task.
Use `steps` only as a compact stage overview; use `workflow inspect` when
structural debugging detail is needed.

```bash
bun run src/main.ts workflow list --workflow-definition-dir ./examples
```

```bash
bun run src/main.ts workflow validate <workflow-name> --workflow-definition-dir ./examples
```

```bash
bun run src/main.ts workflow inspect <workflow-name> \
  --workflow-definition-dir ./examples \
  --output json
```

```bash
bun run src/main.ts workflow run <workflow-name> \
  --workflow-definition-dir ./examples \
  --output json
```

Recommended supervised execution:

```bash
bun run src/main.ts workflow run <workflow-name> \
  --workflow-definition-dir ./examples \
  --auto-improve \
  --nested-supervisor \
  --max-supervised-attempts 3 \
  --workflow-mutation-mode execution-copy \
  --output json
```

Use this recommended path when the workflow may need retries, stall detection, remediation, or a supervisor workflow to drive recovery. Use plain `workflow run` for quick local checks, deterministic mock runs, or cases where supervision is intentionally disabled.

For deterministic local testing:

```bash
bun run src/main.ts workflow run <workflow-name> \
  --workflow-definition-dir ./examples \
  --mock-scenario ./examples/<workflow-name>/mock-scenario.json \
  --output json
```

## Session Operations

After `workflow run`, capture the returned `sessionId`.

```bash
bun run src/main.ts session progress <session-id>
```

```bash
bun run src/main.ts session status <session-id> --output json
```

```bash
bun run src/main.ts session resume <session-id>
```

```bash
bun run src/main.ts session rerun <session-id> <step-id>
```

```bash
bun run src/main.ts session continue <session-id> \
  --start-step <step-id> \
  --after-step-run <step-run-id>
```

## User-Safe Defaults

- Validate before running unless the user explicitly asks to skip validation.
- For AI-guided workflow selection, prefer `workflow usage --output json` before `workflow run`.
- Prefer supervised execution with `--auto-improve --nested-supervisor` for real work where failure recovery matters.
- Prefer `--output json` when the result will be parsed, saved, or compared.
- Prefer `--mock-scenario` for demos, tests, and docs because it avoids real backend calls.
- Do not combine `--mock-scenario` with `--endpoint`; mock scenarios are local-only.
- Use `--working-dir` when workflow execution must happen relative to a specific project directory.
- Use `--artifact-root` and `--session-store` when the user wants isolated runtime state.
- Use `session rerun` when restarting from a step with variables only; use `session continue` only when intentionally importing history up to a concrete prior step-run.

## Remote And Server Use

Start the local control plane:

```bash
bun run src/main.ts serve --workflow-definition-dir ./examples
```

Then target it:

```bash
bun run src/main.ts workflow run <workflow-name> \
  --workflow-definition-dir ./examples \
  --endpoint http://127.0.0.1:43173/graphql \
  --output json
```

Remote-capable operations include `workflow run`, `session resume`, and `session rerun`. `call-step` and `session continue` remain local-only; detailed diagnostics are GraphQL queries rather than separate session subcommands.

## Troubleshooting

- If a workflow is not found, check `--workflow-definition-dir`, `DIVEDRA_WORKFLOW_DEFINITION_DIR`, and scope lookup.
- If an AI cannot tell how to call a workflow, run `workflow usage --output json` and inspect the description, compact `steps`, and callable input/output contract.
- If validation fails, fix the workflow bundle before running; do not bypass schema errors for normal usage.
- If a run fails, inspect `session status`, `session progress`, and GraphQL detail queries.
- If backend calls should not happen, rerun with `--mock-scenario` or `--dry-run` when appropriate.
- If paths are surprising, check `--working-dir`, `--artifact-root`, `--session-store`, and the command invocation directory.
