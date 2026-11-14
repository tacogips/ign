# Verified Defect Remediation Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/architecture.md#verified-defect-remediation`, `design-docs/specs/command.md#defect-fix-cli-semantics`
**Created**: 2026-07-12
**Last Updated**: 2026-07-12

---

## Design Document Reference

**Sources**:

- `design-docs/specs/architecture.md#verified-defect-remediation`
- `design-docs/specs/command.md#defect-fix-cli-semantics`

### Summary

Implement the accepted issue-resolution design for verified defects in
`internal/app`, `internal/cli`, `internal/config`, and `internal/template`.
All HIGH findings are mandatory. The listed MID findings are included because
they fit one coherent defect-remediation change set and share verification
scope with the HIGH fixes.

### Scope

**Included**:

- HIGH-1 update transaction safety and manifest persistence after partial
  cleanup failure.
- HIGH-2 crash-safe config/template JSON writes.
- HIGH-3 rewind fallback ownership safety.
- HIGH-4 and HIGH-5 CLI error output and checkout exit behavior.
- HIGH-6 relative include resolution.
- MID-1 through MID-7 parser, provider, CLI, switch, update hash, and prompt
  fixes.
- Regression tests for each implemented finding.
- Progress-log update under `docs/progress/`.

**Excluded**:

- Unrelated feature work.
- New third-party dependencies.
- Public CLI changes except checkout non-zero existing-config behavior and
  `ign switch --var/-V`.

### Codex-Agent References

- `.agents/agents/go-coding.md`: required for Step 6 Go implementation with
  Purpose, Reference Document, Implementation Target, and Completion Criteria.
- `.agents/agents/go-check-and-test-after-modify.md`: required after Go file
  modifications and for requested Go checks.

No external codex-agent reference repository or behavior input was supplied.

---

## Modules

### 1. Update Transaction And Hash Validation

**Files**:

- `internal/app/update.go`
- `internal/app/update_cleanup.go`
- `internal/app/manifest.go`
- `internal/app/update_test.go`

**Status**: COMPLETED

```go
func PrepareUpdate(ctx context.Context, opts UpdateOptions) (*PrepareUpdateResult, error)
func CompleteUpdate(ctx context.Context, opts CompleteUpdateOptions) (*UpdateResult, error)
func validateTemplateHash(hash string) error
func cleanupRemovedManagedFilesForUpdate(ctx context.Context, opts cleanupRemovedManagedFilesOptions) (*cleanupRemovedManagedFilesResult, error)
func saveManifestFromGenerateResultExcluding(path string, result *generator.GenerateResult, excludedCanonicalPaths map[string]struct{}) error
```

**Checklist**:

- [x] Validate new update template hashes with checkout's SHA-256 contract.
- [x] Generate and clean up removed managed files before saving `.ign/ign.json`
      or `.ign/ign-var.json`.
- [x] Reuse existing checkout rollback machinery or an app-layer equivalent for
      update file rollback.
- [x] Preserve previous `.ign` state when generation fails.
- [x] Persist manifest entries for successful writes and successful deletions
      even when cleanup returns a partial failure.
- [x] Add regression tests for HIGH-1 and MID-6.

### 2. Atomic JSON Persistence

**Files**:

- `internal/config/loader.go`
- `internal/config/config_test.go`
- `internal/app/template_update.go`
- `internal/app/template_update_test.go`

**Status**: COMPLETED

```go
func SaveIgnVarJson(path string, ignVar *model.IgnVarJson) error
func SaveIgnManifest(path string, manifest *model.IgnManifest) error
func SaveIgnConfig(path string, ignConfig *model.IgnConfig) error
func updateIgnJson(path string, result *UpdateTemplateResult, existing *model.IgnJson, merge bool) error
```

**Checklist**:

- [x] Add one shared same-directory temp-file, fsync, and rename helper.
- [x] Use the helper for `.ign/ign-var.json`, `.ign/ign-files.json`,
      `.ign/ign.json`, and template-side `ign-template.json`.
- [x] Preserve existing marshaling, validation, permissions, directory creation,
      and error wrapping behavior.
