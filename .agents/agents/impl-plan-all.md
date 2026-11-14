---
name: impl-plan-all
description: Generate all implementation plans from design documents in parallel. Spawns multiple impl-plan agents as concurrent subtasks to create all plans at once. Validates PROGRESS.json after all plans are created.
tools: Read, Write, Edit, Glob, Grep, Task, TaskOutput
model: sonnet
skills: impl-plan
---

# Batch Plan Generator Subagent

## Overview

This subagent generates all implementation plans from design documents by spawning multiple `impl-plan` agents in parallel. It reads the design documentation, identifies all features that need implementation plans, and creates them concurrently.

## MANDATORY: Required Information in Task Prompt

**CRITICAL**: When invoking this subagent via the Task tool, the caller MUST include the following information in the `prompt` parameter.

### Required Information

1. **Design Directory**: Path to the design documents directory (default: `design-docs/`)
2. **Output Directory**: Where to save implementation plans (default: `impl-plans/`)

### Optional Information

- **Features**: Specific features to generate plans for (if not provided, derives from design docs)
- **Exclude**: Features to skip
- **Dry Run**: If true, only list plans that would be created without creating them

### Example Task Tool Invocation

```
Task tool prompt parameter should include:

Design Directory: design-docs/
Output Directory: impl-plans/
Features: (auto-detect from design documents)
```

---

## Execution Workflow

### Phase 1: Analyze Design Documents

1. **Read DESIGN.md**: Understand overall architecture and phases
2. **Read all spec-*.md files**: Identify feature specifications
3. **Extract feature list**: Build list of features requiring implementation plans

### Phase 2: Plan Feature Mapping

**IMPORTANT**: Implementation plans and spec files do NOT need 1:1 mapping.

Mapping strategies:
- **1:N** (one spec -> multiple plans): Split large specs into smaller, focused implementation units
- **N:1** (multiple specs -> one plan): Combine related specs when they share dependencies
- **1:1** (one spec -> one plan): For well-bounded features

Recommended plan granularity:
- Each plan should be completable in 1-3 sessions
- Each plan should have 3-10 subtasks
- Subtasks should be as parallelizable as possible

Example mapping:

| Design Document(s) | Implementation Plans | Rationale |
|-------------------|---------------------|-----------|
| DESIGN.md, spec-infrastructure.md | foundation-and-core.md | Combined - shared dependencies |
| spec-session-groups.md | session-groups-types.md, session-groups-runner.md | Split - large feature |
| spec-command-queue.md | command-queue.md | 1:1 - well bounded |
| spec-sdk-api.md | markdown-parser.md, http-api.md, daemon-auth.md | Split by domain |
| spec-viewers.md | realtime-monitoring.md | 1:1 - specific feature |
| spec-infrastructure.md | bookmarks.md, caching.md | Split by feature |
| spec-changed-files.md | file-changes.md | 1:1 - specific feature |
| DESIGN.md (CLI section) | cli-core.md, cli-commands.md | Split - large surface area |

### Phase 3: Check Existing Plans

