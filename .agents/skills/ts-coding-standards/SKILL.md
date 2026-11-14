---
name: ts-coding-standards
description: Use when writing, reviewing, or refactoring TypeScript code. Provides type safety patterns, error handling, project layout, and async programming guidelines.
allowed-tools: Read, Grep, Glob
---

# TypeScript Coding Standards

This skill provides modern TypeScript coding guidelines and best practices for this project.

## When to Apply

Apply these standards when:

- Writing new TypeScript code
- Reviewing or refactoring existing TypeScript code
- Designing module APIs and interfaces
- Implementing error handling strategies

## Core Principles

1. **Type Safety Over Convenience** - Never sacrifice type safety for shorter code
2. **Explicit Over Implicit** - Make types and intentions clear
3. **Simple Over Clever** - Prefer readable code over clever abstractions
4. **Fail Fast** - Catch errors at compile time, not runtime

## Source file size

- **Target limit**: TypeScript source files under `src/` should stay below **1000 lines**. If a touched file is already over that size, avoid making it substantially larger and prefer a focused split when the task scope allows it.
- **How to split**: Prefer clear module boundaries (feature, layer, or cohesive helpers) and choose meaningful file names that describe each split file's responsibility. When many imports point at one path, use a **thin facade** file that re-exports from `*-helpers.ts`, `*-types.ts`, or a small subdirectory so callers keep stable import paths.
- **Agents**: When editing or reviewing code, if a touched file is **1000+ lines**, call this out and either split it in the same change set or record why the split is a separate follow-up.
- **Automation**: Non-test sources under `src/` are checked by **Biome** (`noExcessiveLinesPerFile`, **1000** lines) as an error during local linting. `*.test.ts` files are exempt in Biome, but the target limit still applies during review.

## After coding (agents)

After modifying TypeScript under `src/` or `vitest.config.ts`:

1. Run **`biome check . --diagnostic-level=warn`** (or **`bun run lint:biome`**, which sets that threshold). Use Biome from `nix develop` / flake devShell, or `bunx biome ...` when the platform binary works.
2. Run **`bun run typecheck`**.
3. Run **`bun run test`** (or the subset relevant to the change).
4. Run Biome formatting when you touch formatted paths: **`bun run format`** or `biome format --write` on the files you edited.

If Biome reports errors or typecheck fails, fix them before declaring the task complete. Biome warnings should be fixed when they are in touched code or otherwise recorded as follow-up migration work.

## Quick Reference

### Must-Use Patterns

| Pattern                   | Use Case                                                                                |
| ------------------------- | --------------------------------------------------------------------------------------- |
| Discriminated Unions      | State machines, API responses, Result types                                             |
| Enums                     | Fixed value domains reused by validation sets, config kinds, providers, modes, statuses |
| Branded Types             | IDs, emails, validated strings                                                          |
| `readonly`                | Data that should not mutate                                                             |
| `unknown` in catch        | Safe error handling                                                                     |
| Explicit undefined checks | Array/object indexed access                                                             |

### Must-Avoid Anti-Patterns

| Anti-Pattern                           | Alternative                   |
| -------------------------------------- | ----------------------------- |
| `any` type                             | `unknown` with type guards    |
| `string` for fixed known values        | string enum                   |
| Throwing exceptions for control flow   | Result type pattern           |
| Optional chaining without null check   | Explicit narrowing            |
| Deep folder nesting (>3 levels)        | Flat, feature-based structure |
| Implicit `undefined` in optional props | Explicit `T \| undefined`     |

## Detailed Guidelines

For comprehensive guidance, see:

- [Error Handling Patterns](./error-handling.md) - Result types, discriminated unions, neverthrow
- [Type Safety Best Practices](./type-safety.md) - Branded types, strict config, type guards
- [Project Layout Conventions](./project-layout.md) - Directory structure, file naming, imports
- [Async Programming Patterns](./async-patterns.md) - Promise handling, concurrent execution
- [Security Guidelines](./security.md) - Credential protection, path sanitization, sensitive data handling

## tsconfig.json Strict Mode

This project uses maximum TypeScript strictness. Ensure your code compiles with:

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noPropertyAccessFromIndexSignature": true,
    "noFallthroughCasesInSwitch": true,
    "noImplicitOverride": true,
    "noImplicitReturns": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true
  }
}
```

## References

- [TypeScript Advanced Patterns 2025](https://dev.to/frontendtoolstech/typescript-advanced-patterns-writing-cleaner-safer-code-in-2025-4gbn)
- [The Strictest TypeScript Config](https://whatislove.dev/articles/the-strictest-typescript-config/)
- [neverthrow - Type-Safe Errors](https://github.com/supermacro/neverthrow)
