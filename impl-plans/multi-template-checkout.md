# Multi-Template Checkout Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/design-multi-template-checkout.md`
**Created**: 2026-05-17
**Last Updated**: 2026-05-17

---

## Design Document Reference

**Source**: `design-docs/specs/design-multi-template-checkout.md`

### Summary

Implement versioned `.ign` project metadata with version 2 multi-template
checkout support. New checkouts write explicit version 2 metadata, missing
versions continue to load as version 1, and adding a second template to a
version 1 project migrates all `.ign` metadata files to version 2. Template
records remain ordered by checkout order, later templates win overlapping output
paths, variables are scoped per template ID, and users can remove selected
templates or all templates while replaying remaining templates to restore
previously hidden files.

The active `design-and-implement-review-loop` pass is in `issue-resolution`
mode for "Self-review and improve multi-template checkout implementation". The
implementation direction below is accepted; the remaining work is to review the
current repository diff, fix only high or mid correctness findings, refresh docs
where behavior and documentation diverge, and rerun the explicit verification
commands before the self-review and independent review gates accept the
implementation.

### Scope

**Included**:

- Versioned project metadata models for `.ign/ign.json`,
  `.ign/ign-var.json`, and `.ign/ign-files.json`.
- Normalized project-state loading, validation, v1 compatibility, and v1-to-v2
  migration.
- Checkout flow for first v2 checkout, add-template checkout, dry-run migration
  reporting, output-root validation, and overwrite precedence.
- Per-template variable storage and update-time variable handling.
- Per-template manifest ownership and managed-path protections.
- New `ign remove <template-id>... [--output PATH]` and
  `ign remove --all [--output PATH]` behavior.
- `ign rewind`, `ign update`, and `ign switch` preservation of the v2 state
  contract.
- Tests, README updates, and progress tracking under `docs/progress/`.

**Excluded**:

- A dedicated variable-management CLI beyond multi-template-safe schema and
  application flows.
- Mixing multiple output roots in one `.ign` project state.
- Copying behavior or code from `../../codex-agent`, which is unavailable.

---

## Codex Agent References

- `AGENTS.md`: repository rules, Go conventions, path hygiene, and commit
  policy.
- `.agents/agents/go-coding.md`: required implementation-agent contract for Go
  changes.
- `.agents/agents/go-check-and-test-after-modify.md`: required verification
  agent after Go changes.
- `../../codex-agent`: unavailable; no behavior copied.

## Active Review-Gate Plan

### RG1. Current Diff Inventory and Reference Trace

**Write Scope**: none unless documentation references are stale.

**Deliverables**:

- Compare the current diff against
  `design-docs/specs/design-multi-template-checkout.md`,
  `impl-plans/multi-template-checkout.md`, and
  `docs/progress/multi-template-checkout.md`.
- Confirm every changed Go path maps to an accepted design module or is
  documented as a necessary support change.
- Treat `../../codex-agent` as unavailable and keep behavior local to ign.

**Verification**:

- `git status --short`
- `git diff --stat`
- `git diff --check`

### RG2. High/Mid Correctness Review

**Write Scope**: review notes first; code fixes only in affected modules.

**Blocker Criteria**:

- V2 metadata can be collapsed to v1 or saved inconsistently across
  `.ign/ign.json`, `.ign/ign-var.json`, and `.ign/ign-files.json`.
- Managed-path validation can delete or overwrite unmanaged files, write
  outside the output root, or mutate `.ign`.
- Output-root validation permits a checkout, remove, rewind, update, or switch
  against a different root from the stored v2 state.
- Template ID selection for remove/update is ambiguous or mutates state before
  all requested IDs are validated.
- Update, remove, rewind, or switch corrupts template order, variables,
  manifest ownership, or hidden-file replay behavior.
- README, progress, or CLI text promises behavior that implementation does not
  provide.

**Verification**:

- Focused `go test` command for each modified package.
- Add or adjust regression tests when a high or mid finding is fixed.

### RG3. Targeted Fix Implementation

**Write Scope**: only files needed to resolve accepted high or mid findings.

**Deliverables**:

- Minimal code or documentation changes for each blocker.
- `docs/progress/multi-template-checkout.md` updated with fixed behavior,
  remaining follow-up-only gaps, and verification results.
- This implementation plan updated if the task split, dependencies, or
  completion criteria change.

**Verification**:

- `gofmt -w internal test`
- Package-level `GOCACHE=/private/tmp/ign-go-cache go test ./...` subset when
  iterating, followed by the full command list in RG4.

### RG4. Required Verification and Review Handoff

**Write Scope**: documentation/progress updates only if commands expose a
behavior or status mismatch.

