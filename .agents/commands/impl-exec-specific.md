---
description: Execute specific tasks from an implementation plan by task ID
argument-hint: "<plan-path> <task-ids...>"
---

## Execute Specific Implementation Tasks Command

This command executes **specific tasks by ID** from an implementation plan. Use this when you want to explicitly choose which tasks to run.

For automatic task selection based on dependencies and parallelization, use `/impl-exec-auto` instead.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `impl-exec-specific` subagent using the Task tool.

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **Plan Path** (required): Path to implementation plan
   - Can be relative: `impl-plans/foundation-and-core.md`
   - Can be short name: `foundation-and-core` (auto-resolves to `impl-plans/foundation-and-core.md`)

2. **Task IDs** (required): Specific tasks to execute
   - Space-separated: `TASK-001 TASK-002 TASK-003`

### Path Resolution

If plan path does not contain `/`:
- Assume it's a short name
- Resolve to: `impl-plans/<name>.md`

Examples:
- `foundation-and-core` -> `impl-plans/foundation-and-core.md`
- `impl-plans/session-groups.md` -> use as-is

### Invoke Subagent

```
Task tool parameters:
  subagent_type: impl-exec-specific
  prompt: |
    Implementation Plan: <resolved-plan-path>
    Task IDs: <task-ids as comma-separated list>
    Execution Mode: parallel (if tasks are parallelizable)
```

### Usage Examples

**Execute specific tasks**:
```
/impl-exec-specific foundation-and-core TASK-001 TASK-002
```
Executes only the specified tasks (in parallel if possible).

**Execute single task**:
```
/impl-exec-specific foundation-and-core TASK-005
```

**Execute with full path**:
```
/impl-exec-specific impl-plans/session-groups.md TASK-001
```

### Error Handling

**If no task IDs provided**:
```
Usage: /impl-exec-specific <plan-path> <task-ids...>

This command requires explicit task IDs.

Examples:
  /impl-exec-specific foundation-and-core TASK-001
  /impl-exec-specific foundation-and-core TASK-001 TASK-002 TASK-003

For automatic task selection, use:
  /impl-exec-auto foundation-and-core
```

**If plan not found**:
```
Error: Implementation plan not found: <plan-path>

Searched locations:
  - impl-plans/<plan-path>
  - impl-plans/<plan-path>.md
  - <plan-path>

Available plans:
  (list plans from impl-plans/)
```

**If task not found**:
```
Error: Task not found in plan: <task-id>

Available tasks in <plan-name>:
  (list tasks with status)
```

### After Subagent Completes

1. Report execution results:
   - Tasks completed
   - Tasks failed (if any)
   - Tasks now available (unblocked by completed tasks)

2. Show updated plan status:
   - Overall progress (X/Y tasks completed)
   - Next executable tasks

3. **Update PROGRESS.json** (with lock):
   - Acquire lock: `while [ -f impl-plans/.progress.lock ]; do sleep 1; done && echo "<plan>:<task>" > impl-plans/.progress.lock`
   - Change completed task status from "Not Started" to "Completed"
   - Update `lastUpdated` timestamp
   - Release lock: `rm -f impl-plans/.progress.lock`

4. If plan completed:
   - Confirm plan status updated to "Completed" in PROGRESS.json
   - Suggest next implementation plan
