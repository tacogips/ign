# Selective Overwrite

## Overview

`ign update` supports selective overwrite so template authors can protect
user-owned generated paths while still allowing safe template updates.

The template root may define `.ign-overwrite-ignore`. The file uses
gitignore-style patterns against generated output paths. During
`ign update --overwrite`, existing output paths matched by this file are skipped.

## Command Behavior

- `ign update --overwrite` performs selective overwrite.
- `ign update --overwrite-all` preserves the previous overwrite-all behavior.
- `ign update --force` regenerates even when the template hash is unchanged and
  uses overwrite-all semantics.
- `ign update --overwrite --yes` and `ign update --overwrite-all --yes` skip the
  overwrite confirmation prompt.
- `--yes` affects confirmation only. It does not alter overwrite mode, hash
  comparison, or dry-run behavior.
- Without `--yes`, a writing update in overwrite mode previews planned changes
  before mutation using `A` for new files, `M` for overwritten existing files,
  and `D` for removed managed files.
- In overwrite mode, update removes files recorded in `.ign/ign-files.json` when
  the current template no longer generates them. Selective overwrite preserves
  existing removed-template paths matched by `.ign-overwrite-ignore`, but prunes
  absent ignored paths from the next manifest.

## Template Metadata

- `.ign-overwrite-ignore` is recognized only at the remote template root.
- Root `.ign-overwrite-ignore` is template metadata and is not emitted into
  generated project output.
- Nested files named `.ign-overwrite-ignore` are normal template files.
- Root `.ign-overwrite-ignore` is included in template hash calculation so
  overwrite-policy changes are detected by `ign update`.
- Update reads overwrite-ignore policy from the fetched remote template, not from
  any local project copy.

## Matching Rules

Selective overwrite matching uses normalized slash-separated generated paths.

The supported gitignore-style behavior includes:

- anchored patterns such as `/config/app.yaml`
- directory patterns such as `config/`
- basename patterns such as `.env`
- glob patterns such as `*.local`
- recursive `**` patterns
- negation ordering such as `config/` followed by `!config/default.yaml`

## Ignored Path Creation Boundary

