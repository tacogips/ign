# Issue 45 Managed Directory-To-Symlink Transition Implementation Plan

## Latest Progress: Atomic Ordinary Rollback Remediation

### Session: 2026-08-11 (Atomic Ordinary Rollback Remediation)

**Tasks Completed**: Replaced ordinary generated and overwritten rollback's
pathname deletion and truncating restoration with same-filesystem quarantine,
post-move identity/content fingerprint validation, and no-replace preimage
restoration. Concurrent substitutions, destination recreation, and in-place
changes now fail closed. Private rollback archives are created beside the
output root and retained after archival: automatic cleanup could delete an
in-place mutation made after validation. Their pruning is deliberately
deferred. Non-Darwin/Linux builds retain a fail-closed stub.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/checkout.go`,
`internal/app/checkout_rollback_nodes.go`,
`internal/app/checkout_rollback_nodes_unix.go`,
`internal/app/checkout_rollback_nodes_other.go`,
`internal/app/checkout_rollback_nodes_unix_test.go`, `internal/app/update.go`,
`docs/progress/selective-overwrite.md`, and this plan.
**Verification**: The focused rollback suite passed, including historical
checkout/config/manifest rollback coverage, the three deterministic race
regressions, and post-validation private-archive retention coverage. `mise run
test`, `mise exec -- go vet ./...`, the Windows cross-build, scoped diff hygiene,
and Go formatting passed. The latest host build wrapper timed out without
diagnostics; the full suite compiled all host packages successfully.

---

### Session: 2026-08-11 (Step 7 Rollback And Artifact-Root Remediation)

**Tasks Completed**: Revalidated manifest ownership against the exact
descriptor-relative tree renamed into each transaction backup, closing the
classifier-to-fingerprint addition window. Generic generation rollback now
captures generated-node identities, refuses concurrent substitutions, and
leaves parent directories in place when their identity cannot be proven.
Committed-journal persistence failure now invokes complete generation rollback.
Tracking artifacts are opened through the validated parent of the actual
`ign-files.json` location; `ign.json` and `ign-var.json` must be sibling files
or the transition fails before mutation.
**Tasks In Progress**: Focused verification and renewed independent and
adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/checkout.go`, `internal/app/update.go`,
`internal/app/update_symlink_transition.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_fingerprint_unix.go`,
`internal/app/update_symlink_transaction_artifacts_unix.go`,
`internal/app/update_symlink_transaction_other.go`,
`internal/app/update_symlink_transition_review_test.go`,
`internal/app/update_symlink_transaction_artifacts_test.go`,
`internal/app/checkout_rollback_nodes.go`, `docs/progress/selective-overwrite.md`,
and this plan.
**Verification**: `mise exec -- gofmt -w` completed. Focused new and checkout
rollback compatibility regressions passed. `mise exec -- go build -o /dev/null
./...` and `GOOS=windows GOARCH=amd64 mise exec -- go build -o /dev/null ./...`
passed. `mise run test`, `mise exec -- go vet ./...`, and issue-scoped `git diff
--check` reached the 60-second runner cap without failure diagnostics after
their observable output; renewed independent and adversarial reviews remain
required.

---

- [x] Persist a descriptor-relative cleanup snapshot for each renamed managed directory.
- [x] Archive each committed transaction backup into collision-resistant `.ign` storage instead of recursively deleting it.
- [x] Use a descriptor-relative cross-directory no-replace archive rename; occupied archive names retain the backup and durable journal.
- [x] Use no-replace restoration so concurrent destination recreation is preserved.
- [x] Archive replacement symlinks with a no-replace rename before rollback or
  uncommitted recovery; an inode or target substitution is retained in archive
  storage with the backup and journal left actionable.
- [x] Add focused Darwin/Linux archival regressions for descendant creation, in-place file mutation, leaf replacement, and recovery finalization.
- [x] Preserve durable regular-file content digests in the journal for source-state validation and recovery diagnostics.
- [x] Persist and restore original tracking-artifact permission modes through rollback and uncommitted recovery.
- [ ] Renew independent implementation review.
- [ ] Complete required adversarial review.

**Status**: In Progress
**Design Reference**: `design-docs/specs/selective-overwrite.md#managed-directory-to-symlink-transitions`
**Created**: 2026-08-10
**Last Updated**: 2026-08-11

### Session: 2026-08-11 (Tracking-Artifact Permission Recovery Remediation)

**Tasks Completed**: Preserved the original regular-file permission modes of
`ign-files.json`, `ign.json`, and `ign-var.json` in the durable transaction
journal. Uncommitted recovery and rollback now restore both their preimage
contents and modes. The primary success regression now asserts the complete
issue-scoped manifest and exactly one replacement-symlink record.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transaction_artifacts_unix.go`,
`internal/app/update_symlink_transaction_artifacts_test.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transition_test.go`, `docs/progress/selective-overwrite.md`,
and this active plan.
**Verification**: `mise exec -- gofmt -w` completed for the changed Go files.
`mise exec -- go test ./internal/app -run
'TestRecoverSymlinkTransitionJournalRestoresArtifactPreimageModes|TestCompleteUpdate_OverwriteSymlinkManagedDirectory'
-count=1 -v` passed. The broader focused app symlink/recovery suite, `mise run
test`, `mise exec -- go build -o /dev/null ./...`, `mise exec -- go vet ./...`,
and `GOOS=windows GOARCH=amd64 mise exec -- go build -o /dev/null ./...`
passed. `git diff --check` had passed before this final documentation update;
the renewed global invocation timed out without diagnostics.

---

## Design Document Reference

**Source**: `design-docs/specs/selective-overwrite.md`
**Issue**: `tacogips/ign#45`

### Summary

Allow an overwrite update to replace a non-empty, fully manifest-managed
directory with a template symlink. One shared transition classification must
drive preview, generation, cleanup, rollback, and manifest persistence so that
the update either commits the symlink and matching tracking state or preserves
the prior tree and tracking state.

