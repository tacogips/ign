---
description: Audit and refactor tests for better maintainability (duplicates, DRY fixtures, assertions, file splitting)
argument-hint: "[scope] [--category=<cat>] [--threshold=<n>] [--max-lines=<n>]"
---

## Test Refactoring Command

This command audits test files for refactoring opportunities and optionally executes the refactoring.

### Refactoring Categories

- **Duplicates**: Duplicate test code across files
- **Fixtures**: Repeated test fixtures that can be shared
- **Assertions**: Common assertion patterns to extract
- **Structure**: Test organization improvements
- **Naming**: Test naming consistency
- **File Split**: Test files exceeding line limit (default 800 lines)

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`
- Test file count: !`find src -name "*.test.ts" | wc -l`

### Arguments Received

$ARGUMENTS

---

## Instructions

### Step 1: Parse Arguments

Parse `$ARGUMENTS` to extract:

1. **Scope** (optional): Directory or glob pattern
   - Example: `src/sdk/`
   - Example: `src/**/*.test.ts`
   - Default: `src/**/*.test.ts`

2. **Category** (optional): `--category=<cat>` or `-c <cat>`
   - Options: `duplicates`, `fixtures`, `assertions`, `structure`, `naming`, `split`, `all`
   - Default: `all`

3. **Threshold** (optional): `--threshold=<n>` or `-t <n>`
   - Minimum occurrences to report
   - Default: `2`

4. **Max Lines** (optional): `--max-lines=<n>` or `-m <n>`
   - Line threshold for file split recommendation
   - Default: `800`

5. **Execute** (optional): `--execute` or `-e`
   - If present, execute refactoring after audit
   - Default: audit only

### Step 2: Run Audit

Invoke the `test-refactor-audit` subagent using the Task tool:

```
Task tool parameters:
  subagent_type: test-refactor-audit
  prompt: |
    Audit test files for refactoring opportunities.

    Scope: <parsed-scope>
    Categories: <parsed-categories>
    Threshold: <parsed-threshold>
    Max Lines: <parsed-max-lines or 800>

    Please analyze test files and report findings in structured format.
    Include file split recommendations for files exceeding the max lines limit.
```

### Step 3: Present Findings

After the audit completes:

1. Display summary table of findings
2. Show high-priority items first
3. List affected files with occurrence counts
4. Show recommended execution order

### Step 4: Execute Refactoring (if requested)

If `--execute` flag is present:

1. Confirm with user before proceeding
2. For each finding (in recommended order):
   - Spawn `ts-coding` agent to perform the refactoring
   - Run tests to verify no regressions
   - Report progress
3. Run full test suite after all refactoring
4. Report final summary

---

## Usage Examples

**Basic audit (all categories)**:
```
/test-refactor
```

**Audit specific directory**:
```
/test-refactor src/sdk/
```

**Audit specific category**:
```
/test-refactor --category=fixtures
/test-refactor -c duplicates
/test-refactor --category=split
```

**File split with custom line limit**:
```
/test-refactor --category=split --max-lines=500
/test-refactor -c split -m 600
```

**With threshold**:
```
/test-refactor --threshold=3
/test-refactor -t 5
```

**Audit and execute**:
```
/test-refactor --execute
/test-refactor src/daemon/ -e
```

**Combined options**:
```
/test-refactor src/sdk/ --category=fixtures --threshold=3 --execute
```

---

## Expected Output Format

### Audit-Only Output

```
## Test Refactoring Audit Results

### Summary
- Files scanned: 45
- Findings: 12
- High priority: 4
- Medium priority: 5
- Low priority: 3

### High Priority Findings

#### FINDING-001: Duplicate Session Fixture (12 occurrences)
- Category: Fixtures
- Difficulty: Easy
- Files: 5 (session.test.ts, repository.test.ts, ...)
- Action: Extract to src/test/fixtures/session.ts

#### FINDING-002: Repeated Error Assertions (8 occurrences)
- Category: Assertions
- Difficulty: Easy
- Files: 4
- Action: Extract to src/test/helpers/assertions.ts

#### FINDING-003: Oversized Test File (1247 lines)
- Category: File Split
- Difficulty: Medium
- File: src/services/api-service.test.ts
- Lines: 1247 (exceeds 800 limit)
- Action: Split into focused test files by feature

[... more findings ...]

### Recommended Next Steps

1. Run `/test-refactor --execute` to perform refactoring
2. Or manually address findings in priority order
3. Run `bun test` after each change to verify
```

### Execute Output

```
## Test Refactoring Execution

### Phase 1: Creating Shared Helpers
- [x] Created src/test/fixtures/session.ts
- [x] Created src/test/helpers/assertions.ts
- Tests passing: Yes

### Phase 2: Updating Imports
- [x] Updated src/sdk/session.test.ts (4 changes)
- [x] Updated src/repository/session-repository.test.ts (3 changes)
- Tests passing: Yes

### Phase 3: Parameterizing Tests
- [x] Converted 7 tests to parameterized in parser.test.ts
- Tests passing: Yes

### Phase 4: File Splitting
- [x] Split api-service.test.ts (1247 lines) into:
  - api-service-user.test.ts (280 lines)
  - api-service-product.test.ts (350 lines)
  - api-service-order.test.ts (320 lines)
  - api-service-payment.test.ts (297 lines)
- Tests passing: Yes

### Final Summary
- Findings addressed: 13/13
- Lines reduced: ~300
- Files split: 1 -> 4
- All tests passing: Yes
- Coverage unchanged: Yes
```

---

## Error Handling

If no test files found:
```
No test files found matching pattern: <scope>

Available test files in project:
  - src/**/*.test.ts (45 files)

Try:
  /test-refactor src/
  /test-refactor "**/*.test.ts"
```

If audit fails:
```
Audit failed: <error message>

Possible causes:
- Invalid scope pattern
- No read access to files
- Syntax errors in test files

Try running with a more specific scope or check file permissions.
```

---

## Related Commands

- `/test-plan` - Create test plans from implementation
- `/test-exec-auto` - Execute tests from test plans
- `/impl-refactor-libs` - Refactor implementations to use libraries
