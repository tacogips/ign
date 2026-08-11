# Issue 45 Partial-State Symlink Recovery Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/issue45-partial-state-symlink-recovery.md`
**Created**: 2026-08-11
**Last Updated**: 2026-08-11

## Feature Contract

- Workflow mode: `issue-resolution`
- Issue reference: `tacogips/ign#47`
- Feature id: `issue45-partial-state-symlink-recovery`
- Fanout feature id: `issue45-partial-state-symlink-recovery`
- Implementation plan path: `impl-plans/issue45-partial-state-symlink-recovery.md`
- Target areas: `internal/app`, `internal/template/generator`, `internal/cli`
- Codex agent references:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`

## Design Document Reference

**Source**: `design-docs/specs/issue45-partial-state-symlink-recovery.md`

### Summary

Recover projects left in issue-45 partial state when a stale directory is no
longer manifest-owned but is byte-equivalent to the rendered current template
tree exposed by the replacement symlink. If equivalence is not provable, stop
with a specific issue-45 partial-state diagnostic instead of allowing the
generic generator skip message to imply that overwrite flags were ignored.

### Scope

**Included**:

- App-layer detection for directory-to-symlink candidates whose manifest
  ownership proof fails because terminal entries are untracked.
- Rendered current-template tree equivalence proof that reuses generation
  semantics for processed file paths, rendered file bytes, binary raw bytes, and
  symlink target comparison.
- Preservation of current generator symlink target semantics:
  `model.TemplateFile.SymlinkTarget` is compared and written as-is unless a
  separate design intentionally changes target rendering.
- Recovery only through the existing symlink transition transaction and journal
  path.
- Validation-style diagnostics in dry-run and real update before mutation when
  recovery is not provable.
- Regression tests for successful recovery, failed recovery diagnostics,
  dry-run immutability, unsafe candidates, rollback, and misleading skip-output
  suppression.

**Excluded**:

- Broad recursive directory deletion.
- Any `--force` behavior that deletes unproven directory contents.
- Following symlinks while inspecting stale directories or expected template
  trees.
- Unrelated generator overwrite behavior changes.
- Issue `tacogips/ign#46` output-path config-root handling.

## Modules

### 1. Transition Blocker Capture

#### `internal/app/update_symlink_transition.go`

**Status**: COMPLETED

```go
type symlinkTransitionBlockerKind string

const (
	symlinkTransitionBlockerLostManifestOwnership symlinkTransitionBlockerKind = "lost_manifest_ownership"
	symlinkTransitionBlockerProtectedPath         symlinkTransitionBlockerKind = "protected_path"
	symlinkTransitionBlockerUnsafePath            symlinkTransitionBlockerKind = "unsafe_path"
	symlinkTransitionBlockerUnreadablePath        symlinkTransitionBlockerKind = "unreadable_path"
	symlinkTransitionBlockerUnsupportedNode       symlinkTransitionBlockerKind = "unsupported_node"
)

type symlinkTransitionBlockers struct {
	Kind  symlinkTransitionBlockerKind
	Paths []string
}
```

**Checklist**:

- [x] Extend directory-to-symlink classification to retain why
      `ownedDirectoryTree` failed.
- [x] Separate lost manifest ownership from protected, unsafe, unreadable, and
      unsupported cases.
- [x] Cap representative blocker paths for diagnostics while retaining enough
      internal detail for tests.
- [x] Keep `Lstat`-based traversal and output-root containment checks.

### 2. Rendered Template Equivalence

#### `internal/app/update_symlink_transition.go`

**Status**: COMPLETED

```go
type renderedTemplateTreeEntry struct {
	RelPath       string
	Kind          renderedTemplateEntryKind
	Content       []byte
	SymlinkTarget string
}

type renderedTemplateEntryKind string

const (
	renderedTemplateEntryFile    renderedTemplateEntryKind = "file"
	renderedTemplateEntrySymlink renderedTemplateEntryKind = "symlink"
)

func proveRenderedTemplateTreeEquivalent(
	ctx context.Context,
	outputDir string,
	staleRoot string,
	symlinkTarget string,
	template *model.Template,
	variables parser.Variables,
) (bool, []string, error)
```

**Checklist**:

- [x] Build expected entries from `model.Template.Files` below the replacement
      symlink target subtree, not from `<output-dir>/<target>`.
- [x] Process template file paths with the same filename processor used by
      generation.
- [x] Render text template content with the same variable processor used by
      generation.
- [x] Compare binary template files using raw bytes.
- [x] Compare symlink targets as `file.SymlinkTarget` as-is to preserve current
      generator semantics; do not apply filename processing to the target.
- [x] Require identical normalized terminal relative paths and reject empty
      expected or stale trees.
- [x] Reject unreadable nodes, unsupported file types, path escapes, and
      symlinked ancestors without following symlinks.

