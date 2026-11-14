# Selective Overwrite

## Transaction Backup Archival And No-Replace Remediation

- [x] Revalidated transition ownership against the descriptor-relative renamed snapshot, preventing an untracked addition between classification and fingerprinting from being replaced.
- [x] Made ordinary generation rollback identity-aware and fail closed when a generated or overwritten node is concurrently substituted; unverified parent directories are retained.
- [x] Replaced ordinary rollback's check-then-remove and truncating restoration with same-filesystem quarantine, post-move identity/content validation, and no-replace restoration; private archives are retained because automatic cleanup could delete a post-validation mutation (`internal/app/checkout_rollback_nodes.go`, `internal/app/checkout_rollback_nodes_unix.go`).
- [x] Bound each successfully persisted ordinary tracking artifact to its post-write identity before a later artifact save can fail, so rollback restores the prior manifest or configuration without deleting a concurrent replacement (`internal/app/update.go`, `internal/app/checkout.go`).
- [x] Allowed ordinary generation on platforms without atomic generic rollback support; rollback primitives fail closed at rollback time and retain the private archive, while only eligible directory-to-symlink transitions are rejected before mutation (`internal/app/checkout_rollback_nodes_other.go`, `internal/app/checkout_rollback_nodes_other_test.go`).
- [x] Bound tracking-artifact rollback to the exact written bytes and mode whenever the atomic rename committed, including when parent-directory syncing fails after a successful rename, so a reported-failed but committed write is still restored and a concurrent replacement is never adopted (`internal/config/loader.go`, `internal/app/update.go`, `internal/app/checkout.go`, `internal/app/update_artifact_rollback_test.go`).
- [x] Anchored test rollback archives under test-owned temporary directories so the app test suite no longer leaks `.ign-checkout-rollback-*` archives into the package source directory (`internal/app/checkout_rollback_nodes_unix_test.go`, `internal/app/update_symlink_transition_test.go`).
- [x] Roll back ordinary created and overwritten generation results when committed-journal persistence fails.
- [x] Bound transaction artifact capture and recovery to the validated tracking-artifact parent; non-sibling `ign-files.json`, `ign.json`, and `ign-var.json` paths are rejected before mutation.
- [x] Added regressions for rollback substitution safety, complete-update persistence failure rollback, and explicit tracking-artifact roots.
- [x] Split identity-aware rollback helpers into `internal/app/checkout_rollback_nodes.go` so every touched Go file remains below the 1000-line limit.
- [x] Replaced committed recursive transition-backup cleanup with a durable, descriptor-relative `.ign` archive rename; no post-commit backup node is deleted automatically.
- [x] Made archive placement cross-directory no-replace so a concurrent `.ign` archive entry is retained with its backup and journal.
- [x] Made restoration fail closed when a concurrent destination exists instead of overwriting it.
- [x] Replaced replacement-symlink unlink with durable no-replace archival; a
  substituted node is retained in `.ign` and leaves the backup and journal
  actionable for safe recovery.
- [x] Added Darwin/Linux regressions for post-snapshot descendant creation, regular-file mutation, leaf replacement, archival, and destination recreation before restoration.
- [x] Kept durable source-tree digests in the journal for validation while archival preserves every post-rename backup mutation for inspection.
- [x] Preserved original permission modes for transaction-journal tracking-artifact preimages during rollback and interrupted-transition recovery (`internal/app/update_symlink_transaction_artifacts_unix.go`).

**Status**: Completed

## Spec Reference

- User request: add selective overwrite for `ign update` using template-side `.ign-overwrite-ignore`

## Implemented

