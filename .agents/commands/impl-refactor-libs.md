---
description: Find redundant custom implementations and replace them with well-known libraries
argument-hint: "[--scope=path] [--dry-run] [--category=name]"
---

## Refactor to Libraries Command

This command audits the codebase for custom implementations that could be replaced with well-known, battle-tested libraries. It then orchestrates concurrent replacements with full review and testing cycles.

### Purpose

- Reduce code redundancy and maintenance burden
- Replace custom utilities with battle-tested libraries
- Improve code quality through standardization
- Leverage community-maintained solutions

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Workflow

```
/impl-refactor-libs
        |
        v
Phase 1: Audit (impl-refactor-libs-audit)
        |
        v
[Show findings, confirm with user]
        |
        v
Phase 2: Replace (impl-refactor-libs-orchestrator)
        |
        +--> ts-coding (concurrent)
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
| `--scope=<path>` | Directory or pattern to audit | `src/` |
| `--dry-run` | Show audit findings only, no replacements | false |
| `--category=<name>` | Focus on specific category | all |
| `--finding=<ids>` | Execute specific findings only | all |

#### Category Options

- `error-handling` - Result types, error utilities
- `validation` - Validation logic, schemas
- `utilities` - Debounce, throttle, deep clone, etc.
- `date-time` - Date formatting, parsing
- `async` - Promise utilities, retry logic
- `id-gen` - ID/UUID generation
- `all` - All categories (default)

### Step 2: Run Audit Phase

Invoke `impl-refactor-libs-audit` to scan the codebase:

```
Task tool parameters:
  subagent_type: impl-refactor-libs-audit
  prompt: |
    Scope: [resolved scope from arguments]
    Categories: [categories from arguments or "all"]
    Exclude: *.test.ts, *.spec.ts, node_modules
```

### Step 3: Present Findings

After audit completes, present findings to the user:

```
## Library Redundancy Audit Results

### Findings Summary
| Category | Count | High Priority | Parallelizable |
|----------|-------|---------------|----------------|
| Error Handling | 2 | 1 | Yes |
| Utilities | 3 | 2 | Partial |

### Top Findings

1. **FINDING-001** (High Priority)
   - Custom Result type -> neverthrow
   - 15 affected files

2. **FINDING-002** (Medium Priority)
   - Custom debounce -> perfect-debounce
   - 3 affected files

[... more findings ...]

### Dependencies to Add
- neverthrow
- perfect-debounce
- nanoid

### Estimated Changes
- Files to modify: 18
- Lines to remove: ~300
- Parallel execution groups: 3
```

### Step 4: User Confirmation

**If `--dry-run` flag present**: Stop after presenting findings.

**Otherwise**: Ask user to confirm:

```
Would you like to proceed with the replacements?

Options:
1. Execute all findings
2. Execute specific findings (provide IDs)
3. Cancel
```

Use `AskUserQuestion` tool to get confirmation.

### Step 5: Execute Replacements

If user confirms, invoke `impl-refactor-libs-orchestrator`:

**For all findings**:
```
Task tool parameters:
  subagent_type: impl-refactor-libs-orchestrator
  prompt: |
    Mode: all

    Audit Report:
    [Full audit report from Step 2]
```

**For specific findings**:
```
Task tool parameters:
  subagent_type: impl-refactor-libs-orchestrator
  prompt: |
    Mode: specific
    Specific Findings: FINDING-001, FINDING-003

    Audit Report:
    [Full audit report from Step 2]
```

### Step 6: Report Results

After orchestrator completes, summarize:

```
## Refactoring Complete

### Summary
- Findings processed: X
- Successfully replaced: Y
- Failed: Z (if any)

### Changes Made
[Summary of files modified, dependencies added]

### Test Status
All tests passing / X tests failing

### Next Steps
[Recommendations based on results]
```

---

## Usage Examples

### Basic Usage (Full Audit and Replace)

```
/impl-refactor-libs
```

Audits entire `src/` directory, presents findings, asks for confirmation, then executes replacements.

### Dry Run (Audit Only)

```
/impl-refactor-libs --dry-run
```

Audits and presents findings without making any changes.

### Scoped Audit

```
/impl-refactor-libs --scope=src/utils/
```

Focuses audit on specific directory.

### Category-Specific

```
/impl-refactor-libs --category=error-handling
```

Only looks for error handling patterns (Result types, etc.).

### Execute Specific Findings

```
/impl-refactor-libs --finding=FINDING-001,FINDING-003
```

Only replaces specified findings (assumes prior audit).

---

## Error Handling

### No Findings

If audit finds no redundant implementations:

```
## Audit Complete - No Action Needed

No significant custom implementations found that warrant library replacement.

The codebase already follows best practices for library usage.

Minor observations:
[List any minor notes from audit]
```

### Partial Failure

If some replacements fail:

```
## Partial Completion

Successfully replaced: FINDING-001, FINDING-002
Failed: FINDING-003

### Failure Details
FINDING-003: Test failures after replacement
[Details from orchestrator]

### Recommendations
1. Review FINDING-003 manually
2. Consider incremental migration
```

### User Cancellation

If user cancels after audit:

```
## Audit Saved

Findings have been reviewed. No changes made.

To execute replacements later, run:
/impl-refactor-libs --finding=FINDING-001,FINDING-002,...
```

---

## Notes

- All replacements go through the full review cycle (up to 3 iterations)
- Tests must pass after each replacement group
- Parallelizable findings are executed concurrently for efficiency
- Dependencies are installed before any code changes
- Failed replacements are rolled back automatically
