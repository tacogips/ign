---
description: Execute specific tests from a test plan by test ID
argument-hint: "<plan-name> <TEST-ID> [TEST-ID...] [--run-only] [--implement-only]"
---

## Execute Specific Tests Command

This command executes specific tests by ID from a test plan.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **plan-name** (required): Name of test plan (without `.md` extension)
2. **TEST-IDs** (required): One or more test IDs (e.g., TEST-001 TEST-002)
3. **--run-only** (optional): Only run existing tests, do not implement
4. **--implement-only** (optional): Only implement tests, do not run

### Invoke Subagent

```
Task tool parameters:
  subagent_type: test-exec-specific
  prompt: |
    Test Plan: <plan-name>
    Test IDs: <comma-separated test IDs>
    Run Only: <true if --run-only, else false>
    Implement Only: <true if --implement-only, else false>
```

---

## Usage Examples

**Execute specific tests (implement and run)**:
```
/test-exec-specific queue-unit TEST-001 TEST-002
```

**Execute single test**:
```
/test-exec-specific daemon-unit TEST-005
```

**Only implement tests (skip running)**:
```
/test-exec-specific group-unit TEST-001 --implement-only
```

**Only run existing tests (skip implementation)**:
```
/test-exec-specific queue-unit TEST-001 TEST-002 --run-only
```

---

## Argument Validation

| Check | Error Message |
|-------|---------------|
| No plan name | "Error: Plan name required. Usage: /test-exec-specific <plan> <TEST-ID...>" |
| No test IDs | "Error: At least one TEST-ID required. Usage: /test-exec-specific <plan> <TEST-ID...>" |
| Invalid test ID format | "Error: Invalid test ID format. Expected TEST-XXX (e.g., TEST-001)" |
| Plan not found | "Error: Test plan not found: test-plans/<plan>.md" |

---

## After Subagent Completes

1. Report test execution results
2. Show pass/fail status for each test
3. Report PROGRESS.json update
4. List any newly unblocked tests
5. Suggest next steps

### Success Output

```markdown
## Test Execution Complete

### Plan: queue-unit

### Tests Executed
| Test ID | Status | Attempts |
|---------|--------|----------|
| TEST-001 | Passing | 1 |
| TEST-002 | Passing | 2 |

### Test Output
```
  ✓ Queue Add Operation (3 tests)
  ✓ Queue Runner (2 tests)

  5 pass | 0 fail
```

### PROGRESS.json Updated
- Tests passing: +2
- lastUpdated: 2026-01-09T17:00:00Z

### Newly Unblocked
- TEST-003 (dependency on TEST-001 satisfied)
```

### Failure Output

```markdown
## Test Execution Partial Failure

### Plan: queue-unit

### Tests Executed
| Test ID | Status | Attempts |
|---------|--------|----------|
| TEST-001 | Passing | 1 |
| TEST-002 | Failing | 3 |

### Failure Details
**TEST-002**: Assertion error after 3 attempts
- Expected: 5
- Received: 4
- File: src/sdk/queue/runner.test.ts:25

### Recommended Actions
1. Review source implementation
2. Check test logic
3. Re-run: `/test-exec-specific queue-unit TEST-002`
```

---

## Error Handling

If test plan not found:
- List available test plans
- Suggest correct plan name

If test ID not found in plan:
- List available test IDs in the plan
- Suggest correct test ID

If test already passing:
- Report that test is already passing
- Ask if user wants to force re-run
