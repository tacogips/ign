---
name: impl-plan
description: Use when creating implementation plans from design documents. Provides plan structure, status tracking, and progress logging guidelines.
allowed-tools: Read, Write, Glob, Grep
---

# Implementation Plan Skill

This skill provides guidelines for creating and managing implementation plans from design documents.

## When to Apply

Apply this skill when:
- Translating design documents into actionable implementation plans
- Planning multi-session implementation work
- Breaking down large features into parallelizable subtasks
- Tracking implementation progress across sessions

## Purpose

Implementation plans bridge the gap between design documents (what to build) and actual implementation (how to build). They provide:
- Clear deliverables with TypeScript type definitions
- Simple status tracking tables
- Checklist-based completion criteria
- Progress tracking across sessions

## Plan Granularity

**IMPORTANT**: Implementation plans and spec files do NOT need 1:1 mapping.

| Mapping | When to Use |
|---------|-------------|
| **1:N** (one spec -> multiple plans) | Large specs should be split into smaller, focused units |
| **N:1** (multiple specs -> one plan) | Related specs sharing dependencies can be combined |
| **1:1** (one spec -> one plan) | Well-bounded features with clear scope |

**Recommended granularity**:
- Each plan should be completable in 1-3 sessions
- Each plan should have 3-10 subtasks
- Maximize parallelizable subtasks

## File Size Limits

**CRITICAL**: Large implementation plan files can make agent execution brittle
and hard to review.

### Hard Limits

| Metric | Limit | Reason |
|--------|-------|--------|
| **Line count** | MAX 1000 lines | Keeps plans readable while allowing realistic implementation detail |
| **Modules per plan** | MAX 8 modules | Keeps plans focused and manageable |
| **Tasks per plan** | MAX 10 tasks | Enables completion in 1-3 sessions |

### When to Split Plans

Split a plan into multiple files when ANY of these conditions are met:

1. **Line count exceeds 1000 lines**: Split by phase or module category
2. **More than 8 modules**: Group related modules into separate plans
3. **More than 10 tasks**: Break into logical sub-plans
4. **Multiple phases with dependencies**: Create separate plans per phase

### Splitting Strategy

```
BEFORE (one large plan):
impl-plans/foundation-and-core.md (1100+ lines)

AFTER (split by phase):
impl-plans/foundation-interfaces.md (~200 lines)
impl-plans/foundation-mocks.md (~150 lines)
impl-plans/foundation-types.md (~150 lines)
impl-plans/foundation-core-services.md (~200 lines)
```

### Split Plan Naming Convention

When splitting, use consistent naming:
- `{feature}-{phase}.md` - For phase-based splits
- `{feature}-{category}.md` - For category-based splits

Example:
- `session-groups-types.md`
- `session-groups-repository.md`
- `session-groups-manager.md`

### Cross-References Between Split Plans

Each split plan MUST include:
```markdown
## Related Plans
- **Previous**: `impl-plans/foundation-interfaces.md` (Phase 1)
- **Next**: `impl-plans/foundation-core-services.md` (Phase 3)
- **Depends On**: `foundation-interfaces.md`, `foundation-types.md`
```

## Output Location

**IMPORTANT**: All implementation plans MUST be stored directly under `impl-plans/`.

```
impl-plans/
├── README.md              # Index of all implementation plans
├── PROGRESS.json          # Task status index (single source of truth)
├── <feature>.md           # Implementation plan files
├── <feature>-types.md     # Split plans use consistent naming
└── templates/             # Plan templates
    └── plan-template.md   # Standard plan template
```

## Directory Rules

| Location | Purpose |
|----------|---------|
| `impl-plans/*.md` | All implementation plan files (no subdirectories) |
| `impl-plans/PROGRESS.json` | Single source of truth for plan/task status |
| `impl-plans/templates/` | Plan templates and examples |

**Plan status is tracked in PROGRESS.json, not by file location.**

**DO NOT** create implementation plan files outside `impl-plans/`.

## Implementation Plan Structure

Each implementation plan file MUST include:

### 1. Header Section
```markdown
# <Feature Name> Implementation Plan

**Status**: Planning | Ready | In Progress | Completed
**Design Reference**: design-docs/<file>.md#<section>
**Created**: YYYY-MM-DD
**Last Updated**: YYYY-MM-DD
```

### 2. Design Document Reference
- Link to specific design document section
- Summary of what is being implemented
- Scope boundaries (what is NOT included)

### 3. Modules and Types

List each module with its TypeScript type definitions. **USE ACTUAL TYPESCRIPT CODE** for interfaces and types - not prose descriptions.

```markdown
## Modules

### 1. Core Interfaces

#### src/interfaces/filesystem.ts

**Status**: NOT_STARTED

```typescript
interface FileSystem {
  readFile(path: string): Promise<string>;
  writeFile(path: string, content: string): Promise<void>;
  exists(path: string): Promise<boolean>;
  watch(path: string): AsyncIterable<WatchEvent>;
}

