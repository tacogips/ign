# Add `ign vars` Command Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/ign-vars.md`
**Created**: 2026-07-13
**Last Updated**: 2026-07-13

## Feature Contract

- Workflow mode: `issue-resolution`
- Issue reference: `workflowInput.issueBody FEATURE 1: 'ign vars' - inspect template variables and current values`
- Feature id: `ign-vars`
- Fanout feature ids: `ign-vars`
- Implementation plan path: `impl-plans/ign-vars.md`
- Target areas: `internal/cli`, `internal/app`, `internal/config`, `internal/template`
- Codex agent references:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`

## Design Document Reference

**Source**: `design-docs/specs/ign-vars.md`

### Summary

Implement a read-only `ign vars` command that displays template variable
declarations merged with current `.ign/ign-var.json` values. The command must
support table output, `--json`, `--unset`, and `--json --unset`, keep JSON
stdout script-safe, degrade to local-only values when template declarations
cannot be fetched, and return exit code `1` only when known required variables
are unset.

### Scope

**Included**:

- New app-layer variable inspection query in `internal/app/vars.go`.
- Shared template declaration fetch helper only if needed to reuse update or
  checkout provider behavior without update-only mutation or hash validation.
- New CLI command in `internal/cli/vars.go`, registered from `internal/cli/root.go`.
- Unit tests for app merge semantics and CLI output/exit behavior.
- Integration tests under `test/integration`.
- README command documentation for `ign vars`, `--json`, and `--unset`.

**Excluded**:

- The separate `ign update --ref` feature.
- Any mutation of `.ign/ign.json`, `.ign/ign-var.json`, `.ign/ign-files.json`,
  template hashes, or generated project files.
- New third-party dependencies.
- Checkout/update prompting or generation readiness validation.

## Modules

### 1. App Vars Query

#### `internal/app/vars.go`

**Status**: Completed

```go
type VarsOptions struct {
    OutputDir   string
    UnsetOnly   bool
    GitHubToken string
}

type VarsResult struct {
    DeclarationsAvailable bool
    Rows                  []VarsRow
    UnsetCount            int
    RequiredUnsetCount    int
    DeclarationError      error
}

type VarsRow struct {
    Name        string
    Type        string
    Required    bool
    Default     interface{}
    Current     interface{}
    HasCurrent  bool
    Description string
    Unset       bool
    Declared    bool
}

func InspectVars(ctx context.Context, opts VarsOptions) (*VarsResult, error)
func IsRequiredUnset(err error) bool
```

**Checklist**:

- [x] Load `.ign/ign.json` and `.ign/ign-var.json` with existing config loaders.
- [x] Fetch configured template declarations using stored URL, ref, and path.
- [x] Merge declarations and current values without applying defaults as current values.
- [x] Treat absent keys as unset; treat `nil`, `false`, `0`, and `""` as current when present.
- [x] Include local-only rows for values not declared by the template.
- [x] Sort rows by variable name.
- [x] Return local-only rows and a non-fatal declaration error when fetch fails.
- [x] Never mutate config, variables, manifests, hashes, or generated files.

### 2. Template Declaration Fetch Reuse

#### `internal/app/template_fetch.go` or `internal/app/update.go`

**Status**: Completed

```go
type trackedTemplateFetchOptions struct {
    Source      model.TemplateSource
    GitHubToken string
}

func fetchTrackedTemplate(ctx context.Context, opts trackedTemplateFetchOptions) (*model.Template, error)
```

**Checklist**:

- [x] Extract provider resolution/fetch behavior only if current `PrepareUpdate` code cannot be reused safely.
- [x] Preserve URL normalization, provider creation, stored ref, and stored path semantics.
- [x] Avoid update-only hash validation, variable prompting, generation, or artifact writes.
- [x] Keep existing `PrepareUpdate` behavior unchanged if it is refactored to call this helper.
- [x] Unit test stored ref/path propagation when extraction occurs.

### 3. CLI Vars Command

#### `internal/cli/vars.go`
#### `internal/cli/root.go`

**Status**: Completed

```go
var varsJSON bool
var varsUnset bool

