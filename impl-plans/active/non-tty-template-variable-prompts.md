# Non-TTY Template Variable Prompt Remediation Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/command.md#redirected-stdin-prompt-failures`, `design-docs/specs/command.md#non-interactive-template-variables`
**Created**: 2026-08-04
**Last Updated**: 2026-08-04

---

## Design Document Reference

**Source**: `design-docs/specs/command.md`

### Summary

Implement the accepted issue-resolution design for GitHub issue
[#41](https://github.com/tacogips/ign/issues/41): unresolved template-variable
prompts in `checkout`, `init`, and `switch` must not call `survey/v2` when
stdin is redirected or otherwise non-interactive. Fully supplied `--var`/`-V`
runs must remain valid in non-interactive execution.

### Scope

**Included**:

- Shared prompt-layer preflight before any variable prompt is constructed or
  executed.
- Controlled actionable error when unresolved variables need a TTY.
- Preservation of fully supplied `--var`/`-V` non-interactive flows.
- Regression coverage for the shared prompt boundary plus `checkout` and
  `switch` command paths; cover `init` directly or through the shared helper.
- User-facing documentation and progress tracking updates.
- Step 3 low finding cleanup only if the implementation step edits the nearby
  `design-docs/specs/command.md` context.

**Excluded**:

- Treating redirected empty lines as prompt answers or default acceptance.
- Changing existing TTY prompt defaults, validation, ordering, or help text.
- Reworking update/new-variable confirmation prompts outside this issue scope.
- Adding new third-party dependencies unless the existing module already needs
  one for TTY detection.

### Codex-Agent References

- `.agents/agents/go-coding.md`: Step 6 implementation must provide Purpose,
  Reference Document, Implementation Target, and Completion Criteria.
- `.agents/agents/go-check-and-test-after-modify.md`: Step 6 verification must
  run after Go modifications and include targeted plus full Go checks.

The codex-agent references are process references only. Product behavior is
defined by `ign` design docs and issue #41.

---

## Modules

### 1. Shared Prompt Preflight

**Files**:

- `internal/cli/prompt.go`
- `internal/cli/prompt_test.go`
- `internal/cli/variables_test.go`

**Status**: COMPLETED

```go
func PromptForVariables(ignJson *model.IgnJson) (map[string]interface{}, error)
func PromptForVariablesWithProvided(ignJson *model.IgnJson, providedVars map[string]interface{}) (map[string]interface{}, error)
func promptForVariable(name string, varDef model.VarDef) (interface{}, error)
func promptInputIsTerminal() bool
func newNonInteractivePromptError(missingVarNames []string) error
```

**Checklist**:

- [x] Classify missing variable names before printing prompt headers.
- [x] Return early when every variable is supplied by `providedVars`.
- [x] Detect non-interactive stdin before any `survey.AskOne` call.
- [x] Return an error that mentions interactive prompts require a TTY and
      repeatable `--var key=value` / `-V key=value`.
- [x] Keep nil config, empty variable list, sorted prompt order, defaults, and
      validation behavior unchanged for interactive paths.

### 2. Command Path Regression Coverage

**Files**:

- `internal/cli/checkout.go`
- `internal/cli/init.go`
- `internal/cli/switch.go`
- `internal/cli/checkout_test.go`
- `internal/cli/switch_test.go`
- `internal/cli/init_test.go` or existing shared CLI test file

**Status**: COMPLETED

```go
func runCheckout(cmd *cobra.Command, args []string) error
func runSwitch(cmd *cobra.Command, args []string) error
func runInit(cmd *cobra.Command, args []string) error
```

**Checklist**:

- [x] Verify missing variables under non-interactive stdin return the controlled
      prompt error for `checkout`.
- [x] Verify missing variables under non-interactive stdin return before current
      output or `.ign` replacement for `switch`.
- [x] Cover `init` through a command-path test or through shared prompt-helper
      coverage that proves `init` uses the guarded boundary.
- [x] Verify fully supplied `--var`/`-V` paths do not prompt and still succeed
      with redirected/non-interactive stdin where existing fixtures permit.
- [x] Assert prompt preflight failures leave `.ign`, backups, and output files
      unchanged.

### 3. User-Facing Documentation And Progress

**Files**:

- `README.md`
- `docs/progress/non-interactive-template-variables.md`
- `impl-plans/active/non-tty-template-variable-prompts.md`
- `impl-plans/PROGRESS.json`

**Status**: COMPLETED

**Checklist**:

- [x] Document that missing interactive variables require a TTY.
- [x] Document `--var`/`-V` as the supported scripted path for `checkout`,
      `init`, and `switch`.
- [x] Update the progress file with implemented behavior, tests, verification
      notes, and remaining work.
- [x] Mark task status changes in this plan and `impl-plans/PROGRESS.json`
      during implementation.

### 4. Final Go Verification

**Files**:

- `internal/cli/*`
- `README.md`
- `docs/progress/non-interactive-template-variables.md`

**Status**: COMPLETED

**Checklist**:

- [x] Run targeted CLI regression tests.
- [x] Run full Go tests.
- [x] Run build and vet checks.
- [x] Invoke `.agents/agents/go-check-and-test-after-modify.md` after Go file
      modifications.
- [x] Record executed commands and outcomes in the progress log.

---

## Task Breakdown

### TASK-001: Shared Non-TTY Prompt Guard

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/cli/prompt.go`, `internal/cli/prompt_test.go`
**Dependencies**: None

**Description**:
Add shared prompt preflight in `PromptForVariablesWithProvided` so unresolved
variables are identified before printing prompt text or calling `survey/v2`.
When stdin is non-interactive, return a normal actionable error.

**Completion Criteria**:

- [x] Missing variables are calculated before `survey.AskOne`.
- [x] Fully supplied variable maps bypass TTY detection and prompting.
- [x] Non-interactive missing-variable tests fail with a controlled error, not a
      panic.

### TASK-002: Checkout And Init Command Regressions

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/cli/checkout_test.go`, `internal/cli/init_test.go` or existing equivalent
**Dependencies**: TASK-001

**Description**:
Add command-level regression coverage for redirected/non-TTY prompt failures and
fully supplied scripted variable execution through checkout and init coverage.

**Completion Criteria**:

- [x] `checkout` missing-variable non-TTY path returns the controlled prompt
      error.
- [x] `checkout` fully supplied `--var` path avoids prompting.
- [x] `init` is covered directly or the test proves it uses the same guarded
      prompt helper.
- [x] Prompt preflight failure does not create `.ign` or backup state.

### TASK-003: Switch Command Regressions

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `internal/cli/switch_test.go`
**Dependencies**: TASK-001

**Description**:
Add switch-specific regression coverage for non-interactive missing variables
and mutation ordering before rewind/replacement.

**Completion Criteria**:

- [x] `switch` missing-variable non-TTY path returns the controlled prompt
      error.
- [x] `switch` fully supplied `--var`/`-V` path avoids prompting.
- [x] Prompt preflight failure leaves current `.ign` and output untouched.

### TASK-004: Documentation And Progress Update

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `README.md`, `docs/progress/non-interactive-template-variables.md`
**Dependencies**: None

**Description**:
Refresh user-facing command documentation and progress tracking for issue #41.
Carry Step 3's low stale-context note only if editing the nearby design section.

**Completion Criteria**:

- [x] README documents TTY requirement for unresolved prompts and `--var`/`-V`
      scripted usage.
- [x] Progress file references issue #41, implemented behavior, regression
      tests, and verification commands.
- [x] No machine-local paths or environment variable values are introduced.

### TASK-005: Final Verification And Handoff

**Status**: Completed
**Parallelizable**: No
**Deliverables**: verification output, updated plan/progress statuses
**Dependencies**: TASK-001, TASK-002, TASK-003, TASK-004

**Description**:
Run targeted and full verification, update implementation-plan status and
progress logs, and prepare the Step 6 handoff summary.

**Completion Criteria**:

- [x] `go test ./internal/cli -run 'TestPrompt|TestCheckout|TestSwitch|TestInit'`
      passes or any narrower missing-test pattern is justified.
- [x] `go test ./...` passes.
- [x] `go build -o /dev/null ./...` passes.
- [x] `go vet ./...` passes.
- [x] `.agents/agents/go-check-and-test-after-modify.md` verification is
      invoked after Go modifications.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Shared prompt preflight | `internal/cli/prompt.go` | COMPLETED | `internal/cli/prompt_test.go` |
| Checkout/init command regressions | `internal/cli/checkout.go`, `internal/cli/init.go` | COMPLETED | `internal/cli/checkout_test.go`, `internal/cli/init_test.go` |
| Switch command regressions | `internal/cli/switch.go` | COMPLETED | `internal/cli/switch_test.go` |
| Documentation and progress | `README.md`, `docs/progress/non-interactive-template-variables.md` | COMPLETED | Review plus verification notes |
| Final verification | `internal/cli/*` | COMPLETED | Targeted tests, full tests, build, vet |

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| TASK-001 shared prompt guard | None | COMPLETED |
| TASK-002 checkout/init regressions | TASK-001 | COMPLETED |
| TASK-003 switch regressions | TASK-001 | COMPLETED |
| TASK-004 docs/progress | None | COMPLETED |
| TASK-005 final verification | TASK-001, TASK-002, TASK-003, TASK-004 | COMPLETED |

## Parallelizable Tasks

- `TASK-001` and `TASK-004` can run in parallel because their write scopes are
  disjoint.
- `TASK-002` and `TASK-003` can run in parallel after `TASK-001` if assigned to
  separate workers and each worker keeps to its command-specific test file.
- `TASK-005` is serial because it depends on all implementation and docs tasks.

## Verification Plan

- `go test ./internal/cli -run 'TestPrompt|TestCheckout|TestSwitch|TestInit'`
- `go test ./...`
- `go build -o /dev/null ./...`
- `go vet ./...`

Use `.agents/agents/go-check-and-test-after-modify.md` after Go modifications
with modified packages `internal/cli` and modified files listed explicitly.

## Completion Criteria

- [x] `checkout`, `init`, and `switch` never call `survey/v2` for unresolved
      variables when stdin is non-interactive.
- [x] Controlled error text mentions TTY requirement and `--var`/`-V` guidance.
- [x] Fully supplied `--var`/`-V` non-interactive runs continue without prompts.
- [x] Prompt preflight failures occur before `.ign` creation, backups, rewind,
      or switch replacement.
- [x] Regression tests cover prompt boundary plus command paths required by the
      accepted design.
- [x] README, progress docs, this plan, and `impl-plans/PROGRESS.json` are
      updated with implementation status and verification outcomes.
- [x] Targeted tests, full tests, build, and vet pass or any environmental
      blocker is documented.

## Progress Log

### Session: 2026-08-04 00:00

**Tasks Completed**: Plan creation for Step 4.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Step 3 accepted `design-docs/specs/command.md` with one low stale
context note at line 76 that is not blocking implementation planning.

### Session: 2026-08-04 Step 6

**Tasks Completed**: TASK-001, TASK-002, TASK-003, TASK-004, TASK-005.
**Tasks In Progress**: None.
**Blockers**: None.
**Notes**: Added shared non-TTY prompt preflight and direct regressions for
`checkout`, `init`, and `switch`. Verification passed:
`go test ./internal/cli -run 'TestPrompt|TestCheckout|TestSwitch|TestInit'`,
`go test ./...`, `go build -o /dev/null ./...`, and `go vet ./...`.

## Related Plans

- **Previous**: `impl-plans/ign-vars.md` and
  `impl-plans/active/verified-defect-remediation.md`
- **Next**: None
- **Depends On**: `design-docs/specs/command.md`
