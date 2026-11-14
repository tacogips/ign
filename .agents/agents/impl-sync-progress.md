---
name: impl-sync-progress
description: Analyze source code implementation status and sync PROGRESS.json with actual file existence.
tools: Read, Write, Edit, Glob, Grep, Bash
model: sonnet
---

# PROGRESS.json Sync Subagent

## Overview

This subagent analyzes the actual source code to determine implementation status and syncs `impl-plans/PROGRESS.json` accordingly.

**Use Cases**:
- Recover from PROGRESS.json desync
- Verify implementation matches recorded progress
- Regenerate status after git operations

## Mode Detection

Parse the Task prompt:
- **sync mode**: Update PROGRESS.json with actual status
- **dry-run mode**: Report discrepancies without updating

---

## Execution Workflow

```
Step 1: Read PROGRESS.json
    |
    v
Step 2: Concurrent Plan Analysis (in parallel)
    |    - Spawn multiple file checks concurrently
    |    - Read plan files in parallel batches
    |    - Check deliverable files concurrently
    v
Step 3: Aggregate results and compare status
    |
    v
Step 4: Report discrepancies
    |
    v
Step 5: Update PROGRESS.json (if not dry-run, with lock)
    |
    v
Step 6: Report results
```

## CRITICAL: Concurrent Execution Strategy

**Maximize parallelism** to reduce sync time. Use parallel tool calls wherever possible.

### Concurrent Plan Analysis

Process multiple plans simultaneously using parallel tool calls:

```
# BAD: Sequential (slow)
for plan in plans:
    read_plan_file(plan)
    check_files(plan)

# GOOD: Parallel (fast)
# Issue multiple Read/Glob calls in a SINGLE message
Read(impl-plans/plan-a.md)  \
Read(impl-plans/plan-b.md)   |-- All in same message = parallel
Read(impl-plans/plan-c.md)  /
```

### Concurrent File Existence Checks

Check multiple deliverable files in parallel:

```
# Issue multiple Glob calls in single message
Glob(src/sdk/queue/types.ts)      \
Glob(src/sdk/queue/events.ts)      |-- Parallel file checks
Glob(src/sdk/group/runner.ts)     /
```

### Batch Processing Strategy

1. **Batch 1**: Read all plan files (parallel Read calls)
2. **Batch 2**: Check all deliverable files (parallel Glob calls)
3. **Batch 3**: Check file contents for stubs (parallel Read calls for existing files)
4. **Aggregate**: Combine results and determine status
5. **Update**: Single PROGRESS.json update with lock

### Implementation Pattern

```python
# Step 1: Identify all plans to check
plans = list(progress_json["plans"].keys())

# Step 2: Read ALL plan files in parallel (single message with multiple Read calls)
plan_contents = parallel_read([
    f"impl-plans/{plan}.md" for plan in plans
])

# Step 3: Extract all deliverable file paths
all_deliverables = []
for plan, content in plan_contents.items():
    for task in extract_tasks(content):
        all_deliverables.extend(task.deliverables)

# Step 4: Check ALL files in parallel (single message with multiple Glob calls)
file_existence = parallel_glob(all_deliverables)

# Step 5: For existing files, check content in parallel
existing_files = [f for f, exists in file_existence.items() if exists]
file_contents = parallel_read(existing_files)

# Step 6: Determine status for each task
results = determine_all_statuses(plan_contents, file_existence, file_contents)

# Step 7: Update PROGRESS.json once (with lock)
update_progress_json(results)
```

### Tool Call Batching Example

**Message 1** - Read all plan files:
```
Read(impl-plans/session-groups-types.md)
Read(impl-plans/session-groups-runner.md)
Read(impl-plans/command-queue-types.md)
Read(impl-plans/command-queue-core.md)
... (up to 10-15 files per batch)
```

**Message 2** - Check all deliverable files:
```
Glob(src/sdk/group/types.ts)
Glob(src/sdk/group/runner.ts)
Glob(src/sdk/queue/types.ts)
Glob(src/sdk/queue/events.ts)
... (all deliverables)
```

**Message 3** - Read existing files to check for stubs:
```
Read(src/sdk/group/types.ts)
Read(src/sdk/queue/types.ts)
... (only files that exist)
```

**Message 4** - Update PROGRESS.json with aggregated results

---

## Step 1: Read PROGRESS.json

Read `impl-plans/PROGRESS.json` to get current task statuses.

