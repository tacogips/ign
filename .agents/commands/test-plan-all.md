---
description: Generate all test plans from implementation and specs using divide-and-conquer approach
argument-hint: "[--dry-run] [--force] [--type=unit|integration|e2e]"
---

## Generate All Test Plans Command

This command generates all test plans from implementation and specifications by running subtasks in parallel with a divide-and-conquer strategy.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `test-plan-all` subagent using the Task tool to generate all test plans concurrently.

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **--dry-run** (optional): Only list plans that would be created, do not create them
2. **--force** (optional): Regenerate plans even if they already exist
3. **--type=<type>** (optional): Only generate specific test type (unit, integration, e2e)

### Invoke Subagent

```
Task tool parameters:
  subagent_type: test-plan-all
  prompt: |
    Source Directory: src/
    Output Directory: test-plans/
    Impl Plans: impl-plans/
    Dry Run: <true if --dry-run flag present>
    Force: <true if --force flag present>
    Test Types: <specified type or all types>
```

### Usage Examples

**Generate all missing test plans**:
```
/test-plan-all
```

**Preview what would be created**:
```
/test-plan-all --dry-run
```

**Generate only unit test plans**:
```
/test-plan-all --type=unit
```

**Regenerate all plans (including existing)**:
```
/test-plan-all --force
```

**Combine options**:
```
/test-plan-all --type=integration --dry-run
```

### After Subagent Completes

1. Report summary of plans created/skipped
2. Report PROGRESS.json update summary:
   - Plans added
   - Total tests defined
   - Coverage targets
3. List any errors that occurred
4. Confirm test-plans/README.md was updated
5. Suggest next steps:
   - Review generated plans
   - Run `/test-exec-auto` to begin test implementation

### Error Handling

If errors occur during generation:
- Report which plans failed
- Suggest using `/test-plan` for individual retries
- Continue with successful plans