### Scope

**Included**:

- directory-to-symlink transitions during `--overwrite`, `--overwrite-all`, and
  `--force` updates;
- manifest ownership and output-root containment checks without following
  symlinks;
- selective-overwrite protection for the transition root and descendants;
- recursive rollback capture and restoration that preserves nested symlinks;
- transition-aware stale cleanup and atomic persistence of `.ign/ign-files.json`,
  `.ign/ign.json`, and `.ign/ign-var.json`;
- an opaque app execution plan handed from CLI confirmation preview to the real
  update, with pre-mutation source-state validation that aborts on divergence;
- focused generator, app, and CLI regression coverage, full Go verification,
  and progress-document refresh.

**Excluded**:

- recursive deletion of ordinary stale manifest directories;
- directory replacement outside template-defined symlink transitions;
- changes to unrelated `GenerateResult.Errors` behavior;
- release work, publishing, or modification of `internal/build/VERSION`;
- unrelated worktree cleanup or refactoring.

### Reference Traceability And Boundaries

| Reference | Planned Trace |
|-----------|---------------|
| Issue #45: `.claude/` becomes `.claude -> .agents` | App regression fixture starts with a non-empty manifest-managed `.claude/` and verifies the resulting symlink. |
| Accepted design ownership boundary | `internal/app/update_symlink_transition.go` classifies the complete existing tree from `.ign/ign-files.json` using contained, normalized paths and `Lstat`-style inspection. |
| Accepted shared-classification data flow | One opaque `app.UpdateExecutionPlan` is created by preview, returned to the CLI, passed unchanged into the confirmed real `CompleteUpdate`, and consumed by generator options/results, rollback preparation, stale cleanup, and manifest persistence. A source-state mismatch aborts before mutation instead of reclassifying. |
| Existing generation path | Extend `internal/template/generator/generator.go` and `internal/template/generator/writer.go` only for explicitly classified update transitions. |
| Existing update transaction path | Integrate through `internal/app/update.go`, `internal/app/checkout.go`, `internal/app/update_cleanup.go`, and `internal/app/manifest.go`; update `internal/cli/update.go` only to retain and return the opaque preview plan. |
| Agent requirements | Implementation uses `.agents/agents/go-coding.md`; verification uses `.agents/agents/go-check-and-test-after-modify.md`; both remain governed by `AGENTS.md`. |

The reference repository is the current repository (`.`). There is no external
Codex implementation or Cursor adapter in this feature, so no adapter divergence
is required. The only intentional behavior extension is the ownership-validated
directory-to-symlink exception accepted in the design.

## Planned Modules And Contracts

### 1. Shared Generator Transition Contract

#### `internal/template/generator/generator.go`

**Status**: IMPLEMENTED

```go
type SymlinkTransitionDisposition string

const (
	SymlinkTransitionEligible  SymlinkTransitionDisposition = "eligible"
	SymlinkTransitionPreserved SymlinkTransitionDisposition = "preserved"
)

type SymlinkTransition struct {
	Path                string
	Disposition         SymlinkTransitionDisposition
	RetiredManagedPaths []string
}

type GenerateOptions struct {
	// Existing fields remain unchanged.
	SymlinkTransitions map[string]SymlinkTransition
}

type DryRunFile struct {
	// Existing fields remain unchanged.
	SymlinkTarget string
}
```

The implementation may refine field names while preserving the accepted data
flow: path-keyed disposition plus the retired manifest entries must be available
to every consuming phase. No unrelated exported generator behavior may change.

**Checklist**:

- [x] Represent eligible and preserved transitions explicitly.
- [x] Expose symlink identity in dry-run output for app-layer classification.
- [x] Preserve backward compatibility when no transition plan is supplied.

### 2. App-Layer Transition Classification

#### `internal/app/update_symlink_transition.go`

**Status**: IMPLEMENTED

```go
type updateSymlinkTransitionPlan struct {
	Transitions map[string]generator.SymlinkTransition
}

type updateSymlinkTransitionOptions struct {
	OutputDir       string
	ManifestPath    string
	Template        *model.Template
	Preview         *generator.GenerateResult
	Overwrite       bool
	OverwriteMode   generator.OverwriteMode
}
```

**Checklist**:

- [x] Identify only current-template symlinks whose existing output path is a directory.
- [x] Prove every non-directory descendant is in the prior manifest.
- [x] Reject empty, unrelated, unreadable, uncontained, or untracked subtrees as preserved.
- [x] Apply selective overwrite-ignore matching to the root and candidate descendants.
- [x] Use `Lstat`/non-following traversal and normalized output-root-contained paths.
- [x] Record every prior manifest descendant as transition-retired only for eligible transitions.

### 3. Generator Mutation Boundary

#### `internal/template/generator/generator.go`
#### `internal/template/generator/writer.go`

**Status**: IMPLEMENTED

```go
type Writer interface {
	WriteFile(path string, content []byte, mode os.FileMode) error
	WriteSymlink(path string, target string) error
	ReplaceDirectoryWithSymlink(path string, target string) error
	CreateDir(path string) error
	Exists(path string) bool
}
```

**Checklist**:

- [x] Skip preserved transition roots in both preview and real generation.
- [x] Report an eligible transition as one overwrite modification at the symlink path.
- [x] Recursively remove only an explicitly eligible directory without following nested symlinks.
- [x] Return eligible-transition removal or symlink-creation failures through
      `Generator.Generate`'s `error` return so `CompleteUpdate` rolls back; never
      append these failures to non-fatal `GenerateResult.Errors`.
- [x] Keep existing non-transition generation errors and ordinary symlink replacement behavior unchanged.

### 4. Update Transaction, Rollback, Cleanup, And Manifest

#### `internal/app/update.go`
#### `internal/app/checkout.go`
#### `internal/app/update_cleanup.go`
#### `internal/app/manifest.go`
#### `internal/cli/update.go`

**Status**: IMPLEMENTED