1. **Read impl-plans/**: List existing plans
2. **Read PROGRESS.json**: Check plan statuses
3. **Skip existing**: Do not regenerate plans that already exist

### Phase 4: Spawn Parallel Subtasks

For each feature that needs a plan, spawn a `impl-plan` agent:

```
For each feature in features_to_generate:
  spawn Task(
    subagent_type: impl-plan,
    prompt: |
      Design Document: <design-doc-path>
      Feature Scope: <feature-description>
      Output Path: impl-plans/<feature-name>.md
    run_in_background: true
  )
```

**IMPORTANT**: Use `run_in_background: true` to run all subtasks concurrently.

### Phase 5: Collect Results

1. Wait for all subtasks to complete using TaskOutput
2. Collect success/failure status for each plan
3. Each impl-plan agent updates PROGRESS.json individually

### Phase 6: Validate and Finalize PROGRESS.json

**IMPORTANT**: After all impl-plan agents complete, validate PROGRESS.json consistency.

1. **Read PROGRESS.json**:
   ```
   Read impl-plans/PROGRESS.json
   ```

2. **Validate all plans are present**:
   - Compare plans in PROGRESS.json with plans created
   - Report any missing entries

3. **Validate cross-plan dependencies**:
   - For each task with cross-plan deps like `"other-plan:TASK-001"`
   - Verify the referenced plan and task exist
   - Report any broken references

4. **Update phase statuses**:
   ```json
   "phases": {
     "1": { "status": "COMPLETED" },
     "2": { "status": "READY" },      // Has plans with no blocked deps
     "3": { "status": "BLOCKED" },    // All plans depend on Phase 2
     "4": { "status": "BLOCKED" }     // All plans depend on Phase 3
   }
   ```

5. **Update impl-plans/README.md** with new plans:
   - Add entries to Active Plans section
   - Update Phase to Plans Mapping if needed

---

## Feature Detection Logic

### From DESIGN.md Implementation Phases

Extract features from implementation phases:
- Phase 4: Session Groups -> `session-groups.md`
- Phase 5: Command Queue -> `command-queue.md`
- Phase 6: Markdown Parser -> `markdown-parser.md`
- Phase 7: Real-Time Monitoring -> `realtime-monitoring.md`
- Phase 8: Bookmark System -> `bookmarks.md`
- Phase 9: File Change Service -> `file-changes.md`
- Phase 10: SDK Entry Point -> `sdk-entry.md`
- Phase 11: HTTP API -> `http-api.md`
- Phase 12: CLI -> `cli.md`

### Skip If Exists

Do not generate plans for:
- Features with existing plans in `impl-plans/`
- Features already tracked in PROGRESS.json
- Foundation layer (already covered by `foundation-*.md` plans)

---

## Output Requirements

### Success Response

```
## Batch Plan Generation Complete

### Plans Created
| Plan | Design Reference | Tasks | Phase | Status |
|------|------------------|-------|-------|--------|
| session-groups.md | spec-session-groups.md | 5 | 2 | Created |
| command-queue.md | spec-command-queue.md | 6 | 2 | Created |
| markdown-parser.md | spec-sdk-api.md#markdown | 4 | 2 | Created |

### Plans Skipped (Already Exist)
| Plan | Reason |
|------|--------|
| foundation-and-core.md | Already exists in active/ |

### PROGRESS.json Updated
- Plans added: 7
- Total tasks added: 42
- Cross-plan dependencies validated: 15
- Broken references: 0
- lastUpdated: 2026-01-06T16:00:00Z

### Phase Status
| Phase | Status | Plans |
|-------|--------|-------|
| 1 | COMPLETED | foundation-* |
| 2 | READY | session-groups-*, command-queue-*, etc. |
| 3 | BLOCKED | daemon-core, http-api, sse-events |
| 4 | BLOCKED | browser-viewer-*, cli-* |

### Summary
- Total features detected: 10
- Plans created: 7
- Plans skipped: 3
- Errors: 0

### Next Steps
1. Review generated plans in impl-plans/
2. Run `/impl-exec-auto` to begin implementation
3. Monitor progress via PROGRESS.json
```

### Failure Response

```
## Batch Plan Generation Partial Failure

### Plans Created Successfully
(list of successful plans)

### Plans Failed
| Plan | Error |
|------|-------|
| feature.md | Reason for failure |

### Recommended Actions
- Review failed plans
- Retry with /impl-plan for specific failures
```

---

## Important Guidelines

1. **Parallel execution**: Always spawn impl-plan agents with `run_in_background: true`
2. **Skip existing**: Never overwrite existing plans without explicit request
3. **Update README**: Always update impl-plans/README.md with new plans
4. **Error handling**: Continue with other plans if one fails
5. **Dry run support**: Support listing plans without creating them
6. **TypeScript-first format**: Plans must use actual TypeScript code blocks, not prose descriptions
7. **Simple tables**: Use simple status tables (Module | File Path | Status | Tests)
8. **Checklist-based**: Use checkboxes for completion tracking
9. **File size limits**: Each plan MUST stay under 400 lines - split large features

## File Size Limits (CRITICAL)

**Large implementation plan files cause Claude Code OOM errors.**

### Hard Limits Per Plan

| Metric | Limit |
|--------|-------|
| **Line count** | MAX 400 lines |
| **Modules per plan** | MAX 8 modules |
| **Tasks per plan** | MAX 10 tasks |

### Splitting Large Features

When a feature would exceed limits, create multiple plans:

```
BEFORE (one large feature):
foundation-and-core.md (1100+ lines) -> OOM RISK

AFTER (split by phase):
foundation-interfaces.md (~200 lines)
foundation-mocks.md (~150 lines)
foundation-types.md (~150 lines)
foundation-core-services.md (~200 lines)
```

### Updated Mapping Example

| Design Document(s) | Implementation Plans | Line Estimate |
|-------------------|---------------------|---------------|
| DESIGN.md, spec-infrastructure.md | foundation-interfaces.md, foundation-mocks.md, foundation-types.md, foundation-services.md | ~200 each |
| spec-session-groups.md | session-groups-types.md, session-groups-runner.md | ~250 each |
| spec-command-queue.md | command-queue.md | ~300 |
