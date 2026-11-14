---
name: exec-impl-plan-ref
description: Use when executing tasks from implementation plans. Provides task selection, parallel execution, progress tracking, and review cycle guidelines.
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, Task, TaskOutput
---

# Implementation Execution Skill

This skill provides guidelines for executing implementation plans created by the `impl-plan` agent.

## When to Apply

Apply this skill when:
- Executing tasks from an implementation plan in `impl-plans/`
- Tracking progress during multi-session implementation work
- Coordinating sequential execution of tasks
- Updating implementation plan status and progress logs

## Purpose

This skill bridges implementation plans (what to build) and actual code implementation. It provides:
- Task selection based on dependencies and parallelization markers
- Execution via Claude subtasks (parallel or sequential)
- Progress tracking via PROGRESS.json and plan updates
- Completion verification and plan finalization

## CRITICAL: Use PROGRESS.json to Prevent Context Overflow

**NEVER read all plan files at once.** This causes context overflow (>200K tokens).

Use `impl-plans/PROGRESS.json` instead, which contains:
- Phase status (COMPLETED, READY, BLOCKED)
- All task statuses across all plans (~2K tokens)
- Task dependencies

**Workflow**:
1. Read `PROGRESS.json` (~2K tokens) to find executable tasks
2. Read ONLY the specific plan file when executing a task (~10K tokens)
3. After execution:
   - `impl-exec-specific` updates the **plan file** immediately
   - **Main conversation** updates `PROGRESS.json` after impl-exec-specific completes

## Responsibility Split: Plan File vs PROGRESS.json

**Clear ownership to avoid conflicts**:
- `impl-exec-specific` agent: Updates **plan file** (task status, progress log)
- Main conversation: Updates **PROGRESS.json** (after impl-exec-specific completes)

### File Locking Protocol (for main conversation PROGRESS.json updates)

Lock file path: `impl-plans/.progress.lock`

**Before updating PROGRESS.json**:
```bash
# Acquire lock (retry if busy)
while [ -f impl-plans/.progress.lock ]; do sleep 1; done
echo "<plan-name>:<TASK-ID>" > impl-plans/.progress.lock
```

**After updating PROGRESS.json**:
```bash
# Release lock
rm -f impl-plans/.progress.lock
```

This prevents race conditions when multiple conversations update concurrently.

### PROGRESS.json Structure

```json
{
  "lastUpdated": "2026-01-06T16:00:00Z",
  "phases": {
    "1": { "status": "COMPLETED" },
    "2": { "status": "READY" },
    "3": { "status": "BLOCKED" },
    "4": { "status": "BLOCKED" }
  },
  "plans": {
    "session-groups-types": {
      "phase": 2,
      "status": "Ready",
      "tasks": {
        "TASK-001": { "status": "Not Started", "parallelizable": true, "deps": [] },
        "TASK-002": { "status": "Not Started", "parallelizable": true, "deps": [] },
        "TASK-007": { "status": "Not Started", "parallelizable": false, "deps": ["TASK-001"] }
      }
    }
  }
}
```

### Dependency Format

- Same-plan dependency: `"TASK-001"` (task in same plan)
- Cross-plan dependency: `"session-groups-types:TASK-001"` (task in different plan)

## Execution Modes

Two execution modes are available:

### Auto Mode (`impl-exec-auto`)

**Architecture**: Analysis-only subagent + Main conversation orchestration via `impl-exec-specific`.

Claude Code does not support nested subagent spawning (subagents cannot use Task tool). Therefore:
- `impl-exec-auto` agent: Analyzes plans and returns executable task list
- Main conversation: **Invokes `impl-exec-specific`** to execute tasks (NOT direct ts-coding spawning)

**Cross-Plan Mode** (no argument - recommended):
```bash
/impl-exec-auto
```
Analyzes ALL active plans and returns executable tasks across plans.

**Single-Plan Mode** (with argument):
```bash
/impl-exec-auto foundation-and-core
```
Focuses on one specific plan only.

Use this mode when:
- Starting implementation (cross-plan mode)
- Continuing work after completing some tasks
- You want automatic task selection across the entire project

**The auto mode workflow**:

