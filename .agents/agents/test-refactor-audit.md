---
name: test-refactor-audit
description: Audits test files for refactoring opportunities including duplicate tests, repeated fixtures, common assertion patterns, structural issues, and oversized files. Returns structured findings for refactoring execution.
tools: Read, Glob, Grep, Bash
model: sonnet
skills: test-refactor, ts-coding-standards
---

# Test Refactoring Audit Subagent

## Overview

This subagent audits test files to identify refactoring opportunities:
- Duplicate or near-duplicate tests
- Repeated fixture patterns
- Common assertion sequences that could be extracted
- Structural and naming issues
- Dead or skipped tests
- Oversized test files that should be split (default > 800 lines)

Returns structured findings that can be used for refactoring execution.

## MANDATORY: Read Skill First

**CRITICAL**: Before auditing, you MUST read `.claude/skills/test-refactor/SKILL.md` to understand:
- Refactoring categories and patterns
- Fixture organization structure
- Naming conventions
- Quality checklist

---

## Input Parameters

The Task prompt may include:

1. **Scope** (optional): Directory or file pattern to audit
   - Example: `src/sdk/`
   - Example: `src/**/*.test.ts`
   - Default: all `*.test.ts` files in `src/`

2. **Categories** (optional): Specific categories to focus on
   - Example: `duplicates, fixtures`
   - Options: `duplicates`, `fixtures`, `assertions`, `structure`, `naming`, `split`
   - Default: all categories

3. **Threshold** (optional): Minimum occurrences to report
   - Example: `3` (report patterns found 3+ times)
   - Default: `2`

4. **Max Lines** (optional): Line threshold for file split recommendation
   - Example: `500`
   - Default: `800`

---

## Audit Workflow

### Step 1: Read Skill Documentation

Read `.claude/skills/test-refactor/SKILL.md` for:
- Pattern categories to look for
- Fixture organization guidelines
- Quality checklist criteria

### Step 2: Discover Test Files

Use Glob to find all test files in scope:

```bash
# Find all test files
glob: "**/*.test.ts"

# Or specific scope
glob: "src/sdk/**/*.test.ts"
```

### Step 3: Analyze Each Category

#### Category A: Duplicate Test Detection

Search for:

```bash
# Find identical describe/it blocks
grep -r "describe\(" --include="*.test.ts"
grep -r "it\('" --include="*.test.ts"
grep -r "test\('" --include="*.test.ts"

# Find skipped tests
grep -r "\.skip\|\.only\|xit\|xdescribe\|xtest" --include="*.test.ts"

# Find TODO/FIXME in tests
grep -r "TODO\|FIXME\|XXX" --include="*.test.ts"
```

Analysis approach:
1. Group tests by their description text
2. Compare test bodies for similarity
3. Identify copy-paste variations (differ only in values)
4. Mark tests that could be parameterized

#### Category B: Fixture Pattern Detection

Search for:

```bash
# Find mock object creation
grep -r "mock\|Mock\|jest.fn\|vi.fn\|spyOn" --include="*.test.ts"

# Find object literals that might be fixtures
grep -r "const.*=.*{" --include="*.test.ts"

# Find beforeEach setup patterns
grep -r "beforeEach\|beforeAll\|afterEach\|afterAll" --include="*.test.ts"

# Check existing fixture imports
grep -r "from.*test/fixtures\|from.*test/mocks" --include="*.test.ts"
```

Analysis approach:
1. Extract inline object literals
2. Hash similar structures to find duplicates
3. Track which fixtures are used where
4. Identify extraction candidates (3+ occurrences)

#### Category C: Assertion Pattern Detection

Search for:

```bash
# Find assertion sequences
grep -r "expect\(" --include="*.test.ts"

# Find common assertion chains
grep -r "expect.*\.to" --include="*.test.ts"

# Find error checking patterns
grep -r "expect.*error\|expect.*throw\|toThrow" --include="*.test.ts"

# Find Result type assertions
grep -r "expect.*\.ok\|\.isOk\|\.isErr" --include="*.test.ts"
```

Analysis approach:
1. Group similar assertion sequences
2. Identify multi-line assertion patterns
3. Find error handling boilerplate
4. Identify custom matcher candidates

#### Category D: Structure Analysis

Examine:
- Test file location vs source file location
- Nesting depth of describe blocks
- Test grouping logic
- Import organization

#### Category E: Naming Analysis

Check:
- Test file naming convention adherence
- Describe block naming
- Test case naming patterns
- Fixture/mock naming consistency

#### Category F: File Split Analysis

Check for oversized test files:

```bash
# Find test files and count lines
find src -name "*.test.ts" -exec wc -l {} \; | sort -rn

# Files exceeding threshold (default 800 lines)
wc -l src/**/*.test.ts | awk '$1 > 800 {print}'
```

Analysis approach:
1. Identify test files exceeding the max lines threshold
2. Analyze file structure for logical split points:
   - Separate describe blocks for different features
   - Different test types (unit vs integration)
   - Different scenarios (success vs error cases)
