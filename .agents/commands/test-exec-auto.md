---
description: Automatically select and execute parallelizable tests from test plan(s)
argument-hint: "[--plan=<plan-name>] [--priority=critical|high|medium|low]"
---

## Auto Execute Tests Command

This command automatically selects and executes all parallelizable tests from test plans.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

### Step 1: Invoke test-exec-auto Subagent

First, analyze available tests:

```
Task tool parameters:
  subagent_type: test-exec-auto
  prompt: |
    Analyze test plans and return executable tests.
    Plan Filter: <--plan value if provided, or "all">
    Priority Filter: <--priority value if provided, or "all">
```

### Step 2: Parse Analysis Results

From the test-exec-auto response, extract:
- List of executable tests grouped by plan
- Priority order
- Dependencies

### Step 3: Execute Tests via test-exec-specific

For each plan with executable tests:

```
Task tool parameters:
  subagent_type: test-exec-specific
  prompt: |
    Test Plan: <plan-name>
    Test IDs: <comma-separated test IDs>
```

Execute in priority order:
1. Critical priority tests first
2. High priority tests
3. Medium priority tests
4. Low priority tests

### Step 4: Update PROGRESS.json

After all tests complete, update `test-plans/PROGRESS.json`:
- Update test statuses
- Update summary counts
- Update lastUpdated timestamp

### Step 5: Report Results

Report:
1. Tests executed and their results
2. Tests that passed/failed
3. Newly unblocked tests
4. Updated PROGRESS.json summary

---

## Usage Examples

**Execute all available tests**:
```
/test-exec-auto
```

**Execute tests from specific plan**:
```
/test-exec-auto --plan=queue-unit
```

**Execute only critical priority tests**:
```
/test-exec-auto --priority=critical
```

**Combine filters**:
```
/test-exec-auto --plan=daemon-unit --priority=high
```

---

## Argument Parsing

| Argument | Description | Default |
|----------|-------------|---------|
| `--plan=<name>` | Only execute tests from this plan | All plans |
| `--priority=<level>` | Only execute tests of this priority or higher | All priorities |

Priority levels (from highest to lowest):
1. `critical` - Only critical tests
2. `high` - Critical and high
3. `medium` - Critical, high, and medium
4. `low` - All tests

---

## After Execution

Report summary:

```markdown
## Test Execution Summary

### Results
| Plan | Executed | Passing | Failing | Skipped |
|------|----------|---------|---------|---------|
| queue-unit | 5 | 4 | 1 | 0 |
| group-unit | 3 | 3 | 0 | 0 |

### Failing Tests
| Plan | Test ID | Error |
|------|---------|-------|
| queue-unit | TEST-003 | Assertion failed |

### Newly Unblocked
- group-unit:TEST-004 (dependency TEST-001 now passing)

### Next Steps
1. Fix failing test: queue-unit:TEST-003
2. Run `/test-exec-auto` again for newly unblocked tests
```

---

## Error Handling

If no tests are executable:
- Report why (all passing, dependencies not met, etc.)
- Suggest using `/test-exec-specific` for manual execution

If tests fail:
- Report failure details
- Continue with remaining tests
- Summarize all failures at end
