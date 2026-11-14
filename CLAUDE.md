# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rule of the Responses

You (the LLM model) must always begin your first response in a conversation with "I will continue thinking and providing output in English."

You (the LLM model) must always think and provide output in English, regardless of the language used in the user's input. Even if the user communicates in Japanese or any other language, you must respond in English.

You (the LLM model) must acknowledge that you have read CLAUDE.md and will comply with its contents in your first response.

## Language Instructions

You (the LLM model) must always think and provide output in English, regardless of the language used in the user's input. Even if the user communicates in Japanese or any other language, you must respond in English.

## Session Initialization Requirements

When starting a new session, you (the LLM model) should be ready to assist the user with their requests immediately without any mandatory initialization process.

## Git Commit Policy

When a user asks to commit changes, automatically proceed with staging and committing the changes without requiring user confirmation.

**IMPORTANT**: Do NOT add any Claude Code attribution or co-authorship information to commit messages. All commits should appear to be made solely by the user. Specifically:

- Do NOT include `🤖 Generated with [Claude Code](https://claude.ai/code)`
- Do NOT include `Co-Authored-By: Claude <noreply@anthropic.com>`
- The commit should appear as if the user made it directly

**Automatic Commit Process**: When the user requests a commit, automatically:

a) Stage the files with `git add`
b) Show a summary that includes:

- The commit message
- Files to be committed with diff stats (using `git diff --staged --stat`)
  c) Create and execute the commit with the message
  d) Show the commit result to the user

Summary format example:

```
📝 COMMIT SUMMARY

📁 FILES TO BE COMMITTED:

────────────────────────────────────────────────────────

[output of git diff --staged --stat]

────────────────────────────────────────────────────────

📋 COMMIT MESSAGE:
[commit message summary]

📌 UNRESOLVED TODOs:
- [ ] [TODO item 1 with file location]
- [ ] [TODO item 2 with file location]
```

Note: When displaying file changes, use color coding where possible:

- 🔴 Red for deletions (D status)
- 🟢 Green for additions (A status) and modifications (M status)
- 🟡 Yellow for renames (R status)

### Git Commit Message Guide

Git commit messages should follow this structured format to provide comprehensive context about the changes:

Create a detailed summary of the changes made, paying close attention to the specific modifications and their impact on the codebase.
This summary should be thorough in capturing technical details, code patterns, and architectural decisions.

Before creating your final commit message, analyze your changes and ensure you've covered all necessary points:

1. Identify all modified files and the nature of changes made
2. Document the purpose and motivation behind the changes
3. Note any architectural decisions or technical concepts involved
4. Include specific implementation details where relevant

Your commit message should include the following sections:

1. Primary Changes and Intent: Capture the main changes and their purpose in detail
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks involved
3. Files and Code Sections: List specific files modified or created, with summaries of changes made
4. Problem Solving: Document any problems solved or issues addressed
5. Impact: Describe the impact of these changes on the overall project
6. Unresolved TODOs: If there are any remaining tasks, issues, or incomplete work, list them using TODO list format with checkboxes `- [ ]`

Example commit message format:

```
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

This is a Golang project with Nix flake development environment support.

## Development Environment
- **Language**: Go
- **Build Tool**: go-task (Task runner)
- **Environment Manager**: Nix flakes + direnv
- **Development Shell**: Run `nix develop` or use direnv to activate

## Project Structure
```
.
├── flake.nix          # Nix flake configuration for Go development
├── flake.lock         # Locked flake dependencies
├── .envrc             # direnv configuration
└── .gitignore         # Git ignore patterns
```

## Development Tools Available
- `go` - Go compiler and toolchain
- `gopls` - Go language server (LSP)
- `gotools` - Additional Go development tools
- `task` - Task runner (go-task)

## Coding Standards
- Follow standard Go conventions and idioms
- Use `gofmt` for code formatting
- Write clear, concise comments for exported functions
- Keep functions focused and single-purpose
- Avoid over-engineering - implement only what's requested

## Go Code Development
**IMPORTANT**: When writing Go code, you (the LLM model) MUST use the specialized go-coding sub agent located at `/g/gits/tacogips/ign/.claude/agents/go-coding.md`.

Use the Task tool with the go-coding agent for:
- Writing new Go code
- Refactoring existing Go code
- Implementing Go packages and modules
- Following Standard Go Project Layout
- Implementing layered architecture (Clean Architecture, Hexagonal Architecture, etc.)

The go-coding agent has comprehensive knowledge of:
- Standard Go Project Layout conventions
- Go best practices and idioms
- Layered architecture patterns
- CLI/TUI application structures
- Package management with go modules

## Task Management
- Use `task` command for build automation
- Define tasks in `Taskfile.yml` (to be created as needed)

## Git Workflow
- Create meaningful commit messages
- Keep commits focused and atomic
- Follow conventional commit format when appropriate

## Notes
- This project uses Nix flakes for reproducible development environments
- Use direnv for automatic environment activation
- All development dependencies are managed through flake.nix