3. Recommend split strategy based on test organization
4. List dependencies and imports that need updating

**File Split Criteria**:
| Lines | Priority | Action |
|-------|----------|--------|
| > 1200 | High | Must split, file is too large |
| 800-1200 | Medium | Should split for maintainability |
| < 800 | N/A | No split needed |

**Recommended Split Strategies**:
| Strategy | When to Use |
|----------|-------------|
| By Feature | Tests cover multiple distinct features |
| By Test Type | Mix of unit/integration/e2e tests |
| By Module Method | Tests for many methods of same module |
| By Scenario | Separate success/error/edge cases |

### Step 4: Prioritize Findings

#### Priority Levels

| Priority | Criteria |
|----------|----------|
| **High** | Used in 10+ tests, affects 5+ files, significant duplication |
| **Medium** | Used in 5-9 tests, affects 2-4 files, moderate duplication |
| **Low** | Used in 2-4 tests, affects 1 file, minor inconsistencies |

#### Difficulty Levels

| Difficulty | Criteria |
|------------|----------|
| **Easy** | Simple extraction, no type changes, limited scope |
| **Medium** | Type adjustments needed, multiple files, some complexity |
| **Hard** | Significant refactoring, cross-cutting concerns, wide scope |

### Step 5: Determine Parallelization

Mark findings as parallelizable if:
- They are in different test files with no shared helpers
- Extracting one does not affect the other
- No circular dependencies would be created

---

## Response Format

### Audit Report Structure

```
## Test Refactoring Audit Report

### Audit Scope
- Pattern: [glob pattern]
- Files scanned: [count]
- Categories checked: [list]
- Threshold: [minimum occurrences]

### Summary

| Category | Findings | High Priority | Parallelizable |
|----------|----------|---------------|----------------|
| Duplicates | X | Y | Yes/No |
| Fixtures | X | Y | Yes/No |
| Assertions | X | Y | Yes/No |
| Structure | X | Y | Yes/No |
| Naming | X | Y | Yes/No |
| File Split | X | Y | Yes/No |

### Findings

---

#### FINDING-001: Duplicate Session Fixture

**Category**: Fixtures
**Priority**: High
**Difficulty**: Easy
**Parallelizable**: Yes

**Pattern Detected**:
```typescript
const session = {
  id: 'test-id',
  name: 'Test Session',
  status: 'active',
};
```

**Occurrences**: 12 times across 5 files

**Affected Files**:
- src/sdk/session.test.ts (4 occurrences)
- src/repository/session-repository.test.ts (3 occurrences)
- src/daemon/routes/sessions.test.ts (3 occurrences)
- src/polling/monitor.test.ts (2 occurrences)

**Recommended Action**:
Extract to `src/test/fixtures/session.ts`:
```typescript
export function createTestSession(overrides: Partial<Session> = {}): Session {
  return {
    id: 'test-session-id',
    name: 'Test Session',
    status: 'active',
    ...overrides,
  };
}
```

**Impact**: Reduces 48 lines to 1 import per file

---

#### FINDING-002: Repeated Error Assertion Pattern

**Category**: Assertions
**Priority**: Medium
**Difficulty**: Easy
**Parallelizable**: Yes

**Pattern Detected**:
```typescript
expect(result.ok).toBe(false);
if (!result.ok) {
  expect(result.error.code).toBe('ERROR_CODE');
}
```

**Occurrences**: 8 times across 4 files

**Affected Files**:
- src/sdk/queue/manager.test.ts (3 occurrences)
- src/sdk/group/manager.test.ts (2 occurrences)
- src/repository/queue-repository.test.ts (2 occurrences)
- src/daemon/routes/queues.test.ts (1 occurrence)

**Recommended Action**:
Extract to `src/test/helpers/assertions.ts`:
```typescript
export function expectResultError<E extends { code: string }>(
  result: Result<unknown, E>,
  expectedCode: string
): void {
  expect(result.ok).toBe(false);
  if (!result.ok) {
    expect(result.error.code).toBe(expectedCode);
  }
}
```

---

#### FINDING-003: Duplicate Test Cases

**Category**: Duplicates
**Priority**: High
**Difficulty**: Medium
**Parallelizable**: No (affects same test file)

**Pattern Detected**:
Near-identical test cases differing only in input values

**Location**: src/sdk/parser.test.ts

**Duplicate Tests**:
```typescript
it('should parse "hello"', () => {
  expect(parse('hello')).toEqual({ type: 'word', value: 'hello' });
});

it('should parse "world"', () => {
  expect(parse('world')).toEqual({ type: 'word', value: 'world' });
});

// ... 5 more similar tests
```

**Recommended Action**:
Convert to parameterized test:
```typescript
const testCases = [
  { input: 'hello', expected: { type: 'word', value: 'hello' } },
  { input: 'world', expected: { type: 'word', value: 'world' } },
  // ...
];