- [x] Added `.ign-overwrite-ignore` as template metadata (`internal/template/model/types.go`)
- [x] Added selective overwrite mode and gitignore-style matching (`internal/template/generator/`)
- [x] Wired `ign update --overwrite`, `--overwrite-all`, and `--yes` (`internal/cli/update.go`)
- [x] Included `.ign-overwrite-ignore` changes in template hash calculation (`internal/app/template_update.go`)
- [x] Documented update overwrite behavior (`README.md`)
- [x] Omitted unchanged existing files from update overwrite confirmation and overwrite counts (`internal/template/generator/generator.go`)
- [x] Removed previously managed files during overwrite updates when the latest template no longer contains them (`internal/app/update_cleanup.go`)
- [x] Pruned already-missing stale managed files from `.ign/ign-files.json`, including stale paths protected by `.ign-overwrite-ignore` (`internal/app/update_cleanup.go`)
- [x] Added issue #45 manifest-owned directory-to-symlink transition handling, including no-follow ancestor refusal, descriptor-relative rename transactions, rollback restoration through artifact persistence, and stale-cleanup exclusions (`internal/app/update_symlink_transition.go`, `internal/app/update_symlink_transaction_unix.go`, `internal/app/update.go`, `internal/app/checkout.go`, `internal/app/update_cleanup.go`)
- [x] Added issue #45 app regressions for eligible and preserved directory-to-symlink transitions, unreadable and uncontained preservation, dry-run no-mutation output, cleanup safety, source divergence, and nested-symlink rollback (`internal/app/update_symlink_transition_test.go`)
- [x] Added issue #45 confirmation-plan handoff and manifest-persistence rollback regressions (`internal/cli/update_test.go`, `internal/app/update_symlink_transition_test.go`)
- [x] Added focused generator regressions for eligible, preserved, fatal, and ordinary symlink behavior (`internal/template/generator/generator_symlink_transition_test.go`)
- [x] Restored non-Darwin/Linux fail-closed compilation and added platform-scoped transaction coverage (`internal/app/update_symlink_transaction_other.go`, `internal/app/update_symlink_transition_test.go`, `internal/app/update_symlink_transaction_other_test.go`)
- [x] Hardened issue #45 preview fingerprints and recovery ordering against crafted source-state collisions, interrupted journal persistence, divergent recovery state, and config-only recovery bypasses (`internal/app/update_symlink_transition.go`, `internal/app/update_symlink_transaction_unix.go`, `internal/app/update.go`, `internal/app/update_symlink_transition_review_test.go`)
- [x] Closed the transition validation-to-rename race with descriptor-relative backup fingerprint validation, and reject divergent committed recovery states without mutation (`internal/app/update_symlink_transaction_unix.go`, `internal/app/update_symlink_transaction_fingerprint_unix.go`, `internal/app/update_symlink_transition_review_test.go`)

## Remaining

- [x] Complete independent and adversarial review for issue #45. Both review
  passes ran; every high and medium finding was remediated in code, and the
  remediations were re-verified after the automated workflow run was cancelled
  mid-verification.

## Design Decisions

- `--overwrite` performs selective overwrite and respects the remote template's `.ign-overwrite-ignore`.
- `--overwrite-all` preserves the previous overwrite-all behavior.
- `--force` remains the explicit regenerate option and uses overwrite-all semantics.
- Existing files are only reported as overwrite targets when generated content or permissions differ.
- Overwrite updates delete stale ign-managed files that are no longer generated by the current template.
- Overwrite previews report stale managed removals with `D`.
- Already-missing stale managed paths are pruned from `.ign/ign-files.json` without reporting a deletion.
- Preview plans fingerprint every generated symlink destination and the prior manifest using non-following traversal; inaccessible source nodes have a stable opaque fingerprint, and preserved roots are excluded from stale cleanup and manifest pruning.
- Preview fingerprints use typed length-prefixed fields so arbitrary file bytes cannot impersonate directory entries or manifest records.

## Notes

- `.ign-overwrite-ignore` is not generated into project output.
- Selective overwrite preserves stale managed files whose generated paths match `.ign-overwrite-ignore`.
- Already-missing stale manifest entries are pruned without reporting a file
  deletion.
- A non-empty directory can become a template symlink only after all terminal
  entries are proven manifest-managed; ambiguous trees are preserved.
