# Add `ign update --ref <ref>` Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/ign-update-ref.md`
**Created**: 2026-07-13
**Last Updated**: 2026-07-13

## Feature Contract

- Workflow ID: `design-and-implement-review-loop-feature-plan`
- Workflow mode: `issue-resolution`
- Issue reference: `workflowInput.issueBody FEATURE 2: 'ign update --ref <ref>' - retarget the tracked branch/tag`
- Feature id: `ign-update-ref`
- Fanout feature id: `ign-update-ref`
- Implementation plan path: `impl-plans/ign-update-ref.md`
- Target areas: `internal/cli`, `internal/app`
- Codex agent references:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`

## Design Document Reference

**Source**: `design-docs/specs/ign-update-ref.md`

### Summary

Add `ign update --ref <ref>` so project users can retarget the template branch,
tag, or commit stored in `.ign/ign.json` without invoking destructive
`ign switch` behavior. The requested ref must use the existing update flow,
compose with dry-run and overwrite protections, validate before provider fetch,
persist after successful non-dry-run completion, and preserve current behavior
when `--ref` is omitted.

### Scope

**Included**:

- Add `--ref` and `-r` parsing to `internal/cli/update.go`.
- Thread the requested ref through `app.UpdateOptions` to `PrepareUpdate`.
- Validate refs before network/provider fetch in CLI and defensively in app.
- Fetch the configured template URL/path at the requested ref.
- Persist `.ign/ign.json.template.ref` and `.ign/ign.json.hash`
  transactionally after successful non-dry-run completion.
- Persist identical-content ref retargets without regenerating project files.
- Add unit tests, integration tests, README command documentation, and feature
  progress documentation.

**Excluded**:

- The separate `ign vars` command.
- Destructive `ign switch` or `ign rewind` behavior changes.
- New third-party dependencies.
- Provider redesign beyond passing the requested ref into the existing fetch
  path.

## Modules

### 1. Shared Git Ref Validation

#### `internal/app/ref_validation.go`

**Status**: Completed

```go
func ValidateGitRef(ref string) error
```

**Checklist**:

- [x] Move neutral git ref validation into `internal/app` or another non-CLI
      package accessible to app callers.
- [x] Keep CLI update validation before `app.PrepareUpdate`.
- [x] Keep app validation before provider resolve/fetch.
- [x] Cover branches, tags, commit SHAs, empty refs, and malformed refs.

### 2. CLI Update Command

#### `internal/cli/update.go`

**Status**: Completed

```go
var updateRef string

func runUpdate(cmd *cobra.Command, args []string) error
func shouldRegenerate(hashChanged, force, overwrite bool) bool
func shouldCompleteUpdate(prep *app.PrepareUpdateResult, force bool, overwrite bool) bool
```

**Checklist**:

- [x] Register `--ref` with shorthand `-r`.
- [x] Reset `updateRef` in tests alongside existing update flag globals.
- [x] Validate non-empty `--ref` before update preparation.
- [x] Pass `TargetRef` through `app.UpdateOptions`.
- [x] Show the effective requested ref when `--ref` is supplied, including
      explicit `main`.
- [x] Avoid the unchanged-template early return when only the tracked ref
      changed.
- [x] Print an identical-content note when the requested ref resolves to the
      same template hash.

### 3. Update Preparation

#### `internal/app/update.go`

**Status**: Completed

```go
type UpdateOptions struct {
    OutputDir     string
    Overwrite     bool
    OverwriteMode generator.OverwriteMode
    DryRun        bool
    Verbose       bool
    GitHubToken   string
    TargetRef     string
}

type PrepareUpdateResult struct {
    Template             *model.Template
    IgnJson              *model.IgnJson
    ExistingVars         map[string]interface{}
    NewVars              []string
    RemovedVars          []string
    CurrentHash          string
    NewHash              string
    HashChanged          bool
    IgnConfigPath        string
    IgnVarPath           string
    IgnConfig            *model.IgnConfig
    PreviousRef          string
    RequestedRef         string
    EffectiveRef         string
    RefOverrideRequested bool
    RefChanged           bool
}
```

**Checklist**:

- [x] Add `TargetRef` to `UpdateOptions`.
- [x] Snapshot the stored ref before applying overrides.
- [x] Validate `TargetRef` before provider work.
- [x] Override only `Template.Ref`, preserving stored URL and path.
- [x] Fetch and compare hashes using the requested ref.
- [x] Populate ref metadata for CLI messaging and completion decisions.
- [x] Preserve no-`--ref` behavior.

### 4. Completion And Artifact Persistence

#### `internal/app/update.go`

**Status**: Completed

```go
type UpdateResult struct {
    HashChanged          bool
    NewVariables         []string
    RemovedVariables     []string
    FilesCreated         int
    FilesSkipped         int
    FilesOverwritten     int
    FilesDeleted         int
    Errors               []error
    Files                []string
    DeletedFiles         []string
    DryRunFiles          []DryRunFile
    Directories          []string
    RefChanged           bool
    RefOverrideRequested bool
}