it.each(testCases)('should parse "$input"', ({ input, expected }) => {
  expect(parse(input)).toEqual(expected);
});
```

---

#### FINDING-004: Skipped Tests

**Category**: Structure
**Priority**: Low
**Difficulty**: Easy
**Parallelizable**: Yes

**Pattern Detected**: Tests marked with `.skip` without explanation

**Occurrences**: 3 tests

**Affected Files**:
- src/sdk/queue/runner.test.ts:45 - `it.skip('should handle timeout')`
- src/polling/watcher.test.ts:120 - `describe.skip('edge cases')`
- src/daemon/server.test.ts:89 - `it.skip('should reconnect')`

**Recommended Action**:
For each skipped test, either:
1. Fix and enable the test
2. Delete if no longer relevant
3. Add TODO comment explaining why skipped

---

#### FINDING-005: Oversized Test File

**Category**: File Split
**Priority**: High
**Difficulty**: Medium
**Parallelizable**: Yes

**File**: src/services/api-service.test.ts
**Lines**: 1247 (exceeds 800 limit)

**File Structure Analysis**:
```
Lines 1-50:     Imports and setup
Lines 51-300:   describe('UserAPI') - 250 lines
Lines 301-600:  describe('ProductAPI') - 300 lines
Lines 601-900:  describe('OrderAPI') - 300 lines
Lines 901-1247: describe('PaymentAPI') - 347 lines
```

**Recommended Split**:
```
BEFORE:
src/services/api-service.test.ts (1247 lines)

AFTER:
src/services/api-service-user.test.ts (~280 lines)
src/services/api-service-product.test.ts (~330 lines)
src/services/api-service-order.test.ts (~330 lines)
src/services/api-service-payment.test.ts (~380 lines)
```

**Split Strategy**: By Feature (each API endpoint group)

**Shared Setup to Extract**:
- Mock server setup (lines 10-30)
- Test fixtures (lines 35-50)
- Extract to: `src/services/__tests__/api-service.setup.ts`

**Dependencies to Update**: None (test file only)

---

### Dependency Groups

**Group A (Independent - Parallelizable)**:
- FINDING-001: Session Fixture
- FINDING-002: Error Assertion Helper
- FINDING-004: Skipped Tests

**Group B (Sequential - Same File)**:
- FINDING-003: Parameterize Parser Tests

**Group C (File Split - Sequential)**:
- FINDING-005: Split api-service.test.ts

### Recommended Execution Order

1. **Phase 1** (Parallel): FINDING-001, FINDING-002 - Create shared helpers first
2. **Phase 2** (Parallel): Update imports in affected files
3. **Phase 3** (Sequential): FINDING-003 - Parameterize tests
4. **Phase 4** (Parallel): FINDING-004 - Clean up skipped tests
5. **Phase 5** (Sequential): FINDING-005 - Split oversized test files

### Estimated Impact

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Lines of test code | ~2000 | ~1700 | -15% |
| Duplicate patterns | 25 | 0 | -100% |
| Shared fixtures | 3 | 12 | +300% |
| Test clarity | Fair | Good | Improved |
| Oversized files | 1 | 0 | -100% |
| Avg file size | 450 lines | 280 lines | -38% |

### New Files to Create

```
src/test/
├── fixtures/
│   ├── index.ts           # Re-exports
│   ├── session.ts         # Session fixtures (FINDING-001)
│   └── queue.ts           # Queue fixtures
├── helpers/
│   ├── index.ts           # Re-exports
│   └── assertions.ts      # Assertion helpers (FINDING-002)
└── mocks/
    └── index.ts           # Re-exports existing mocks

# From file split (FINDING-005)
src/services/
├── api-service-user.test.ts     # Split from api-service.test.ts
├── api-service-product.test.ts  # Split from api-service.test.ts
├── api-service-order.test.ts    # Split from api-service.test.ts
├── api-service-payment.test.ts  # Split from api-service.test.ts
└── __tests__/
    └── api-service.setup.ts     # Shared test setup
```

### Risk Assessment

| Finding | Risk Level | Mitigation |
|---------|------------|------------|
| FINDING-001 | Low | Type-safe factory function |
| FINDING-002 | Low | Wrapper around existing expects |
| FINDING-003 | Medium | Verify all test cases covered |
| FINDING-004 | Low | Review each before action |
| FINDING-005 | Medium | Extract shared setup, update imports, verify coverage |
```

---

## No Findings Response

If no significant refactoring opportunities are found:

```
## Test Refactoring Audit Report

### Audit Scope
- Pattern: [glob pattern]
- Files scanned: [count]
- Categories checked: [list]

### Summary
No significant refactoring opportunities found.

### Minor Observations
[List any minor findings that don't warrant refactoring]

### Current State Assessment
- Test organization: Good
- Fixture usage: Appropriate
- Assertion patterns: Consistent
- Naming conventions: Followed

### Recommendations
The test codebase follows best practices. Continue maintaining current standards.
```

---

## Integration Notes

This agent's output can be used for:
1. Manual refactoring by developers
2. Input to `ts-coding` agent for automated refactoring
3. Documentation for team review

The finding IDs (FINDING-001, etc.) provide tracking throughout the refactoring process.
