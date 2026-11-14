# Fix `ign update <output-path>` Configuration Root Handling Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/update-output-path-config-root.md`
**Created**: 2026-08-11
**Last Updated**: 2026-08-11

## Feature Contract

- Workflow mode: `issue-resolution`
- Issue reference: `tacogips/ign#46`
- Feature id: `update-output-path-config-root`
- Fanout feature id: `update-output-path-config-root`
- Implementation plan path: `impl-plans/update-output-path-config-root.md`
- Target areas: `internal/app`, `internal/cli`
- Codex agent references:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`

## Design Document Reference

**Source**: `design-docs/specs/update-output-path-config-root.md`

### Summary

Make `ign update [output-path]` load every `.ign` tracking artifact from the
requested output project instead of from the caller current working directory.
The update workflow must root config, variables, manifest derivation, rollback
snapshots, symlink transition journals, and generation variable defaults under
`opts.OutputDir`.

### Scope

**Included**:

- Root `PrepareUpdate` config directory checks and artifact loads under
  `opts.OutputDir`.
- Root `CompleteUpdate` variable preparation, manifest derivation, rollback
  snapshots, and config-only artifact persistence under `opts.OutputDir`.
- Preserve CLI documented behavior for `ign update ./my-project`.
- Add app and CLI regressions that run update from a different working
  directory than the target project.
- Inspect `ign rewind`, `ign switch`, and `ign vars` for the same bare `.ign`
  pattern; only change update behavior unless a shared defect is directly
  proven and covered.

**Excluded**:

- Issue `tacogips/ign#47` partial-state directory-to-symlink recovery.
- Checkout output path semantics.
- Public CLI behavior unrelated to `ign update [output-path]`.
- New third-party dependencies.

## Modules

### 1. Update Project Path Rooting

#### `internal/app/update.go`

**Status**: COMPLETED

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
	IgnConfigPath string
	IgnVarPath    string
	IgnConfig     *model.IgnConfig
}
```

**Checklist**:

- [x] Build `configDir` as `filepath.Join(opts.OutputDir, model.IgnConfigDir)`.
- [x] Build `ignConfigPath` and `ignVarPath` from the output-rooted
      `configDir`.
- [x] Run the prior-checkout existence check against the output-rooted
      `configDir`.
- [x] Keep user-facing output path display behavior stable.
- [x] Keep `PrepareUpdateResult.IgnConfigPath` and `PrepareUpdateResult.IgnVarPath`
      as the authoritative artifact paths for later update phases.

### 2. Completion Path Rooting

#### `internal/app/update.go`

**Status**: COMPLETED

```go
type CompleteUpdateOptions struct {
	PrepareResult *PrepareUpdateResult
	NewVariables  map[string]interface{}
	OutputDir     string
	Overwrite     bool
	OverwriteMode generator.OverwriteMode
	DryRun        bool
	Verbose       bool
	ExecutionPlan *UpdateExecutionPlan
}