Step 1: `impl-exec-auto` agent (analysis only):
1. Reads `impl-plans/PROGRESS.json`
2. Identifies executable tasks (deps satisfied, status "Not Started")
3. Reads plan files for task details
4. Returns structured task list to main conversation

Step 2: Main conversation (orchestration via impl-exec-specific):
1. Parses the executable tasks list from impl-exec-auto
2. Groups tasks by plan
3. For each plan with executable tasks:
   a. Invokes `impl-exec-specific` with task IDs
   b. `impl-exec-specific` handles: ts-coding, check-and-test, ts-review cycle, plan updates
4. After `impl-exec-specific` completes:
   a. Updates PROGRESS.json status (with lock)
   b. Reports completion and newly unblocked tasks
5. Repeats for remaining plans/tasks

**Why impl-exec-specific?**
- Main conversation lacks review cycle logic and plan update format
- impl-exec-auto cannot spawn subagents (Claude Code limitation)
- impl-exec-specific handles the full implementation cycle

### Specific Mode (`impl-exec-specific`)

Executes specific tasks by ID:

```bash
/impl-exec-specific foundation-and-core TASK-001 TASK-002
```

Use this mode when:
- Re-running a failed task
- Testing a specific implementation
- You know exactly which tasks to run

## Execution Workflow

### Phase 1: Plan Analysis

1. **Read the implementation plan**: Load from `impl-plans/<plan-name>.md`
2. **Parse task status**: Identify tasks by status (Not Started, In Progress, Completed)
3. **Build dependency graph**: Understand which tasks depend on others
4. **Select executable tasks**: Tasks with status "Not Started" and all dependencies satisfied

### Phase 2: Task Selection Strategy

Select tasks for execution based on:

| Criterion | Priority | Rationale |
|-----------|----------|-----------|
| Dependencies satisfied | Required | Cannot start blocked tasks |
| Marked as parallelizable | High | Can run concurrently |
| Small estimated effort | Medium | Quick wins build momentum |
| Foundation/core tasks | High | Unblock other tasks |

### Phase 3: Sequential Execution

Execute tasks one at a time to avoid LLM errors:

1. **Spawn ts-coding agent** for first task (WITHOUT `run_in_background`)
2. **Provide complete context** to the agent:
   - Purpose: From task description
   - Reference Document: The implementation plan path
   - Implementation Target: Deliverables from the task
   - Completion Criteria: From task completion criteria
3. **Wait for completion** before proceeding
4. **Run tests** (check-and-test-after-modify)
5. **Run review cycle** (ts-review, up to 3 iterations)
6. **Update task status** in plan
7. **Repeat** for next task

### Phase 4: Progress Update (IMMEDIATELY After Each Task)

**CRITICAL**: Execute this phase IMMEDIATELY after each task completes. DO NOT batch updates.

**Responsibility Split (when using impl-exec-specific)**:
- `impl-exec-specific` agent: Updates **plan file** only
- Main conversation: Updates **PROGRESS.json** after impl-exec-specific completes

After task execution (within impl-exec-specific):

1. **Update task status in the plan file**:
   - Not Started -> In Progress (when started)
   - In Progress -> Completed (when all criteria met)

2. **Add progress log entry** to plan file:
   ```markdown
   ### Session: YYYY-MM-DD HH:MM
   **Tasks Completed**: TASK-001, TASK-002
   **Tasks In Progress**: TASK-003
   **Blockers**: None
   **Notes**: Implementation notes and decisions made
   ```

3. **Check completion criteria** for the overall plan

4. **Update plan status** in PROGRESS.json if all tasks done (set to "Completed")

**NOTE**: PROGRESS.json is updated by **main conversation** after impl-exec-specific completes. See "Main Conversation Orchestration Protocol" section for details.

**No file move is required.** PROGRESS.json is the single source of truth for plan status.

## Task Invocation Format

When invoking the `ts-coding` agent for a task:

```
Task tool parameters:
  subagent_type: ts-coding
  prompt: |
    Purpose: <task description from implementation plan>
    Reference Document: impl-plans/<plan-name>.md
    Implementation Target: <deliverables list>
    Completion Criteria:
      - <criterion 1 from task>
      - <criterion 2 from task>
      - <criterion N from task>
  run_in_background: true  # For parallel execution of independent tasks
```