```go
type UpdateExecutionPlan struct {
	outputDir              string
	templateHash           string
	overwriteMode          generator.OverwriteMode
	symlinkTransitions     updateSymlinkTransitionPlan
	sourceStateFingerprint string
}

type CompleteUpdateOptions struct {
	// Existing fields remain unchanged.
	ExecutionPlan *UpdateExecutionPlan
}

type UpdateResult struct {
	// Existing fields remain unchanged.
	ExecutionPlan *UpdateExecutionPlan
}

type checkoutRollbackEntry struct {
	path       string
	existed    bool
	mode       os.FileMode
	backupPath string
	linkTarget string
	isSymlink  bool
	isDir      bool
	children   []checkoutRollbackEntry
}

type cleanupRemovedManagedFilesOptions struct {
	// Existing fields remain unchanged.
	SymlinkTransitions map[string]generator.SymlinkTransition
}
```

**Checklist**:

- [x] When no plan is supplied, preview and classify once; return the resulting
      opaque plan in dry-run output or reuse it directly for `--yes` execution.
- [x] Retain `preview.ExecutionPlan` in `internal/cli/update.go` and pass the same
      pointer into the confirmed real `CompleteUpdate` call.
- [x] Bind the plan to output directory, template identity, overwrite mode, and
      a non-symlink-following source-state fingerprint; reject mismatches before
      mutation rather than silently reclassifying.
- [x] Reuse the supplied transition map unchanged for generator dry-run,
      mutation, rollback, cleanup, and persistence.
- [x] Capture eligible directory trees and all three tracking artifacts before mutation.
- [x] Snapshot and restore nested symlinks as link nodes without traversing targets.
- [x] Exclude eligible retired descendants from post-symlink filesystem cleanup.
- [x] Retain preserved-root descendants in the manifest and do not claim the symlink was written.
- [x] After success, prune retired descendants and record the symlink exactly once.
- [x] On transition or persistence failure, restore the prior tree and tracking artifacts.

### 5. Regression Tests And Progress Documentation

#### `internal/app/update_test.go`
#### `internal/app/update_symlink_transition_test.go`
#### `internal/cli/update_test.go`
#### `internal/template/generator/generator_test.go`
#### `docs/progress/selective-overwrite.md`

**Status**: IMPLEMENTED

No new production API is planned in this module group.

**Checklist**:

- [x] Add the accepted issue #45 success regression and manifest assertions.
- [x] Add no-overwrite, protected, untracked, empty-subtree, dry-run, cleanup-safety, nested-symlink rollback, and failed-transition cases.
- [x] Add a confirmation-flow regression proving the CLI passes the exact
      preview execution plan into mutation and a changed source-state proof
      aborts before any write or tracking update.
- [x] Add focused generator coverage for eligible, preserved, fatal-failure, and ordinary symlink behavior.
- [x] Update progress status, spec/issue reference, implemented paths, remaining work, decisions, and notes.

## Task Breakdown

| ID | Task | Deliverables | Dependencies | Write Scope | Status |
|----|------|--------------|--------------|-------------|--------|
| T1 | Establish the shared transition contract | Generator transition types and dry-run symlink metadata | Accepted design | `internal/template/generator/generator.go` | COMPLETED |
| T2 | Implement app classification | Ownership, containment, ignore-policy, and retired-entry classifier | T1 | `internal/app/update_symlink_transition.go` | COMPLETED |
| T3 | Implement generator transition execution | Preserved skip and transaction-aware generated-file accounting | T1 | `internal/template/generator/generator.go`, `internal/template/generator/writer.go` | COMPLETED |
| T4 | Integrate the update transaction and confirmation handoff | Opaque execution-plan return/input contract, descriptor-relative no-follow rename transaction, rollback, cleanup exclusions, and persistence | T2, T3 | `internal/app/update.go`, `internal/app/update_symlink_transaction_*.go`, `internal/app/checkout.go`, `internal/app/update_cleanup.go`, `internal/cli/update.go` | COMPLETED |
| T5 | Add app and CLI regressions | End-to-end success, metadata, cleanup-safety, and preview-plan coverage | T4 | `internal/app/update_symlink_transition_test.go`, `internal/cli/update_test.go` | COMPLETED |
| T6 | Add generator regressions | Focused eligible/preserved/failure and compatibility tests | T3 | `internal/template/generator/generator_symlink_transition_test.go` | COMPLETED |
| T7 | Refresh progress documentation | Issue #45 status and implemented/remaining path inventory | T5, T6 | `docs/progress/selective-overwrite.md` | COMPLETED |
| T8 | Verify and review | Formatting, focused/full tests, build/vet, self-review, independent review, adversarial review | T5, T6, T7 | Read-only except fixes limited to issue #45 files | IN_PROGRESS |

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Shared transition contract | `internal/template/generator/generator.go` | IMPLEMENTED | T1, T6 |
| Transition classifier | `internal/app/update_symlink_transition.go` | IMPLEMENTED | T2, T5 |
| Authorized filesystem mutation | `internal/app/update_symlink_transaction_*.go` | IMPLEMENTED | T5 |
| Update orchestration and confirmation handoff | `internal/app/update.go`, `internal/cli/update.go` | IMPLEMENTED | T5 |
| Recursive rollback | `internal/app/checkout.go` | IMPLEMENTED | T5 |
| Cleanup exclusions | `internal/app/update_cleanup.go` | IMPLEMENTED | T5 |
| Manifest persistence | `internal/app/manifest.go` | REUSED | T5 |
| Progress record | `docs/progress/selective-overwrite.md` | IMPLEMENTED | Review |

## Dependencies

| Task | Depends On | Status |
|------|------------|--------|
| T1 | Accepted design and Step 3 approval | COMPLETED |
| T2 | T1 shared contract | COMPLETED |
| T3 | T1 shared contract | COMPLETED |
| T4 | T2 classifier and T3 generator execution | COMPLETED |
| T5 | T4 integrated update transaction | COMPLETED |
| T6 | T3 generator execution | COMPLETED |
| T7 | T5 and T6 verified behavior | COMPLETED |
| T8 | T5, T6, and T7 | IN_PROGRESS |

