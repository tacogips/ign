---
description: Sync PROGRESS.json status with actual source code implementation
argument-hint: "[--dry-run]"
---

## Sync PROGRESS.json with Source Code

This command analyzes the actual source code implementation and syncs PROGRESS.json task statuses accordingly.

**Use Cases**:
- Recover from PROGRESS.json desync (e.g., after manual changes)
- Verify implementation status matches recorded progress
- Regenerate PROGRESS.json after git operations (merge, rebase)

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `impl-sync-progress` subagent using the Task tool.

### Argument Parsing

Parse `$ARGUMENTS`:

1. **If `--dry-run` flag present**: Analyze and report but do not update PROGRESS.json
2. **If no argument**: Perform analysis and update PROGRESS.json

### Invoke Subagent

```
Task tool parameters:
  subagent_type: impl-sync-progress
  prompt: |
    Mode: <sync or dry-run>
    Analyze source code implementation status and sync with PROGRESS.json
```

### What the Subagent Does

**Concurrent Processing** - The agent maximizes parallelism:

1. **Read PROGRESS.json** to get current task status
2. **Batch 1 (Parallel)**: Read ALL plan files concurrently
3. **Batch 2 (Parallel)**: Check ALL deliverable files concurrently
4. **Batch 3 (Parallel)**: Read existing files to check for stubs/incomplete code
5. **Aggregate** results and determine actual status
6. **Compare** recorded status vs actual status
7. **Report discrepancies**
8. **Update PROGRESS.json once** (unless dry-run, with file lock)

### Status Determination Logic (Priority Order)

**1. Completion Criteria (Highest Priority)**
```
Plan file has completion criteria checkboxes:
  - All [x] checked -> Completed
  - Some [x] checked -> In Progress
  - All [ ] unchecked -> Not Started or In Progress
```

**2. File Content Quality (Secondary)**
```
If files exist, check content:
  - Contains TODO/FIXME/notImplemented -> In Progress
  - File < 100 chars (stub) -> In Progress
  - Substantial implementation -> Completed
```

**3. File Existence (Lowest Priority)**
```
Only if completion criteria unavailable:
  - All files exist -> In Progress (NOT Completed)
  - Some files exist -> In Progress
  - No files exist -> Not Started
```

**IMPORTANT**: File existence alone does NOT mean "Completed".

### Usage Examples

**Sync PROGRESS.json with actual source**:
```
/impl-sync-progress
```

**Preview changes without updating**:
```
/impl-sync-progress --dry-run
```

### Output Format

```
## PROGRESS.json Sync Report

### Analysis Summary
- Plans analyzed: 21
- Tasks checked: 85
- Discrepancies found: 3

### Discrepancies

| Plan | Task | Recorded | Actual | Action |
|------|------|----------|--------|--------|
| session-groups-types | TASK-001 | Not Started | Completed | Updated |
| command-queue-core | TASK-003 | Completed | Not Started | Updated |
| realtime-watcher | TASK-002 | Completed | In Progress | Updated |

### Changes Applied
- Updated 3 task statuses in PROGRESS.json
- Updated lastUpdated timestamp

### Next Steps
- Run `/impl-exec-auto` to continue implementation
```

### Error Handling

**If plan file not found**:
```
Warning: Plan file not found: impl-plans/foo.md
Skipping plan 'foo' in sync
```

**If deliverable path is ambiguous**:
```
Warning: Cannot determine status for TASK-001
Deliverables contain wildcards: src/sdk/**/*.ts
Keeping current status: Not Started
```

### After Subagent Completes

1. Report sync results:
   - Number of plans/tasks analyzed
   - Discrepancies found and resolved
   - Warnings for ambiguous cases

2. If changes were made:
   - Show summary of updates
   - Confirm PROGRESS.json was updated

3. Suggest next steps:
   - Run `/impl-exec-auto` if tasks are ready
   - Review manual intervention needed for warnings
