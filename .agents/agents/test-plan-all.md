---
name: test-plan-all
description: Generate all test plans from implementations and specs in parallel. Uses divide-and-conquer to avoid context overflow. Spawns multiple test-plan agents as concurrent subtasks. Validates PROGRESS.json after all plans are created.
tools: Read, Write, Edit, Glob, Grep, Task, TaskOutput
model: sonnet
skills: test-plan
---

# Batch Test Plan Generator Subagent

## Overview

This subagent generates all test plans from implementations and specifications by spawning multiple `test-plan` agents in parallel. It uses a divide-and-conquer strategy to avoid context overflow by analyzing the codebase in phases rather than all at once.

## MANDATORY: Required Information in Task Prompt

**CRITICAL**: When invoking this subagent via the Task tool, the caller MUST include the following information in the `prompt` parameter.

### Required Information

1. **Source Directory**: Path to source code (default: `src/`)
2. **Output Directory**: Where to save test plans (default: `test-plans/`)

### Optional Information

- **Impl Plans**: Path to implementation plans for reference (default: `impl-plans/`)
- **Test Types**: Which test types to generate (Unit, Integration, E2E)
- **Focus Areas**: Specific modules or features to prioritize
- **Exclude**: Patterns to skip (e.g., test files, mocks)
- **Dry Run**: If true, only list plans that would be created

### Example Task Tool Invocation

```
Task tool prompt parameter should include:

Source Directory: src/
Output Directory: test-plans/
Impl Plans: impl-plans/
Test Types: Unit, Integration
Exclude: src/test/mocks/*, src/viewer/browser/ui/*
```

---

## Divide and Conquer Strategy

**CRITICAL**: This agent MUST NOT analyze the entire codebase at once.

### Phase 1: Module Discovery (Minimal Context)

Only use Glob to identify module areas. DO NOT read source files yet.

```
Glob: src/**/
Result:
  - src/sdk/queue/
  - src/sdk/group/
  - src/daemon/
  - src/cli/
  ...
```

Group modules by area for test plan generation.

### Phase 2: Impl-Plan Reference (Lightweight)

Read impl-plans/PROGRESS.json to understand implementation structure.
DO NOT read individual implementation plan files.

### Phase 3: Test Discovery (Minimal Context)

Identify existing tests to avoid duplication:

```
Glob: src/**/*.test.ts
```

### Phase 4: Plan Mapping

Map modules to test plans WITHOUT reading source files:

| Module Area | Unit Plan | Integration Plan | E2E Plan |
|-------------|-----------|------------------|----------|
| src/sdk/queue/ | queue-unit.md | queue-integration.md | - |
| src/sdk/group/ | group-unit.md | group-integration.md | group-e2e.md |
| src/daemon/ | daemon-unit.md | daemon-integration.md | - |

### Phase 5: Spawn Parallel Subtasks

For each planned test plan, spawn a `test-plan` agent:

```
For each plan in plans_to_generate:
  spawn Task(
    subagent_type: test-plan,
    prompt: |
      Target: <module-path>
      Test Type: <unit|integration|e2e>
      Output Path: test-plans/<plan-name>.md
    run_in_background: true
  )
```

**IMPORTANT**: Use `run_in_background: true` to run all subtasks concurrently.

### Phase 6: Collect Results

Wait for all subtasks using TaskOutput.
Collect success/failure status for each plan.

### Phase 7: Validate and Finalize PROGRESS.json

After all agents complete, validate PROGRESS.json consistency.

---

## Module to Plan Mapping Strategy

### Granularity Rules

1. **One source directory -> One or more test plans**
2. **Split by test type**: Unit, Integration, E2E
3. **Keep plans focused**: MAX 15 test cases per plan

### Recommended Mapping

| Source Area | Test Plans | Rationale |
|-------------|------------|-----------|
| src/sdk/queue/ | queue-unit.md | Unit tests for queue logic |
| src/sdk/group/ | group-types-unit.md, group-runner-unit.md | Split large module |
| src/daemon/ | daemon-unit.md, daemon-routes-integration.md | Split by type |
| src/cli/ | cli-unit.md | CLI logic tests |
| src/repository/ | repository-unit.md, repository-file-integration.md | Split by type |

### Skip Patterns

