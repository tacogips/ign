---
description: Generate all implementation plans from design documents in parallel
argument-hint: "[--dry-run] [--force]"
---

## Generate All Implementation Plans Command

This command generates all implementation plans from design documents by running subtasks in parallel.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `impl-plan-all` subagent using the Task tool to generate all implementation plans concurrently.

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **--dry-run** (optional): Only list plans that would be created, do not create them
2. **--force** (optional): Regenerate plans even if they already exist

### Invoke Subagent

```
Task tool parameters:
  subagent_type: impl-plan-all
  prompt: |
    Design Directory: design-docs/
    Output Directory: impl-plans/
    Dry Run: <true if --dry-run flag present>
    Force: <true if --force flag present>
```

### Usage Examples

**Generate all missing plans**:
```
/impl-plan-all
```

**Preview what would be created**:
```
/impl-plan-all --dry-run
```

**Regenerate all plans (including existing)**:
```
/impl-plan-all --force
```

### After Subagent Completes

1. Report summary of plans created/skipped
2. Report PROGRESS.json update summary:
   - Plans added
   - Total tasks added
   - Cross-plan dependencies validated
3. List any errors that occurred
4. Confirm impl-plans/README.md was updated
5. Suggest next steps:
   - Review generated plans
   - Run `/impl-exec-auto` to begin implementation

### Error Handling

If errors occur during generation:
- Report which plans failed
- Suggest using `/impl-plan` for individual retries
- Continue with successful plans