## Parallelizable Tasks

- T2 and T3 may run in parallel after T1 because their write scopes are disjoint.
- T5 and T6 may run in parallel after their dependencies because their test-file
  write scopes are disjoint.
- T4, T7, and T8 remain sequential because they integrate or validate outputs
  from multiple preceding tasks.

## Verification

Run formatting and diff hygiene before the required verification suite:

```bash
gofmt -w \
  internal/app/checkout.go \
  internal/app/manifest.go \
  internal/app/update.go \
  internal/app/update_cleanup.go \
  internal/app/update_symlink_transition.go \
  internal/app/update_symlink_transaction_unix.go \
  internal/app/update_symlink_transaction_other.go \
  internal/app/update_symlink_transition_test.go \
  internal/app/update_test.go \
  internal/cli/update.go \
  internal/cli/update_test.go \
  internal/template/generator/generator.go \
  internal/template/generator/generator_test.go \
  internal/template/generator/generator_symlink_transition_test.go \
  internal/template/generator/writer.go
git diff --check -- \
  design-docs/specs/selective-overwrite.md \
  docs/progress/selective-overwrite.md \
  impl-plans/active/issue-45-managed-directory-to-symlink.md \
  internal/app/checkout.go \
  internal/app/manifest.go \
  internal/app/update.go \
  internal/app/update_cleanup.go \
  internal/app/update_symlink_transition.go \
  internal/app/update_symlink_transaction_unix.go \
  internal/app/update_symlink_transaction_other.go \
  internal/app/update_symlink_transition_test.go \
  internal/app/update_test.go \
  internal/cli/update.go \
  internal/cli/update_test.go \
  internal/template/generator/generator.go \
  internal/template/generator/generator_test.go \
  internal/template/generator/writer.go
go test ./internal/template/generator -run 'Symlink'
go test ./internal/app -run 'TestCompleteUpdate_OverwriteSymlink'
go test ./internal/cli -run 'TestRunUpdate_ConfirmationPreviewReusesSymlinkTransitionPlan'
mise run test
go build -o /dev/null ./...
go vet ./...
GOOS=windows GOARCH=amd64 go build -o /dev/null ./...
GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/app
git status --short -- \
  design-docs/specs/selective-overwrite.md \
  docs/progress/selective-overwrite.md \
  impl-plans/active/issue-45-managed-directory-to-symlink.md \
  internal/app/checkout.go \
  internal/app/manifest.go \
  internal/app/update.go \
  internal/app/update_cleanup.go \
  internal/app/update_symlink_transition.go \
  internal/app/update_symlink_transition_test.go \
  internal/app/update_test.go \
  internal/cli/update.go \
  internal/cli/update_test.go \
  internal/template/generator/generator.go \
  internal/template/generator/generator_test.go \
  internal/template/generator/generator_symlink_transition_test.go \
  internal/template/generator/writer.go \
  internal/build/VERSION
```

The implementation review must also inspect the issue-scoped diff and confirm
that `internal/build/VERSION` and unrelated dirty paths remain untouched. Use the
Go verification agent from `.agents/agents/go-check-and-test-after-modify.md`
after every Go modification cycle.

## Completion Criteria

- [x] A regression starts with a non-empty, fully manifest-managed directory at a path the new template defines as a symlink.
- [x] Overwrite replaces the eligible directory with the expected symlink and emits no generation error.
- [x] No-overwrite, protected, untracked, or empty transitions preserve cleanly without a generation error.
- [x] Dry-run, confirmation preview, and real generation use the same opaque
      execution plan and report one `M` for an eligible transition.
- [x] Confirmation passes the exact preview plan into mutation; source-state and
      manifest divergence coverage is implemented.
- [x] Successful tracking records the symlink once, removes retired descendants, and advances all three artifacts consistently.
- [x] Preserved and failed transitions retain or restore the prior tree and tracking state without partial metadata.
- [x] Nested symlink capture, removal, and restoration never traverse the link target.
- [x] Focused tests, `mise run test`, `go build -o /dev/null ./...`, and `go vet ./...` pass.
- [x] `internal/build/VERSION` and unrelated pre-existing worktree changes remain untouched.
- [x] The non-Darwin/Linux fail-closed transaction path compiles, and its
      platform-specific tests do not break cross-platform app-test compilation.
- [ ] Independent implementation and adversarial reviews report zero unresolved high or medium findings.
- [x] `docs/progress/selective-overwrite.md` reflects the completed issue #45 work.

## Risks And Controls

| Risk | Control |
|------|---------|
| Divergent preview and mutation decisions | Return an opaque execution plan from preview, pass that same plan through CLI confirmation into mutation, and test pointer-identical handoff. |
| Filesystem changes after confirmation preview | Bind a non-symlink-following source-state fingerprint to the plan and abort before mutation when validation differs. |
| Cleanup follows the new symlink into its target | Treat eligible retired entries as cleanup filesystem exclusions before manifest pruning. |
| Recursive traversal follows nested symlinks | Use `Lstat`/directory-entry traversal and record links as terminal nodes. |
| User-owned data is removed | Require complete manifest ownership and preserve the whole root on any uncertainty. |
| Failed persistence leaves mixed state | Snapshot the tree and three tracking artifacts before mutation and restore them together. |
| Broad behavior or release drift | Keep ordinary cleanup/error behavior unchanged and prohibit `internal/build/VERSION` or release edits. |

## Progress Log Expectations

For every implementation session, append a dated entry containing tasks
completed, tasks in progress, blockers, files changed, verification commands and
results, review findings, and decisions. Update task/module status and completion
checkboxes in the same change. When complete, add the final review evidence,
change status to `Completed`, and move this file to `impl-plans/completed/`.