**Note**: Use `run_in_background: true` only when executing multiple independent tasks in parallel. Omit for sequential execution.

### Extracting Task Information

Extract prompt content from the task structure in the implementation plan:

```markdown
### TASK-001: Core Interfaces

**Status**: Not Started
**Parallelizable**: Yes
**Deliverables**:        <- Use for Implementation Target
- `src/interfaces/filesystem.ts`
- `src/interfaces/process-manager.ts`

**Description**:         <- Use for Purpose
Define all core interfaces for abstracting external dependencies.

**Completion Criteria**: <- Use for Completion Criteria
- [ ] FileSystem interface defined
- [ ] ProcessManager interface defined
- [ ] Type checking passes
```

## Sequential Execution Pattern

**CRITICAL**: Execute tasks ONE AT A TIME to avoid LLM errors.

```
For each task in the executable tasks list:

1. Invoke ts-coding (without run_in_background):
   Task tool parameters:
     subagent_type: ts-coding
     prompt: |
       Purpose: <TASK-001 description>
       Reference Document: <implementation-plan-path>
       Implementation Target: <TASK-001 deliverables>
       Completion Criteria:
         - <criterion 1>
         - <criterion 2>

2. Wait for ts-coding to complete

3. Run check-and-test-after-modify:
   Task tool parameters:
     subagent_type: check-and-test-after-modify
     prompt: |
       Verify changes for TASK-001

4. Run ts-review (up to 3 iterations):
   Task tool parameters:
     subagent_type: ts-review
     prompt: |
       Review TASK-001 implementation

5. Update task status in plan

6. Proceed to TASK-002 (repeat steps 1-5)
```

**All tasks run sequentially** - each task completes fully (including review) before the next begins.

## Result Handling

After each task completes:

1. Parse the task's result (success/failure)
2. Verify completion criteria are met
3. Update task status immediately
4. Record any issues or partial completion
5. Proceed to next task

## Dependency Detection

Parse dependencies from plan:
```markdown
**Parallelizable**: No (depends on TASK-001)
```
or
```markdown
**Parallelizable**: No (depends on TASK-001, TASK-002)
```

## Dependency Resolution

### Dependency Types

| Type | Example | Resolution |
|------|---------|------------|
| **Data dependency** | Types must exist before using them | Execute sequentially |
| **File dependency** | Interface before implementation | Execute sequentially |
| **None** | Independent modules | Execute in parallel |

### Dependency Graph Example

```
TASK-001 (Interfaces)     TASK-002 (Errors)     TASK-003 (Types)
    |                          |                     |
    +----------+---------------+                     |
               |                                     |
    TASK-004 (Mocks)                        TASK-007 (Repo Interfaces)
```

From this graph:
- TASK-001, TASK-002, TASK-003 can run in parallel (Group A)
- TASK-004 must wait for TASK-001
- TASK-007 must wait for TASK-003

## Execution Order Rules

### All Tasks Execute Sequentially

**NOTE**: To avoid LLM errors, all tasks now execute sequentially regardless of parallelization markers.

The "Parallelizable: Yes" marker is still used to:
- Identify tasks with no dependencies (can run in any order)
- Track which tasks would theoretically be safe to parallelize

### Dependency Order

When multiple tasks are executable, prefer this order:
1. Foundation/core tasks first (unblock other tasks)
2. Tasks with no dependencies
3. Tasks with fewer downstream dependents

## Progress Tracking Format

### Task Status Values

| Status | Meaning |
|--------|---------|
| `Not Started` | Task not yet begun |
| `In Progress` | Currently being implemented |
| `Completed` | All completion criteria met |
| `Blocked` | Waiting on dependencies |

### Module Status Table