Issue [#48](https://github.com/tacogips/ign/issues/48) extends the selective
overwrite contract from "do not overwrite existing ignored paths" to "do not
materialize ignored paths at all" during `ign update --overwrite`.

The ignore decision must be applied before both creation and overwrite:

- A generated path matched by `.ign-overwrite-ignore`, or by an ignored
  directory ancestor, is protected even when the destination does not exist.
- A missing protected file or symlink remains absent after the update.
- An existing protected file, symlink, or directory descendant remains unchanged.
- A protected generated path is reported as skipped, not created or overwritten.
- Protected paths skipped because of this rule are not added to
  `.ign/ign-files.json`.
- Dry-run and confirmation previews use the same protected-path decision as the
  write path, so previewed `A` entries cannot become ignored creations during
  mutation.

This filtering belongs at the generator/update write boundary before filesystem
mutation and before manifest persistence. It must not be implemented by writing
placeholders, deleting generated files after the fact, or manually editing the
manifest to hide created files.

`--overwrite-all` and `--force` continue to bypass `.ign-overwrite-ignore`
matching for generated paths. No-overwrite mode keeps its existing behavior of
skipping existing paths without using `.ign-overwrite-ignore` as a creation
filter.

## Validation

The dry-run/confirmation preview path and final write path must share the same
template rendering, file filtering, overwrite mode, and ignore-pattern decisions
so the displayed plan matches the eventual mutation.

## Removed Managed File Cleanup

Overwrite update cleanup uses `.ign/ign-files.json` as the record of paths that
`ign` owns from previous generation runs. After rendering the current template,
`ign update` compares the prior manifest with the newly generated file set and
classifies manifest entries missing from the new render as stale managed files.

Issue [#49](https://github.com/tacogips/ign/issues/49) clarifies the selective
overwrite migration path for projects updated by older releases. A path may be
both stale and matched by the current template's `.ign-overwrite-ignore` because
a previous manifest recorded it before ignored-path creation was fixed. During
`ign update --overwrite --yes`, an absent ignored stale path must be removed from
the next `.ign/ign-files.json` without creating the file. If the ignored stale
path still exists on disk, selective overwrite preserves it and retains its prior
manifest entry rather than deleting user-owned content.

Cleanup is intentionally narrower than general overwrite behavior:

- It runs only for overwrite updates. No-overwrite update mode must not delete
  stale managed files.
- `--overwrite-all` removes stale managed files without consulting
  `.ign-overwrite-ignore`.
- selective `--overwrite` removes stale managed files only when their generated
  paths are not matched by the remote template's `.ign-overwrite-ignore`.
- Stale cleanup acts only on manifest-recorded files. User-created files outside
  `.ign/ign-files.json` are outside the cleanup set.
- Directory manifest entries are rejected instead of recursively removed.
- Missing stale paths are still removed from the manifest so future updates do
  not keep reporting already-absent files. This includes stale paths matched by
  `.ign-overwrite-ignore`, because pruning an absent entry does not mutate or
  claim ownership of user content.
- After a real update, removed paths are excluded from the saved manifest while
  paths still generated by the current template remain tracked.

Dry-run and confirmation previews must expose the same stale cleanup decisions
as the real update by reporting removed managed files with `D`. Preview output
must not mutate files, remove directories, or rewrite `.ign/ign-files.json`.

## Managed Directory-To-Symlink Transitions

Issue [#45](https://github.com/tacogips/ign/issues/45) adds one overwrite type
transition: a path generated as a directory by an older template revision may
become a symlink in the current template. A representative transition is a
non-empty `.claude/` directory becoming `.claude -> .agents`.

### Eligibility And Ownership Boundary

- The transition is considered only in `--overwrite`, `--overwrite-all`, or
  `--force` mode. Without overwrite, the existing directory is skipped.
- Selective overwrite must skip the transition when the symlink path or any
  candidate descendant is protected by `.ign-overwrite-ignore`.
- A non-empty directory may be replaced only when every non-directory entry
  below it is represented by the prior `.ign/ign-files.json` manifest. The
  transition root and directory ancestors of managed entries do not need
  separate manifest records.
- Every descendant directory must contain at least one manifest-managed
  non-directory entry. An empty directory subtree or a directory subtree
  unrelated to any managed entry lacks ownership proof and preserves the entire
  transition root.
- Ownership validation uses normalized, output-root-contained paths and must not
  follow symlinks while inspecting or removing the existing tree.
- If any non-directory entry is untracked, any directory subtree lacks ownership
  proof, or any descendant is protected, unreadable, or escapes the output root,
  preserve the directory and classify the new symlink as skipped. This is a
  safety decision, not a generation error.
- This exception does not broaden stale cleanup into general recursive directory
  deletion. It applies only when the current template defines the same path as a
  symlink and the complete existing tree passes the ownership checks above.
- `--overwrite-all` and `--force` bypass overwrite-ignore matching, but they do
  not bypass manifest ownership, containment, or no-symlink-following checks.

### Planning, Mutation, And Rollback

- Dry-run, confirmation preview, and real generation share one transition
  classification so the preview cannot authorize a different mutation.
- Classification produces either an eligible transition or a preserved
  transition root. The same result, including its transition-retired manifest
  entries, is consumed by generation, stale cleanup, rollback preparation and
  execution, and manifest persistence; those phases must not independently
  reclassify it.
- For an eligible transition, every prior manifest entry below the transition
  root is classified as transition-retired. Stale cleanup must treat those
  entries as filesystem-deletion exclusions: it must not inspect, traverse, or
  remove them after the replacement symlink exists. Their old on-disk instances
  are removed only as part of the ownership-validated directory removal before
  symlink creation.
- This exclusion prevents a retired path such as `.claude/settings.json` from
  resolving through the new `.claude -> .agents` symlink and deleting the
  current `.agents/settings.json` target. Cleanup may prune transition-retired
  entries from tracking only after the symlink transition and update commit
  succeed.
- An eligible transition is reported as one modification (`M`) at the symlink
  path rather than separate deletions for every managed descendant.
- Before mutation, rollback uses the shared eligible-transition classification
  to capture the complete existing directory tree and the three tracking
  artifacts. Preserved transitions do not enter the mutation or rollback set.
- Recursive rollback capture records each nested symlink as a symlink node and
  its link target, without traversing or copying the target tree. Restoration
  reconstructs the prior directory contents and symlink nodes without following
  their targets, leaving any data reachable only through those links untouched.
- The managed directory is removed without following nested symlinks, then the
  new symlink is created.
- If removal, symlink creation, later generation, or tracking persistence fails,
  update restores the prior directory and prior tracking artifacts where
  rollback state is available.
- A failure while executing an eligible directory-to-symlink transition is a
  fatal `CompleteUpdate` error: update rolls back and does not persist new
  `.ign` tracking artifacts. It must not be downgraded into the generator's
  non-fatal `GenerateResult.Errors` collection.
- Issue #45 does not change commit behavior for unrelated pre-existing kinds of
  `GenerateResult.Errors`; broad generation-error policy remains outside this
  narrowly scoped transition fix.

### Tracking Metadata

- After successful replacement, `.ign/ign-files.json` contains the symlink path
  once and removes transition-retired descendant entries from the replaced
  directory without performing separate stale-cleanup deletion through the new
  symlink.
- `.ign/ign.json` and `.ign/ign-var.json` advance only with the same successful
  update commit.
- For a preserved transition root, generation records the new symlink as
  skipped rather than written. Stale cleanup must exempt every prior manifest
  entry at or below that root from deletion and pruning, and manifest persistence
  retains those entries without claiming that the new symlink was created.
- A failed transition must not leave a manifest entry for an unwritten symlink
  or advance template metadata past the on-disk project state.

### Regression And Verification Contract

Regression coverage belongs primarily in `internal/app/update_test.go` because
manifest ownership, overwrite mode, rollback, and artifact persistence are
app-layer responsibilities. Focused generator coverage may remain in
`internal/template/generator/generator_test.go` for low-level symlink behavior.

Required cases:

- overwrite replaces a non-empty, fully manifest-managed directory with the
  expected symlink and emits no generation error;
- the successful manifest records the symlink and removes obsolete descendants;
- stale cleanup does not delete a current target file by resolving a retired
  descendant through the newly created symlink;
- no-overwrite mode preserves and skips the directory;
- selective overwrite preserves a protected transition;
- an untracked descendant causes a clean skip with prior metadata retained;
- an empty or unrelated directory subtree causes the same clean skip;
- dry-run reports the same modification without changing files or metadata;
- rollback of a transition tree containing a nested symlink restores the nested
  symlink itself without reading, replacing, or removing its target tree;
- an unexpected mutation failure does not persist a failed partial transition.

Required verification commands:

```bash
go test ./internal/template/generator -run 'Symlink'
go test ./internal/app -run 'TestCompleteUpdate_OverwriteSymlink'
mise run test
go build -o /dev/null ./...
go vet ./...
```

The change must not modify `internal/build/VERSION`, publish a release, or alter
unrelated pre-existing worktree changes.

## Review Follow-Up Scope

The review for `v0.1.16..HEAD`, with focus on commit `1f8bba8`, keeps the
stale managed file cleanup contract narrow. Any follow-up change should address
only a concrete defect in update overwrite cleanup or release packaging.

Review validation should cover these boundaries:

- cleanup is driven by `.ign/ign-files.json`, not by directory scans.
- cleanup compares the prior manifest against files rendered by the current
  template before pruning stale manifest entries.
- no-overwrite update mode does not delete stale managed files.
- selective overwrite keeps existing removed paths protected by the remote
  template's `.ign-overwrite-ignore`.
- selective overwrite prunes absent removed paths protected by the remote
  template's `.ign-overwrite-ignore` from the next manifest without recreating
  them.
- overwrite-all removes stale manifest files without overwrite-ignore filtering.
- dry-run and confirmation previews report the same deletion candidates as a
  real update, using `D`, without mutating files or manifest state.
- deleted-file paths are reported relative to the requested output path.
- release packaging follow-up must stay independent from update cleanup unless
  a packaging defect directly affects the release artifact or generated formula.

Later implementation review must preserve user-created files and continue to
refuse ordinary stale-cleanup directory removal. The only directory-removal
exception is the ownership-validated directory-to-symlink transition defined
above. Review must also avoid unrelated refactors and verify the Go surface with
`go test ./...`, `go build ./cmd/ign`, and `go vet ./...`.
