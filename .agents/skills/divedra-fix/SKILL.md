---
name: divedra-fix
description: Use when a divedra workflow, CLI command, GraphQL control plane, event listener, or submodule integration appears to fail because of an upstream divedra bug. Requires reproducing with the local submodule, filing an issue in tacogips/divedra when the bug is real, fixing the submodule, validating it, and committing the divedra change.
---

# Divedra Fix Workflow

Use this skill when work in this repository indicates that `divedra` itself may be wrong, incomplete, or misaligned with its documented behavior.

## Source Boundary

- Treat `./divedra` as the authoritative `divedra` checkout for this repository.
- Do not use `github:tacogips/divedra` or a globally installed `divedra` to diagnose or fix behavior unless comparing versions is necessary.
- Prefer `task divedra -- <args>` from the parent repository, or `bun run src/main.ts <args>` from `./divedra` when editing the submodule directly.

## Triage

1. Reproduce the failure with the local submodule checkout.
2. Determine whether the fault is in this repository's integration or in `./divedra`.
3. If the fault is integration-only, fix this repository and do not file an upstream issue.
4. If the fault is in `./divedra`, collect the command, input files, expected behavior, actual behavior, and relevant logs.

## Upstream Issue

When the fault is in `divedra`, create an issue in `tacogips/divedra` before or alongside the fix:

```bash
gh issue create --repo tacogips/divedra --title "<concise bug title>" --body-file <issue-body-file>
```

The issue body should include:

- Reproduction steps using `./divedra` or `task divedra --`.
- Expected behavior and actual behavior.
- Environment details that affect execution, such as Bun, Nix, OS, workflow root, and relevant command flags.
- Links or paths to affected workflow fixtures when they are safe to reference.

If `gh` is unavailable or authentication fails, write the issue body to a temporary markdown file, report the blocker, and continue only if the requested fix can still be validated locally.

## Submodule Fix

1. Work inside `./divedra`.
2. Follow `./divedra/AGENTS.md` and its implementation-plan requirements for non-trivial TypeScript changes.
3. Keep fixes minimal and covered by targeted tests.
4. Run the smallest meaningful validation first, then broader checks when practical:
   - `bun test <targeted-test>`
   - `bun run typecheck`
   - `task ci`
5. Commit the submodule fix in `./divedra` with no automated-assistant attribution or co-authorship trailers.
6. Return to the parent repository, stage the updated submodule pointer, and commit the parent change when the parent repository should track the fixed submodule revision.

## Reporting

In the final response, include the issue URL or issue creation blocker, the submodule commit hash, the parent repository files or submodule pointer updated, and validation commands that passed or could not be run.
