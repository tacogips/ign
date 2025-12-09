---
description: Plan and execute implementation based on spec and progress tracking
argument-hint: [feature-name]
---

## Implementation Progress Command

You are tasked with continuing the implementation of the ign project by:
1. Checking documentation and progress files
2. Planning what to implement in this session
3. Executing the implementation
4. Updating progress tracking

### Current Context

- Working directory: !`pwd`
- Current branch: !`git branch --show-current`
- Existing source files: !`find . -name "*.go" -type f 2>/dev/null | head -20 || echo "No Go files yet"`

### User Request

Feature to focus on (optional): $ARGUMENTS

---

## Phase 1: Gather Documentation and Progress

Read the following files to understand the current state:

1. **Specification**: Read `docs/spec.md` for overall requirements
2. **Reference Documentation**: Check `docs/reference/` for detailed syntax/command specs
3. **Architecture**: Read `docs/implementation/architecture.md` for design patterns
4. **Progress Files**: Check `docs/progress/` for implementation status

If `$ARGUMENTS` specifies a feature name, focus on that feature's progress file (e.g., `docs/progress/<feature-name>.md`).

If no feature is specified, scan all progress files and identify the highest priority incomplete feature.

---

## Phase 2: Plan Implementation

Based on the documentation and progress analysis:

1. **Identify Target**: Determine which feature/component to implement next
   - If a feature is specified via arguments, use that
   - Otherwise, prioritize: "Not Started" > "In Progress" features
   - Consider dependencies between features

2. **Define Scope**: Break down the work into concrete implementation tasks
   - What files need to be created or modified?
   - What interfaces or types are needed?
   - What tests should be written?

3. **Set Completion Criteria**: Define clear success criteria
   - Code compiles without errors
   - Tests pass
   - Matches specification requirements

4. **Create/Update Todo List**: Use TodoWrite to track the implementation tasks

Present the plan to the user with:
- Feature being implemented
- Spec reference
- Implementation tasks (numbered list)
- Completion criteria
- Estimated file changes

Ask for user confirmation before proceeding to Phase 3.

---

## Phase 3: Execute Implementation

Once the plan is approved, execute the implementation using the **go-coding subagent**:

For each implementation task, invoke the Task tool with:
- `subagent_type`: `go-coding`
- `prompt`: Must include all required fields:
  - **Purpose**: What this implementation achieves
  - **Reference Document**: Path to relevant spec/reference doc
  - **Implementation Target**: Specific files/functions to implement
  - **Completion Criteria**: What defines "done"

Example invocation format:
```
Purpose: Implement the TemplateProvider interface for GitHub sources
Reference Document: docs/spec.md (Section 2.3), docs/implementation/architecture.md
Implementation Target: Create internal/provider/github.go with GitHubProvider struct
Completion Criteria:
  - Implements TemplateProvider interface (Fetch, List, Validate methods)
  - Handles github.com/owner/repo URL parsing
  - Returns TemplateRoot with file contents
  - Unit tests cover success and error cases
  - go build and go test pass
```

After each go-coding subagent completes:
- Review the implementation result
- Mark the corresponding todo item as completed
- If there were issues, note them for the progress file

---

## Phase 4: Update Progress

After implementation is complete (or partially complete), update the progress file:

1. **Create progress file if needed**: If `docs/progress/<feature-name>.md` doesn't exist, create it using the format from CLAUDE.md

2. **Update status**:
   - `Not Started` -> `In Progress` (if work began)
   - `In Progress` -> `Completed` (if all items done)

3. **Update Implemented list**: Add completed items with file paths
   ```markdown
   - [x] TemplateProvider interface (`internal/provider/provider.go:15`)
   - [x] GitHubProvider implementation (`internal/provider/github.go`)
   ```

4. **Update Remaining list**: Mark completed items, add any new discovered items

5. **Add Design Decisions**: Document any notable decisions made during implementation

6. **Add Notes**: Record any issues, considerations, or follow-up items

---

## Progress File Template

If creating a new progress file, use this structure:

```markdown
# <Feature Name>

**Status**: Not Started | In Progress | Completed

## Spec Reference
- docs/spec.md Section X.X
- docs/reference/xxx.md

## Implemented
- [ ] Sub-feature A
- [ ] Sub-feature B

## Remaining
- [ ] Sub-feature C
- [ ] Sub-feature D

## Design Decisions
- (none yet)

## Notes
- (none yet)
```

---

## Important Guidelines

1. **Always read before implementing**: Never propose changes to code you haven't read
2. **Follow existing patterns**: Match the project's coding standards from CLAUDE.md
3. **Minimal changes**: Only implement what's needed for the current task
4. **Test coverage**: Ensure tests are written for new functionality
5. **Atomic progress**: Update progress file after each logical unit of work
6. **No over-engineering**: Implement to spec, no extras unless requested
