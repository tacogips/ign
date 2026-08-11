# Recover Or Diagnose Issue-45 Partial Directory-To-Symlink State

Design for resolving
[`tacogips/ign#47`](https://github.com/tacogips/ign/issues/47): projects that
were partially updated by the issue-45 bug may have a stale template-owned
directory on disk while `.ign/ign-files.json` no longer contains the directory's
descendant entries.

## Overview

Issue #45 introduced transactional replacement of a managed directory with a
template symlink, for example `.claude/` becoming `.claude -> .agents`. The
current safety boundary requires every terminal entry in the existing directory
to be present in `.ign/ign-files.json` before the directory can be removed.

That boundary is still correct, but issue #47 exposes a partial-state variant:
v0.1.20 could rewrite the manifest after failing the replacement, so v0.1.21
can no longer prove manifest ownership. In `--overwrite-all` or `--force`, the
transition is preserved and the generator emits the generic message:

```text
# SKIP: .claude (file exists, use --overwrite or --force to overwrite)
```

This is misleading because overwrite mode is already active and the real
problem is lost ownership metadata.

## Goals

- Preserve the safety guarantee: `ign update` must never silently delete a
  directory tree it cannot prove is template-owned or content-equivalent to the
  new template state.
- Recover issue-45 partial state automatically only when ownership is provable
  by content equivalence.
- Emit a specific recovery diagnostic when automatic proof fails.
- Keep the existing symlink transition transaction and journal recovery path as
  the only mutation mechanism for replacing the directory.
- Cover dry-run, confirmation preview, and real update with the same
  classification result.

## Non-Goals

- Do not add a broad recursive directory deletion feature.
- Do not make `--force` bypass manifest ownership, containment, symlink, or
  equivalence checks.
- Do not follow symlinks while inspecting the stale directory or the replacement
  target tree.
- Do not alter unrelated generator overwrite behavior for regular files or
  ordinary symlink replacements.

## Technical Design

### Partial-State Detection

Extend `classifyUpdateSymlinkTransitions` in
`internal/app/update_symlink_transition.go` with a third preserved reason for
directory-to-symlink candidates:

- existing path is a real directory;
- current template preview defines the same path as a symlink;
- overwrite mode is `--overwrite`, `--overwrite-all`, or `--force`;
- `ownedDirectoryTree` fails because one or more terminal entries are absent
  from `.ign/ign-files.json`.

The classifier should collect diagnostic evidence while it inspects the tree:

- untracked terminal paths under the stale directory;
- empty or unrelated directory subtrees that cannot be proven owned;
- protected paths in selective overwrite mode;
- unreadable or unsafe paths.

The issue-47 recovery path applies only when the blocker is lost manifest
ownership. Protected, unsafe, unreadable, or symlink-ancestor cases remain
preserved diagnostics without automatic replacement.

### Content-Equivalence Ownership Proof

When manifest ownership fails because descendants are untracked, attempt a
second proof: the existing directory tree must be byte-equivalent to the
rendered current-template tree that the replacement symlink would expose after
this update.

For `.claude -> .agents`, compare:

- stale existing root: `<output-dir>/.claude/`;
- expected target tree: the current template entries below `.agents/` after
  filename and content variable rendering with the same variables that will be
  passed to generation.

Do not compare against the pre-update on-disk `<output-dir>/.agents/` tree. It
may be stale, absent, locally modified, or not yet generated, and therefore
cannot prove that deleting `.claude/` loses no user data.

Equivalence rules:

- compare normalized relative paths below each root;
- require the same set of terminal entries;
- regular files must match the rendered bytes that `generator.Generate` would
  write for the current template file;
- binary template files compare their raw bytes, while text template files
  compare bytes after the normal variable processor has run;
- symlink entries must match the template symlink target after filename
  processing and must not be followed;
- directories must have matching structure and at least one terminal entry;
- template entries outside the symlink target subtree are irrelevant to this
  equivalence proof;
- any unreadable node, unsupported file type, path escape, or symlinked
  ancestor rejects equivalence;
- ignore `.ign/ign-files.json` for this second proof, because the issue state
  is defined by corrupted or advanced manifest ownership.

The expected target tree should be built from `model.Template.Files` and the
same `parser.Variables`/file processor inputs used by generation. This keeps
classification independent of current output state while preserving consistency
with generated bytes.

If equivalence passes, classify the transition as
`generator.SymlinkTransitionEligible` with a new recovery provenance marker such
as `RecoveredByContentEquivalence`. The retired managed paths should include
prior manifest paths at or below the transition root when present; missing
descendants do not need synthetic manifest entries because the transaction
removes the stale directory atomically before creating the symlink.

### Diagnostic When Recovery Is Not Provable

If equivalence cannot prove ownership, return a clear update diagnostic instead
of allowing the generic generator skip line to stand alone.

Diagnostic requirements:

- name the transition root and symlink target, for example
  `.claude -> .agents`;
- state that the project appears to be in issue-45 partial state when lost
  manifest ownership is detected;
- list representative blocking paths, capped to avoid noisy output;
- state that `--overwrite-all` and `--force` cannot remove unproven content;
- provide exact manual recovery steps:

```text
Remove or move the stale directory, then rerun update:
  mv .claude .claude.backup
  ign update --overwrite-all --yes
After verifying the generated .claude symlink and backup contents, delete the backup.
```

Dry-run must report the same diagnostic text and must not mutate the project.

The diagnostic is reported, not fatal. An unprovable transition preserves the
directory and all of its contents, and every other template change still
applies: aborting the whole update would leave a project unable to take any
template change merely because one directory holds a file ign cannot attribute,
which is a worse outcome than the partial state being repaired. The command
exits non-zero and lists each preserved path so an unresolved transition is
never mistaken for a completed one. Both dry-run and real update behave
identically here, so a preview always matches what an apply would do.
so the command does not also emit the misleading generic `# SKIP` for the same
path.

### Transaction And Rollback

Recovered transitions must flow through the existing mutation path:

- `prepareSymlinkTransitionTransactions`;
- journal persistence under `.ign/`;
- rollback snapshot capture;
- directory removal without following nested symlinks;
- replacement symlink creation;
- committed journal cleanup.

No new ad hoc removal path is permitted. If generation, manifest persistence, or
transaction commit fails, rollback restores the previous directory and tracking
artifacts using the existing transaction implementation.

### User-Facing Documentation

Update the `ign update` help or README only if the implementation introduces
documented recovery wording. The help text should not imply that `--force`
deletes untracked directories; it should say that directory-to-symlink recovery
requires manifest ownership or content equivalence.

## Implementation Targets

- `internal/app/update_symlink_transition.go`: classification, blocker capture,
  rendered-template equivalence proof, execution-plan fingerprint coverage.
- `internal/app/update.go`: collect unresolved transition diagnostics into the
  update result and carry recovered transitions into the existing transaction
  preparation.
- `internal/template/generator/generator.go`: avoid the generic skip message for
  transition paths that app classification already rejected with a specific
  diagnostic.
- `internal/cli/update.go`: render the diagnostic clearly in dry-run and real
  update output.
- `design-docs/specs/selective-overwrite.md`: optional cross-reference only if
  a later scoped revision is requested.

## Regression Tests

Required focused cases:

- issue-45 partial state with stale `.claude/` byte-identical to generated
  `.agents/` is recovered to `.claude -> .agents` under `--overwrite-all`;
- the same automatic recovery works under `--force`;
- dry-run reports the recovered transition without mutating files or metadata;
- stale `.claude/` containing a divergent untracked file fails with the
  issue-45 partial-state diagnostic and preserves all content;
- dry-run with an unrecoverable partial-state transition exits through the same
  validation-style diagnostic path without mutating files or metadata;
- unreadable, unsafe, or symlink-ancestor candidates do not use equivalence
  recovery;
- rollback after a recovered transition failure restores the stale directory and
  prior `.ign` artifacts;
- output does not include the misleading
  `file exists, use --overwrite or --force to overwrite` line for a classified
  unrecoverable partial-state transition.

Suggested commands:

```bash
go test ./internal/app -run 'SymlinkTransition|PartialState|Issue45'
go test ./internal/template/generator -run 'Symlink'
go test ./...
go build ./...
gofmt -w internal/app internal/cli internal/template/generator
```

## Decisions

- Content equivalence is sufficient ownership proof only for this narrow
  directory-to-symlink transition case.
- Equivalence is computed against the rendered current-template target tree,
  not against the existing on-disk symlink target directory.
- `--force` and `--overwrite-all` do not bypass safety checks.
- Unrecoverable partial state is a command diagnostic, not a generic generation
  skip.
- Dry-run returns the same validation-style diagnostic for unrecoverable partial
  state as real update, while preserving the project unchanged.
- Recovered replacements must use the existing transaction and journal system.

## Open Questions

- None for the safe baseline. A future explicit recovery flag can be considered
  if users need to override divergent local contents, but it is outside this
  issue-resolution scope.

## Risks

- Equivalence comparison must use `Lstat` and never follow symlinks, otherwise
  recovery could inspect or affect data outside the output directory.
- The app and generator output paths must avoid duplicate or contradictory
  diagnostics for the same transition.
- Fingerprint validation must include enough source and target state to reject
  changes between preview and confirmation.