- [x] Clean up temporary files on failure.
- [x] Add regression tests for HIGH-2 behavior that can be exercised without
      process-kill tests.

### 3. Rewind Fallback Ownership Safety

**Files**:

- `internal/app/rewind.go`
- `internal/app/rewind_test.go`

**Status**: COMPLETED

```go
func Rewind(ctx context.Context, opts RewindOptions) (*RewindResult, error)
func buildManagedFilesFromCurrentTemplate(ctx context.Context, opts RewindOptions) ([]string, error)
```

**Checklist**:

- [x] Keep manifest-backed rewind behavior authoritative.
- [x] In the missing-manifest fallback, compare current on-disk content with
      dry-run generated content before deleting.
- [x] Skip fallback candidates whose content differs.
- [x] Keep existing containment and empty-directory cleanup behavior.
- [x] Add regression tests for HIGH-3.

### 4. CLI Error, Checkout, Switch, And Prompt Semantics

**Files**:

- `internal/cli/root.go`
- `internal/cli/output.go`
- `internal/cli/checkout.go`
- `internal/cli/template.go`
- `internal/cli/switch.go`
- `internal/cli/prompt.go`
- `internal/cli/cli_test.go`
- `internal/cli/checkout_test.go`
- `internal/cli/switch_test.go`
- `internal/cli/prompt_test.go`

**Status**: COMPLETED

```go
func Execute()
func printError(err error)
func runCheckout(cmd *cobra.Command, args []string) error
func runSwitch(cmd *cobra.Command, args []string) error
func promptForVariable(name string, varDef model.VarDef) (interface{}, error)
func matchPattern(pattern string, message string) survey.Validator
```

**Checklist**:

- [x] Ensure `--quiet` suppresses non-error output only.
- [x] Make command failures print once through the root error path.
- [x] Return an error when checkout finds existing `.ign` without `--force`.
- [x] Replace `os.Exit` inside `RunE` with returned errors.
- [x] Delay switch `.ign` setup until after variable parsing and prompting.
- [x] Add `ign switch --var/-V` using checkout/init variable semantics.
- [x] Compile invalid prompt regex patterns before prompting and fail fast with
      the variable name.
- [x] Add regression tests for HIGH-4, HIGH-5, MID-4, MID-5, and MID-7.

### 5. Template Parser And Generator Safety

**Files**:

- `internal/template/generator/processor.go`
- `internal/template/generator/generator_test.go`
- `internal/template/parser/directive.go`
- `internal/template/parser/include.go`
- `internal/template/parser/parser_test.go`

**Status**: COMPLETED

```go
func (p *FileProcessor) Process(ctx context.Context, file model.TemplateFile, vars parser.Variables, templateRoot string) ([]byte, error)
func resolveIncludePath(includePath, templateRoot, currentFile string) (string, error)
func findDirectives(input []byte) []DirectiveMatch
```

**Checklist**:

- [x] Pass a full template-root-relative filesystem path as parser
      `CurrentFile` during generation.
- [x] Use `filepath.Rel` style containment for include path validation.
- [x] Parse same-line raw directives independently with nearest valid `@@`
      termination.
- [x] Add regression tests for HIGH-6, MID-1, and include containment in MID-3.

### 6. GitHub URL And Archive Hardening

**Files**:

- `internal/template/provider/url.go`
- `internal/template/provider/provider_test.go`
- `internal/template/provider/github.go`
- `internal/template/provider/github_test.go`

**Status**: COMPLETED

```go
func ParseGitHubURL(url string) (*model.TemplateRef, error)
func (p *GitHubProvider) extractArchive(archivePath string) (string, error)
```

**Checklist**:

- [x] Parse branch-only `/tree/<branch>` URLs as an empty template path with
      `Ref=<branch>`.
- [x] Trim trailing `.git` for all GitHub URL forms.
- [x] Preserve documented ambiguity for slashed branch names; require explicit
      `--ref` for those cases.
- [x] Reject archive entries whose cleaned target escapes extraction root.
- [x] Reject absolute symlink targets and symlink targets resolving outside the
      extraction root.
- [x] Add regression tests for MID-2 and provider-side MID-3.

### 7. Documentation And Progress Tracking