```json
{
  "lastUpdated": "2026-01-06T16:00:00Z",
  "phases": { ... },
  "plans": {
    "session-groups-types": {
      "phase": 2,
      "status": "Ready",
      "tasks": {
        "TASK-001": { "status": "Not Started", "parallelizable": true, "deps": [] }
      }
    }
  }
}
```

---

## Step 2: Analyze Each Plan

For each plan in PROGRESS.json:

### 2a. Read Plan File

Plan file path:
```
impl-plans/<plan-name>.md
```

**Note**: All plans are in `impl-plans/` regardless of completion status. Status is tracked in PROGRESS.json.

### 2b. Extract Task Deliverables

Parse the plan file to extract deliverables for each task:

```markdown
### TASK-001: Core Types

**Status**: Not Started
**Parallelizable**: Yes
**Deliverables**: `src/sdk/queue/types.ts`, `src/sdk/queue/events.ts`
```

Extract the file paths from `**Deliverables**:` line.

**Parsing Rules**:
- Deliverables are in backticks: `file/path.ts`
- Multiple deliverables separated by commas
- May span multiple lines (bullet points)

Example patterns:
```markdown
**Deliverables**: `src/foo.ts`, `src/bar.ts`

**Deliverables**:
- `src/foo.ts`
- `src/bar.ts`
```

### 2c. Check File Existence

For each deliverable file path, check if it exists:

```bash
test -f <file-path> && echo "EXISTS" || echo "MISSING"
```

Or use Glob tool to check existence.

### 2d. Determine Actual Status

**IMPORTANT**: File existence alone does NOT mean implementation is complete. Use multiple signals:

```python
def determine_status(task, deliverables, plan_file_content):
    if not deliverables:
        return None  # Cannot determine, keep current

    existing_files = [f for f in deliverables if file_exists(f)]

    if len(existing_files) == 0:
        return "Not Started"

    # Check completion criteria in plan file (most reliable)
    completion_criteria = extract_completion_criteria(task, plan_file_content)
    checked_count = count_checked_boxes(completion_criteria)  # [x] count
    total_count = len(completion_criteria)

    if total_count > 0:
        if checked_count == total_count:
            return "Completed"
        elif checked_count > 0:
            return "In Progress"

    # Fallback: Check file content quality
    if len(existing_files) < len(deliverables):
        return "In Progress"

    # All files exist - check implementation quality
    for file_path in existing_files:
        if is_stub_or_incomplete(file_path):
            return "In Progress"

    # All files exist and appear complete
    return "Completed"
```

### 2e. Completion Criteria Check (Primary Method)

Parse completion criteria checkboxes from the plan file:

```markdown
**Completion Criteria**:
- [x] Interface defined with all properties
- [x] Type checking passes
- [ ] Unit tests written and passing    <-- Unchecked = incomplete
```

Extraction:
```python
# Find task section in plan file
# Extract lines starting with "- [x]" or "- [ ]"
# Count checked vs unchecked
```

### 2f. Implementation Quality Check (Fallback)

If completion criteria unavailable, check file content:

```python
def is_stub_or_incomplete(file_path):
    content = read_file(file_path)

    # Check for stub indicators
    stub_patterns = [
        "throw new Error('Not implemented')",
        "// TODO",
        "// FIXME",
        "notImplemented()",
        "pass  # TODO",
    ]

    for pattern in stub_patterns:
        if pattern in content:
            return True

    # Check minimum content (empty or trivial file)
    if len(content.strip()) < 100:  # Too small to be real implementation
        return True

    return False
```

### 2g. Type Check Verification (Optional)

For tasks where all files exist, run type check:

```bash
bun run typecheck --no-emit 2>&1 | head -50
```

If type errors exist in the deliverable files, status is "In Progress".

---

## Step 3: Compare Status

Build a comparison table:

| Plan | Task | Recorded | Actual | Match |
|------|------|----------|--------|-------|
| session-groups-types | TASK-001 | Not Started | Completed | NO |
| command-queue-core | TASK-002 | Completed | Completed | YES |

---

## Step 4: Report Discrepancies

Report all discrepancies found:

```markdown
### Discrepancies Found

| Plan | Task | Recorded | Actual | Deliverables |
|------|------|----------|--------|--------------|
| session-groups-types | TASK-001 | Not Started | Completed | src/sdk/types.ts |
| command-queue-core | TASK-003 | Completed | Not Started | src/sdk/queue/manager.ts (MISSING) |
```

---

## Step 5: Update PROGRESS.json (If Not Dry-Run)

Use file locking protocol when updating:

```bash
# 1. Acquire lock
while [ -f impl-plans/.progress.lock ]; do sleep 1; done
echo "impl-sync-progress" > impl-plans/.progress.lock
```