Do NOT generate test plans for:
- `src/test/mocks/*` - Mock implementations (already tests)
- `src/viewer/browser/ui/*` - UI components (different testing approach)
- Files ending in `.test.ts` - Already test files

---

## Existing Test Detection

Before generating plans, check what tests already exist:

```
Glob: src/**/*.test.ts

Organize by module:
  src/sdk/queue/manager.test.ts -> tests exist
  src/sdk/queue/runner.test.ts -> tests exist
  src/sdk/queue/types.test.ts -> NO tests (need plan)
```

For modules with existing tests:
1. Still generate plan if coverage is incomplete
2. Mark existing test cases as "Passing" or "Existing"
3. Focus on gaps in coverage

---

## Concurrency Management

### Parallel Agent Limits

- Maximum concurrent agents: 5
- Group related modules in same batch

### Batching Strategy

```
Batch 1 (parallel):
  - queue-unit (test-plan agent)
  - group-types-unit (test-plan agent)
  - bookmarks-unit (test-plan agent)

Wait for Batch 1...

Batch 2 (parallel):
  - daemon-unit (test-plan agent)
  - cli-unit (test-plan agent)

Wait for Batch 2...
```

---

## PROGRESS.json Validation

After all agents complete:

1. **Read test-plans/PROGRESS.json**
2. **Validate all plans are present**
3. **Validate test dependencies**
4. **Update summary statistics**:
   ```json
   "summary": {
     "totalPlans": 12,
     "totalTests": 180,
     "passing": 0,
     "failing": 0,
     "notStarted": 180
   }
   ```
5. **Update test-plans/README.md**

---

## Output Requirements

### Success Response

```
## Batch Test Plan Generation Complete

### Plans Created
| Plan | Source | Test Type | Tests | Status |
|------|--------|-----------|-------|--------|
| queue-unit.md | src/sdk/queue/ | Unit | 12 | Created |
| group-types-unit.md | src/sdk/group/ | Unit | 8 | Created |
| daemon-integration.md | src/daemon/ | Integration | 5 | Created |

### Plans Skipped
| Plan | Reason |
|------|--------|
| mocks-unit.md | Mock implementations - no testing needed |

### Existing Tests Detected
| Module | Tests Found | Coverage |
|--------|-------------|----------|
| src/sdk/queue/manager.ts | 15 | ~80% |
| src/sdk/queue/runner.ts | 8 | ~60% |

### PROGRESS.json Updated
- Plans added: 10
- Total tests defined: 125
- lastUpdated: 2026-01-09T16:00:00Z

### Summary
- Modules analyzed: 15
- Plans created: 10
- Plans skipped: 3
- Errors: 0

### Next Steps
1. Review generated plans in test-plans/
2. Run `/test-exec-auto` to begin test implementation
3. Monitor progress via PROGRESS.json
```

### Failure Response

```
## Batch Test Plan Generation Partial Failure

### Plans Created Successfully
(list of successful plans)

### Plans Failed
| Plan | Error |
|------|-------|
| feature.md | Reason for failure |

### Recommended Actions
- Review failed plans
- Retry with /test-plan for specific failures
```

---

## Important Guidelines

1. **Divide and conquer**: NEVER analyze entire codebase at once
2. **Module discovery first**: Use Glob before reading files
3. **Parallel execution**: Spawn test-plan agents with `run_in_background: true`
4. **Batch appropriately**: Limit concurrent agents to 5
5. **Skip existing**: Do not overwrite existing plans without explicit request
6. **Update README**: Always update test-plans/README.md
7. **Error handling**: Continue with other plans if one fails
8. **Dry run support**: Support listing plans without creating them

## Context Management

### Minimize Context Usage

- Phase 1-3: Only use Glob and minimal reads
- Phase 4: Planning only, no source reads
- Phase 5-7: Delegate to subagents

### Clear Separation

Each test-plan subagent handles its own source analysis.
The orchestrator (this agent) only coordinates and validates.

### Memory-Safe Pattern

```
1. Glob all modules (light)
2. Read PROGRESS.json (one file)
3. Glob existing tests (light)
4. Plan mapping (no file reads)
5. Spawn agents (delegate complexity)
6. Collect results (structured data only)
7. Update PROGRESS.json (one file)
```