func CompleteUpdate(ctx context.Context, opts CompleteUpdateOptions) (*UpdateResult, error)
func saveCompleteUpdateArtifacts(prep *PrepareUpdateResult, rawVars map[string]interface{}, genResult *generator.GenerateResult, removedManagedFiles *cleanupRemovedManagedFilesResult, rollback *checkoutGenerationRollback) error
func saveCompleteUpdateConfigOnlyArtifacts(prep *PrepareUpdateResult, rawVars map[string]interface{}) error
```

**Checklist**:

- [x] Persist `IgnConfig.Hash = NewHash` and
      `IgnConfig.Template.Ref = RequestedRef` after successful non-dry-run
      completion.
- [x] Keep generation and stale-file cleanup before `.ign` artifact writes.
- [x] Use rollback snapshots for `.ign/ign.json`, `.ign/ign-var.json`, and
      `.ign/ign-files.json`.
- [x] Add a config-only completion path for `RefChanged && !HashChanged` when
      no force or overwrite regeneration is requested.
- [x] Avoid project-file regeneration for identical-content ref retargets.
- [x] Leave all artifacts unchanged during dry runs.
- [x] Restore the previous ref and hash on artifact-save failure.

### 5. Tests

#### `internal/cli/update_test.go`
#### `internal/app/update_test.go`
#### `test/integration/update_ref_test.go`

**Status**: Completed

```go
func TestUpdateCmd_FlagRegistration(t *testing.T)
func TestUpdateCmd_FlagParsing(t *testing.T)
func TestPrepareUpdate_TargetRefOverridesStoredRef(t *testing.T)
func TestPrepareUpdate_InvalidTargetRefFailsBeforeFetch(t *testing.T)
func TestCompleteUpdate_IdenticalHashRefRetargetPersistsConfigOnly(t *testing.T)
func TestCompleteUpdate_DryRunRefRetargetDoesNotPersist(t *testing.T)
func TestCompleteUpdate_SaveFailureRollsBackPreviousRef(t *testing.T)
func TestUpdateRefRetargetsTemplateAndPersistsRef(t *testing.T)
func TestUpdateRefIdenticalContentPersistsRefWithoutRewritingFiles(t *testing.T)
func TestUpdateRefDryRunDoesNotPersistRef(t *testing.T)
func TestUpdateRefOverwriteKeepsOverwriteProtections(t *testing.T)
```

**Checklist**:

- [x] Cover long and short ref flag registration and parsing.
- [x] Assert invalid refs fail before app update execution.
- [x] Assert requested ref overrides stored ref while preserving URL/path.
- [x] Assert identical-hash ref retarget persists only configuration.
- [x] Assert dry-run leaves `.ign` artifacts unchanged.
- [x] Assert artifact save failures roll back previous ref and hash.
- [x] Assert overwrite protections still apply with `--ref --overwrite --yes`.

### 6. Documentation And Progress

#### `README.md`
#### `docs/progress/ign-update-ref.md`

**Status**: Completed

```markdown
ign update --ref v2.0.0
ign update --ref v2.0.0 --dry-run
ign update --ref v2.0.0 --overwrite --yes
```

**Checklist**:

- [x] Document `ign update --ref <ref>` and `-r` in README command examples
      and flags.
- [x] Document non-destructive retargeting behavior.
- [x] Document identical-content ref pin persistence on non-dry-run success.
- [x] Document that dry-run leaves the stored ref unchanged.
- [x] Add `docs/progress/ign-update-ref.md` with status, spec reference,
      implemented work, remaining work, design decisions, and notes.

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Shared git ref validation | `internal/app/ref_validation.go` | Completed | `internal/app/update_test.go` |
| CLI update command | `internal/cli/update.go` | Completed | `internal/cli/update_test.go` |
| Update preparation | `internal/app/update.go` | Completed | `internal/app/update_test.go` |
| Completion persistence | `internal/app/update.go` | Completed | `internal/app/update_test.go` |
| Integration behavior | `test/integration/update_ref_test.go` | Completed | `go test ./test/integration` |
| Documentation | `README.md`, `docs/progress/ign-update-ref.md` | Completed | review + `go test ./...` |

## Task Breakdown

### TASK-001: Centralize Git Ref Validation

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/app/ref_validation.go`, `internal/app/update_test.go`, `internal/cli/flags.go`
**Dependencies**: None