## Progress Log

### Session: 2026-08-10

**Tasks Completed**: Plan created from the accepted issue #45 design.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Ready for Step 5 implementation-plan review. No production code was changed.

### Session: 2026-08-10 (Self-Review Revision)

**Tasks Completed**: Replaced verification placeholders with explicit issue #45
paths and fixed the dedicated classifier test-file scope.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Addressed the Step 4 self-review mid finding; no production code was changed.

### Session: 2026-08-10 (Step 5 Review Revision)

**Tasks Completed**: Added the opaque confirmation-preview execution-plan
handoff, CLI write scope, divergence-abort contract, regression ownership, and
focused CLI verification command.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Addressed the Step 5 high finding; no production code was changed.

### Session: 2026-08-10 (Implementation)

**Tasks Completed**: T1-T4. Added shared transition dispositions, validated
manifest ownership using non-following traversal, fatal authorized mutation,
preview-to-confirmation plan reuse and source-state validation, recursive
rollback snapshots, and cleanup exclusions for retired descendants.
**Tasks In Progress**: T5-T8 verification and review.
**Blockers**: None.
**Files Changed**: `internal/template/generator/generator.go`,
`internal/template/generator/writer.go`, `internal/app/update_symlink_transition.go`,
`internal/app/update.go`, `internal/app/update_cleanup.go`,
`internal/app/checkout.go`, `internal/cli/update.go`, and
`internal/app/update_test.go`.
**Verification**: `mise exec -- go test ./internal/template/generator -run 'Symlink'`
and `mise exec -- go test ./internal/app -run 'TestCompleteUpdate_OverwriteSymlink'`
passed.
**Notes**: T5/T6 retain the design-required preservation, rollback, and failure
coverage for the implementation-review loop. Full `mise run test` exceeded the
120-second runner cap after reporting passing packages; independent verification
also observed passing focused generator and app tests before its wrapper timed out.

### Session: 2026-08-10 (Self-Review Finding Revision)

**Tasks Completed**: Preserved-transition cleanup retains all prior manifest
entries below the root; execution-plan validation fingerprints every generated
symlink destination, regular-file content, and manifest ownership; dedicated
app and generator transition regressions were added.
**Tasks In Progress**: T5 CLI confirmation-handoff coverage and T8 verification,
independent review, and adversarial review.
**Blockers**: The focused app test command was interrupted at the user's request.
**Verification**: `mise exec -- gofmt -w` completed. `mise exec -- go test
./internal/template/generator -run 'Symlink'` passed. Full suite, CLI, build,
vet, and diff hygiene remain pending.

### Session: 2026-08-11 (Regression Completion)

**Tasks Completed**: T5. Added a confirmation-flow regression that proves the
exact opaque preview execution-plan pointer reaches confirmed mutation. Added a
manifest-persistence-failure regression that verifies rollback restores the
managed directory and all three `.ign` artifacts without recording the unwritten
symlink.
**Tasks In Progress**: T8 verification, independent review, and adversarial
review.
**Blockers**: None.
**Files Changed**: `internal/cli/update.go`, `internal/cli/update_test.go`,
`internal/app/manifest.go`, and `internal/app/update_symlink_transition_test.go`.
**Verification**: `mise exec -- gofmt -w internal/cli/update.go
internal/cli/update_test.go internal/app/manifest.go
internal/app/update_symlink_transition_test.go` completed. `mise exec -- go
test ./internal/app -run 'TestCompleteUpdate_OverwriteSymlinkManifestPersistenceFailureRestoresPriorState|TestCompleteUpdate_OverwriteSymlink'`
and `mise exec -- go test ./internal/cli -run
'TestRunUpdate_ConfirmationPreviewReusesSymlinkTransitionPlan'` passed.

### Session: 2026-08-11 (Self-Review Safety Revision)

**Tasks Completed**: Updated classifier and fingerprint behavior so unreadable
or uncontained ownership state preserves the existing directory without a
generation error. Added unreadable, uncontained, and dry-run no-mutation
regressions, and reconciled completed module checklists with the task matrix.
**Tasks In Progress**: T8 verification, independent review, and adversarial
review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transition.go`,
`internal/app/update_cleanup.go`, `internal/app/update_symlink_transition_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: Focused app tests, `mise run test`, `go build -o /dev/null
./...`, and scoped `git diff --check` passed. `go vet ./...` exceeded the
environment wrapper timeout without diagnostics; final formatter diff output was
also unavailable after its wrapper timeout, although the preceding `gofmt -w`
completed successfully.

### Session: 2026-08-11 (Adversarial Review Remediation)

**Tasks Completed**: Replaced the generator's recursive directory deletion with
an app-layer transaction. On Darwin and Linux it opens every ancestor with
`openat(..., O_NOFOLLOW)`, renames the owned source into a same-directory backup,
creates the symlink through the verified parent descriptor, restores by rename
on generation or artifact-persistence failure, and recursively removes the
backup only after artifacts commit. Unsupported platforms refuse eligible
destructive transitions. Eligible preprepared links are still reported as one
overwrite and persisted in the manifest.
**Tasks In Progress**: T8 focused/full verification and renewed independent and
adversarial review.
**Blockers**: Tool-environment formatting and verification commands are being
rerun after this remediation.
**Files Changed**: `internal/app/update.go`, `internal/app/checkout.go`,
`internal/app/update_symlink_transition.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_other.go`,
`internal/app/update_symlink_transition_test.go`,
`internal/template/generator/generator.go`,
`internal/template/generator/writer.go`, and
`internal/template/generator/generator_symlink_transition_test.go`.
**Verification**: The repository verification agent attempted direct `gofmt -d`
and the focused app, generator, build, vet, and diff-hygiene commands. The
shell runner blocked before any gate completed or emitted diagnostics; all
post-remediation verification remains unconfirmed and must be rerun.

### Session: 2026-08-11 (Durable Interrupted-Transaction Recovery)

