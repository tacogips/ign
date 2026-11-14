# Update Overwrite Cleanup Review Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/selective-overwrite.md#removed-managed-file-cleanup`
**Created**: 2026-05-31
**Last Updated**: 2026-05-31

---

## Design Document Reference

**Source**: `design-docs/specs/selective-overwrite.md`

### Summary

Review `v0.1.16..HEAD`, with focus on commit `1f8bba8`, for concrete defects in
stale managed file cleanup during `ign update` overwrite flows. Implement only
targeted fixes found by the review, refresh documentation only if behavior or
user-facing commands change, and leave runtime behavior unchanged when no defect
is identified.

### Scope

**Included**:

- Stale managed file cleanup driven by `.ign/ign-files.json`.
- `--overwrite`, `--overwrite-all`, `--force`, no-overwrite, dry-run, and
  confirmation preview cleanup behavior.
- Manifest pruning after real update and no mutation during preview.
- Release packaging scripts only when a concrete packaging defect is found in
  the `v0.1.16..HEAD` review.
- Go verification and progress-log updates.

**Excluded**:

- Broad refactors outside update overwrite cleanup.
- Changes to workflow runtime behavior.
- User-facing documentation churn when no behavior or packaging change is made.

### Codex-Agent References

- `.agents/agents/go-coding.md`: use only if Go implementation changes are
  required.
- `.agents/agents/go-check-and-test-after-modify.md`: use after any Go file
  modification or when Go checks are requested.
- `.agents/agents/go-code-review-file.md`: use as focused review guidance for
  the cleanup diff.

### Upstream Feedback Trace

- Step 1 intake established `v0.1.16..HEAD` as the review range, commit
  `1f8bba8` as the focus commit, and required concrete fixes only when review
  finds a defect.
- Step 2 design update made `design-docs/specs/selective-overwrite.md` the
  source of truth for stale managed file cleanup boundaries.
- Step 2 self-review accepted the design with residual implementation risks
  around manifest path classification, preview parity, and release packaging.
- Step 3 design review accepted the design with no high or mid findings and no
  additional feedback to revise.
- Codex-agent references are workflow guidance only. No Cursor adapter behavior
  or external implementation reference is being copied into this plan.

---

## Modules and Review Targets

### 1. Cleanup Classification

**Files**:

- `internal/app/update_cleanup.go`
- `internal/app/update.go`
- `internal/app/manifest.go`

**Status**: COMPLETED

```go
type cleanupRemovedManagedFilesOptions struct
type cleanupRemovedManagedFilesResult struct
func cleanupRemovedManagedFilesForUpdate(ctx context.Context, opts cleanupRemovedManagedFilesOptions) (*cleanupRemovedManagedFilesResult, error)
func shouldRemoveManagedPathDuringUpdate(path string, mode generator.OverwriteMode, overwriteIgnorePatterns []string) bool
func saveManifestFromGenerateResultExcluding(path string, result *generator.GenerateResult, excludedCanonicalPaths map[string]struct{}) error
```

**Checklist**:

- [x] Confirm cleanup compares prior manifest entries against the current
      rendered file set before manifest pruning.
- [x] Confirm cleanup candidates are limited to manifest-recorded files.
- [x] Confirm no-overwrite updates do not delete stale managed files.
- [x] Confirm `--overwrite-all` bypasses `.ign-overwrite-ignore`.
- [x] Confirm selective `--overwrite` preserves stale paths matched by the
      remote template `.ign-overwrite-ignore`.
- [x] Confirm directory manifest entries are refused, not recursively removed.
- [x] Confirm already-missing stale managed files are pruned from the manifest,
      including paths matched by remote `.ign-overwrite-ignore`.

### 2. Preview and CLI Reporting Parity

**Files**:

- `internal/app/update.go`
- `internal/cli/update.go`
- `internal/app/update_test.go`
- `internal/cli/update_test.go`

**Status**: COMPLETED

```go
type UpdateResult struct
type DryRunFile struct
func printUpdateDryRunResult(result *app.UpdateResult)
func printUpdateResult(result *app.UpdateResult)
```

**Checklist**:

- [x] Confirm dry-run reports the same delete candidates as real update.
- [x] Confirm confirmation preview reports `D` entries before mutation.
- [x] Confirm preview paths are relative to the requested output path.
- [x] Confirm preview does not mutate files or `.ign/ign-files.json`.
- [x] Add focused regression tests only for uncovered defects.

### 3. Release Packaging Follow-Up Gate

**Files**:

- `Taskfile.yml`
- `scripts/build-homebrew-release.sh`
- `scripts/render-homebrew-formula.sh`
- `packaging/homebrew/README.md`
- `internal/build/VERSION`

**Status**: COMPLETED

**Checklist**:

- [x] Review release packaging changes in `v0.1.16..HEAD`.
- [x] Fix only concrete release artifact or formula-generation defects.
- [x] Keep packaging changes independent from update cleanup unless a direct
      dependency is found.
- [x] Document packaging changes only when user-facing commands or artifacts
      change.

### 4. Documentation, Progress, Verification, and Commit Handoff

**Files**:

- `README.md`
- `docs/progress/selective-overwrite.md`
- `design-docs/specs/selective-overwrite.md`
- Go files changed by TASK-001 or TASK-002, if any

**Status**: COMPLETED

**Checklist**:

- [x] Update docs only for implemented behavior or packaging changes.
- [x] Update `docs/progress/selective-overwrite.md` with review result and any
      completed fix.
- [x] Run Go verification commands explicitly.
- [x] Prepare commit/push handoff only when an improvement is made.
- [x] If no improvement is needed, leave code and docs unchanged except
      workflow-owned planning artifacts and report the no-change review result.

---

## Task Breakdown

### TASK-001: Review Cleanup Classification Against Design

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: review findings for `internal/app/update_cleanup.go`,
`internal/app/update.go`, and `internal/app/manifest.go`
**Dependencies**: None

**Description**:
Inspect cleanup classification and manifest handling against the accepted
design boundaries.

**Completion Criteria**:

- [x] Findings classify whether implementation changes are required.
- [x] Any defect has exact file path, behavior impact, and expected fix scope.
- [x] No code is changed when review finds no defect.

### TASK-002: Review Preview and CLI Reporting Parity

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: review findings for update dry-run, confirmation preview, and
CLI result reporting
**Dependencies**: None

**Description**:
Verify that dry-run and confirmation preview expose the same stale managed file
delete decisions as real update without mutating files or manifest state.

**Completion Criteria**:

- [x] Preview parity is confirmed or a focused defect is filed for implementation.
- [x] Path reporting is checked for requested output path behavior.
- [x] Regression-test targets are identified only for real gaps.

### TASK-003: Review Release Packaging Follow-Up

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: review findings for release packaging files in `v0.1.16..HEAD`
**Dependencies**: None

**Description**:
Inspect release packaging changes separately from cleanup behavior and only
route concrete packaging defects into implementation.

**Completion Criteria**:

- [x] Packaging review outcome is explicit.
- [x] Cleanup and packaging scopes remain independent unless a direct defect
      requires crossing them.
- [x] No packaging files are changed without a concrete defect.

### TASK-004: Implement Targeted Fixes, If Findings Require Them

**Status**: Completed
**Parallelizable**: No
**Deliverables**: targeted Go, script, docs, or progress updates only as needed
**Dependencies**: TASK-001, TASK-002, TASK-003

**Description**:
Use `.agents/agents/go-coding.md` for Go implementation if TASK-001 or
TASK-002 finds a concrete Go defect. Keep changes minimal and preserve user
files by design.

**Completion Criteria**:

- [x] Each code change maps to an accepted finding.
- [x] Regression tests cover every fixed cleanup or preview defect.
- [x] Documentation changes are limited to behavior or packaging changes.
- [x] No unrelated refactors are included.

### TASK-005: Verify and Record Outcome

**Status**: Completed
**Parallelizable**: No
**Deliverables**: verification results, progress-log update, commit/push handoff
or no-change outcome
**Dependencies**: TASK-004

**Description**:
Run required Go verification and record whether the review produced a committed
improvement or a no-change result.

**Completion Criteria**:

- [x] `go test ./...` completes and result is recorded.
- [x] `go build ./cmd/ign` completes and result is recorded.
- [x] `go vet ./...` completes and result is recorded.
- [x] `docs/progress/selective-overwrite.md` captures implemented improvements
      when any code or behavior changed.
