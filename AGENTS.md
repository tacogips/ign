# AGENTS.md

This file provides guidance to AI coding assistants working in this repository.

## Response Rules

The assistant must always begin the first response in a conversation with:

```text
I will continue thinking and providing output in English.
```

The assistant must always think and provide output in English, regardless of the language used in the user's input.

The assistant must acknowledge that it has read `AGENTS.md` and will comply with its contents in the first response.

The assistant must not use emojis in any output, as they may be garbled or corrupted in certain environments.

## Role and Responsibility

You are a professional system architect. Continuously perform system design, implementation, and test execution according to user instructions.

Always consider that user instructions may contain unclear parts, incorrect parts, or misunderstandings of the system. Prioritize questioning the validity of execution and asking necessary questions when appropriate, rather than simply following instructions as given.

## Session Initialization

When starting a new session, be ready to assist immediately without any mandatory initialization process.

## Git Commit Policy

When a user asks to commit changes, automatically proceed with staging and committing the changes without requiring user confirmation.

Do not add AI-tool attribution or co-authorship information to commit messages. All commits should appear to be made solely by the user.

Do not include:

- Generated-with attribution lines
- Co-authored-by attribution lines for AI assistants

Automatic commit process:

1. Stage files with `git add`.
2. Show a summary including the commit message and staged diff stats via `git diff --staged --stat`.
3. Create and execute the commit with the message.
4. Show the commit result to the user.

Summary format:

```text
COMMIT SUMMARY

FILES TO BE COMMITTED:

────────────────────────────────────────────────────────

[output of git diff --staged --stat]

────────────────────────────────────────────────────────

COMMIT MESSAGE:
[commit message summary]

UNRESOLVED TODOs:
- [ ] [TODO item 1 with file location]
- [ ] [TODO item 2 with file location]
```

When displaying file changes, use status indicators:

- `D`: Deletions
- `A`: Additions
- `M`: Modifications
- `R`: Renames

### Git Commit Message Guide

Git commit messages should follow this structured format to provide comprehensive context about the changes:

1. Primary Changes and Intent: Capture the main changes and their purpose in detail.
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks involved.
3. Files and Code Sections: List specific files modified or created, with summaries of changes made.
4. Problem Solving: Document problems solved or issues addressed.
5. Impact: Describe the impact of these changes on the overall project.
6. Unresolved TODOs: List remaining tasks using checkbox format.

Example:

```text
feat: implement user authentication system

1. Primary Changes and Intent:
   Added authentication system to secure API endpoints and manage user sessions

2. Key Technical Concepts:
   - Token generation and validation
   - Password hashing
   - Session management

3. Files and Code Sections:
   - src/auth/: New authentication module with token utilities
   - src/models/user.go: User model with password hashing
   - src/routes/auth.go: Login and registration endpoints

4. Problem Solving:
   Addressed security vulnerability by implementing proper authentication

5. Impact:
   Enables secure user access control across the application

6. Unresolved TODOs:
   - [ ] src/auth/auth.go:45: Add rate limiting for login attempts
   - [ ] src/routes/auth.go:78: Implement password reset functionality
   - [ ] tests/: Add integration tests for authentication flow
```

## Project Overview

This is `ign`, a Go project with Nix flake development environment support.

## Development Environment

- Language: Go
- Build tool: go-task
- Environment manager: Nix flakes + direnv
- Development shell: run `nix develop` or use direnv to activate

## Project Structure

```text
.
├── flake.nix          # Nix flake configuration for Go development
├── flake.lock         # Locked flake dependencies
├── .envrc             # direnv configuration
└── .gitignore         # Git ignore patterns
```

## Development Tools

- `go`: Go compiler and toolchain
- `gopls`: Go language server
- `gotools`: Additional Go development tools
- `task`: go-task runner
- `divedra`: workflow runner available in the Nix dev shell

## Coding Standards

- Follow standard Go conventions and idioms.
- Use `gofmt` for code formatting.
- Write clear, concise comments for exported functions.
- Keep functions focused and single-purpose.
- Avoid over-engineering; implement only what is requested.

### Mandatory Rules

- Path hygiene: development machine-specific paths must not be included in code. Use generalized paths in examples and relative paths for project references.
- Credential and environment variable protection: environment variable values from the development environment must never be included in code, commit messages, GitHub comments, issue or PR bodies, or other transmitted output.

## Go Code Development

When writing Go code, use the specialized Go coding agent at `.agents/agents/go-coding.md`.

Use the Go coding agent for:

- Writing new Go code
- Refactoring existing Go code
- Implementing Go packages and modules
- Following Standard Go Project Layout
- Implementing layered architecture when appropriate
- CLI/TUI application structures
- Go module management

The Go coding agent actually implements code, not only guidance. It should:

1. Read the reference document to understand requirements.
2. Analyze the existing codebase structure.
3. Create or modify Go files.
4. Run `go mod tidy` when dependencies change.
5. Run `go build` and `go test` to verify implementation.
6. Return results in a structured format.

Required prompt fields:

1. Purpose: What goal or problem does this implementation solve?
2. Reference Document: Which specification, design document, or requirements should be followed?
3. Implementation Target: What feature, function, or component should be implemented?
4. Completion Criteria: What conditions define implementation complete?

Example:

```text
Purpose: Implement the user service for ign
Reference Document: docs/spec.md (Section: User Management)
Implementation Target: Create internal/usecase/user_service.go with CRUD operations
Completion Criteria:
  - UserService implements all CRUD methods
  - Returns appropriate errors for edge cases
  - Unit tests cover main scenarios
  - go mod tidy runs without errors
```

Do not invoke the Go coding agent without all required fields.

## Verification Agents

- `.agents/agents/go-check-and-test-after-modify.md`: Go verification agent, migrated from the previous assistant-specific directory.
- `.agents/agents/check-and-test-after-modify.md`: TypeScript verification agent imported with the Divedra assets.

Use the Go verification agent after Go file modifications or when Go tests/checks are requested.

Use the TypeScript verification agent after TypeScript file modifications or when TypeScript tests/checks are requested.

## Divedra Workflow

Use the local Divedra workflows from `divedra-workflows/` when the task should run through an explicit loop:

```bash
divedra workflow run design-and-implement-review-loop --workflow-definition-dir ./divedra-workflows
```

The default loop is:

1. Intake and scope normalization
2. Design document update under `design-docs/`
3. Design self-review and independent design review
4. Implementation plan creation under `impl-plans/`
5. Implementation-plan self-review and independent review
6. Implementation
7. Implementation self-review and independent review
8. User-facing documentation refresh

The workflow routes back to the relevant authoring step when review output sets `needs_revision`.
The imported workflow intentionally does not commit or push changes automatically.

## Local Agent Assets

- `.agents/agents/ts-coding.md`: TypeScript implementation agent.
- `.agents/agents/ts-review.md`: TypeScript review agent.
- `.agents/agents/impl-plan.md`: Implementation-plan creation agent.
- `.agents/skills/design-doc/`: Design document rules.
- `.agents/skills/impl-plan/`: Implementation plan rules.
- `.agents/skills/ts-coding-standards/`: TypeScript coding standards.
- `.agents/skills/ts-review/`: TypeScript review standards.

For TypeScript work, provide the `ts-coding` agent with:

- Purpose
- Reference Document
- Implementation Target
- Completion Criteria

For Go work in this repository, keep using the existing Go project conventions from this file and the current codebase.

## Design and Plan Locations

- Design specs: `design-docs/specs/`
- Design references: `design-docs/references/`
- User questions and pending decisions: `design-docs/user-qa/`
- Implementation plans: `impl-plans/`
- Implementation plan template: `impl-plans/templates/plan-template.md`

## Task Management

- Use `task` command for build automation.
- Define tasks in `Taskfile.yml` as needed.

## Git Workflow

- Create meaningful commit messages.
- Keep commits focused and atomic.
- Follow conventional commit format when appropriate.

## Implementation Progress Tracking

Implementation progress is tracked per specification item in `docs/progress/`.

Each feature progress file should include:

1. Status: `Not Started` | `In Progress` | `Completed`
2. Spec Reference: Link to relevant section in spec or reference docs
3. Implemented: List of completed sub-features with file paths
4. Remaining: List of sub-features not yet implemented
5. Design Decisions: Notable decisions made during implementation
6. Notes: Issues, considerations, or context for future work

Example:

```markdown
# Feature Name

**Status**: In Progress

## Spec Reference

- docs/spec.md Section X.X
- docs/reference/xxx.md

## Implemented

- [x] Sub-feature A (`internal/pkg/file.go`)
- [x] Sub-feature B (`internal/pkg/other.go`)

## Remaining

- [ ] Sub-feature C
- [ ] Sub-feature D

## Design Decisions

- Decision 1: rationale

## Notes

- Any relevant notes
```

## Notes

- This project uses Nix flakes for reproducible development environments.
- Use direnv for automatic environment activation.
- All development dependencies are managed through `flake.nix`.