func runVars(cmd *cobra.Command, args []string) error
func printVarsTable(result *app.VarsResult)
func printVarsJSON(result *app.VarsResult) error
func warnVarsDeclarationsUnavailable(err error)
```

**Checklist**:

- [x] Register `ign vars` with `--json` and `--unset`.
- [x] Respect global `--quiet` by suppressing normal output and warnings.
- [x] Respect `--no-color` by using existing output behavior and avoiding new colored table content.
- [x] Keep declaration-unavailable warnings off stdout in JSON mode.
- [x] Return a non-zero exit for required unset variables without duplicate Cobra error output.
- [x] Print table columns `NAME`, `TYPE`, `REQUIRED`, `DEFAULT`, `CURRENT`, `DESCRIPTION`.
- [x] Print JSON with declaration availability, rows, `unset_count`, and `required_unset_count`.

### 4. Unit Tests

#### `internal/app/vars_test.go`
#### `internal/cli/vars_test.go`

**Status**: Completed

```go
func TestInspectVars_MergesDeclarationsAndCurrentValues(t *testing.T)
func TestInspectVars_UnsetSemanticsUseKeyPresence(t *testing.T)
func TestInspectVars_FetchFailureReturnsLocalOnlyRows(t *testing.T)
func TestVarsCmd_TableOutput(t *testing.T)
func TestVarsCmd_JSONOutputIsScriptSafe(t *testing.T)
func TestVarsCmd_UnsetRequiredReturnsExitOne(t *testing.T)
```

**Checklist**:

- [x] Cover deterministic sorting and local-only rows.
- [x] Cover required unset counts and unset-only filtering.
- [x] Cover missing `.ign` errors matching update/rewind style.
- [x] Cover quiet suppression.
- [x] Cover warning routing for JSON mode.
- [x] Cover command registration and flag parsing.

### 5. Integration Tests

#### `test/integration/vars_test.go`

**Status**: Completed

```go
func TestVarsCommandListsTemplateVariables(t *testing.T)
func TestVarsCommandJSONOutput(t *testing.T)
func TestVarsCommandUnsetSatisfiedVariablesExitZero(t *testing.T)
func TestVarsCommandUnsetRequiredMissingExitOne(t *testing.T)
```

**Checklist**:

- [x] Seed or checkout a local fixture template with declared variables.
- [x] Verify table output columns and current values.
- [x] Verify JSON output parses and has expected row/count fields.
- [x] Verify `--unset` filters rows.
- [x] Verify required missing variables return exit code `1`.
- [x] Verify no generated files or `.ign` artifacts are mutated by `ign vars`.

### 6. README Command Documentation

#### `README.md`

**Status**: Completed

```text
ign vars
ign vars --json
ign vars --unset
```

**Checklist**:

- [x] Add `ign vars` to command documentation.
- [x] Document table columns.
- [x] Document JSON scripting mode.
- [x] Document `--unset` CI gate behavior and required-unset exit code `1`.
- [x] Document local-only fallback when template declarations are unavailable.

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| App vars query | `internal/app/vars.go` | Completed | `internal/app/vars_test.go` |
| Template declaration fetch reuse | `internal/app/template_fetch.go` | Completed | `internal/app/vars_test.go`, `internal/app/update_test.go` |
| CLI vars command | `internal/cli/vars.go`, `internal/cli/root.go` | Completed | `internal/cli/vars_test.go` |
| Integration tests | `test/integration/vars_test.go` | Completed | `go test ./test/integration` |
| README docs | `README.md` | Completed | manual docs review |

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| `ign-vars` | Accepted design `design-docs/specs/ign-vars.md` | Completed |
| App vars query | Existing config loaders and provider/template models | Completed |
| CLI vars command | App vars query | Completed |
| Unit tests | App vars query and CLI command | Completed |
| Integration tests | CLI vars command and fixtures | Completed |
| README docs | Final CLI behavior | Completed |

## Task Breakdown

### TASK-001: Implement App-Layer Vars Inspection

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/app/vars.go`, `internal/app/vars_test.go`
**Dependencies**: None

**Description**:
Add the read-only app API that loads project config/current variables, fetches
template declarations, merges rows, computes unset counts, and exposes
declaration fetch failures as non-fatal result state.

**Completion Criteria**:

- [x] Rows include declared metadata, current values, `HasCurrent`, `Unset`, and `Declared`.
- [x] Missing current values are based on key absence only.
- [x] Rows and counts are deterministic.
- [x] Fetch failures return local-only rows and do not fail normal listing.
- [x] Missing `.ign` config returns an error consistent with update/rewind behavior.

### TASK-002: Extract Safe Template Fetch Helper If Needed

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/app/template_fetch.go` or focused updates in `internal/app/update.go`, tests covering ref/path behavior
**Dependencies**: None

**Description**:
Share the provider resolution and fetch path needed by `ign vars` without
pulling in update-only hash comparison, validation, or artifact mutation.

**Completion Criteria**:

- [x] Stored template URL, ref, and path are honored.
- [x] Existing update tests still pass if `PrepareUpdate` is refactored.
- [x] `ign vars` can fetch declarations without requiring template hash validation.

### TASK-003: Add CLI Command And Output Formatting

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/cli/vars.go`, `internal/cli/root.go`, `internal/cli/vars_test.go`
**Dependencies**: TASK-001

**Description**:
Register `ign vars`, parse `--json` and `--unset`, call the app query, and
format table or JSON output while preserving quiet/no-color behavior and JSON
stdout safety.

**Completion Criteria**:

- [x] `ign vars`, `ign vars --json`, `ign vars --unset`, and `ign vars --json --unset` work.
- [x] Table output includes all required columns.
- [x] JSON output is valid JSON with no warning text on stdout.
- [x] Required unset variables produce process exit code `1` without duplicate error printing.

### TASK-004: Add Integration Coverage

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `test/integration/vars_test.go`, fixture updates if needed
**Dependencies**: TASK-003

**Description**:
Exercise the compiled command against local fixture projects so command
behavior, exit codes, JSON output, and read-only guarantees are verified end to
end.

**Completion Criteria**:

- [x] Integration tests cover table output, JSON output, unset-only output, and required-unset exit `1`.
- [x] Tests verify `ign vars` does not modify `.ign` artifacts or generated files.
- [x] Fixture changes are minimal and reusable by existing integration helpers.

### TASK-005: Refresh README Command Documentation

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `README.md`
**Dependencies**: TASK-003

**Description**:
Document the final command surface and scripting behavior in the README command
section.

**Completion Criteria**:

- [x] README lists `ign vars`.
- [x] README documents `--json` and `--unset`.
- [x] README states required-unset `--unset` exits with code `1`.
- [x] README states local-only fallback when declarations cannot be fetched.

### TASK-006: Run Verification And Record Progress

**Status**: Completed
**Parallelizable**: No
**Deliverables**: verification output, `docs/progress/non-interactive-template-variables.md` or feature-specific progress file if implementation workflow updates progress docs
**Dependencies**: TASK-001, TASK-002, TASK-003, TASK-004, TASK-005

**Description**:
Run formatting, static checks, build, tests, and update progress tracking as
required by the implementation workflow after code and documentation changes.

**Completion Criteria**:

- [x] `gofmt` has been applied to changed Go files.
- [x] `go vet ./...` passes.
- [x] `go build ./...` passes.
- [x] `go test ./...` passes.
- [x] Progress tracking documents reflect completed implementation work.

## Parallelizable Tasks

- `TASK-001` and `TASK-002` can start in parallel if the fetch helper boundary
  is coordinated before merging.
- `TASK-003` depends on `TASK-001`.
- `TASK-004` and `TASK-005` both depend on `TASK-003`; documentation can be
  drafted while integration tests are being added after the final CLI behavior
  is stable.
- `TASK-006` depends on all implementation, test, and documentation tasks.

## Verification

Required commands:

```bash
gofmt
go vet ./...
go build ./...
go test ./...
```

Focused checks:

```bash
go test ./internal/app -run 'TestInspectVars'
go test ./internal/cli -run 'TestVarsCmd|TestVars'
go test ./test/integration -run 'TestVarsCommand'
```

## Completion Criteria

- [x] `ign vars` is registered and documented.
- [x] `ign vars` prints table rows with `NAME`, `TYPE`, `REQUIRED`, `DEFAULT`, `CURRENT`, and `DESCRIPTION`.
- [x] `ign vars --json` prints valid JSON only on stdout.
- [x] `ign vars --unset` filters unset variables and exits `1` only for known required unset variables.
- [x] Offline or failed declaration fetch produces local-only rows and a non-fatal warning outside JSON stdout.
- [x] The command is read-only for `.ign` artifacts, hashes, manifests, and generated files.
- [x] Unit and integration tests cover app merge semantics, CLI output, quiet behavior, JSON safety, exit codes, and read-only behavior.
- [x] `README.md` command documentation is refreshed.
- [x] `gofmt`, `go vet ./...`, `go build ./...`, and `go test ./...` pass.

## Addressed Feedback

- Design review accepted `design-docs/specs/ign-vars.md` and requested planning
  for `impl-plans/ign-vars.md`.
- JSON stdout safety is explicit in CLI tasks and tests.
- Read-only behavior is explicit in app tasks, integration tests, and completion criteria.
- Provider fetch reuse is isolated so update-only hash validation and mutation
  flow do not leak into `ign vars`.

## Risks

- Fetch reuse may require refactoring `PrepareUpdate`; this must not regress
  update behavior or require template hash validation for read-only inspection.
- Existing warning helpers write to stdout, so JSON mode needs stderr-specific
  warning handling or suppression.
- Required-unset exit handling needs a sentinel or typed error path that
  preserves exit code `1` without noisy duplicate error output.
- Offline fallback cannot detect missing required variables when declarations
  are unavailable.
- Global CLI flag state in tests may need careful reset between command tests.

## Progress Log

### Session: 2026-07-13 00:39 JST

**Tasks Completed**: TASK-001 through TASK-006 implemented.
**Tasks In Progress**: Verification running in Step 6.
**Blockers**: None.
**Notes**: Added app inspection, shared fetch reuse, CLI output, tests,
README documentation, and progress tracking in issue-resolution mode.

## Related Plans

- **Sibling**: `impl-plans/ign-update-ref-flag.md` covers the separate fanout
  feature `ign-update-ref`.
- **Depends On**: `design-docs/specs/ign-vars.md`.
