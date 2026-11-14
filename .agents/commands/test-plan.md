---
description: Generate a test plan for a specific module or feature
argument-hint: "<source-path> [--type=unit|integration|e2e]"
---

## Generate Test Plan Command

This command generates a test plan for a specific module or feature using divide-and-conquer analysis.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `test-plan` subagent using the Task tool to generate a test plan for the specified target.

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **source-path** (required): Path to source file, directory, or impl-plan
2. **--type=<type>** (optional): Test type - unit (default), integration, or e2e

### Determine Output Path

Generate output path based on input:

| Input | Output |
|-------|--------|
| `src/sdk/queue/manager.ts` | `test-plans/queue-manager-unit.md` |
| `src/sdk/queue/` | `test-plans/queue-unit.md` |
| `impl-plans/session-groups.md` | `test-plans/session-groups-unit.md` |

### Invoke Subagent

```
Task tool parameters:
  subagent_type: test-plan
  prompt: |
    Target: <parsed source-path>
    Test Type: <parsed type or 'Unit'>
    Output Path: test-plans/<generated-name>.md
```

### Usage Examples

**Generate unit test plan for a module**:
```
/test-plan src/sdk/queue/
```

**Generate integration test plan**:
```
/test-plan src/daemon/ --type=integration
```

**Generate test plan for specific file**:
```
/test-plan src/sdk/queue/manager.ts
```

**Generate test plan from impl-plan**:
```
/test-plan impl-plans/session-groups.md --type=unit
```

### After Subagent Completes

1. Report the plan file created
2. Show summary of test cases defined
3. Report PROGRESS.json update
4. Suggest next steps:
   - Review the generated plan
   - Implement tests following the plan
   - Run tests with `bun test`

### Error Handling

If source path does not exist:
- Report the error
- Suggest valid paths

If test plan already exists:
- Ask if user wants to overwrite
- Or suggest reviewing existing plan
