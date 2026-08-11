# Issue 45 Partial-State Symlink Recovery

**Status**: Completed

## Spec Reference

- `design-docs/specs/issue45-partial-state-symlink-recovery.md`
- `impl-plans/issue45-partial-state-symlink-recovery.md`
- GitHub issue `tacogips/ign#47`

## Implemented

- [x] Directory-to-symlink classification records lost manifest ownership blockers (`internal/app/update_symlink_transition.go`).
- [x] Content-equivalence recovery compares the stale directory with the rendered current-template symlink target tree (`internal/app/update_symlink_transition.go`).
- [x] Recovered transitions carry provenance and use the existing symlink transition transaction and rollback path (`internal/app/update_symlink_transition.go`, `internal/app/update.go`).
- [x] Unprovable partial states preserve the directory, report a specific diagnostic with manual recovery steps, and exit non-zero without blocking unrelated template changes (`internal/app/update_symlink_transition.go`, `internal/cli/update.go`).
- [x] Generator exposes shared content rendering and suppresses generic skip output for app-classified preserved transitions (`internal/template/generator/generator.go`).
- [x] Regression tests cover recovery, force-equivalent overwrite mode, dry-run immutability, divergent diagnostics, and generic skip suppression (`internal/app/update_issue45_partial_state_test.go`, `internal/app/update_symlink_transition_test.go`).

## Remaining

- None.

## Design Decisions

- Automatic recovery is allowed only when rendered-template content equivalence proves the stale directory can be removed without data loss.
- `--overwrite-all` and `--force` do not bypass ownership, path safety, or equivalence checks.
- Recovery uses the existing journaled symlink transition transaction rather than a separate deletion path.

## Notes

- README now documents safe directory-to-symlink recovery behavior for `ign update`.