### 3. Recovery Classification And Plan Fingerprint

#### `internal/app/update_symlink_transition.go`
#### `internal/template/generator/generator.go`

**Status**: COMPLETED

```go
type SymlinkTransitionRecoveryReason string

const (
	SymlinkTransitionRecoveredByContentEquivalence SymlinkTransitionRecoveryReason = "content_equivalence"
)

type SymlinkTransition struct {
	Disposition         SymlinkTransitionDisposition
	RetiredManagedPaths []string
	SourceFingerprint   string
	Target              string
	RecoveryReason      SymlinkTransitionRecoveryReason
}
```

**Checklist**:

- [x] Mark equivalence-proven transitions eligible with recovery provenance.
- [x] Retire existing manifest paths at or below the transition root when
      present; do not synthesize missing manifest descendants.
- [x] Include enough stale-root and expected-template state in the execution
      plan fingerprint to reject preview-to-confirmation divergence.
- [x] Preserve existing manifest-owned transition behavior.
- [x] Preserve generator behavior when no app transition plan is supplied.

### 4. Unrecoverable Diagnostic Flow

#### `internal/app/update.go`
#### `internal/cli/update.go`
#### `internal/template/generator/generator.go`

**Status**: COMPLETED

```go
type UpdateSymlinkTransitionDiagnostic struct {
	Path          string
	Target        string
	BlockingPaths []string
	RecoverySteps []string
}

func (d UpdateSymlinkTransitionDiagnostic) Error() string
```

**Checklist**:

- [x] Return a validation-style diagnostic before generation when partial-state
      recovery is not provable.
- [x] Include transition root, symlink target, representative blocking paths,
      overwrite safety wording, and exact manual recovery steps.
- [x] Keep dry-run mutation-free and use the same diagnostic path as real
      update.
- [x] Prevent the generic `file exists, use --overwrite or --force to
      overwrite` skip output for classified unrecoverable transitions.
- [x] Ensure CLI output renders the diagnostic clearly without duplicating
      generator skip lines.

### 5. Transaction And Rollback Integration

#### `internal/app/update.go`
#### `internal/app/update_symlink_transaction_*.go`

**Status**: COMPLETED

```go
func prepareSymlinkTransitionTransactions(
	outputDir string,
	manifestPath string,
	transitions map[string]generator.SymlinkTransition,
) ([]symlinkTransitionTransaction, error)
```

**Checklist**:

- [x] Route recovered transitions through the existing transaction preparation,
      journal persistence, rollback snapshot capture, directory removal, symlink
      creation, and journal cleanup.
- [x] Avoid any ad hoc removal path for recovered stale directories.
- [x] Verify rollback restores stale directory contents and `.ign` artifacts
      after a failure following recovered transition preparation.
- [x] Keep dry-run out of all transaction mutation paths.

### 6. Regression Tests And Documentation

#### `internal/app/update_symlink_transition_test.go`
#### `internal/app/update_symlink_transition_review_test.go`
#### `internal/template/generator/generator_test.go`
#### `internal/cli/update_test.go`
#### `docs/progress/issue45-partial-state-symlink-recovery.md`

**Status**: COMPLETED

```go
func TestCompleteUpdateRecoversIssue45PartialStateByContentEquivalence(t *testing.T)
func TestCompleteUpdateRecoversIssue45PartialStateWithForce(t *testing.T)
func TestPrepareUpdateDryRunReportsRecoveredTransitionWithoutMutation(t *testing.T)
func TestCompleteUpdateDiagnosesDivergentIssue45PartialState(t *testing.T)
func TestCompleteUpdateDryRunDiagnosesPartialStateWithoutMutation(t *testing.T)
func TestCompleteUpdateDoesNotRecoverUnsafePartialStateCandidates(t *testing.T)
func TestCompleteUpdateRollbackRestoresRecoveredPartialStateDirectory(t *testing.T)
func TestUpdatePartialStateDiagnosticSuppressesGenericSymlinkSkip(t *testing.T)
```

**Checklist**:

- [x] Add fixtures where stale `.claude/` is byte-equivalent to rendered
      `.agents/` and is recovered to `.claude -> .agents` under
      `--overwrite-all`.
- [x] Repeat automatic recovery under `--force`.
- [x] Assert dry-run recovery preview does not mutate files or metadata.
- [x] Assert divergent untracked content produces the issue-45 partial-state
      diagnostic and preserves all content.
- [x] Assert unreadable, unsafe, and symlink-ancestor candidates do not use
      equivalence recovery.
- [x] Assert rollback restores the stale directory and prior `.ign` artifacts
      after a recovered transition failure.
- [x] Assert output does not include the misleading generic skip line for a
      classified unrecoverable partial-state transition.
- [x] Add progress tracking under `docs/progress/` during implementation.
- [x] Update README or command help only if final wording documents recovery
      behavior.