- Eligible directory-to-symlink transitions use a same-filesystem backup rename;
  the backup survives generation and tracking-artifact persistence, then is
  moved into a collision-resistant `.ign` archive after commit. Symlinked
  ancestors are preserved rather than traversed.
- Interrupted eligible transitions persist a validated, output-root-relative
  recovery journal under `.ign`; the next non-dry-run update restores an
  uncommitted backup or finalizes a committed one before classification.
- Recovery validates the complete journal document and its exact tracking
  artifact allowlist before any recovery mutation; failed rollback or committed
  backup archival retains the journal for a later safe recovery.
- Tracking-artifact preimages retain their prior regular-file modes so rollback
  and uncommitted recovery restore metadata permissions as well as contents.
- The symlinked-ancestor recovery regression uses a complete valid journal and
  proves recovery reaches the no-follow ancestor refusal while preserving the
  external sentinel and durable journal record; focused recovery tests pass.
- A newly introduced template symlink remains an ordinary generator write; only
  an existing directory can be classified as a managed transition.
- A dry-run with a pending transition journal refuses without mutating the
  project, manifest, replacement symlink, or transaction backup.
- Recovery preflights every durable transaction record and its replacement target
  before restoring metadata or mutating any backup; config-only updates follow
  the same recovery gate.
- Eligible transitions carry a descriptor-relative source-tree fingerprint from
  planning. After rename, the backup must still match before a replacement
  symlink is created; otherwise the directory is restored and the update aborts.
- Committed recovery accepts only the expected replacement symlink. Missing or
  directory destinations, and backups that are not directories, retain the
  durable journal and require manual inspection without metadata mutation.
- Rollback and uncommitted recovery archive an expected replacement symlink
  instead of unlinking it. If its inode or target changes before the atomic
  archive rename completes, the substituted node is retained in the archive
  and the backup plus journal remain for later recovery.
- Ordinary generation rollback uses a private sibling rollback archive rather
  than deleting an output pathname after checking it. Archives are retained
  after rollback because a writer can still modify an archived inode after
  validation; pruning is deferred to a separate safe lifecycle. A substitution,
  recreation, or changed in-place content fails closed.
- On platforms without the descriptor-relative atomic rollback primitives,
  ordinary generation proceeds; a rollback that becomes necessary fails closed
  at rollback time and retains the private archive for manual recovery. Only
  eligible directory-to-symlink transitions are rejected before mutation.
- Private ordinary rollback archives are created beside the absolute output
  root, so a relative output such as `.` does not place retained archives inside
  the generated project.
- The final artifact-rollback remediation passed focused tests, direct full Go
  tests, host and Windows builds, vet, and scoped diff hygiene; `mise run test`
  remains runner-capped after reporting no failures.
- Latest ordinary-rollback verification passed the focused race and historical
  rollback suite, `mise run test`, `mise exec -- go vet ./...`, the Windows
  cross-build, formatting, and scoped diff hygiene. The final host-build
  wrapper timed out without diagnostics after the full suite compiled every
  host package.
- Post-recovery checks passed: `gofmt -d`, scoped `git diff --check`, `go mod
  verify`, focused app and generator tests, direct `go test ./...`, `mise run
  test`, `go build -o /dev/null ./...`, and `go vet ./...`.
- The non-Darwin/Linux transaction stub implements the complete update call
  surface and rejects eligible replacement transitions before mutation. Unix
  transaction and recovery regressions are limited to Darwin/Linux; a portable
  non-Darwin/Linux regression asserts the fail-closed branch. The remediation
  verification includes Windows cross-build and app-test compilation gates.
- Final takeover verification (after the automated review workflow was
  cancelled): project-wide `gofmt` clean, `mise exec -- go vet ./...`, host
  build, `GOOS=windows GOARCH=amd64` cross-build and vet, and the full
  `go test -count=1 ./...` suite all passed, with zero retained
  `.ign-checkout-rollback-*` archives left in the repository after the run.