- [x] Commit/push handoff is ready only when an improvement was made.

---

## Implementation Outcome

- Found and fixed a cleanup defect where an already-missing stale managed file
  matched by remote `.ign-overwrite-ignore` remained in `.ign/ign-files.json`.
- Preserved existing selective overwrite behavior for ignored stale files that
  still exist on disk.
- Added regression coverage for manifest pruning without reporting a deletion.
- Reviewed Homebrew packaging changes in `v0.1.16..HEAD`; no concrete packaging
  defect required implementation.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Cleanup classification | `internal/app/update_cleanup.go` | COMPLETED | `go test ./internal/app` |
| Update manifest integration | `internal/app/update.go`, `internal/app/manifest.go` | COMPLETED | `go test ./internal/app` |
| Preview and CLI reporting | `internal/app/update.go`, `internal/cli/update.go` | COMPLETED | `go test ./internal/app ./internal/cli` |
| Release packaging gate | `Taskfile.yml`, `scripts/*.sh`, `packaging/homebrew/README.md` | COMPLETED | reviewed; no script change required |
| Progress and docs | `docs/progress/selective-overwrite.md`, `README.md` | COMPLETED | progress doc updated for fix |

## Dependencies

| Task | Depends On | Status |
|------|------------|--------|
| TASK-001 | None | COMPLETED |
| TASK-002 | None | COMPLETED |
| TASK-003 | None | COMPLETED |
| TASK-004 | TASK-001, TASK-002, TASK-003 | COMPLETED |
| TASK-005 | TASK-004 | COMPLETED |

## Parallelization Notes

- TASK-001, TASK-002, and TASK-003 can run in parallel because their initial
  review scopes are disjoint and produce findings only.
- TASK-004 must wait for review findings so implementation remains targeted.
- TASK-005 must run after implementation or after the explicit no-change
  decision.

## Verification Plan

- `go test ./...`
- `go build ./cmd/ign`
- `go vet ./...`
- If scripts are changed, run the smallest available packaging render/build
  command that exercises the changed path without publishing artifacts.

## Progress-Log Expectations

- Keep `impl-plans/PROGRESS.json` pointing at this plan with status `Ready`
  until implementation starts, then update the status to match actual progress.
- Update `docs/progress/selective-overwrite.md` only when a concrete fix or
  behavior-affecting documentation change is made.
- If the later review finds no improvement is needed, leave product code and
  user-facing docs unchanged and report the no-change outcome in workflow
  output instead of creating documentation churn.
- Record verification command results in the implementation step output, and
  in `docs/progress/selective-overwrite.md` only when that file is otherwise
  updated for an implemented fix.

## Completion Criteria

- [x] Full `v0.1.16..HEAD` review completed with focus on commit `1f8bba8`.
- [x] Cleanup, preview, and packaging findings are explicit.
- [x] Any implemented fix has targeted regression coverage.
- [x] User-created files remain outside cleanup candidates.
- [x] Documentation is refreshed only when behavior or packaging changes.
- [x] Required Go verification commands pass or failures are reported with
      concrete diagnostics.
- [x] Commit/push handoff is performed only when implementation changes are
      made; otherwise the no-change review outcome is reported.

## Progress Log

### Session: 2026-05-31

**Tasks Completed**: Plan creation for review and improvement of update
overwrite cleanup follow-up.

**Tasks In Progress**: None.

**Blockers**: None.

**Notes**: Step 3 accepted `design-docs/specs/selective-overwrite.md` as source
of truth. The later implementation step should preserve the existing completed
selective-overwrite plan history and use this plan for the focused follow-up.

### Session: 2026-05-31 Step 6

**Tasks Completed**: TASK-001, TASK-002, TASK-003, TASK-004, TASK-005.

**Tasks In Progress**: None.

**Blockers**: None.

**Notes**: Found and fixed one cleanup defect from focus commit `1f8bba8`.
Selective overwrite now prunes already-missing stale managed paths from
`.ign/ign-files.json` even when remote `.ign-overwrite-ignore` would preserve an
existing file at that path. Added focused regression coverage and reviewed
release packaging changes without finding a packaging defect requiring code
changes.
