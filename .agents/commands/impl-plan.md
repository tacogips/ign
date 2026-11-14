---
description: Generate implementation plan from design document
argument-hint: "<design-doc-path> [feature-name]"
---

## Generate Implementation Plan Command

This command creates an implementation plan from a design document.

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`

### Arguments Received

$ARGUMENTS

---

## Instructions

Invoke the `impl-plan` subagent using the Task tool.

### Argument Parsing

Parse `$ARGUMENTS` to extract:

1. **Design Document Path** (required): Path to design document
   - Can be relative path: `design-docs/DESIGN.md`
   - Can include section: `design-docs/spec-session-groups.md#lifecycle`

2. **Feature Name** (optional): Short name for the feature
   - If not provided, derive from design document section
   - Used for output file naming

### Determine Output Path

Generate the output path based on feature name:
- If feature name provided: `impl-plans/<feature-name>.md`
- If not provided: Derive from design document path

### Invoke Subagent

```
Task tool parameters:
  subagent_type: impl-plan
  prompt: |
    Design Document: <parsed-design-doc-path>
    Feature Scope: <parsed-or-derived-feature-scope>
    Output Path: <generated-output-path>
```

### Usage Examples

**Basic usage with design doc path**:
```
/impl-plan design-docs/spec-session-groups.md
```
Creates: `impl-plans/session-groups.md`

**With explicit feature name**:
```
/impl-plan design-docs/DESIGN.md foundation-layer
```
Creates: `impl-plans/foundation-layer.md`

**With section reference**:
```
/impl-plan design-docs/spec-infrastructure.md#testability testability
```
Creates: `impl-plans/testability.md`

### After Subagent Completes

1. Report the created plan file path to the user
2. Summarize the tasks defined with their dependencies
3. Confirm PROGRESS.json was updated with the new plan
4. Suggest next steps:
   - Review the generated plan
   - Run `/impl-exec-auto` to begin implementation

### Error Handling

If no arguments provided, respond with usage instructions:
```
Usage: /impl-plan <design-doc-path> [feature-name]

Examples:
  /impl-plan design-docs/spec-session-groups.md
  /impl-plan design-docs/DESIGN.md foundation-layer
  /impl-plan design-docs/spec-infrastructure.md#testability testability

The design document path is required. Feature name is optional.
```
