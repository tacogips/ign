---
description: Automatically select and execute parallelizable tasks from implementation plan(s)
argument-hint: "[plan-path]"
---

## Execute Implementation Plan (Auto-Select) Command

This command **automatically analyzes** implementation plans via `impl-plans/PROGRESS.json` and selects tasks that can be executed based on:
- Task status (Not Started)
- Dependency satisfaction (all dependencies completed)
- Cross-plan dependencies (phase-based ordering)
- Parallelization markers (Parallelizable: Yes)

**IMPORTANT**: Uses PROGRESS.json (~2K tokens) instead of reading all plan files (~200K+ tokens) to prevent context overflow.

For executing specific tasks by ID, use `/impl-exec-specific` instead.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `impl-exec-auto` subagent using the Task tool.

### Argument Parsing

Parse `$ARGUMENTS`:

1. **If no argument provided**: Analyze ALL active plans and auto-select executable tasks across plans
2. **If plan path provided**: Focus on that specific plan only
   - Can be relative: `impl-plans/foundation-and-core.md`
   - Can be short name: `foundation-and-core` (auto-resolves to `impl-plans/foundation-and-core.md`)
3. **If `--dry-run` flag present**: Analyze and report but do not execute

### Path Resolution

If plan path does not contain `/`:
- Assume it's a short name
- Resolve to: `impl-plans/<name>.md`

Examples:
- `foundation-and-core` -> `impl-plans/foundation-and-core.md`
- `impl-plans/session-groups.md` -> use as-is
- (no argument) -> analyze all plans via `impl-plans/PROGRESS.json`

### Invoke Subagent

**When no argument provided (cross-plan mode)**:
```
Task tool parameters:
  subagent_type: impl-exec-auto
  prompt: |
    Mode: cross-plan auto-select
    Analyze ALL plans via impl-plans/PROGRESS.json
    Respect cross-plan dependencies from PROGRESS.json phases
```

**When plan path provided (single-plan mode)**:
```
Task tool parameters:
  subagent_type: impl-exec-auto
  prompt: |
    Implementation Plan: <resolved-plan-path>
    Mode: single-plan auto-select parallelizable tasks
```

### Usage Examples

**Execute across ALL plans (recommended)**:
```
/impl-exec-auto
```
Analyzes all active plans, finds all tasks that:
- Belong to plans whose phase dependencies are satisfied
- Have status "Not Started"
- Have all task-level dependencies satisfied
- Are marked as parallelizable

Then executes them sequentially using Claude subtasks.

**Execute within a specific plan**:
```
/impl-exec-auto foundation-and-core
```
Focuses on tasks within the specified plan only.

**Dry run (preview without executing)**:
```
/impl-exec-auto --dry-run
/impl-exec-auto foundation-and-core --dry-run
```

### What the Subagent Does (Analysis Only)

**IMPORTANT**: The `impl-exec-auto` subagent is **analysis-only**. It does NOT execute tasks - it returns a structured list of executable tasks. The main conversation then uses `impl-exec-specific` to execute them.

#### Cross-Plan Mode (no argument)

1. **Reads impl-plans/PROGRESS.json** (~2K tokens) for phase/task status
2. **Determines phase eligibility** from PROGRESS.json phases:
   - Phase 1: Check if COMPLETED
   - Phase 2: READY when Phase 1 is COMPLETED
   - Phase 3: BLOCKED until Phase 2 critical tasks complete
   - Phase 4: BLOCKED until Phase 3 is COMPLETED
3. **Reads ONLY plan files for executable tasks** (not all plans)
4. **Returns structured task list** to main conversation

#### Single-Plan Mode (with argument)

1. **Reads the implementation plan file**
2. **Builds dependency graph** from task definitions
3. **Identifies executable tasks** within that plan only
4. **Returns structured task list** to main conversation

### Main Conversation Orchestration

After receiving the executable tasks list from `impl-exec-auto`, the main conversation:

1. **Groups tasks by plan name**
2. **For each plan with executable tasks**:
   - Invokes `impl-exec-specific` with task IDs
   - Example: `/impl-exec-specific session-groups-runner TASK-008`
3. **impl-exec-specific handles internally**:
   - ts-coding spawning
   - check-and-test-after-modify
   - ts-review cycle (up to 3 iterations)
   - Plan file status updates
4. **Main conversation updates PROGRESS.json** (with lock) after each plan
5. **Reports completion and newly unblocked tasks**

**Why this architecture?**
- impl-exec-auto cannot spawn subagents (Claude Code limitation)
- impl-exec-specific has the full implementation cycle logic
- Main conversation coordinates between the two

### Cross-Plan Dependencies (from impl-plans/README.md)

```
Phase 1: foundation-and-core (no dependencies)
    |
    v
Phase 2: session-groups, command-queue, markdown-parser,
         realtime-monitoring, bookmarks, file-changes
    |    (can run in parallel)
    v
Phase 3: daemon-and-http-api
    |
    v
Phase 4: browser-viewer, cli
```

### Error Handling

**If no executable tasks across all plans**:
```
No executable tasks found across all active plans.

Current status by phase:

Phase 1:
- foundation-and-core: In Progress (X/Y tasks)
  - In Progress: TASK-001, TASK-002
  - Blocked: TASK-003 (waiting on TASK-001)

Phase 2: (blocked by Phase 1)
- session-groups: Blocked (waiting on foundation-and-core)
- command-queue: Blocked (waiting on foundation-and-core)
...

Recommended Actions:
1. Wait for in-progress tasks to complete
2. Use /impl-exec-specific to run specific tasks
```

### After impl-exec-auto Analysis Completes

1. **Parse executable tasks** from the subagent output
2. **Group tasks by plan name**
3. **Execute via impl-exec-specific** for each plan:
   ```
   /impl-exec-specific <plan-name> <TASK-IDs>
   ```
4. **Update PROGRESS.json** after each impl-exec-specific completes
5. **Report results**:
   - Tasks completed successfully
   - Tasks failed (if any)
   - Tasks/Plans now unblocked
6. **If more tasks available**:
   - List next executable tasks/plans
   - Suggest re-running `/impl-exec-auto`
7. **If a plan completed**:
   - Confirm plan status updated to "Completed" in PROGRESS.json
   - Note newly unblocked plans
8. **If all plans completed**:
   - Report implementation completion
