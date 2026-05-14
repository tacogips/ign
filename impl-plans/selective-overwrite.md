# Selective Overwrite Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/selective-overwrite.md`
**Created**: 2026-05-14
**Last Updated**: 2026-05-14

---

## Design Document Reference

**Source**: `design-docs/specs/selective-overwrite.md`

### Summary

Review and implement selective overwrite for `ign update`. The accepted behavior
is:

- `ign update --overwrite` uses the remote template root
  `.ign-overwrite-ignore`.
- `ign update --overwrite-all` preserves the previous overwrite-all behavior.
- `ign update --force` keeps overwrite-all semantics.
- `--yes` / `-y` skips confirmation only.
- Non-`--yes` overwrite updates preview planned writes with `A` and `M`.
- Root `.ign-overwrite-ignore` is template metadata: included in template hash
  calculation, followed from the remote template, and not emitted into project
  output.

### Scope

**Included**:

- CLI flag parsing and overwrite mode selection.
- Update orchestration and confirmation preview.
- Generator overwrite filtering.
- Template metadata and hash behavior.
- User-facing documentation and progress tracking.
- Unit and integration-level regression tests.

**Excluded**:

- Changes to Divedra runtime behavior.
- Broad rewrites outside the `ign update` selective overwrite surface.

---

## Modules

### 1. CLI and Update Mode Semantics

**Files**:

- `internal/cli/update.go`
- `internal/cli/update_test.go`
- `internal/app/update.go`
- `internal/app/update_test.go`

**Status**: COMPLETED

**Checklist**:

- [x] `--overwrite` selects selective overwrite mode.
- [x] `--overwrite-all` and `--force` select overwrite-all mode.
- [x] `--yes` bypasses confirmation without changing update decisions.
- [x] Preview output lists planned writes with `A` and `M`.

### 2. Generator Selective Overwrite Filtering

**Files**:

- `internal/template/generator/filter.go`
- `internal/template/generator/generator.go`
- `internal/template/generator/generator_test.go`
- `internal/template/model/types.go`

**Status**: COMPLETED

**Checklist**:

- [x] Protected existing paths are skipped in selective mode.
- [x] New unprotected paths are created in selective mode.
- [x] Existing unprotected paths are overwritten in selective mode.
- [x] Overwrite-all mode ignores `.ign-overwrite-ignore`.
- [x] Root `.ign-overwrite-ignore` is metadata; nested same-name files remain
      normal template files.
- [x] Matching covers anchored, directory, negation, nested, and `**` patterns.

### 3. Template Metadata and Hash Behavior

**Files**:

- `internal/app/template_update.go`
- `internal/app/template_update_test.go`
- `internal/template/generator/generator.go`
- `internal/template/generator/generator_test.go`

**Status**: COMPLETED

**Checklist**:

- [x] Hash changes when root `.ign-overwrite-ignore` changes.
- [x] Project output does not receive root `.ign-overwrite-ignore`.
- [x] Update reads ignore policy from the current fetched template.

### 4. Documentation and Progress

**Files**:

- `README.md`
- `design-docs/specs/selective-overwrite.md`
- `docs/progress/selective-overwrite.md`

**Status**: COMPLETED

**Checklist**:

- [x] Docs explain selective versus overwrite-all modes.
- [x] Docs explain confirmation preview and skip behavior.
- [x] Docs explain metadata hash participation and non-emission.

---

## Completion Criteria

- [x] Focused CLI/app/generator/template-update tests pass.
- [x] `go test ./...` passes.
- [x] `go build -o /dev/null ./...` passes.
- [x] `go vet ./...` passes.
- [x] `git diff --check` passes.

## Progress Log

### Session: 2026-05-14

**Tasks Completed**: Selective overwrite implementation, Divedra review loop,
documentation updates, and verification.

**Blockers**: The packaged `divedra` binary failed to load a runtime module, so
the Divedra review workflow was run from the local source checkout at
`../divedra`.

**Notes**: Accidental Divedra repository design/plan artifacts were removed and
their relevant content was incorporated here.