**Description**:
Create app-accessible git ref validation and keep CLI validation aligned with
the same rule before provider or network access.

**Completion Criteria**:

- [x] `app.ValidateGitRef` or equivalent rejects empty and malformed refs.
- [x] CLI can validate refs before invoking `app.PrepareUpdate`.
- [x] Tests cover valid branch, tag, SHA, and invalid inputs.

### TASK-002: Add CLI `--ref` Flag

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/cli/update.go`, `internal/cli/update_test.go`
**Dependencies**: TASK-001

**Description**:
Register `--ref`/`-r`, parse and validate it, pass it through
`app.UpdateOptions.TargetRef`, and adjust output plus early-return decisions.

**Completion Criteria**:

- [x] `ign update --ref <ref>` parses successfully.
- [x] `ign update -r <ref>` parses successfully.
- [x] Invalid refs fail before provider fetch.
- [x] Ref-only changes do not trigger the unchanged-template early return.

### TASK-003: Implement PrepareUpdate Ref Override

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/app/update.go`, `internal/app/update_test.go`
**Dependencies**: TASK-001

**Description**:
Extend update preparation to validate and apply `TargetRef` while retaining the
stored template URL and path.

**Completion Criteria**:

- [x] `UpdateOptions.TargetRef` exists.
- [x] `PrepareUpdateResult` exposes previous/requested/effective ref metadata.
- [x] Target ref is used for provider fetch and hash comparison.
- [x] No-`--ref` update behavior remains unchanged.

### TASK-004: Add Ref-Only Transactional Completion

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/app/update.go`, `internal/app/update_test.go`
**Dependencies**: TASK-003

**Description**:
Persist requested refs through rollback-safe update artifact writes, including
identical-content retargets that should not rewrite project files.

**Completion Criteria**:

- [x] Successful non-dry-run update writes requested ref and new hash.
- [x] Identical-hash ref retarget persists config without requiring overwrite
      or force.
- [x] Dry-run leaves `.ign` artifacts unchanged.
- [x] Save failure restores previous ref and hash.

### TASK-005: Preserve Existing Update Flag Compositions

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/cli/update.go`, `internal/app/update.go`, `internal/app/update_test.go`
**Dependencies**: TASK-002, TASK-004

**Description**:
Ensure `--ref` composes with `--dry-run`, `--overwrite`, `--overwrite-all`,
`--yes`, and `--force` without regressing current update behavior.

**Completion Criteria**:

- [x] `--dry-run --ref` previews against requested ref and writes nothing.
- [x] `--overwrite --yes --ref` keeps selective overwrite behavior and
      `.ign-overwrite-ignore` protections.
- [x] `--force --ref` regenerates and persists the requested ref.
- [x] Existing update tests without `--ref` continue to pass.

### TASK-006: Add Integration Coverage

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `test/integration/update_ref_test.go`, integration fixtures under `test/testdata`
**Dependencies**: TASK-004

**Description**:
Add end-to-end coverage for ref retargeting, identical-content pin updates,
dry-run non-persistence, and overwrite protections.

**Completion Criteria**:

- [x] Integration test verifies generated files and stored ref after retarget.
- [x] Integration test verifies identical-content retarget avoids project-file
      rewrites while persisting config.
- [x] Integration test verifies dry-run leaves config unchanged.
- [x] Integration test verifies overwrite protections still apply.

### TASK-007: Refresh User Documentation And Progress

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `README.md`, `docs/progress/ign-update-ref.md`
**Dependencies**: None

**Description**:
Document the new command behavior and maintain feature progress tracking.

**Completion Criteria**:

- [x] README documents `--ref` and `-r` on `ign update`.
- [x] README includes dry-run and overwrite examples.
- [x] Progress file records implemented/remaining work and design decisions.

### TASK-008: Run Final Verification

**Status**: Completed
**Parallelizable**: No
**Deliverables**: verification logs
**Dependencies**: TASK-005, TASK-006, TASK-007

**Description**:
Run formatting, static checks, build, and full tests after implementation.

**Completion Criteria**:

- [x] `gofmt -w <modified-go-files>` completed.
- [x] `go vet ./...` passes.
- [x] `go build ./...` passes.
- [x] `go test ./...` passes.

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| TASK-001 validation | None | COMPLETED |
| TASK-002 CLI flag | TASK-001 | COMPLETED |
| TASK-003 prepare override | TASK-001 | COMPLETED |
| TASK-004 persistence | TASK-003 | COMPLETED |
| TASK-005 flag compositions | TASK-002, TASK-004 | COMPLETED |
| TASK-006 integration tests | TASK-004 | COMPLETED |
| TASK-007 docs and progress | None | COMPLETED |
| TASK-008 final verification | TASK-005, TASK-006, TASK-007 | COMPLETED |

## Parallelizable Tasks

- `TASK-001`: shared validation.
- `TASK-007`: README and progress documentation.
- After `TASK-001`, `TASK-002` and `TASK-003` can proceed in parallel if
  coordinated to avoid conflicting edits in `internal/app/update.go`.
- After `TASK-004`, `TASK-006` can proceed while `TASK-005` finishes CLI/app
  composition polish, if integration fixtures do not depend on unresolved CLI
  output strings.

## Completion Criteria

- [x] `ign update --ref <ref>` and `ign update -r <ref>` are available.
- [x] Requested refs are validated before provider fetch in CLI and app paths.
- [x] Retargeting preserves configured template URL/path and changes only ref.
- [x] Existing update behavior is unchanged when `--ref` is omitted.
- [x] `--dry-run --ref` writes no `.ign` artifacts.
- [x] Successful non-dry-run retarget persists the requested ref even when the
      content hash is unchanged.
- [x] Rollback restores previous ref/hash on artifact-save failure.
- [x] Existing overwrite, overwrite-all, yes, and force protections still apply.
- [x] README and `docs/progress/ign-update-ref.md` are updated.
- [x] Required verification commands pass.

## Verification

Required commands:

```bash
gofmt -w <modified-go-files>
go vet ./...
go build ./...
go test ./...
```

Focused commands during implementation:

```bash
go test ./internal/cli ./internal/app
go test ./test/integration
```

## Addressed Feedback

- Accepted design review finding: the design's `-r` shorthand is treated as an
  accepted compatible extension because checkout and switch already use `-r`
  for refs.
- Scoped this plan to `ign-update-ref`; it intentionally excludes the separate
  `ign vars` feature.
- Called out rollback-safe identical-content retargeting because design review
  identified it as the primary implementation risk.
- Kept validation explicit in both CLI and app paths to avoid non-CLI caller
  drift.

## Risks

- A config-only identical-content retarget can regress rollback safety if it
  bypasses existing artifact snapshot helpers.
- Ref validation can drift if CLI and app validation are not centralized.
- Local integration fixtures may not fully model remote git ref switching, so
  tests should separately assert persisted config and provider fetch behavior.
- Existing dirty planning files in the workspace may be unrelated; implementation
  should avoid reverting or depending on unrelated changes.

## Progress Log

### Session: 2026-07-13

**Tasks Completed**: TASK-001 through TASK-008 implemented.
**Tasks In Progress**: Verification running in Step 6.
**Blockers**: None.
**Notes**: Added centralized validation, CLI flag handling, target-ref update
preparation, identical-content config-only persistence, tests, README
documentation, and progress tracking.

### Session: 2026-07-13 Step 7 Remediation

**Tasks Completed**: Strengthened TASK-006 integration coverage.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Added compiled CLI integration cases for changed-template generated
file updates, identical-content no-rewrite ref persistence, dry-run preview and
non-persistence, and selective overwrite protection with `.ign-overwrite-ignore`.

## Related Plans

- **Previous**: None.
- **Next**: None.
- **Depends On**: `design-docs/specs/ign-update-ref.md`.
- **Related Historical Plans**: `impl-plans/ign-update-ref-flag.md`,
  `impl-plans/active/update-ref.md`.
