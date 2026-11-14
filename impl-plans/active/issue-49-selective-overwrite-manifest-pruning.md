# Issue 49 Selective Overwrite Manifest Pruning Implementation Plan

**Status**: Implemented
**Design Reference**: `design-docs/specs/selective-overwrite.md#removed-managed-file-cleanup`
**Created**: 2026-08-12
**Last Updated**: 2026-08-12

---

## Design Document Reference

**Source**: `design-docs/specs/selective-overwrite.md`

### Summary

Issue [#49](https://github.com/tacogips/ign/issues/49) requires
`ign update --overwrite --yes` to remove absent paths matched by the remote
template's `.ign-overwrite-ignore` from the next `.ign/ign-files.json`, including
entries inherited from manifests written by older releases. The update must not
recreate ignored files, and it must preserve existing ignored stale paths and
their prior manifest entries.

### Scope

**Included**:

- Selective overwrite stale cleanup and manifest reconciliation for absent
  ignored paths in `internal/app/update_cleanup.go` and `internal/app/manifest.go`.
- Preview/write parity for stale cleanup decisions produced by
  `internal/app/update.go` and surfaced through `internal/cli/update.go`.
- Regression coverage in `internal/app/update_test.go`.
- Progress tracking under `docs/progress/`.

**Excluded**:

- Changes to `.ign-overwrite-ignore` matching semantics in
  `internal/template/generator/filter.go`.
- Changes to ignored-path creation filtering implemented for issue #48.
- Broad rewrites of issue #45 managed directory-to-symlink transitions.
- Changes to `--overwrite-all` and `--force`; they continue to bypass
  `.ign-overwrite-ignore`.
- Release packaging, version bumps, or publication.

### Reference Trace

- **Workflow mode**: issue-resolution.
- **Issue reference**: `tacogips/ign#49`.
- **Requested behavior**: remove ignored stale manifest paths when absent,
  without recreating ignored files.
- **Codex-agent references**:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`
- **Primary repository modules**:
  - `internal/app/update_cleanup.go`
  - `internal/app/manifest.go`
  - `internal/app/update.go`
  - `internal/app/update_test.go`
  - `internal/template/generator/generator.go`
  - `internal/template/generator/filter.go`
  - `internal/cli/update.go`

No intentional divergence from the accepted design is planned. Product adapter
boundaries are local to this repository: Codex-agent references guide Go
implementation and verification only, and no external implementation behavior
is copied.

---

## Modules

### 1. Cleanup Classification And Pruning

#### `internal/app/update_cleanup.go`

**Status**: COMPLETED

```go
type cleanupRemovedManagedFilesOptions struct {
	OutputDir       string
	ManifestPath    string
	GeneratedPaths  []string
	OverwriteMode   generator.OverwriteMode
	TransitionPlan  *UpdateExecutionPlan
}

type cleanupRemovedManagedFilesResult struct {
	FilesDeleted          int
	DeletedFiles          []string
	ExcludedManifestPaths map[string]struct{}
}
```

**Checklist**:

- [x] Distinguish stale manifest paths that exist on disk from stale manifest
      paths that are already absent.
- [x] Preserve existing stale paths matched by `.ign-overwrite-ignore` during
      selective `--overwrite`.
- [x] Prune absent stale paths from the next manifest even when matched by
      `.ign-overwrite-ignore`.
- [x] Keep directory manifest entries rejected instead of recursively removed.
- [x] Preserve existing symlink-transition exclusions and never follow symlink
      targets during cleanup.

### 2. Manifest Save Integration

#### `internal/app/manifest.go`

**Status**: COMPLETED

```go
func saveManifestFromGenerateResultWithResult(
	path string,
	result *generator.GenerateResult,
	excludedCanonicalPaths map[string]struct{},
) (config.AtomicWriteResult, error)

func isExcludedManifestPath(path string, excludedCanonicalPaths map[string]struct{}) bool
```

**Checklist**:

- [x] Ensure absent ignored stale paths classified for pruning are passed as
      excluded manifest paths.
- [x] Ensure existing ignored stale paths are not excluded when selective
      overwrite preserves them.
- [x] Keep current generated paths tracked when they are written or retained by
      the generator result.
- [x] Preserve absolute/canonical path handling used by rollback and manifest
      persistence.

### 3. Preview And CLI Parity

#### `internal/app/update.go`
#### `internal/cli/update.go`

**Status**: COMPLETED

```go
type UpdateResult struct {
	FilesDeleted int
	DeletedFiles []string
}

func printUpdateDryRunResult(result *app.UpdateResult)
func printUpdateResult(result *app.UpdateResult)
```

**Checklist**:

- [x] Confirm dry-run uses the same stale cleanup classification as real update.
- [x] Confirm confirmation preview reports the same `D` candidates that a real
      update will apply.
- [x] Report absent stale paths pruned from the manifest with `D`, matching the
      accepted design's removed-managed-file preview contract.
- [x] Keep `--yes` scoped to prompt suppression only.

### 4. Regression Tests

#### `internal/app/update_test.go`

**Status**: COMPLETED

```go
func TestCompleteUpdate_OverwritePrunesMissingIgnoredManagedFile(t *testing.T)
func TestCompleteUpdate_SelectiveOverwriteRespectsTemplateIgnore(t *testing.T)
func TestCompleteUpdate_OverwriteDeleteRespectsTemplateIgnore(t *testing.T)
```

**Checklist**:

- [x] Add a focused issue #49 fixture with a prior manifest containing an
      ignored stale path that is absent on disk.
- [x] Assert `ign update --overwrite --yes` does not recreate the ignored path.
- [x] Assert the next `.ign/ign-files.json` omits the absent ignored stale path.
- [x] Assert an existing ignored stale path is preserved and retained in the
      manifest.
- [x] Assert `--overwrite-all` stale cleanup behavior remains unchanged.
- [x] Assert dry-run and write-path cleanup classifications stay consistent.

### 5. Progress Documentation

#### `docs/progress/selective-overwrite.md`

**Status**: COMPLETED

**Checklist**:

- [x] Record issue #49 status, spec reference, implemented behavior, remaining
      work, design decisions, and verification notes.
- [x] Mention that issue #49 is a manifest migration/reconciliation fix, not a
      change to ignored-path creation rules.
- [x] Leave unrelated progress files untouched.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Cleanup classification and pruning | `internal/app/update_cleanup.go` | COMPLETED | `internal/app/update_test.go` |
| Manifest save integration | `internal/app/manifest.go` | COMPLETED | `internal/app/update_test.go` |
| Preview and CLI parity | `internal/app/update.go`, `internal/cli/update.go` | COMPLETED | `internal/app/update_test.go` |
| Regression tests | `internal/app/update_test.go` | COMPLETED | focused issue #49 tests |
| Progress documentation | `docs/progress/selective-overwrite.md` | COMPLETED | review only |

## Dependencies

| Task | Depends On | Status |
|------|------------|--------|
| TASK-001: cleanup classification | Accepted design review | COMPLETED |
| TASK-002: manifest save integration | TASK-001 | COMPLETED |
| TASK-003: preview and CLI parity | TASK-001 | COMPLETED |
| TASK-004: regression tests | TASK-001, TASK-002, TASK-003 | COMPLETED |
| TASK-005: progress documentation | Final implementation behavior | COMPLETED |
| TASK-006: verification and handoff | TASK-004, TASK-005 | COMPLETED |

## Task Breakdown

### TASK-001: Adjust Cleanup Classification

**Status**: Completed
**Parallelizable**: No
**Deliverables**: targeted update cleanup change in `internal/app/update_cleanup.go`
**Dependencies**: accepted Step 3 design review

Classify stale manifest entries so selective overwrite preserves existing
ignored stale paths, but marks absent ignored stale paths for manifest pruning.

### TASK-002: Wire Manifest Exclusions

**Status**: Completed
**Parallelizable**: No
**Deliverables**: manifest reconciliation change in `internal/app/manifest.go`
**Dependencies**: TASK-001

Pass the cleanup pruning set into manifest persistence without removing existing
ignored stale paths that selective overwrite preserves.

### TASK-003: Preserve Preview/Write Consistency

**Status**: Completed
**Parallelizable**: No
**Deliverables**: parity review or focused changes in `internal/app/update.go`
and `internal/cli/update.go`
**Dependencies**: TASK-001

Ensure dry-run, confirmation preview, `--yes`, and real update use one cleanup
decision path and surface consistent deletion/pruning output.

### TASK-004: Add Regression Coverage

**Status**: Completed
**Parallelizable**: No
**Deliverables**: issue #49 tests in `internal/app/update_test.go`
**Dependencies**: TASK-001, TASK-002, TASK-003

Cover absent ignored stale manifest pruning, existing ignored stale path
preservation, no recreation of ignored paths, and unchanged overwrite-all
cleanup behavior.

### TASK-005: Update Progress Tracking

**Status**: Completed
**Parallelizable**: Yes, after TASK-004
**Deliverables**: `docs/progress/selective-overwrite.md`
**Dependencies**: final implementation behavior

Record issue #49 implementation status and verification notes after behavior is
confirmed.

### TASK-006: Verification And Handoff

**Status**: Completed
**Parallelizable**: No
**Deliverables**: completed verification record for implementation review
**Dependencies**: TASK-004, TASK-005

Run focused and full Go verification commands, then update this plan's progress
log and completion checkboxes.

## Parallelization

| Task | Parallelizable | Reason |
|------|----------------|--------|
| TASK-001 | No | Defines cleanup semantics consumed by manifest and preview tasks. |
| TASK-002 | No | Depends on cleanup classification outputs. |
| TASK-003 | No | Must consume the same classification as TASK-001. |
| TASK-004 | No | Tests depend on final cleanup, manifest, and preview behavior. |
| TASK-005 | Yes, after TASK-004 | Documentation write scope is disjoint from Go code once behavior is final. |
| TASK-006 | No | Verification runs after implementation and docs are complete. |

## Verification

```bash
go test ./internal/app -run 'TestCompleteUpdate_OverwritePrunesMissingIgnoredManagedFile|TestCompleteUpdate_SelectiveOverwriteRespectsTemplateIgnore|TestCompleteUpdate_OverwriteDeleteRespectsTemplateIgnore'
go test ./internal/template/generator ./internal/app ./internal/cli
go test ./...
go build -o /dev/null ./...
go vet ./...
git diff --check
```

## Completion Criteria

- [x] `ign update --overwrite --yes` prunes absent ignored stale paths from the
      next `.ign/ign-files.json`.
- [x] The same update does not recreate ignored stale paths.
- [x] Existing ignored stale paths are preserved on disk and retained in the
      manifest.
- [x] `--overwrite-all` and `--force` cleanup behavior is unchanged.
- [x] Dry-run and confirmation preview use the same cleanup classification as
      real update.
- [x] Slash-normalized `.ign-overwrite-ignore` matching is preserved.
- [x] Issue #45 symlink transition cleanup exclusions are not broadened or made
      unsafe.
- [x] Focused issue #49 regression tests pass.
- [x] Full Go verification commands pass.
- [x] `docs/progress/selective-overwrite.md` is refreshed.

## Progress Log

### Session: 2026-08-12

**Tasks Completed**: TASK-001 through TASK-006.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Implemented issue #49 selective overwrite manifest reconciliation.
Absent stale paths now share the removed-managed-path classification used by
preview and write paths, while existing ignored stale paths remain preserved.

## Related Plans

- **Depends On**: `impl-plans/active/issue-48-ignored-descendant-creation.md`
  for existing ignored-path creation filtering behavior.
- **Related**: `impl-plans/update-overwrite-cleanup-review.md` for prior stale
  managed file cleanup boundaries.
