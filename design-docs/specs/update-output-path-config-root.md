# Fix `ign update <output-path>` Configuration Root Handling

Design for issue [tacogips/ign#46](https://github.com/tacogips/ign/issues/46).

## Overview

`ign update [output-path]` documents that the optional argument selects the
generated project to update. The current app-layer preparation path still
constructs `.ign/ign.json` and `.ign/ign-var.json` from the bare
`model.IgnConfigDir`, so those files are loaded relative to the caller current
working directory instead of the requested output project.

The fix is to make the update workflow treat `opts.OutputDir` as the project
root for every `.ign` tracking artifact and every derived update path.

## Required Behavior

- `ign update /path/to/project --dry-run --overwrite --yes --no-color` must load
  `/path/to/project/.ign/ign.json`,
  `/path/to/project/.ign/ign-var.json`, and
  `/path/to/project/.ign/ign-files.json` even when the command is launched from
  another directory.
- `ign update ./my-project` must preserve the documented command behavior in
  `README.md` and command help.
- A missing target project still fails with the existing prior-checkout
  validation, but the diagnostic must refer to the requested output project, not
  the caller current working directory.
- Dry-run, confirmation preview, and real update must use the same output-rooted
  tracking paths so preview and mutation classify the same files.
- Existing safety behavior for managed directory-to-symlink transitions,
  rollback snapshots, stale managed-file cleanup, and ref retargeting must not
  change.

## Path Rooting Contract

`PrepareUpdate` must normalize and validate `opts.OutputDir` before constructing
tracking paths. The root may remain relative for user-facing reporting, but app
internals that touch `.ign` files must use paths joined beneath the requested
output directory.

Required update tracking paths:

- `configDir`: `filepath.Join(opts.OutputDir, model.IgnConfigDir)`
- `ignConfigPath`: `filepath.Join(configDir, model.IgnProjectConfigFile)`
- `ignVarPath`: `filepath.Join(configDir, model.IgnVarFile)`
- `manifestPath`: derived from the output-rooted `ignConfigPath`
- symlink transition journal path: derived from `opts.OutputDir`
- rollback snapshots for `ign.json`, `ign-var.json`, and `ign-files.json`: based
  on the output-rooted artifact paths
- generation variables with path defaults: resolved using the requested
  `opts.OutputDir`, while reading any persisted variable files from the
  output-rooted `.ign` directory

`CompleteUpdate` must not reintroduce a bare `.ign` root. In particular,
`prepareVariablesForGeneration` must receive the output-rooted config directory
instead of `model.IgnConfigDir`.

## Implementation Notes

- Add a small helper for update project paths if it reduces duplicated joins
  between `PrepareUpdate` and `CompleteUpdate`. The helper should be local to
  the app package and avoid changing public CLI behavior.
- Keep `PrepareUpdateResult.IgnConfigPath` and `PrepareUpdateResult.IgnVarPath`
  as the authoritative tracking artifact paths for later phases.
- Keep `manifestPathFromConfigPath(prep.IgnConfigPath)` as the manifest source
  so manifest, cleanup, transition classification, and rollback all follow the
  same output-rooted config path.
- Do not change checkout output path semantics. `ign checkout` creates the
  target project and writes `.ign` beneath that target already.
- `ign vars` currently has no `[output-path]` argument, so its current-directory
  project-root behavior is out of scope for this issue. `ign rewind` and
  `ign switch` should be checked for the same bare `.ign` pattern during
  implementation, but only update code should change unless a shared defect is
  directly proven and covered.

## Regression Coverage

Required tests:

- App-layer update preparation loads `.ign/ign.json` and `.ign/ign-var.json`
  from `opts.OutputDir` while the process working directory is a different
  temporary directory.
- App-layer update dry-run uses the output-rooted manifest path when classifying
  stale cleanup and symlink transitions.
- `CompleteUpdate` resolves variable files and rollback snapshots under
  `opts.OutputDir`, including the config-only ref-retarget path.
- CLI-level `ign update <output-path>` passes the requested path through prepare
  and complete calls without rewriting it to the caller current directory.

Focused verification commands:

```bash
go test ./internal/app -run 'TestPrepareUpdate|TestCompleteUpdate'
go test ./internal/cli -run 'TestRunUpdate'
go test ./...
go build ./...
gofmt -w internal/app internal/cli
```

## Decisions

- Accepted: `opts.OutputDir` is the configuration root for `ign update
  [output-path]`.
- Accepted: update tracking paths are joined beneath the output directory before
  any existence check or config load.
- Accepted: `PrepareUpdateResult` carries rooted artifact paths forward so later
  update phases do not recompute bare `.ign` paths.
- Accepted: this feature fanout item covers issue #46 only. Issue #47's
  partial-state symlink recovery is independent and must be designed in its own
  feature item.

## Risks

- Relative output paths can still appear in user-facing dry-run output; tests
  should assert filesystem behavior rather than over-constraining display text.
- If a later phase mixes absolute and relative path forms, execution-plan
  validation may reject a valid confirmation preview. Normalize comparison paths
  consistently.
- Config-only updates have less generator activity, so rollback coverage must
  explicitly exercise that path.
