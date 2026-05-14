---
name: ts-review
description: Use when reviewing TypeScript changes for alignment with ts-coding-standards, Biome lint rules, and repository conventions before merge or handoff.
allowed-tools: Read, Grep, Glob
---

# TypeScript review checklist

Use this skill after substantive TypeScript edits (or before approving a PR) when you need a structured pass beyond automated checks.

## Prerequisites

1. Read `.agents/skills/ts-coding-standards/SKILL.md` and scan the linked topic files that apply to the change (error handling, types, layout, async, security).
2. Ensure automated gates were run (or run them now): `biome check . --diagnostic-level=warn` (or `bun run lint:biome`), `bun run typecheck`, `bun run test`, and Prettier on touched paths where formatting applies.

## Compliance with ts-coding workflow

Confirm the implementation matches what the **ts-coding** agent requires:

- **Standards read**: Changes respect strict typing (`no any`, indexed access, optional/`exactOptionalPropertyTypes`), Result/error patterns where appropriate, and project layout conventions.
- **Execution order**: Lint (Biome) before or alongside typecheck; formatting applied where the repo uses Prettier.
- **Security**: No machine-specific paths in examples, no env secrets or private URLs in output; follow `.agents/skills/ts-coding-standards/security.md`.

## Static checks

- **Biome**: Lint runs with **`--diagnostic-level=warn`** (warnings are visible; infos are not). There must be **no errors** in that output for files under `src/` and `vitest.config.ts`. Treat **warnings** as normal review/fix targets unless the team agrees to defer them. Do not recommend weakening rules to hide violations unless the user explicitly approves a config change.
- **File size**: Non-test sources under `src/` should stay **at or below 1000 lines** (Biome enforces this). Test files may temporarily exceed that; still flag unreasonably large `*.test.ts` files as follow-up work.

## Code review focus

- Public APIs: exports are typed; edge cases and failures are handled deliberately.
- Imports and module boundaries match `.agents/skills/ts-coding-standards/project-layout.md`.
- Tests: behavior-changing code has or updates coverage; flaky patterns are avoided.

## Output format

Summarize **Pass** or **Issues** with bullet points referencing file paths and concrete fixes. If Biome or `tsc` failures exist, quote the diagnostic and suggest the minimal correction.
