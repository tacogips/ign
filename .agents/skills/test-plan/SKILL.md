---
name: test-plan
description: Use when creating test plans from implementation and design documents. Provides test plan structure, test case tracking, and coverage guidelines.
allowed-tools: Read, Write, Glob, Grep
---

# Test Plan Skill

This skill provides guidelines for creating and managing test plans from implementation and design documents.

## When to Apply

Apply this skill when:
- Creating test plans from existing implementations
- Planning test coverage for features
- Breaking down test cases into parallelizable units
- Tracking test progress across sessions

## Purpose

Test plans bridge the gap between implementation (what was built) and verification (does it work correctly). They provide:
- Clear test scenarios with expected behaviors
- Test case dependencies and execution order
- Coverage targets and completion criteria
- Progress tracking across test execution sessions

## Divide & Conquer Strategy

**CRITICAL**: Due to context limitations, test plans MUST use a divide-and-conquer approach.

### Analysis Phases

1. **Discovery Phase**: Identify modules to test (file listing, not content)
2. **Module Analysis Phase**: Analyze one module at a time
3. **Test Design Phase**: Design tests for analyzed module
4. **Aggregation Phase**: Combine into test plan

### DO NOT

- Load all source files at once
- Analyze entire codebase in single pass
- Generate tests for multiple modules simultaneously

### DO

- Focus on one module/area per agent invocation
- Use file patterns to identify scope first
- Analyze implementation only when designing specific tests

## Plan Granularity

**IMPORTANT**: Test plans and impl-plans do NOT need 1:1 mapping.

| Mapping | When to Use |
|---------|-------------|
| **1:N** (one impl -> multiple test plans) | Large implementations need focused test plans |
| **N:1** (multiple impls -> one test plan) | Related modules can share integration tests |
| **1:1** (one impl -> one test plan) | Well-bounded features with clear test scope |

**Recommended granularity**:
- Each test plan should be completable in 1-3 sessions
- Each test plan should have 3-15 test cases
- Maximize parallelizable test cases

## File Size Limits

**CRITICAL**: Large test plan files cause Claude Code OOM errors.

### Hard Limits

| Metric | Limit | Reason |
|--------|-------|--------|
| **Line count** | MAX 400 lines | Prevents memory issues |
| **Test cases per plan** | MAX 15 | Keeps plans focused |
| **Modules per plan** | MAX 5 | Enables focused testing |

### When to Split Plans

Split a test plan into multiple files when:

1. **Line count exceeds 400 lines**
2. **More than 15 test cases**
3. **Multiple test types** (unit, integration, e2e)
4. **Multiple feature areas**

### Split Strategy

```
BEFORE: session-groups-tests.md (too large)

AFTER:
- session-groups-unit-tests.md (~200 lines)
- session-groups-integration-tests.md (~200 lines)
- session-groups-e2e-tests.md (~150 lines)
```

## Output Location

**IMPORTANT**: All test plans MUST be stored under `test-plans/`.

```
test-plans/
├── README.md              # Index of all test plans
├── PROGRESS.json          # Test status tracking (single source of truth)
├── <feature>-unit.md      # Unit test plans
├── <feature>-integration.md # Integration test plans
└── <feature>-e2e.md       # End-to-end test plans
```

## Test Plan Structure

Each test plan file MUST include:

### 1. Header Section

```markdown
# <Feature Name> Test Plan

**Status**: Planning | Ready | In Progress | Completed
**Implementation Reference**: impl-plans/<file>.md
**Source Files**: src/<path>/*.ts
**Test Type**: Unit | Integration | E2E
**Created**: YYYY-MM-DD
**Last Updated**: YYYY-MM-DD
```

### 2. Implementation Reference

- Link to implementation plan or source files
- Summary of what is being tested
- Scope boundaries (what is NOT tested in this plan)

### 3. Test Environment

```markdown
## Test Environment

**Runtime**: Bun test
**Mocks Required**: List of mock implementations needed
**Fixtures**: List of test fixtures needed
**Setup/Teardown**: Special requirements
```

### 4. Test Cases

Each test case MUST have:

```markdown
### TEST-001: <Test Case Name>

**Status**: Not Started | In Progress | Passing | Failing | Skipped
**Priority**: Critical | High | Medium | Low
**Parallelizable**: Yes | No
**Dependencies**: None | TEST-XXX | other-plan:TEST-XXX

**Target**: `src/path/file.ts:functionName`

**Description**:
Brief description of what this test verifies.

**Scenarios**:
1. Happy path: <expected behavior>
2. Edge case: <expected behavior>
3. Error case: <expected behavior>

**Assertions**:
- [ ] Assertion 1
- [ ] Assertion 2

**Test Code Location**: `src/path/file.test.ts`
```

### 5. Test Status Table

