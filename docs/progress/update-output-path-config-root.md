# Update Output Path Config Root

**Status**: Completed

## Spec Reference

- `design-docs/specs/update-output-path-config-root.md`
- `impl-plans/update-output-path-config-root.md`
- GitHub issue `tacogips/ign#46`

## Implemented

- [x] `PrepareUpdate` loads `.ign/ign.json` and `.ign/ign-var.json` from `UpdateOptions.OutputDir` (`internal/app/update.go`).
- [x] `CompleteUpdate` resolves persisted variable files, manifest paths, rollback artifacts, cleanup, and symlink transition state from the output-rooted config path (`internal/app/update.go`).
- [x] CLI update preserves and passes the optional `output-path` to prepare, preview, and complete phases (`internal/cli/update.go`, `internal/cli/update_test.go`).
- [x] Regression tests cover update from a different caller working directory (`internal/app/update_output_root_test.go`).
- [x] Related command inspection completed for `vars`, `rewind`, and `switch`.

## Remaining

- None.

## Design Decisions

- `PrepareUpdateResult.IgnConfigPath` remains the authoritative root for derived `.ign` artifact paths.
- `ign vars` has current-directory semantics and no output-path argument.
- `ign rewind` and `ign switch` already have their own output-path contracts and are not changed by this issue.

## Notes

- README now states that `ign update [output-path]` reads and writes the requested project's `.ign/` tracking files.