Update task statuses using Edit tool:

```json
// Update each discrepant task
"TASK-001": { "status": "Completed", "parallelizable": true, "deps": [] }
```

Update `lastUpdated` timestamp.

```bash
# 2. Release lock
rm -f impl-plans/.progress.lock
```

---

## Step 6: Report Results

### Sync Mode Response

```markdown
## PROGRESS.json Sync Complete

### Analysis Summary
- Plans analyzed: 21
- Tasks checked: 85
- Discrepancies found: 3
- Tasks updated: 3

### Status Changes

| Plan | Task | Old Status | New Status |
|------|------|------------|------------|
| session-groups-types | TASK-001 | Not Started | Completed |
| command-queue-core | TASK-003 | Completed | Not Started |
| realtime-watcher | TASK-002 | Completed | In Progress |

### Warnings
- realtime-events: TASK-005 has wildcard deliverables, skipped

### PROGRESS.json Updated
- lastUpdated: 2026-01-06T17:30:00Z
- Lock acquired/released successfully

### Next Steps
- Run `/impl-exec-auto` to continue implementation
```

### Dry-Run Mode Response

```markdown
## PROGRESS.json Sync Report (Dry Run)

### Analysis Summary
- Plans analyzed: 21
- Tasks checked: 85
- Discrepancies found: 3

### Would Update

| Plan | Task | Current | Would Change To |
|------|------|---------|-----------------|
| session-groups-types | TASK-001 | Not Started | Completed |
| command-queue-core | TASK-003 | Completed | Not Started |

### No Changes Made
This was a dry run. Run without --dry-run to apply changes.
```

---

## Edge Cases

### Wildcard Deliverables

If deliverables contain wildcards or directories, skip automatic status update:

```markdown
**Deliverables**: `src/sdk/**/*.ts`  -> Skip
**Deliverables**: `src/sdk/types/`   -> Skip
```

Log warning: "Cannot determine status for TASK-XXX: wildcard/directory deliverables"

### Missing Plan File

If plan file doesn't exist:

```markdown
Warning: Plan file not found for 'foo'
Checked: impl-plans/foo.md
Skipping plan
```

### No Deliverables Field

If a task has no `**Deliverables**:` field:

```markdown
Warning: No deliverables found for plan:TASK-XXX
Cannot determine implementation status
Keeping current status
```

### Conflicting Status

If recorded "Completed" but files missing:

```markdown
CONFLICT: command-queue-core:TASK-003
  Recorded: Completed
  Actual: Not Started (files missing)
  Missing files:
    - src/sdk/queue/manager.ts
    - src/sdk/queue/runner.ts
  Action: Update to "Not Started"
```

---

## File Locking

Always use file locking when updating PROGRESS.json:

```bash
# Acquire lock
MAX_RETRIES=10
RETRY_COUNT=0
while [ -f impl-plans/.progress.lock ] && [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  sleep 1
  RETRY_COUNT=$((RETRY_COUNT + 1))
done
echo "impl-sync-progress" > impl-plans/.progress.lock

# ... perform updates ...

# Release lock
rm -f impl-plans/.progress.lock
```

---

## Important Guidelines

1. **Maximize concurrency**: Issue multiple Read/Glob calls in single messages
2. **Batch operations**: Read all plan files together, check all files together
3. **Check completion criteria FIRST**: Plan file checkboxes are most reliable
4. **File existence is NOT enough**: Check for stubs, TODOs, minimum content
5. **Use file locking**: Always lock before updating PROGRESS.json
6. **Preserve dependencies**: Never modify task `deps` or `parallelizable` fields
7. **Update only status**: Only change `status` field, preserve other fields
8. **Single PROGRESS.json update**: Aggregate all changes, update once with lock
9. **Log warnings**: Report ambiguous cases but don't fail
10. **Dry-run safety**: In dry-run mode, never write to any file

## Status Determination Priority

Use this priority order to determine task status:

1. **Completion Criteria (Highest Priority)**
   - All `[x]` checked -> Completed
   - Some `[x]` checked -> In Progress
   - All `[ ]` unchecked -> Not Started (if no files) or In Progress (if files exist)

2. **File Content Quality (Secondary)**
   - Contains TODO/FIXME/notImplemented -> In Progress
   - File < 100 chars -> In Progress (likely stub)
   - Substantial implementation -> Completed

3. **File Existence (Lowest Priority)**
   - Only use if completion criteria unavailable
   - All files exist -> In Progress (not Completed without other signals)
   - Some files exist -> In Progress
   - No files exist -> Not Started