func CompleteUpdate(ctx context.Context, opts CompleteUpdateOptions) (*UpdateResult, error)
```

**Checklist**:

- [x] Pass the output-rooted config directory to
      `prepareVariablesForGeneration`.
- [x] Continue deriving the manifest path from `prep.IgnConfigPath`.
- [x] Ensure stale managed-file cleanup and symlink transition classification
      use the output-rooted manifest path.
- [x] Ensure symlink transition journals are rooted under `opts.OutputDir`.
- [x] Ensure rollback snapshots for `ign.json`, `ign-var.json`, and
      `ign-files.json` use output-rooted artifact paths.
- [x] Cover config-only ref-retarget artifact rollback under `opts.OutputDir`.

### 3. CLI Update Argument Flow

#### `internal/cli/update.go`

**Status**: COMPLETED

```go
func runUpdate(cmd *cobra.Command, args []string) error
```

**Checklist**:

- [x] Verify `runUpdate` passes the optional positional `output-path` through
      `app.UpdateOptions.OutputDir`.
- [x] Verify preview and confirmed mutation use the same output directory.
- [x] Preserve help and README-documented `ign update ./my-project` behavior.
- [x] Avoid rewriting the requested path to the caller current working
      directory before app preparation.

### 4. Regression Tests

#### `internal/app/update_test.go`
#### `internal/cli/update_test.go`

**Status**: COMPLETED

```go
func TestPrepareUpdateLoadsConfigFromOutputDir(t *testing.T)
func TestCompleteUpdateUsesOutputDirConfigRoot(t *testing.T)
func TestCompleteUpdateConfigOnlyRetargetUsesOutputDirRollback(t *testing.T)
func TestRunUpdatePassesOutputPathToPrepareAndComplete(t *testing.T)
```

**Checklist**:

- [x] Create target projects with `.ign/ign.json`, `.ign/ign-var.json`, and
      `.ign/ign-files.json` under a temporary output directory.
- [x] Change the process working directory to a different temporary directory
      before invoking update.
- [x] Assert app preparation loads config and vars from the requested output
      directory.
- [x] Assert dry-run classification uses the output-rooted manifest path for
      stale cleanup and symlink transitions.
- [x] Assert config-only ref-retarget rollback snapshots and restores
      output-rooted config artifacts.
- [x] Assert CLI preview and mutation calls receive the requested output path.

### 5. Related Command Inspection

#### `internal/app/rewind.go`
#### `internal/app/switch.go`
#### `internal/app/vars.go`

**Status**: COMPLETED

```go
type RewindOptions struct {
	OutputDir string
}
```

**Checklist**:

- [x] Inspect `ign rewind`, `ign switch`, and `ign vars` for bare `.ign`
      patterns related to an accepted output-path argument.
- [x] Document any confirmed related defect separately instead of expanding
      this fanout scope without tests.
- [x] Leave these commands unchanged if they have current-directory semantics
      or no relevant output-path contract.

### 6. Documentation And Progress Tracking

#### `README.md`
#### `docs/progress/update-output-path-config-root.md`

**Status**: COMPLETED

```go
type UpdateResult struct {
	HashChanged          bool
	RefChanged           bool
	RefOverrideRequested bool
}
```

**Checklist**:

- [x] Update README or command help only if documented behavior changes or the
      current output-path example needs clarification.
- [x] Add feature progress tracking under `docs/progress/`.
- [x] Record issue reference, implemented sub-features, remaining work, design
      decisions, and verification notes.

## Task Breakdown

| Task | Deliverable | File Paths | Status |
|------|-------------|------------|--------|
| T1 | Output-rooted update path construction | `internal/app/update.go` | COMPLETED |
| T2 | Output-rooted completion, rollback, and journal behavior | `internal/app/update.go`, `internal/app/update_symlink_transaction_unix.go` | COMPLETED |
| T3 | CLI output-path pass-through verification | `internal/cli/update.go`, `internal/cli/update_test.go` | COMPLETED |
| T4 | App regression tests for different working directory | `internal/app/update_test.go` | COMPLETED |
| T5 | Related command defect inspection | `internal/app/rewind.go`, `internal/app/switch.go`, `internal/app/vars.go` | COMPLETED |
| T6 | Documentation and progress tracking | `README.md`, `docs/progress/update-output-path-config-root.md` | COMPLETED |

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| T1: Rooted prepare paths | Accepted design doc | COMPLETED |
| T2: Rooted completion paths | T1 authoritative `PrepareUpdateResult` paths | COMPLETED |
| T3: CLI verification | T1 app behavior | COMPLETED |
| T4: App regression tests | T1, T2 | COMPLETED |
| T5: Related command inspection | Accepted issue scope | COMPLETED |
| T6: Documentation and progress tracking | Final implementation decisions | COMPLETED |

## Parallelizable Tasks

| Task | Can Run In Parallel With | Notes |
|------|--------------------------|-------|
| T1 | T5 | Different files except inspection only. |
| T3 | T4 | Test scaffolding can be drafted once expected app path contract is clear. |
| T6 | None | Wait for implementation outcome and verification results. |

## Verification

Run formatting before final build/test verification:

```bash
gofmt -w internal/app internal/cli
go test ./internal/app -run 'TestPrepareUpdate|TestCompleteUpdate'
go test ./internal/cli -run 'TestRunUpdate'
go test ./...
go build ./...
```

## Completion Criteria

- [x] `ign update <output-path>` loads `.ign/ign.json`, `.ign/ign-var.json`,
      and `.ign/ign-files.json` from `<output-path>/.ign/` when invoked from a
      different working directory.
- [x] `PrepareUpdate` prior-checkout validation checks the requested output
      project, not the caller current working directory.
- [x] `CompleteUpdate` uses output-rooted variable files, manifest paths,
      rollback snapshots, symlink transition journals, and generation variable
      defaults.
- [x] Dry-run preview and confirmed mutation use the same output-rooted tracking
      paths.
- [x] Config-only ref-retarget rollback is covered under an output-rooted
      project.
- [x] CLI-level update tests prove output-path pass-through.
- [x] Related commands are inspected and any out-of-scope findings are noted.
- [x] `gofmt`, focused tests, `go test ./...`, and `go build ./...` pass.
- [x] `docs/progress/update-output-path-config-root.md` records final status.

## Addressed Review Feedback

- Design review accepted issue `tacogips/ign#46` and feature
  `update-output-path-config-root` with no findings.
- Plan keeps issue `tacogips/ign#47` out of scope for this fanout item.
- Plan keeps `PrepareUpdateResult` artifact paths authoritative.
- Plan calls out execution-plan path normalization, config-only rollback, and
  unrelated working-tree changes as explicit implementation risks.

## Risks

- Mixing absolute and relative path forms may cause execution-plan validation to
  reject a valid confirmation preview.
- Config-only ref-retarget flow has less generator activity, so rollback tests
  must directly cover it.
- Symlink transition journal recovery already has strict root handling; update
  changes must pass `opts.OutputDir` consistently without changing safety
  guarantees.
- Existing unrelated working-tree changes must remain untouched:
  `divedra-workflows/design-and-implement-review-loop/workflow.json` and
  `design-docs/specs/issue45-partial-state-symlink-recovery.md`.

## Progress Log

### Session: 2026-08-11

**Tasks Completed**: Implementation plan created from accepted design.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Ready for feature implementation branch.

### Session: 2026-08-11 Step 6 Implementation

**Tasks Completed**: Implemented output-rooted `PrepareUpdate` and
`CompleteUpdate` path handling, added app and CLI regression tests, inspected
`vars`, `rewind`, and `switch`, updated README documentation, and added
`docs/progress/update-output-path-config-root.md`.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Verification commands are recorded in the Step 6 output.