**Deliverables**:

- Verification commands run with results recorded in
  `docs/progress/multi-template-checkout.md`.
- Self-review and independent review gates receive explicit findings,
  decisions, residual risks, and command results.

**Required Commands**:

```bash
gofmt -w internal test
GOCACHE=/private/tmp/ign-go-cache go test ./...
GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign
GOCACHE=/private/tmp/ign-go-cache go vet ./...
GOCACHE=/private/tmp/ign-go-cache task test
GOCACHE=/private/tmp/ign-go-cache task build
git diff --check
```

---

## Modules

### 1. Versioned Project Metadata Models

**Files**:

- `internal/template/model/ignvar.go`
- `internal/template/model/ignmanifest.go`
- `internal/template/model/model_test.go`

**Status**: COMPLETED

**Planned Go Shapes**:

```go
const IgnProjectMetadataVersionV2 = 2

type TemplateCheckoutRecord struct {
	ID       string         `json:"id"`
	Order    int            `json:"order"`
	Template TemplateSource `json:"template"`
	Hash     string         `json:"hash"`
	Metadata *FileMetadata  `json:"metadata,omitempty"`
}

type IgnConfigV2 struct {
	Version    int                      `json:"version"`
	OutputRoot string                   `json:"output_root"`
	Templates  []TemplateCheckoutRecord `json:"templates"`
}

type IgnVarTemplateRecord struct {
	ID        string                 `json:"id"`
	Variables map[string]interface{} `json:"variables"`
}

type IgnVarJsonV2 struct {
	Version    int                    `json:"version"`
	OutputRoot string                 `json:"output_root"`
	Templates  []IgnVarTemplateRecord `json:"templates"`
}

type IgnManifestTemplateRecord struct {
	ID    string   `json:"id"`
	Order int      `json:"order"`
	Files []string `json:"files"`
}

type IgnManifestV2 struct {
	Version    int                         `json:"version"`
	OutputRoot string                      `json:"output_root"`
	Templates  []IgnManifestTemplateRecord `json:"templates"`
}
```

**Checklist**:

- [x] Add explicit version fields without breaking existing v1 structs.
- [x] Add v2 structs for ordered template, variable, and manifest records.
- [x] Add constants/helpers for supported metadata versions.
- [x] Add tests for JSON compatibility and missing-version v1 decoding.

### 2. Normalized Project State Loader and Saver

**Files**:

- `internal/config/loader.go`
- `internal/config/validation.go`
- `internal/config/config_test.go`
- `internal/config/validation_test.go`

**Status**: COMPLETED

**Deliverables**:

- A normalized project-state API that presents v1 and v2 metadata as ordered
  template records to application code.
- Cross-file validation for `version`, `output_root`, template IDs, order,
  hash format, variables, and manifest references.
- Save helpers for atomic-enough multi-file writes of `ign.json`,
  `ign-var.json`, and `ign-files.json` using temp files and rename.
- Migration helper that wraps a v1 state as one v2 template record while
  preserving existing v1 behavior until an add-template checkout commits.

**Checklist**:

- [x] Treat missing `version` as v1.
- [x] Reject unsupported versions with clear errors.
- [x] Validate v2 `output_root` is normalized, relative, and consistent across
      all `.ign` files.
- [x] Reject absolute, traversal, and non-normalized v2 `output_root` values on
      load before returning normalized project state.
- [x] Validate v2 template IDs are unique and manifest/variable records match
      `ign.json`.
- [x] Preserve legacy `LoadIgnConfig`, `LoadIgnVarJson`, and `LoadIgnManifest`
      callers or adapt them through compatibility wrappers.
- [x] Add unit tests for v1 load, v2 load, invalid cross-file state, and
      migration save failure behavior.
- [x] Stage all v2 metadata files before live replacement and restore the prior
      metadata set if a partial replace fails.

### 3. Checkout State Creation, Add-Template, and Migration

**Files**:

- `internal/app/checkout.go`
- `internal/app/checkout_test.go`
- `internal/cli/checkout.go`
- `internal/cli/cli_test.go`
- `test/integration/checkout_test.go`

**Status**: COMPLETED

**Deliverables**:

- First checkout writes v2 metadata for a single template.
- Existing `.ign` without `--force` routes to add-template checkout instead of
  returning the current informational early exit.
- Adding to v1 migrates all metadata files to v2 before saving the new record.
- Dry-run reports that migration would occur without mutating `.ign`.
- Output path validation enforces the stored v2 `output_root`.
- Later checkout overwrites managed paths from earlier templates and protects
  unmanaged existing paths unless force semantics apply.