**Files**:

- `docs/progress/verified-defect-remediation.md`
- `design-docs/specs/architecture.md`
- `design-docs/specs/command.md`
- `impl-plans/active/verified-defect-remediation.md`
- `impl-plans/PROGRESS.json`

**Status**: COMPLETED

**Checklist**:

- [x] Add or update `docs/progress/verified-defect-remediation.md`.
- [x] Record implemented sub-features, remaining work, design decisions, and
      verification notes.
- [x] Update this plan and `impl-plans/PROGRESS.json` task statuses during Step
      6 implementation.
- [x] Refresh design docs only if implementation reveals an accepted design
      correction.

### 8. Final Verification And Handoff

**Files**:

- All files changed by TASK-001 through TASK-007.

**Status**: COMPLETED

**Checklist**:

- [x] Run `gofmt` on changed Go files.
- [x] Run `go vet ./...`.
- [x] Run `go build ./...`.
- [x] Run `go test ./...`.
- [x] Record command outcomes in the workflow result and progress log.

---

## Task Breakdown

### TASK-001: Fix Update Transaction Safety And Hash Validation

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/app/update.go`, `internal/app/update_cleanup.go`,
`internal/app/manifest.go`, `internal/app/update_test.go`
**Dependencies**: None

**Description**:
Implement HIGH-1 and MID-6 by validating update hashes and making update file
generation/cleanup transactional before `.ign` config persistence.

**Completion Criteria**:

- [x] Update generation failure leaves previous `.ign` state unchanged.
- [x] Cleanup partial failure still records successful writes and deletions in
      `ign-files.json`.
- [x] Malformed update template hash is rejected before persistence.
- [x] Regression tests cover failure ordering and hash validation.

### TASK-002: Add Atomic JSON Write Helper

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/config/loader.go`, `internal/config/config_test.go`,
`internal/app/template_update.go`, `internal/app/template_update_test.go`
**Dependencies**: None

**Description**:
Implement HIGH-2 with one shared atomic write helper and route all four JSON
persistence call sites through it.

**Completion Criteria**:

- [x] Helper writes temp file in target directory, fsyncs, renames, and cleans
      up on failure.
- [x] Existing save functions keep their public signatures and validation.
- [x] Tests cover successful writes and failure cleanup behavior.

### TASK-003: Fix Rewind Missing-Manifest Fallback

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/app/rewind.go`, `internal/app/rewind_test.go`
**Dependencies**: None

**Description**:
Implement HIGH-3 by deleting fallback candidates only when current content
matches dry-run generated content.

**Completion Criteria**:

- [x] User-owned conflicting files are skipped.
- [x] Matching generated files remain removable.
- [x] Skipped files are visible in rewind result/reporting.

### TASK-004: Fix CLI Error And Interactive Semantics

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/cli/root.go`, `internal/cli/output.go`,
`internal/cli/checkout.go`, `internal/cli/template.go`,
`internal/cli/switch.go`, `internal/cli/prompt.go`, related CLI tests
**Dependencies**: None

**Description**:
Implement HIGH-4, HIGH-5, MID-4, MID-5, and MID-7 in the CLI layer.

**Completion Criteria**:

- [x] Errors print under `--quiet`.
- [x] Failures print once.
- [x] Existing `.ign` checkout without `--force` returns non-zero.
- [x] Template check returns errors instead of calling `os.Exit`.
- [x] Switch supports `--var/-V` and does not create `.ign` before successful
      variable handling.
- [x] Invalid regex patterns fail before prompting.