**Tasks Completed**: Added a descriptor-relative `.ign` recovery journal for
eligible transitions. A subsequent non-dry-run update restores a pre-commit
backup or safely finalizes a committed backup after validating the output-root
relative path, backup basename, and replacement target. The journal stores
preimages of all three tracking artifacts and marks commit only after each is
persisted and synced, so manifest-only interruption restores the prior tree and
metadata. Dry runs now refuse a pending recovery journal without mutation.
**Tasks In Progress**: T8 formatting, focused/full verification, and renewed
independent and adversarial review.
**Blockers**: Verification commands must be rerun because the prior shell runner
did not complete them after the durable-recovery changes.
**Files Changed**: `internal/app/update.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_other.go`, and
`internal/app/update_symlink_transition_test.go`.
**Verification**: `mise exec -- gofmt -w internal/app/update.go
internal/app/update_symlink_transaction_unix.go
internal/app/update_symlink_transaction_other.go
internal/app/update_symlink_transition_test.go` passed. `mise exec -- go test
./internal/app -run 'TestRecoverSymlinkTransitionJournal|TestCompleteUpdateDryRunRefusesPendingSymlinkTransitionRecovery' -count=1`
passed in 0.593 seconds; its outer wrapper timed out after emitting success.

### Session: 2026-08-11 (Post-Recovery T8 Verification)

**Tasks Completed**: Re-ran formatting, module-integrity, focused regression,
full direct-test, static-analysis, and scoped diff-hygiene checks after the
durable-recovery changes.
**Tasks In Progress**: T8 exact `mise run test` and project build completion,
then renewed independent and adversarial review.
**Blockers**: The command runner exceeded its cap before `mise run test` and
`go build -o /dev/null ./...` completed; neither command emitted a diagnostic.
**Verification**: `gofmt -d` and issue-scoped `git diff --check` passed with no
output; `go mod verify` passed; focused app and generator tests passed; direct
`go test ./...` passed all packages including integration tests; and `go vet
./...` passed with no diagnostics. `mise run test` showed no failure, including
an integration-test pass, before the cap; build completion remains unconfirmed.

### Session: 2026-08-11 (T8 Verification Completion)

**Tasks Completed**: Completed the exact full-test and project-wide build
verification after expanding the runner window.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Verification**: `mise run test` completed with exit status 0 and all packages
passed. `go build -v -o /dev/null ./...` completed with exit status 0. Earlier
focused tests, direct full tests, `go vet ./...`, `go mod verify`, `gofmt -d`,
and issue-scoped `git diff --check` remain passing.

### Session: 2026-08-11 (Test-Integrity Platform Remediation)

**Tasks Completed**: Resolved the Step 6 test-integrity finding by completing
the non-Darwin/Linux transaction stub's update-facing method set, moving the
Darwin/Linux transaction and recovery suite behind its platform build
constraint, and adding a non-Darwin/Linux fail-closed regression.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: The post-remediation `go test ./...` wrapper reached its
60-second cap after every package, including integration tests, reported pass;
the immediately preceding T8 full-suite, build, and vet gates completed
successfully, and this remediation changes only the non-Darwin/Linux stub and
test build constraints.
**Files Changed**: `internal/app/update_symlink_transaction_other.go`,
`internal/app/update_symlink_transition_test.go`,
`internal/app/update_symlink_transaction_other_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `GOOS=windows GOARCH=amd64 go build -o /dev/null ./...` and
`GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/app` passed.
Focused app and generator tests passed; scoped `go vet`, `gofmt -d`, and
issue-scoped `git diff --check` passed with no diagnostics or diff. The
post-remediation direct full suite reported all packages passing before the
runner cap; the prior exact `mise run test`, project build, and project vet
evidence remains recorded above.

### Session: 2026-08-11 (Transaction Rollback Nested-Symlink Coverage)

**Tasks Completed**: Added a Darwin/Linux regression that drives the eligible
directory-to-symlink rename transaction through its real rollback path with a
nested symlink. The test confirms that rollback restores the symlink node and
its target while leaving an external sentinel reachable only through that link
unchanged.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transition_test.go` and this
active plan.
**Verification**: `mise exec -- go test ./internal/app -run
'^TestSymlinkTransitionTransactionRollbackRestoresNestedSymlinkWithoutTargetTraversal$' -count=1`
passed in 0.471 seconds; `mise exec -- gofmt -d
internal/app/update_symlink_transition_test.go` produced no diff; and
`mise exec -- go test ./internal/app -run
'Symlink|TestCompleteUpdate|TestRecoverSymlinkTransitionJournal' -count=1`
passed in 1.957 seconds. `mise run test` completed with exit status 0, including
integration tests. A fresh project build was terminated after 72 seconds without
compiler diagnostics; the prior completed T8 build and project-wide vet evidence
remains applicable because this remediation changes only a compiled test file.
Scoped `git diff --check` reported no whitespace errors for tracked issue #45
paths; renewed independent/adversarial review remains in T8.

### Session: 2026-08-11 (Step 7 Review Remediation)

