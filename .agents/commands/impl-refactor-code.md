---
description: Refactor changed code files - fix duplicates, naming issues, and split large files
argument-hint: "[--base=branch] [--dry-run] [--category=dup|name|split] [--max-lines=800]"
---

## Code Refactoring Command

This command audits files changed from the base branch and performs refactoring:
- Consolidate duplicate functions within files
- Fix typos and unclear/inappropriate variable/function names
- Split files exceeding the line limit (default 800 lines)

### Purpose

- Improve code maintainability
- Eliminate code duplication
- Enforce consistent naming conventions
- Keep files focused and manageable in size

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`
- Base branch: !`git remote show origin 2>/dev/null | grep 'HEAD branch' | cut -d' ' -f5 || echo "main"`

### Arguments Received

$ARGUMENTS

---

## Workflow

```
/impl-refactor-code
        |
        v
Phase 1: Audit (impl-refactor-code-audit)
        |
        v
[Show findings, confirm with user]
        |
        v
Phase 2: Refactor (impl-refactor-code-orchestrator)
        |
        +--> ts-coding (concurrent for parallelizable)
        +--> check-and-test-after-modify
        +--> ts-review (review cycle)
        |
        v
[Final Report]
```

---

## Instructions

### Step 1: Parse Arguments

Parse `$ARGUMENTS` for options:

| Option | Description | Default |
|--------|-------------|---------|
| `--base=<branch>` | Base branch to compare against | auto-detect (main/master) |
| `--dry-run` | Show audit findings only, no refactoring | false |
| `--category=<name>` | Focus on specific category | all |
| `--finding=<ids>` | Execute specific findings only | all |
| `--max-lines=<n>` | Line threshold for file split | 800 |

#### Category Options

- `dup` - Duplicate function detection and consolidation
- `name` - Naming issues (typos, unclear names, conventions)
- `split` - Large file splitting
- `all` - All categories (default)

### Step 2: Run Audit Phase

Invoke `impl-refactor-code-audit` to analyze changed files:

```
Task tool parameters:
  subagent_type: impl-refactor-code-audit
  prompt: |
    Base Branch: [resolved from arguments or auto-detect]
    Max File Lines: [from arguments or 800]
    Categories: [from arguments or "all"]
    Exclude Patterns: *.test.ts, *.spec.ts, __tests__/*, __mocks__/*
```

### Step 3: Present Findings

After audit completes, present findings to the user:

```
## Code Refactoring Audit Results

### Changed Files
- Base Branch: main
- Files with changes: 12
- Files analyzed (excluding tests): 8

### Findings Summary
| Category | Count | High Priority | Parallelizable |
|----------|-------|---------------|----------------|
| Duplicate Functions | 2 | 1 | Yes |
| Naming Issues | 5 | 2 | Yes |
| File Split | 1 | 1 | Sequential |

### Findings by Category

#### Duplicate Functions (2 findings)
1. **DUP-001** (High Priority)
   - File: src/services/user-service.ts
   - Issue: processUserData and processUserDataForExport share 80% logic

2. **DUP-002** (Medium Priority)
   - File: src/utils/validators.ts
   - Issue: validateEmail and validateEmailFormat nearly identical

#### Naming Issues (5 findings)
1. **NAME-001** (Medium Priority)
   - File: src/utils/parser.ts
   - Issues: 3 unclear variable names (d, temp, result)

2. **NAME-002** (Low Priority)
   - File: src/services/auth-service.ts
   - Issues: 2 typos (recieve, seperate)

[... more findings ...]

#### File Split (1 finding)
1. **SPLIT-001** (High Priority)
   - File: src/services/api-service.ts
   - Lines: 1247 (exceeds 800 limit)
   - Recommended: Split into 6 modules

### Execution Plan
- Group 1 (Parallel): DUP-001, DUP-002, NAME-001, NAME-002, ...
- Group 2 (Sequential): SPLIT-001
```

### Step 4: User Confirmation

**If `--dry-run` flag present**: Stop after presenting findings.

**Otherwise**: Ask user to confirm:

```
Would you like to proceed with the refactoring?

Options:
1. Execute all findings
2. Execute specific findings (provide IDs)
3. Cancel
```

Use `AskUserQuestion` tool to get confirmation.

### Step 5: Execute Refactoring

If user confirms, invoke `impl-refactor-code-orchestrator`:

**For all findings**:
```
Task tool parameters:
  subagent_type: impl-refactor-code-orchestrator
  prompt: |
    Mode: all

    Audit Report:
    [Full audit report from Step 2]
```

**For specific findings**:
```
Task tool parameters:
  subagent_type: impl-refactor-code-orchestrator
  prompt: |
    Mode: specific
    Specific Findings: DUP-001, NAME-002

    Audit Report:
    [Full audit report from Step 2]
```

### Step 6: Report Results

After orchestrator completes, summarize:

```
## Refactoring Complete

### Summary
- Findings processed: X
- Successfully refactored: Y
- Failed: Z (if any)

### Changes Made

#### Duplicate Functions Consolidated
- DUP-001: Extracted normalizeUser() in user-service.ts

#### Naming Issues Fixed
- NAME-001: Renamed 3 variables in parser.ts
- NAME-002: Fixed 2 typos in auth-service.ts

#### Files Split
- SPLIT-001: api-service.ts -> 6 modules in src/services/api/

### Test Status
All tests passing / X tests failing

### Files Modified
[List of all modified files]

### Next Steps
[Recommendations based on results]
```

---

## Usage Examples

### Basic Usage (Full Audit and Refactor)

```
/impl-refactor-code
```

Audits all changed files from base branch, presents findings, asks for confirmation, then executes refactoring.

### Dry Run (Audit Only)

```
/impl-refactor-code --dry-run
```

Audits and presents findings without making any changes.

### Specific Base Branch

```
/impl-refactor-code --base=develop
```

Compares against `develop` branch instead of auto-detected main/master.

### Category-Specific

```
/impl-refactor-code --category=dup
```

Only looks for duplicate functions.

```
/impl-refactor-code --category=name
```

Only looks for naming issues.

```
/impl-refactor-code --category=split
```

Only looks for oversized files.

### Custom Line Limit

```
/impl-refactor-code --max-lines=500
```

Flags files exceeding 500 lines (instead of default 800).

### Execute Specific Findings

```
/impl-refactor-code --finding=DUP-001,NAME-002
```

Only refactors specified findings (assumes prior audit or knows finding IDs).

---

## Error Handling

### No Changed Files

If no files have changed from base branch:

```
## No Changes Detected

No files have changed from the base branch (main).

Current branch: feature/my-feature
Base branch: main

### Next Steps
- Make code changes
- Run the audit again
```

### No Findings

If audit finds no refactoring opportunities:

```
## Audit Complete - No Action Needed

All changed files pass quality checks:
- No duplicate function patterns detected
- Naming conventions followed
- All files under 800 lines

### Files Analyzed
[List of analyzed files]
```

### Partial Failure

If some refactoring fails:

```
## Partial Completion

Successfully refactored: DUP-001, NAME-001
Failed: SPLIT-001

### Failure Details
SPLIT-001: Circular import introduced during split

### Recommendations
1. Review SPLIT-001 manually
2. Consider different module boundaries
```

### User Cancellation

If user cancels after audit:

```
## Audit Complete - No Changes Made

Findings have been reviewed. No refactoring performed.

To execute specific refactoring later, run:
/impl-refactor-code --finding=DUP-001,NAME-002,...
```

---

## Notes

- Only non-test files are analyzed (*.test.ts, *.spec.ts excluded)
- All refactoring goes through the full review cycle (up to 3 iterations)
- Tests must pass after each refactoring group
- Parallelizable findings are executed concurrently for efficiency
- File splits are executed sequentially due to import dependencies
- Failed refactoring does not affect other findings