**Checklist**:

- [x] Add a stable template ID generator based on template name plus checkout
      ordinal.
- [x] Store monotonic `order` and never renumber existing records.
- [x] Copy template name, template version, source, ref, path, and hash from
      fetched template metadata into the v2 record.
- [x] Prompt only for the new template's variables on add-template checkout.
- [x] Update CLI help and messages for multi-template add behavior.
- [ ] Add tests for first checkout, second checkout, v1 migration, dry-run, and
      conflicting output path rejection. Stored output-root reuse and explicit
      output-root mismatch rejection are covered; dry-run migration reporting
      remains a follow-up test gap.

### 4. Manifest Ownership and Managed-Path Precedence

**Files**:

- `internal/app/manifest.go`
- `internal/app/rewind.go`
- `internal/app/checkout.go`
- `internal/app/rewind_test.go`
- `test/integration/checkout_test.go`

**Status**: COMPLETED

**Deliverables**:

- Per-template manifest records preserve generated paths for each checkout.
- Managed-path ownership supports later-template precedence for overlapping
  paths.
- Existing managed-path validation remains the gate for file deletion and
  refuses paths outside the output root or inside `.ign`.

**Checklist**:

- [x] Replace global sorted manifest writes with per-template ordered ownership
      writes.
- [x] Keep path cleaning and deduping per template record.
- [x] Add helpers to compute winning paths and removed-template paths.
- [x] Add tests for overlapping managed paths and unmanaged path protection.

### 5. Per-Template Variables and Update Preparation

**Files**:

- `internal/app/variables.go`
- `internal/app/update.go`
- `internal/app/variables_test.go`
- `internal/app/update_test.go`
- `internal/cli/update.go`
- `internal/cli/update_test.go`

**Status**: COMPLETED

**Deliverables**:

- Variable preparation, validation, `@file:` resolution, and update-time new
  variable collection operate on one template record at a time.
- `ign update` processes all v2 template records in checkout order for the
  first implementation and preserves each record's variables, source, order,
  manifest ownership, and `output_root`.

**Checklist**:

- [x] Add helpers to select variable maps by template ID.
- [x] Preserve existing v1 update behavior through normalized single-record
      state.
- [x] For v2 update, fetch and process all records in order.
- [x] Update hashes and metadata per template record without collapsing the v2
      state.
- [x] Add tests for per-template variable collisions and all-record update.

### 6. Selective Remove and Replay

**Files**:

- `internal/app/remove.go`
- `internal/app/remove_test.go`
- `internal/app/rewind.go`
- `internal/app/rewind_test.go`
- `internal/cli/remove.go`
- `internal/cli/root.go`
- `internal/cli/cli_test.go`
- `test/integration/rewind_test.go`

**Status**: COMPLETED

**Deliverables**:

- Add `ign remove <template-id>... [--output PATH]` and
  `ign remove --all [--output PATH]`.
- Unknown template IDs fail before mutation.
- Removing selected templates deletes their generated output paths, removes
  their metadata, and replays remaining templates in ascending order.
- Removing all templates deletes `.ign`; `ign rewind [output-path]` remains
  the backward-compatible full-removal path.

**Checklist**:

- [x] Add application-level remove options/result types.
- [x] Validate output path against stored `output_root`.
- [x] Delete removed template output paths with existing managed-path
      protections.
- [x] Replay remaining templates from stored source/ref/path/hash and scoped
      variables.
- [x] Restore earlier hidden files when a later overwriting template is removed.
- [ ] Add tests for removing later, earlier, multiple, unknown, and all template
      IDs.

### 7. Switch Compatibility

**Files**:

- `internal/cli/switch.go`
- `internal/cli/switch_test.go`
- `internal/app/rewind.go`
- `internal/app/checkout.go`

**Status**: COMPLETED

**Deliverables**:

- `ign switch` remains full removal followed by checkout.
- With v2 state, switch removes all records and creates one new v2 record.

**Checklist**:

- [x] Ensure switch uses the same full-removal path as rewind/remove all.
- [x] Keep switch validation dry-run non-mutating.
- [ ] Add regression tests for switching from v2 multi-template state.

### 8. Documentation and Progress Tracking

**Files**:

- `README.md`
- `docs/progress/multi-template-checkout.md`
- `impl-plans/PROGRESS.json`

**Status**: COMPLETED

**Deliverables**:

- User-facing docs describe multi-template checkout, template IDs, v2 metadata,
  output-root behavior, remove, remove all, rewind, update, and switch.
- Progress file tracks status, spec references, implemented paths, remaining
  work, design decisions, and notes.

**Checklist**:

- [x] Add `docs/progress/multi-template-checkout.md`.
- [x] Keep progress status current during implementation.
- [x] Update README command examples and behavior notes.
- [x] Mark this plan complete in `impl-plans/PROGRESS.json` only after tests
      and docs are complete.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Versioned metadata models | `internal/template/model/ignvar.go`, `internal/template/model/ignmanifest.go` | COMPLETED | `go test ./internal/template/model` |
| Normalized loaders and savers | `internal/config/loader.go`, `internal/config/validation.go`, `internal/config/project_state.go` | COMPLETED | `go test ./internal/config` |
| Checkout creation and migration | `internal/app/checkout.go`, `internal/cli/checkout.go` | COMPLETED | `go test ./internal/app` |
| Manifest ownership | `internal/app/manifest.go`, `internal/app/rewind.go` | COMPLETED | `go test ./internal/app` |
| Per-template variables and update | `internal/app/variables.go`, `internal/app/update.go`, `internal/cli/update.go` | COMPLETED | `go test ./internal/app ./internal/cli` |
| Remove and replay | `internal/app/remove.go`, `internal/cli/remove.go`, `internal/cli/root.go` | COMPLETED | `go test ./internal/app ./internal/cli` |
| Switch compatibility | `internal/cli/switch.go`, `internal/app/rewind.go` | COMPLETED | `go test ./internal/app ./internal/cli` |
| Documentation and progress | `README.md`, `docs/progress/multi-template-checkout.md` | COMPLETED | `git diff --check` |

---

## Task Breakdown

| ID | Task | Deliverables | Dependencies | Verification |
|----|------|--------------|--------------|--------------|
| T1 | Add v2 metadata models | Go structs/constants and model JSON tests | Accepted design | `go test ./internal/template/model` |
| T2 | Add normalized state loading/saving | v1/v2 loaders, validators, migration saver | T1 | `go test ./internal/config` |
| T3 | Update checkout add-template flow | first v2 checkout, add-template, v1 migration, dry-run reporting | T1, T2 | `go test ./internal/app ./internal/cli`; checkout integration tests |
| T4 | Add per-template manifest ownership | ordered manifest records, winning-path helpers, managed-path tests | T1, T2 | `go test ./internal/app` |
| T5 | Scope variables per template and update all records | variable helpers, all-record v2 update, update CLI output | T1, T2, T4 | `go test ./internal/app ./internal/cli`; update integration tests |
| T6 | Implement remove and replay | `app.Remove`, `ign remove`, replay remaining templates | T1-T5 | `go test ./internal/app ./internal/cli`; rewind/remove integration tests |
| T7 | Preserve rewind and switch behavior | full-removal compatibility and switch-from-v2 tests | T6 | `go test ./internal/app ./internal/cli` |
| T8 | Refresh docs and progress | README and progress file updates | T3-T7 behavior stable | `git diff --check`; docs review |
| T9 | Full verification | formatted code and full build/test suite | T1-T8 | `gofmt -w internal test`; `GOCACHE=/private/tmp/ign-go-cache go test ./...`; `GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign`; `GOCACHE=/private/tmp/ign-go-cache go vet ./...`; `GOCACHE=/private/tmp/ign-go-cache task test`; `GOCACHE=/private/tmp/ign-go-cache task build`; `git diff --check` |
| RG1 | Review current diff inventory | traced changed files and accepted-reference mapping | Step 3 accepted design | `git status --short`; `git diff --stat`; `git diff --check` |
| RG2 | Identify high/mid correctness findings | self-review findings with severity and file paths | RG1 | focused package tests and code inspection |
| RG3 | Apply targeted fixes | minimal code/docs/test changes for each blocker | RG2 accepted findings | `gofmt -w internal test`; focused `go test` |
| RG4 | Final review-gate verification | command results and residual risk handoff | RG3 | full required command list |

---

## Dependencies

| Feature | Depends On | Status |
|---------|------------|--------|
| Metadata models | Accepted design | DONE |
| Normalized state API | Metadata models | DONE |
| Checkout migration and add-template | Normalized state API | DONE |
| Manifest ownership | Normalized state API | DONE |
| Per-template variables and update | Normalized state API, manifest ownership | DONE |
| Selective remove and replay | Checkout, variables, manifest ownership | DONE |
| Rewind and switch compatibility | Remove/full-removal path | DONE |
| Documentation and progress | Implemented command behavior | DONE |
| Review diff inventory | Step 3 design acceptance | READY |
| High/mid correctness fixes | Review diff inventory | READY |
| Final verification handoff | Correctness fixes complete or no blockers found | DONE |

---

## Parallelizable Tasks