**Tasks Completed**: Remediated all Step 7 high and medium findings. Recovery
now validates the complete journal document, including its exact unique tracking
artifact allowlist and all entries, before any artifact or transition mutation.
Absent, regular-file, and symlink destinations now retain ordinary generator
behavior; only existing directories receive transition dispositions. Failed
rollback and committed-backup cleanup preserve the durable journal for recovery.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transition.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transition_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- go test ./internal/app -run
'Symlink|TestCompleteUpdate|TestRecoverSymlinkTransitionJournal' -count=1`,
`mise run test`, `mise exec -- go build -o /dev/null ./...`, `mise exec -- go
vet ./...`, `GOOS=windows GOARCH=amd64 mise exec -- go build -o /dev/null
./...`, and scoped `git diff --check` passed. `mise exec -- gofmt -w` completed
for the three changed Go files.
**Notes**: New regressions cover malformed absolute, parent-relative, duplicate,
unknown, and missing journal artifact names; an absent template symlink target;
and rollback or committed-backup cleanup failures retaining recovery state.

### Session: 2026-08-11 (Actionable Retained-Journal Recovery Coverage)

**Tasks Completed**: Strengthened the injected rollback and committed-cleanup
failure regression to reload and validate the durable journal, execute recovery,
and assert the corresponding restored directory or finalized replacement
symlink, backup removal, and journal cleanup.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transition_test.go` and this
active plan.
**Verification**: `mise exec -- gofmt -w
internal/app/update_symlink_transition_test.go` completed. `mise exec -- go
test ./internal/app -run
'TestSymlinkTransitionTransactionsRetainJournalAfterRollbackOrCommitFailure|TestRecoverSymlinkTransitionJournal' -count=1`
passed. The full `mise run test` suite, `mise exec -- go build -o /dev/null
./...`, `mise exec -- go vet ./...`, and the Windows build gate passed after
this regression change. `mise exec -- gofmt -d` produced no diff. The renewed
scoped `git diff --check` invocation reached the runner timeout without a
diagnostic; its preceding clean result remains recorded. Renewed review is the
only T8 work in progress.

### Session: 2026-08-11 (Symlinked-Ancestor Recovery Regression Repair)

**Tasks Completed**: Repaired the symlinked-ancestor recovery regression so its
journal contains the exact unique `ign-files.json`, `ign.json`, and
`ign-var.json` preimages required for journal validation. The regression now
asserts that recovery reaches the descriptor-relative no-follow ancestor refusal
rather than failing validation, and that both the external sentinel and durable
journal remain unchanged.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transition_test.go`,
`internal/app/update_symlink_recovery_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- gofmt -w internal/app/update_symlink_transition_test.go
internal/app/update_symlink_recovery_test.go` completed. `mise exec -- go test
./internal/app -run '^TestRecoverSymlinkTransitionJournalRefusesSymlinkedAncestor$'
-count=1` and `mise exec -- go test ./internal/app -run
'TestRecoverSymlinkTransitionJournal|TestSymlinkTransitionTransactionsRetainJournalAfterRollbackOrCommitFailure'
-count=1` passed. Scoped `git diff --check` passed for the repaired test and
documentation paths.

### Session: 2026-08-11 (Step 7 State-Integrity Remediation)

**Tasks Completed**: Addressed the renewed Step 7 high and medium findings.
Execution-plan fingerprints now use typed length-prefixed fields, recovery
preflights every record and expected replacement target before metadata or tree
mutation, failed committed-journal persistence leaves rollback enabled, and the
journal gate runs before config-only completion.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update.go`,
`internal/app/update_symlink_transition.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transition_review_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- gofmt -w internal/app/update.go
internal/app/update_symlink_transition.go
internal/app/update_symlink_transaction_unix.go
internal/app/update_symlink_transition_review_test.go` completed. `mise exec --
go test ./internal/app -run
'LegacyDelimiter|MarkArtifacts|PreflightsEvery|ConfigOnlyRecovers' -count=1 -v`
passed.
**Notes**: New regressions prove delimiter-collision source changes reject the
confirmed update, failed committed-state publication still rolls back, divergent
multi-record recovery leaves artifacts and backups untouched, and config-only
dry-run/update paths respect pending-journal recovery.

### Session: 2026-08-11 (Transition Race And Committed-Recovery Remediation)

**Tasks Completed**: Closed the final self-review transition race by binding
eligible transitions to a descriptor-relative source-tree fingerprint and
revalidating that fingerprint after the directory is renamed to its transaction
backup. A mismatch restores the directory and aborts before replacement.
Committed recovery now accepts only the journal's expected symlink target, and
requires every retained backup to be a directory.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/template/generator/generator.go`,
`internal/app/update_symlink_transition.go`,
`internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_fingerprint_unix.go`,
`internal/app/update_symlink_transaction_other.go`,
`internal/app/update_symlink_transition_review_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: Focused Darwin/Linux coverage verifies a descendant created
after preview validation is restored rather than replaced, and committed
missing-destination, directory-destination, and non-directory-backup states
fail without filesystem or journal mutation. `mise exec -- go test
./internal/app -run
'RestoresDirectoryWhenSourceChanges|RejectsDivergentState|PreflightsEvery|LegacyDelimiter'
-count=1 -v` passed (four top-level tests and three committed-recovery
subtests). Fresh full-suite, build, vet, and Windows cross-build completion
remain blocked by silent runner stalls.

### Session: 2026-08-11 (Committed Backup Archival Remediation)

**Tasks Completed**: Replaced the post-commit recursive backup-deletion path
with a descriptor-relative rename into collision-resistant `.ign` archive
storage. The durable journal is cleared only after archival succeeds; committed
recovery archives an extant expected backup or safely clears a journal whose
original-parent backup is already absent. No post-rename descendant is removed
automatically.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_archive_unix.go`,
`internal/app/update_symlink_transaction_fingerprint_unix.go`,
`internal/app/update_symlink_transition_review_test.go`,
`internal/app/update_symlink_transition_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- gofmt -w` completed for the transaction,
archive, and focused test files. `mise exec -- go test ./internal/app -run
'ArchivesCommitted|ArchivesUnexpected|ArchivesRegular|ArchivesLeaf|DoesNotOverwriteConcurrentDestination|FinalizesCommittedBackup'
-count=1 -v` passed (six tests). `git diff --check` passed. Full-suite, build,
vet, and Windows gates remain pending renewed workflow verification.

### Session: 2026-08-11 (Archive Destination No-Replace Remediation)

**Tasks Completed**: Replaced the archive destination check-then-rename with
an atomic descriptor-relative no-replace rename (`renameat2` on Linux and
`renameatx_np` with `RENAME_EXCL` on Darwin). An occupied deterministic archive
destination now retains the collision entry, transaction backup, and journal.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/update_symlink_transaction_archive_unix.go`,
`internal/app/update_symlink_transaction_restore_linux.go`,
`internal/app/update_symlink_transaction_rename_darwin.go`,
`internal/app/update_symlink_transition_review_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- gofmt -w` completed. `mise exec -- go test
./internal/app -run
'CommittedTransitionArchiveCollision|ArchivesCommitted|ArchivesUnexpected|ArchivesRegular|ArchivesLeaf|DoesNotOverwriteConcurrentDestination|FinalizesCommittedBackup'
-count=1 -v` passed (seven tests). `git diff --check` passed.