```markdown
## Test Status

| Test ID | Name | Status | Priority | Dependencies |
|---------|------|--------|----------|--------------|
| TEST-001 | Basic parsing | Not Started | High | None |
| TEST-002 | Error handling | Not Started | High | TEST-001 |
| TEST-003 | Edge cases | Not Started | Medium | TEST-001 |
```

### 6. Coverage Targets

```markdown
## Coverage Targets

| Module | Current | Target | Status |
|--------|---------|--------|--------|
| src/sdk/queue/types.ts | 0% | 90% | Not Started |
| src/sdk/queue/manager.ts | 0% | 85% | Not Started |
```

### 7. Completion Criteria

```markdown
## Completion Criteria

- [ ] All test cases implemented
- [ ] All tests passing
- [ ] Coverage targets met
- [ ] No flaky tests
- [ ] Documentation updated
```

### 8. Progress Log

```markdown
## Progress Log

### Session: YYYY-MM-DD HH:MM
**Tests Completed**: TEST-001, TEST-002
**Tests In Progress**: TEST-003
**Blockers**: None
**Notes**: Discovered edge case requiring additional test
```

## Test Case Definition Guidelines

### Test ID Format

- Format: `TEST-XXX` where XXX is zero-padded (001, 002, etc.)
- IDs are unique within a plan
- Cross-plan references: `<plan-name>:TEST-XXX`

### Dependency Specification

**Same-plan dependency**:
```markdown
**Dependencies**: TEST-001
**Dependencies**: TEST-001, TEST-002
```

**Cross-plan dependency**:
```markdown
**Dependencies**: session-groups-unit:TEST-001
```

**No dependencies**:
```markdown
**Dependencies**: None
```

### Parallelization Rules

Tests can be parallelized when:
1. No shared mutable state
2. No file system side effects
3. No network dependencies between tests
4. No order-dependent setup

**Parallelizable: Yes** - Test can run independently
**Parallelizable: No** - Test depends on other tests or shared state

## PROGRESS.json Integration

**CRITICAL**: After creating a test plan, update `test-plans/PROGRESS.json`.

### PROGRESS.json Structure

```json
{
  "lastUpdated": "2026-01-09T16:00:00Z",
  "summary": {
    "totalPlans": 10,
    "totalTests": 150,
    "passing": 0,
    "failing": 0,
    "notStarted": 150
  },
  "plans": {
    "feature-unit": {
      "status": "Ready",
      "testType": "Unit",
      "implRef": "impl-plans/feature.md",
      "tests": {
        "TEST-001": { "status": "Not Started", "priority": "High", "parallelizable": true, "deps": [] },
        "TEST-002": { "status": "Not Started", "priority": "High", "parallelizable": false, "deps": ["TEST-001"] }
      }
    }
  }
}
```

### Status Values

| Status | Meaning |
|--------|---------|
| `Not Started` | Test not yet implemented |
| `In Progress` | Test being written |
| `Passing` | Test implemented and passing |
| `Failing` | Test implemented but failing |
| `Skipped` | Test intentionally skipped |

### Priority Values

| Priority | Meaning |
|----------|---------|
| `Critical` | Must pass for release |
| `High` | Should pass for release |
| `Medium` | Important but not blocking |
| `Low` | Nice to have |

## Test Type Guidelines

### Unit Tests

- Test single functions/methods in isolation
- Mock all dependencies
- Fast execution (< 100ms per test)
- File naming: `*.test.ts` adjacent to source

### Integration Tests

- Test multiple components together
- May use real dependencies
- Moderate execution time
- Test API contracts between modules

### End-to-End Tests

- Test complete user flows
- Minimal mocking
- May require setup/teardown
- Test from external interface

## Workflow Integration

### Creating a Test Plan

1. Read the implementation or design document
2. Identify test boundaries and types
3. List test scenarios (happy path, edge cases, errors)
4. Define test cases with IDs and dependencies
5. Set coverage targets
6. Create plan file in `test-plans/<feature>-<type>.md`
7. **Update PROGRESS.json with new plan and tests**
8. **Update test-plans/README.md with new plan entry**

### During Test Execution

1. Update test status in PROGRESS.json as tests are written
2. Add progress log entries per session
3. Note blockers and issues
4. Check off completion criteria

### Completing a Test Plan

1. Verify all tests passing
2. Verify coverage targets met
3. Update plan status to "Completed"
4. Add final progress log entry

## Quick Reference

| Section | Required | Format |
|---------|----------|--------|
| Header | Yes | Markdown metadata |
| Implementation Reference | Yes | Link + summary |
| Test Environment | Yes | Requirements list |
| Test Cases | Yes | Structured format with scenarios |
| Status Table | Yes | Simple table |
| Coverage Targets | Yes | Target percentages |
| Completion Criteria | Yes | Checklist |
| Progress Log | Yes | Session entries |