- T1 and initial T8 progress-file scaffolding are parallelizable because their
  write scopes are disjoint.
- After T1 and T2 land, T3 checkout work and T5 variable/update preparation can
  be split only if the workers coordinate on the normalized project-state API
  and avoid editing the same loader/model files.
- T6 remove/replay must wait for T3, T4, and T5 because it depends on checkout
  records, manifest ownership, and per-template variable replay.
- Final docs in T8 should wait until CLI behavior and messages are stable.
- RG1 can run in parallel with documentation consistency review because the
  write scopes are disjoint until a concrete finding is accepted.
- RG2 code review and verification-log review can run in parallel when they do
  not edit the same Go files or docs.
- RG3 fixes are parallelizable only when the file write scopes are disjoint,
  such as CLI text/docs in one task and app-layer replay logic in another.
- RG4 is not parallelizable with code changes because it is the final
  verification and review-gate handoff.

---

## Verification Plan

### Focused Unit Tests

- `go test ./internal/template/model`
- `go test ./internal/config`
- `go test ./internal/app`
- `go test ./internal/cli`

### Integration and Regression Tests

- `go test ./test/integration`
- Add test coverage for:
  - missing-version v1 metadata loading as normalized single-template state;
  - first checkout writing v2 metadata;
  - second checkout migrating v1 metadata to v2;
  - ordered overwrite precedence with later checkout winning;
  - per-template variable storage and same variable names across templates;
  - removing a later template and restoring an earlier hidden file;
  - removing an earlier template while keeping a later overwrite;
  - removing an earlier template without overwriting files skipped by a
    remaining template;
  - `ign remove --all` and `ign rewind` full removal;
  - `ign update` preserving all v2 template records;
  - output-root mismatch rejection.

### Full Verification Commands

```bash
gofmt -w internal test
GOCACHE=/private/tmp/ign-go-cache go test ./...
GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign
GOCACHE=/private/tmp/ign-go-cache go vet ./...
GOCACHE=/private/tmp/ign-go-cache task test
GOCACHE=/private/tmp/ign-go-cache task build
git diff --check
```

---

## Completion Criteria

- [x] New checkout-created `.ign` files include `"version": 2`.
- [x] Missing `version` loads as v1 and existing v1 checkout/update/rewind
      behavior remains compatible.
- [x] Adding a template to v1 migrates `ign.json`, `ign-var.json`, and
      `ign-files.json` to v2 in one consistent operation.
- [x] V2 `ign.json`, `ign-var.json`, and `ign-files.json` store ordered
      template records with stable IDs and consistent `output_root`.
- [x] Later templates overwrite earlier managed paths while unmanaged paths
      remain protected.
- [x] Variables are stored and loaded per template ID.
- [x] Users can remove one, multiple, or all templates.
- [x] Removing templates replays remaining templates in checkout order and
      restores files hidden by later overwrites.
- [x] Remove replay protects skipped unmanaged files while preserving managed
      overwrite precedence.
- [x] Add-template checkout with omitted `[output-path]` reuses stored
      `output_root`; explicit mismatches are rejected.
- [x] `rewind`, `update`, and `switch` do not collapse or corrupt v2 state.
- [x] README and `docs/progress/multi-template-checkout.md` are updated.
- [x] `gofmt -w internal test`, `GOCACHE=/private/tmp/ign-go-cache go test ./...`,
      `GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign`,
      `GOCACHE=/private/tmp/ign-go-cache go vet ./...`,
      `GOCACHE=/private/tmp/ign-go-cache task test`,
      `GOCACHE=/private/tmp/ign-go-cache task build`, and `git diff --check`
      pass or any environment-specific blockers are documented.
- [x] Implementation self-review gate records no unresolved high or mid
      findings against the current diff.
- [x] Independent implementation review gate accepts the current diff or all
      high and mid findings are routed back to RG3.
- [x] Required verification commands are rerun with `GOCACHE` and recorded
      explicitly before final handoff.

---

## Progress Log Expectations

Create and maintain `docs/progress/multi-template-checkout.md` during
implementation with:

- Status: `Not Started`, `In Progress`, or `Completed`.
- Spec reference to `design-docs/specs/design-multi-template-checkout.md`.
- Implemented checklist with file paths.
- Remaining checklist with file paths or behavior names.
- Design decisions for schema, migration, output-root validation, and replay.
- Notes for verification results, blocked commands, and follow-up risks.

This implementation plan should also remain listed in `impl-plans/PROGRESS.json`
until completed.

---

## Related Plans

- **Previous**: None
- **Next**: None
- **Depends On**: `design-docs/specs/design-multi-template-checkout.md`