### Session: 2026-08-11 (Replacement-Symlink No-Replace Archival Remediation)

**Tasks Completed**: Replaced the replacement-symlink validation-to-unlink
path used by rollback, replacement-journal persistence failure handling, and
uncommitted recovery. Each path now atomically moves the validated node into a
collision-resistant `.ign` archive with no replacement. The archived inode and
target must still match the validated symlink; a substitution remains archived,
and the transaction backup plus durable journal remain actionable for recovery.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: Fresh project-wide `mise exec -- go vet ./...` completion remains
unconfirmed after a silent 60-second runner timeout.
**Files Changed**: `internal/app/update_symlink_transaction_unix.go`,
`internal/app/update_symlink_transaction_fingerprint_unix.go`,
`internal/app/update_symlink_transaction_archive_unix.go`,
`internal/app/update_symlink_transition_review_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: `mise exec -- gofmt -w` completed and `gofmt -d` produced no
diff. Focused substitution/recovery coverage and the broad focused app suite
passed. `mise run test`, `mise exec -- go build -o /dev/null ./...`, and
`GOOS=windows GOARCH=amd64 mise exec -- go build -o /dev/null ./...` completed
with exit status 0. Scoped `git diff --check` passed. `mise exec -- go vet
./...` timed out silently after 60 seconds without diagnostics.

### Session: 2026-08-11 (Ordinary Rollback Archive Retention)

**Tasks Completed**: Moved ordinary generation rollback storage to a private
same-filesystem sibling of the output root and routed manifest and tracking
artifact restoration through the owning rollback instance. Atomic rollback
archives now remain after a successful restore or generated-node removal: a
writer can modify a moved inode after it passes validation, so automatic
recursive cleanup would delete post-validation user data. Archive pruning is
explicitly deferred.
**Tasks In Progress**: T8 renewed independent and adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/checkout.go`,
`internal/app/checkout_rollback_nodes.go`,
`internal/app/checkout_rollback_nodes_unix_test.go`, `internal/app/update.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: Focused ordinary rollback, manifest failure, and archive
retention regressions passed. `mise run test`, `mise exec -- go vet ./...`, the
Windows cross-build, formatting, and scoped diff hygiene passed. The final host
build wrapper timed out without diagnostics after the full suite compiled all
host packages.

### Session: 2026-08-11 (Tracking-Artifact Rollback And Platform Guard)

**Tasks Completed**: Captured the post-write identity and fingerprint of each
successfully persisted manifest or configuration artifact before a subsequent
write can fail. Generic rollback now restores those artifacts through the
atomic no-replace path instead of leaving partial tracking state. The checkout
artifact path uses the same snapshots rather than pathname removal. On targets
without atomic generic rollback support, non-dry-run generation now fails
before mutation. Private rollback storage resolves the output root to an
absolute path before selecting its sibling directory, preventing relative
output paths from retaining archives inside the generated project.
**Tasks In Progress**: T8 renewed independent/adversarial review.
**Blockers**: None.
**Files Changed**: `internal/app/checkout.go`,
`internal/app/checkout_rollback_nodes.go`,
`internal/app/checkout_rollback_nodes_unix.go`,
`internal/app/checkout_rollback_nodes_other.go`, `internal/app/update.go`,
`internal/app/update_artifact_rollback_test.go`,
`internal/app/checkout_rollback_nodes_other_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: Focused artifact rollback and atomic rollback race tests
passed. Direct `mise exec -- go test ./...`, host build, project-wide vet,
Windows cross-build and app-test compilation, and scoped diff hygiene passed.
`mise run test` reached the runner cap after reporting no failures; the direct
full suite completed successfully.
**Risks**: Successful ordinary rollback archives remain intentionally retained
until a separate safe pruning lifecycle is designed.

### Session: 2026-08-11 (Takeover Completion And Final Verification)

**Tasks Completed**: The automated review workflow run was cancelled while
verifying the last self-review remediation; this session took over and closed
it out. Confirmed the final two self-review findings are remediated in code:
tracking-artifact rollback now binds to the exact written bytes and mode
whenever the atomic rename committed (including a parent-directory sync error
after a successful rename), and the blanket non-Darwin/Linux pre-generation
guard is removed so ordinary generation proceeds on those platforms while
rollback primitives fail closed at rollback time and only eligible
directory-to-symlink transitions are rejected before mutation. Fixed a test
fixture defect that leaked `.ign-checkout-rollback-*` archives into
`internal/app/` on every test run by anchoring test rollback archives under
test-owned temporary directories, and removed the previously leaked archives.
Refreshed `docs/progress/selective-overwrite.md` to match the final platform
behavior and marked the feature Completed.
**Tasks In Progress**: None. T8 review closure recorded; all high and medium
findings remediated and re-verified.
**Blockers**: None.
**Files Changed**: `internal/app/checkout_rollback_nodes_unix_test.go`,
`internal/app/update_symlink_transition_test.go`,
`docs/progress/selective-overwrite.md`, and this active plan.
**Verification**: Project-wide `gofmt` clean; `mise exec -- go vet ./...`,
host `go build ./...`, `GOOS=windows GOARCH=amd64` cross-build and vet, full
`go test -count=1 ./...`, and `mise run test` all passed. Zero
`.ign-checkout-rollback-*` directories remain after the suite.
**Risks**: Successful ordinary rollback archives remain intentionally retained
until a separate safe pruning lifecycle is designed.