## Task Breakdown

| Task | Deliverable | File Paths | Status |
|------|-------------|------------|--------|
| T1 | Blocker-aware transition classification | `internal/app/update_symlink_transition.go` | COMPLETED |
| T2 | Rendered-template equivalence proof | `internal/app/update_symlink_transition.go` | COMPLETED |
| T3 | Recovery provenance and execution-plan fingerprinting | `internal/app/update_symlink_transition.go`, `internal/template/generator/generator.go` | COMPLETED |
| T4 | Unrecoverable partial-state diagnostic flow | `internal/app/update.go`, `internal/cli/update.go`, `internal/template/generator/generator.go` | COMPLETED |
| T5 | Existing transaction and rollback integration | `internal/app/update.go`, `internal/app/update_symlink_transaction_*.go` | COMPLETED |
| T6 | Regression tests and documentation refresh | `internal/app/*_test.go`, `internal/template/generator/*_test.go`, `internal/cli/*_test.go`, `docs/progress/issue45-partial-state-symlink-recovery.md` | COMPLETED |
| T7 | Full verification and gofmt cleanup | Go packages, docs | COMPLETED |

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| T2: Rendered-template equivalence proof | T1 blocker classification | COMPLETED |
| T3: Recovery provenance and fingerprinting | T2 equivalence proof | COMPLETED |
| T4: Diagnostic flow | T1 blocker classification | COMPLETED |
| T5: Transaction integration | T3 recovery classification | COMPLETED |
| T6: Regression tests | T1-T5 implementation surfaces | COMPLETED |
| T7: Verification | T1-T6 | COMPLETED |

## Parallelizable Tasks

- After T1, T2 equivalence tests and T4 diagnostic wording tests can be drafted
  in parallel.
- T5 rollback integration tests can proceed once T3 exposes recovery provenance.
- Documentation/progress tracking can proceed in parallel with final regression
  work after behavior names and diagnostics stabilize.

## Verification

Run focused checks first, then the full required suite:

```bash
gofmt -w internal/app internal/cli internal/template/generator
go test ./internal/app -run 'SymlinkTransition|PartialState|Issue45'
go test ./internal/template/generator -run 'Symlink'
go test ./internal/cli -run 'Update|PartialState|Symlink'
go test ./...
go build ./...
mise run test
git diff --check
```

## Completion Criteria

- [x] `ign update --dry-run --overwrite-all --yes --no-color` recovers or
      diagnoses issue-45 partial state without the misleading generic skip
      message.
- [x] Byte-equivalent stale directories recover automatically under
      `--overwrite-all` and `--force`.
- [x] Divergent, unsafe, unreadable, protected, and symlink-ancestor candidates
      are preserved and diagnosed without deleting content.
- [x] Dry-run diagnostics and recovery previews do not mutate project files,
      `.ign` metadata, journals, or rollback snapshots.
- [x] Recovered mutations use only the existing symlink transition transaction
      and rollback system.
- [x] Equivalence comparison uses generated file path and byte semantics and
      preserves as-is symlink target comparison.
- [x] New focused regression tests pass.
- [x] `gofmt`, `go test ./...`, `go build ./...`, `mise run test`, and
      `git diff --check` pass or any environment limitation is documented.
- [x] Progress tracking is added under `docs/progress/`.

## Addressed Review Feedback

- The plan carries forward rendered-template equivalence as the source of
  truth and explicitly rejects comparison against stale output-dir target
  contents.
- The plan explicitly preserves current symlink target semantics by comparing
  `file.SymlinkTarget` as-is rather than filename-processing the target.
- The plan requires mutation-free dry-run validation diagnostics.
- The plan requires all recovered replacements to use existing transaction and
  rollback paths.
- The plan requires suppression of the misleading generic generator skip output
  for app-classified unrecoverable partial-state transitions.

## Risks

- Reimplementing generation rendering in the app layer may drift from
  `generator.Generate`; prefer shared helpers or tightly focused tests where
  practical.
- Fingerprint coverage must reject stale-root or expected-template changes
  between preview and confirmed update.
- Diagnostic handling must avoid double-reporting both app and generator
  messages for the same transition.
- Permission-dependent unreadable-path tests may need platform-aware setup or
  targeted skips.

## Progress Log

### Session: 2026-08-11

**Tasks Completed**: Created implementation plan from accepted design and Step
3 review feedback.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Implementation must preserve current as-is symlink target semantics
unless a future design explicitly changes target rendering.

### Session: 2026-08-11 Step 6 Implementation

**Tasks Completed**: Implemented blocker-aware transition classification,
rendered-template equivalence recovery, unrecoverable partial-state diagnostics,
generator support helpers, regression tests, README documentation, and
`docs/progress/issue45-partial-state-symlink-recovery.md`.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Recovered replacements continue through the existing symlink
transition transaction and rollback system.