### TASK-005: Fix Template Parser And Generator Defects

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/template/generator/processor.go`,
`internal/template/parser/directive.go`, `internal/template/parser/include.go`,
related generator/parser tests
**Dependencies**: None

**Description**:
Implement HIGH-6, MID-1, and parser-side MID-3 containment hardening.

**Completion Criteria**:

- [x] Relative includes resolve from the including file.
- [x] Same-line raw directives remain independent.
- [x] Include containment uses `filepath.Rel` style checks.
- [x] Regression tests cover each behavior.

### TASK-006: Fix GitHub Provider URL And Extraction Safety

**Status**: Completed
**Parallelizable**: Yes
**Deliverables**: `internal/template/provider/url.go`,
`internal/template/provider/github.go`, related provider tests
**Dependencies**: None

**Description**:
Implement MID-2 and provider-side MID-3 for GitHub template references and
archive extraction.

**Completion Criteria**:

- [x] Branch-only `/tree/<branch>` URLs and trailing `.git` URLs parse
      correctly.
- [x] Escaping archive entries are rejected.
- [x] Escaping symlink targets are rejected.
- [x] Slashed branch ambiguity remains documented and is not guessed.

### TASK-007: Update Progress Documentation

**Status**: Completed
**Parallelizable**: No
**Deliverables**: `docs/progress/verified-defect-remediation.md`,
`impl-plans/active/verified-defect-remediation.md`,
`impl-plans/PROGRESS.json`
**Dependencies**: TASK-001, TASK-002, TASK-003, TASK-004, TASK-005, TASK-006

**Description**:
Record implemented fixes, remaining work if any, and design decisions after the
code changes are known.

**Completion Criteria**:

- [x] Progress file includes status, spec references, implemented items,
      remaining items, design decisions, and notes.
- [x] Plan and `PROGRESS.json` statuses match implementation state.

### TASK-008: Run Required Verification

**Status**: Completed
**Parallelizable**: No
**Deliverables**: verification command results in workflow output and progress
log
**Dependencies**: TASK-007

**Description**:
Run the required Go verification commands and record outcomes for review.

**Completion Criteria**:

- [x] `gofmt` completed for changed Go files.
- [x] `go vet ./...` passes.
- [x] `go build ./...` passes.
- [x] `go test ./...` passes.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Update transaction and hash validation | `internal/app/update.go` | COMPLETED | `internal/app/update_test.go` |
| Atomic JSON persistence | `internal/config/loader.go`, `internal/app/template_update.go` | COMPLETED | `internal/config/config_test.go`, `internal/app/template_update_test.go` |
| Rewind fallback ownership | `internal/app/rewind.go` | COMPLETED | `internal/app/rewind_test.go` |
| CLI defect semantics | `internal/cli/*.go` targeted files | COMPLETED | `internal/cli/*_test.go` targeted tests |
| Parser and include safety | `internal/template/parser/*.go`, `internal/template/generator/processor.go` | COMPLETED | parser/generator tests |
| GitHub provider safety | `internal/template/provider/*.go` targeted files | COMPLETED | provider tests |
| Progress tracking | `docs/progress/verified-defect-remediation.md` | COMPLETED | workflow review |

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| TASK-001 Update transaction safety | None | COMPLETED |
| TASK-002 Atomic JSON persistence | None | COMPLETED |
| TASK-003 Rewind fallback ownership | None | COMPLETED |
| TASK-004 CLI semantics | None | COMPLETED |
| TASK-005 Parser/generator safety | None | COMPLETED |
| TASK-006 GitHub provider safety | None | COMPLETED |
| TASK-007 Progress documentation | TASK-001 through TASK-006 | COMPLETED |
| TASK-008 Verification | TASK-007 | COMPLETED |

## Completion Criteria

- [x] All HIGH findings HIGH-1 through HIGH-6 are fixed.
- [x] MID findings MID-1 through MID-7 are fixed unless Step 6 documents a
      concrete implementation risk accepted by review.
- [x] Regression tests cover every implemented finding.
- [x] Existing behavior is preserved except where the findings define a defect.
- [x] No new third-party dependencies are added.
- [x] Public CLI behavior changes are limited to checkout existing-config
      non-zero exit and `ign switch --var/-V`.
- [x] `docs/progress/verified-defect-remediation.md` records implementation
      status and verification notes.
- [x] `gofmt`, `go vet ./...`, `go build ./...`, and `go test ./...` pass.

## Progress Log

### Session: 2026-07-12

**Tasks Completed**: TASK-001 through TASK-008 implemented and verified.

**Tasks In Progress**: None.

**Blockers**: None.

**Notes**: `gofmt`, `go vet ./...`, `go build ./...`, and `go test ./...`
passed during Step 6.