interface WatchEvent {
  type: 'create' | 'modify' | 'delete';
  path: string;
}
```

**Checklist**:
- [ ] Define FileSystem interface
- [ ] Define WatchEvent interface
- [ ] Export from interfaces/index.ts
- [ ] Unit tests
```

### 4. Status Tracking Table

Use simple tables for overview tracking:

```markdown
## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| FileSystem interface | `src/interfaces/filesystem.ts` | NOT_STARTED | - |
| ProcessManager interface | `src/interfaces/process-manager.ts` | NOT_STARTED | - |
| Mock implementations | `src/test/mocks/*.ts` | NOT_STARTED | - |
```

### 5. Dependencies

Simple table showing what depends on what:

```markdown
## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| Phase 2: Repository | Phase 1: Interfaces | BLOCKED |
| Phase 3: Core Services | Phase 1, Phase 2 | BLOCKED |
```

### 6. Completion Criteria

Simple checklist:

```markdown
## Completion Criteria

- [ ] All modules implemented
- [ ] All tests passing
- [ ] Type checking passes
- [ ] Integration verified
```

### 7. Progress Log

Track session-by-session progress:

```markdown
## Progress Log

### Session: YYYY-MM-DD HH:MM
**Tasks Completed**: Module 1, Module 2
**Tasks In Progress**: Module 3
**Blockers**: None
**Notes**: Discovered edge case in variable parsing
```

## Content Guidelines

### INCLUDE TypeScript Code

**ALWAYS** include actual TypeScript code for:
- Interface definitions
- Type aliases
- Class structure (public methods, constructor signature)
- Function signatures

Example:
```markdown
```typescript
interface SessionGroup {
  id: string;                    // Format: YYYYMMDD-HHMMSS-{slug}
  name: string;
  status: GroupStatus;
  sessions: GroupSession[];
  config: GroupConfig;
  createdAt: string;             // ISO timestamp
}

type GroupStatus = 'created' | 'running' | 'paused' | 'completed' | 'failed';
```
```

### DO NOT Include

- Implementation logic (function bodies)
- Private methods
- Algorithm details
- Excessive prose descriptions

### Format Comparison

**GOOD** (TypeScript-first):
```markdown
#### src/interfaces/clock.ts

```typescript
interface Clock {
  now(): Date;
  timestamp(): string;
  sleep(ms: number): Promise<void>;
}
```

**Checklist**:
- [ ] Define Clock interface
- [ ] Export from interfaces/index.ts
```

**BAD** (Prose-heavy):
```markdown
**Exports**:
| Name | Type | Purpose | Called By |
|------|------|---------|-----------|
| `Clock` | interface | Time operations | Caching, logging |

**Function Signatures**:
now(): Date
  Purpose: Get current date/time
  Called by: Logger, Cache
```

## Task Definition with Dependencies

**CRITICAL**: Each task MUST have explicit task ID and dependency information for PROGRESS.json integration.

### Task Structure Format

```markdown
### TASK-001: Core Types

**Status**: Not Started
**Parallelizable**: Yes
**Deliverables**: `src/sdk/queue/types.ts`, `src/sdk/queue/events.ts`
**Dependencies**: None

**Description**:
Define core type definitions for the queue system.

**Completion Criteria**:
- [ ] Types defined
- [ ] Type checking passes
- [ ] Unit tests written
```

### Task ID Format

- Format: `TASK-XXX` where XXX is zero-padded number (001, 002, etc.)
- IDs are unique within a plan
- Cross-plan references use format: `<plan-name>:TASK-XXX`

### Dependency Specification

**Same-plan dependency**:
```markdown
**Dependencies**: TASK-001
**Dependencies**: TASK-001, TASK-002
```

**Cross-plan dependency**:
```markdown
**Dependencies**: session-groups-types:TASK-001
**Dependencies**: session-groups-types:TASK-001, command-queue-types:TASK-002
```

**No dependencies**:
```markdown
**Dependencies**: None
```

### Dependency Identification Rules

Identify dependencies by analyzing:

1. **Type dependencies**: Does this task use types defined in another task?
2. **Interface dependencies**: Does this implement an interface from another task?
3. **Import dependencies**: Will the code import from files created by another task?
4. **Execution order**: Must another task complete first for this to be testable?

Example analysis:
```
TASK-001: Define QueueRepository interface
TASK-002: Implement FileQueueRepository (implements QueueRepository)
  -> TASK-002 depends on TASK-001

TASK-003: Define QueueManager class (uses QueueRepository)
  -> TASK-003 depends on TASK-001

TASK-004: Define types (independent)
  -> No dependencies, parallelizable
```

### Parallelization Rules

Tasks can be parallelized when:
1. No data dependencies between tasks
2. No shared file modifications
3. No order-dependent side effects

**Parallelizable: Yes** means the task has no blocking dependencies on other incomplete tasks.
**Parallelizable: No** is used when the task depends on other tasks (list in Dependencies field).

## PROGRESS.json Integration

**CRITICAL**: After creating a plan file, PROGRESS.json MUST be updated to include the new plan and its tasks.

### PROGRESS.json Structure

```json
{
  "lastUpdated": "2026-01-06T16:00:00Z",
  "phases": {
    "1": { "status": "COMPLETED" },
    "2": { "status": "READY" },
    "3": { "status": "BLOCKED" },
    "4": { "status": "BLOCKED" }
  },
  "plans": {
    "plan-name": {
      "phase": 2,
      "status": "Ready",
      "tasks": {
        "TASK-001": { "status": "Not Started", "parallelizable": true, "deps": [] },
        "TASK-002": { "status": "Not Started", "parallelizable": false, "deps": ["TASK-001"] },
        "TASK-003": { "status": "Not Started", "parallelizable": false, "deps": ["other-plan:TASK-001"] }
      }
    }
  }
}
```

### Dependency Format in PROGRESS.json

| Dependency Type | Plan File Format | PROGRESS.json Format |
|-----------------|------------------|---------------------|
| None | `**Dependencies**: None` | `"deps": []` |
| Same-plan | `**Dependencies**: TASK-001` | `"deps": ["TASK-001"]` |
| Same-plan multiple | `**Dependencies**: TASK-001, TASK-002` | `"deps": ["TASK-001", "TASK-002"]` |
| Cross-plan | `**Dependencies**: other-plan:TASK-001` | `"deps": ["other-plan:TASK-001"]` |
| Mixed | `**Dependencies**: TASK-001, other-plan:TASK-002` | `"deps": ["TASK-001", "other-plan:TASK-002"]` |

### Phase Assignment

Assign the plan to a phase based on its dependencies:

| Condition | Phase |
|-----------|-------|
| No cross-plan dependencies | Phase 2 (or current active phase) |
| Depends on Phase 2 plans | Phase 3 |
| Depends on Phase 3 plans | Phase 4 |

### PROGRESS.json Update Protocol

When creating a new plan:

1. **Read current PROGRESS.json**
2. **Extract tasks from plan file**:
   - Parse all `### TASK-XXX:` sections
   - Extract `**Status**`, `**Parallelizable**`, `**Dependencies**`
3. **Convert to PROGRESS.json format**:
   ```python
   for task in plan_tasks:
       task_entry = {
           "status": task.status,  # "Not Started", "In Progress", "Completed"
           "parallelizable": task.parallelizable,  # true/false
           "deps": parse_dependencies(task.dependencies)  # ["TASK-001", "other-plan:TASK-002"]
       }
   ```
4. **Add plan to PROGRESS.json**:
   ```json
   "new-plan-name": {
     "phase": <determined-phase>,
     "status": "Ready",
     "tasks": { ... }
   }
   ```
5. **Update lastUpdated timestamp**
6. **Write PROGRESS.json** (with file locking if concurrent access possible)

### File Locking Protocol

When updating PROGRESS.json, use file locking to prevent race conditions:

```bash
# Acquire lock
while [ -f impl-plans/.progress.lock ]; do sleep 1; done
echo "<plan-name>" > impl-plans/.progress.lock

# ... perform PROGRESS.json update ...

# Release lock
rm -f impl-plans/.progress.lock
```

## Workflow Integration

### Creating a Plan
1. Read the design document
2. Identify feature boundaries
3. Define TypeScript interfaces and types
4. Define tasks with explicit IDs and dependencies
5. List modules with status tracking
6. Set completion criteria
7. Create plan file in `impl-plans/<feature>.md`
8. **Update PROGRESS.json with new plan and tasks**
9. **Update impl-plans/README.md with new plan entry**

### During Implementation
1. Update task status in PROGRESS.json as work progresses
2. Update module status in plan file
3. Add progress log entries per session
4. Note blockers and decisions
5. Check off completion criteria

### Completing a Plan
1. Verify all completion criteria met
2. Update plan status to "Completed" in PROGRESS.json
3. Update plan file header status to "Completed"
4. Add final progress log entry

**Note**: No file move is required. PROGRESS.json is the single source of truth for plan status.

## Quick Reference

| Section | Required | Format |
|---------|----------|--------|
| Header | Yes | Markdown metadata |
| Design Reference | Yes | Link + summary |
| Modules | Yes | TypeScript code blocks + checklist |
| Status Table | Yes | Simple table |
| Dependencies | Yes | Simple table |
| Completion Criteria | Yes | Checklist |
| Progress Log | Yes | Session entries |