```markdown
## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Core Interfaces | `src/interfaces/*.ts` | Completed | Pass |
| Error Types | `src/errors.ts` | In Progress | - |
| Mock Implementations | `src/test/mocks/*.ts` | Not Started | - |
```

### Progress Log Entry

```markdown
### Session: 2026-01-04 14:30
**Tasks Completed**: TASK-001, TASK-002
**Tasks Started**: TASK-004
**Blockers**: None
**Notes**:
- Defined FileSystem, ProcessManager, Clock interfaces
- Added Result type with ok/err helpers
- Discovered need for additional WatchOptions type
```

## Completion Verification

### Per-Task Completion

A task is complete when:
- [ ] All deliverable files exist
- [ ] All completion criteria checkboxes can be checked
- [ ] Type checking passes (`bun run typecheck`)
- [ ] Tests pass (if tests are part of criteria)

### Per-Plan Completion

A plan is complete when:
- [ ] All tasks have status "Completed"
- [ ] Overall completion criteria are met
- [ ] Final type check passes
- [ ] Final test run passes

### Plan Finalization

When a plan is complete:

1. Update status header to "Completed" in plan file
2. Update plan status to "Completed" in PROGRESS.json
3. Add final progress log entry
4. Update `impl-plans/README.md` (move plan entry to Completed section if applicable)

**No file move is required.** PROGRESS.json tracks plan completion status.

## Review Cycle

After task implementation and testing, each task goes through a code review cycle using the `ts-review` agent.

### Review Workflow

```
ts-coding agent (implementation)
    |
    v
check-and-test-after-modify agent (tests pass)
    |
    v
ts-review agent (iteration 1)
    |
    +-- APPROVED --> Task complete
    |
    +-- CHANGES_REQUESTED --> ts-coding (fixes) --> check-and-test --> ts-review (iteration 2)
                                                                            |
                                                                            +-- ... (up to 3 iterations)
```

### Maximum Iterations

The review cycle is limited to **3 iterations** per task to prevent infinite loops:

| Iteration | Review Scope | Outcome |
|-----------|--------------|---------|
| 1 | Full comprehensive review | APPROVED or CHANGES_REQUESTED |
| 2 | Focus on previous issues + new issues from fixes | APPROVED or CHANGES_REQUESTED |
| 3 | Critical issues only | APPROVED (with documented remaining issues) |

### Review Agent Invocation

```
Task tool parameters:
  subagent_type: ts-review
  prompt: |
    Design Reference: <path to design document>
    Implementation Plan: impl-plans/<plan-name>.md
    Task ID: TASK-XXX
    Implemented Files:
      - <file path 1>
      - <file path 2>
    Iteration: 1
```

### Re-Review After Fixes

```
Task tool parameters:
  subagent_type: ts-review
  prompt: |
    Design Reference: <path to design document>
    Implementation Plan: impl-plans/<plan-name>.md
    Task ID: TASK-XXX
    Implemented Files:
      - <file path 1>
      - <file path 2>
    Iteration: 2
    Previous Feedback:
      - C1: Missing readonly modifiers
      - S1: Duplicate validation logic
    Focus Areas: readonly modifiers, duplicate validation
```

### Handling Review Results

**If APPROVED**:
1. Mark task as Completed
2. Update completion criteria checkboxes
3. Add review approval to progress log

**If CHANGES_REQUESTED**:
1. Check current iteration number
2. If iteration < 3:
   - Parse issue list from review
   - Invoke ts-coding with fix instructions
   - Run check-and-test
   - Invoke ts-review with iteration + 1
3. If iteration >= 3:
   - Mark task as Completed
   - Document remaining issues in progress log
   - Note: "Approved after max iterations with documented issues"

### Review Feedback to ts-coding

When re-invoking ts-coding to fix review issues:

```
Task tool parameters:
  subagent_type: ts-coding
  prompt: |
    Purpose: Fix code review issues for TASK-XXX
    Reference Document: impl-plans/<plan-name>.md
    Implementation Target: Fix the following review issues

    Issues to Fix:
    - C1 (Critical): src/foo.ts:25 - Missing required method X
      Suggested Fix: Add method X per design spec section Y
    - C2 (Critical): src/bar.ts:42 - Using `any` type
      Suggested Fix: Replace with `unknown` and add type guard
    - S1 (Improvement): src/foo.ts:30,45 - Duplicate validation logic
      Suggested Fix: Extract to shared validateX function

    Completion Criteria:
      - All critical issues (C1, C2) are resolved
      - Improvement suggestions addressed where reasonable
      - Type checking passes
      - Tests pass
```

### Progress Log with Review

```markdown
### Session: 2026-01-04 14:30
**Tasks Completed**: TASK-001
**Review Iterations**: 2
**Review Summary**:
- Iteration 1: 2 critical issues, 1 improvement suggestion
- Iteration 2: APPROVED (all issues resolved)
**Notes**:
- Fixed missing readonly modifiers
- Extracted duplicate validation to shared utility
```

## Error Handling

### Task Failure

If a ts-coding agent fails:

1. Record the failure in progress log
2. Keep task status as "In Progress" (not completed)
3. Document the error and recommended fix
4. Continue with other tasks if possible
5. Report failures to user for manual intervention

### Partial Completion

If only some tasks complete:

1. Update completed task statuses
2. Update progress log with what completed
3. Document blockers for incomplete tasks
4. Report partial progress to user

## Quick Reference

| Action | Tool | Parameters |
|--------|------|------------|
| Read plan | Read | `impl-plans/<plan>.md` |
| Execute task | Task | `subagent_type: ts-coding` (NO run_in_background) |
| Run tests | Task | `subagent_type: check-and-test-after-modify` |
| Review code | Task | `subagent_type: ts-review` |
| Update plan | Edit | Update status, checkboxes, log |
| Update progress | Edit | Update PROGRESS.json task/plan status |

## Common Response Formats

### Plan Completed Response

```
## Implementation Plan Completed

### Plan
`impl-plans/<plan-name>.md`

### Final Verification
- Type checking: Pass
- Tests: Pass (X/X)

### Plan Finalization
- Plan file status updated to: Completed
- PROGRESS.json status updated to: Completed
- README.md updated

### Next Steps
- Review completed implementation
- Consider integration testing
- Proceed to next implementation plan
```

### Partial Failure Response

```
### Failure Details

**TASK-XXX Failure**:
- Error: <error type>
- Details: <specific error message>
- Files affected: <file paths>

### Recommended Actions
1. Review failure details
2. Fix the issue
3. Re-run with: `/impl-exec-specific <plan-name> TASK-XXX`
```

## Important Guidelines

1. **Read this skill first**: Always read this skill before execution
2. **Execute sequentially**: Run tasks ONE AT A TIME to avoid LLM errors
3. **Complete each task fully**: Run ts-coding -> check-and-test -> ts-review before next task
4. **Update plan file IMMEDIATELY**: Update task status in plan file AFTER EACH task
5. **PROGRESS.json updated by main conversation**: Main conversation updates PROGRESS.json (with lock) after impl-exec-specific completes
6. **Fail gracefully**: If a task fails, document it and proceed to next task
7. **Invoke check-and-test**: After ts-coding completes, invoke `check-and-test-after-modify`
8. **Run review cycle**: After tests pass, invoke `ts-review` for code review (max 3 iterations)
9. **No file move required**: Plan status tracked in PROGRESS.json, not by file location

## Cross-Plan Execution

When running `/impl-exec-auto` without arguments, the system analyzes active plans with **lazy loading** to prevent OOM.

### CRITICAL: Lazy Loading to Prevent OOM

**DO NOT read all plan files.** Only read plans from eligible phases.

1. Read `impl-plans/README.md` FIRST
2. Extract Phase Status table and PHASE_TO_PLANS mapping
3. Identify READY phases (not BLOCKED)
4. Read ONLY plans from READY phases

```python
# Step 1: From README.md, get phase status
PHASE_STATUS = {
    1: "COMPLETED",
    2: "READY",       # Only this phase is eligible
    3: "BLOCKED",
    4: "BLOCKED"
}

# Step 2: Get plan list for eligible phases from PHASE_TO_PLANS
eligible_phases = [p for p, s in PHASE_STATUS.items() if s == "READY"]
plans_to_read = PHASE_TO_PLANS[eligible_phases[0]]  # Only Phase 2 plans

# Step 3: Read ONLY these plans (12 files, not 25)
# DO NOT read Phase 3/4 plans - they are BLOCKED
```

### Phase Dependency Rules

From `impl-plans/README.md`:

```
Phase 1: foundation-* (COMPLETED)
    |
    v
Phase 2: session-groups-*, command-queue-*, markdown-parser-*,
         realtime-*, bookmarks-*, file-changes-*
    |    (can run in parallel with each other)
    v
Phase 3: daemon-core, http-api, sse-events
    |
    v
Phase 4: browser-viewer-*, cli-*
```

### Cross-Plan Task Selection (with Lazy Loading)

1. Read README.md to get Phase Status and PHASE_TO_PLANS
2. Identify READY phases (skip BLOCKED phases entirely)
3. Read ONLY plan files from READY phases
4. Build task graph from those plans only
5. Execute tasks sequentially (one at a time)

### Cross-Plan Progress Tracking

When updating plans after cross-plan execution:

1. Update each affected plan's task statuses in plan files
2. Update PROGRESS.json with new task statuses
3. Add progress log to each affected plan
4. Check if any plan is now complete
5. If plan completes, update PROGRESS.json plan status and check if new phases become eligible

### Phase Transition Handling

When a phase-gating plan completes (e.g., foundation-and-core):

1. Update plan status to "Completed" in PROGRESS.json
2. Update plan file header status to "Completed"
3. Update phase status in PROGRESS.json if all phase plans complete
4. Report newly eligible plans
5. Suggest running `/impl-exec-auto` again for next phase

## Integration with Other Skills

| Skill/Agent | Relationship |
|-------------|--------------|
| `impl-plan/SKILL.md` | Read plans created by this skill |
| `ts-coding-standards/` | ts-coding agent follows these |
| `design-doc/SKILL.md` | Original design reference |
| `ts-review` agent | Code review after implementation |
| `check-and-test-after-modify` agent | Test verification before review |

## Main Conversation Orchestration Protocol

After `impl-exec-auto` returns the executable tasks list, the main conversation handles orchestration **via `impl-exec-specific`**.

**IMPORTANT**: DO NOT spawn ts-coding agents directly from the main conversation. Use `impl-exec-specific` instead.

### Orchestration Workflow

```
1. Receive executable tasks list from impl-exec-auto
2. Group tasks by plan name
3. For each plan with executable tasks:
   a. Invoke impl-exec-specific:
      /impl-exec-specific <plan-name> <TASK-IDs>

      Example:
      /impl-exec-specific session-groups-runner TASK-008
      /impl-exec-specific command-queue-core TASK-005 TASK-006
      /impl-exec-specific bookmarks-types TASK-003 TASK-004

   b. impl-exec-specific handles internally:
      - ts-coding spawning
      - check-and-test-after-modify
      - ts-review cycle (up to 3 iterations)
      - Plan file status updates

4. After impl-exec-specific completes:
   a. Update PROGRESS.json (with lock)
   b. Report results

5. Repeat for remaining plans/tasks
```

### Why Use impl-exec-specific?

| Approach | Problem |
|----------|---------|
| Main spawns ts-coding directly | Main lacks review cycle logic, plan update format |
| impl-exec-auto spawns ts-coding | Claude Code prohibits nested subagent spawning |
| **Main invokes impl-exec-specific** | Correct - full implementation cycle handled |

### PROGRESS.json Update (After impl-exec-specific Completes)

```
1. Acquire lock:
   Bash: while [ -f impl-plans/.progress.lock ]; do sleep 1; done && echo "<plan>:<task>" > impl-plans/.progress.lock

2. Edit PROGRESS.json:
   // Before:
   "TASK-001": { "status": "Not Started", "parallelizable": true, "deps": [] }

   // After:
   "TASK-001": { "status": "Completed", "parallelizable": true, "deps": [] }

   Also update `lastUpdated` timestamp.

3. Release lock:
   Bash: rm -f impl-plans/.progress.lock
```

### Example Main Conversation Response

```markdown
The impl-exec-auto analysis found 9 executable tasks across 6 plans.

Executing via impl-exec-specific:

1. /impl-exec-specific session-groups-runner TASK-008
   Result: TASK-008 completed (2 review iterations)

2. /impl-exec-specific command-queue-core TASK-005 TASK-006
   Result: TASK-005, TASK-006 completed

3. /impl-exec-specific markdown-parser-core TASK-004
   Result: TASK-004 completed

... (continue for remaining plans)

All 9 tasks completed. PROGRESS.json updated.
Newly unblocked tasks: TASK-009, TASK-010
```

### Error Handling

If impl-exec-specific reports a task failure:
1. Keep task status as "In Progress" in PROGRESS.json
2. Report the failure to user
3. Continue with next plan if possible
4. Suggest re-running failed tasks: `/impl-exec-specific <plan-name> <failed-task-id>`
