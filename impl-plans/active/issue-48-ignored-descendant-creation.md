# Issue 48 Ignored Descendant Creation Implementation Plan

**Status**: Completed
**Design Reference**: `design-docs/specs/selective-overwrite.md#ignored-path-creation-boundary`
**Created**: 2026-08-11
**Last Updated**: 2026-08-11

---

## Design Document Reference

**Source**: `design-docs/specs/selective-overwrite.md`

### Summary

Issue [#48](https://github.com/tacogips/ign/issues/48) requires
`ign update --overwrite` to apply `.ign-overwrite-ignore` before both creation
and overwrite. A generated path matched by the remote template's ignore file, or
by an ignored directory ancestor such as `src/`, `Sources/`, or `Tests/`, must
remain absent when missing, remain unchanged when present, and must not be added
to `.ign/ign-files.json`.

### Scope

**Included**:

- Selective overwrite generation filtering in `internal/template/generator/`.
- Manifest persistence behavior through existing app-layer save paths in
  `internal/app/manifest.go`.
- Regression coverage for ignored directories, missing descendants, existing
  ignored paths, symlinks, dry-run, and update manifest tracking.
- Progress tracking for issue 48 under `docs/progress/`.

**Excluded**:

- Changes to `--overwrite-all` and `--force`; they continue to bypass
  `.ign-overwrite-ignore`.
- Placeholder creation, post-generation deletion, or manual manifest edits to
  hide generated ignored files.
- Broad rewrites of stale cleanup, issue 45 symlink transition handling, or
  unrelated release packaging.

### Reference Trace

- **Workflow mode**: issue-resolution.
- **Issue reference**: `tacogips/ign#48`.
- **Codex-agent references**:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`
- **Primary repository modules**:
  - `internal/template/generator/generator.go`
  - `internal/template/generator/filter.go`
  - `internal/template/generator/generator_ignored_descendants_test.go`
  - `internal/app/update.go`
  - `internal/app/manifest.go`
  - `internal/app/update_issue48_ignored_descendants_test.go`

No intentional divergence from the accepted design is planned.

---

## Modules

### 1. Generator Protected-Path Decision

#### `internal/template/generator/generator.go`

**Status**: COMPLETED

```go
type OverwriteMode string

const (
	OverwriteNone      OverwriteMode = "none"
	OverwriteSelective OverwriteMode = "selective"
	OverwriteAll       OverwriteMode = "all"
)

type GenerateResult struct {
	CreatedFiles []string
	WrittenFiles []string
	FilesSkipped int
	Files        []string
	DryRunFiles  []DryRunFile
}
```

**Checklist**:

- [x] Apply the selective overwrite ignore decision before classifying missing
      regular files as creates.
- [x] Apply the same decision before classifying missing symlinks as creates.
- [x] Treat paths under ignored directory patterns as protected descendants.
- [x] Record protected paths as skipped, not created or written.
- [x] Keep protected paths out of `CreatedFiles`, `WrittenFiles`, and generated
      manifest inputs.
- [x] Preserve `OverwriteAll`, `OverwriteNone`, and `SkipUnchanged` behavior.

### 2. Shared Ignore Matching Boundary

#### `internal/template/generator/filter.go`

**Status**: COMPLETED

```go
func MatchesGitIgnorePattern(path string, patterns []string) bool
```

**Checklist**:

- [x] Reuse the existing gitignore-style matcher for creation and overwrite
      decisions.
- [x] Confirm directory patterns protect all descendants after path
      normalization.
- [x] Avoid adding a second matcher or app-layer duplicate policy.

### 3. App Manifest Integration

#### `internal/app/manifest.go`

**Status**: COMPLETED

```go
func saveManifestFromGenerateResultWithResult(
	path string,
	result *generator.GenerateResult,
	excludedCanonicalPaths map[string]struct{},
) (config.AtomicWriteResult, error)
```

**Checklist**:

- [x] Verify skipped protected paths are absent from `GenerateResult` write
      lists consumed by manifest persistence.
- [x] Add or adjust app-layer regression coverage only if generator filtering
      alone does not prove manifest behavior.
- [x] Do not manually remove issue 48 paths from the manifest after generation.

### 4. Regression Tests

#### `internal/template/generator/generator_ignored_descendants_test.go`

**Status**: COMPLETED

```go
func TestGenerateSelectiveOverwriteSkipsMissingIgnoredDescendants(t *testing.T)
func TestDryRunSelectiveOverwriteSkipsMissingIgnoredDescendants(t *testing.T)
```

#### `internal/app/update_issue48_ignored_descendants_test.go`

**Status**: COMPLETED

```go
func TestCompleteUpdate_SelectiveOverwriteDoesNotCreateMissingIgnoredDescendants(t *testing.T)
```

**Checklist**:

- [x] Cover ignored directory patterns for `src/`, `Sources/`, and `Tests/`.
- [x] Cover missing ignored regular files and missing ignored symlinks.
- [x] Cover existing ignored files remaining unchanged.
- [x] Cover dry-run and write path agreement.
- [x] Assert `.ign/ign-files.json` excludes skipped protected paths.
- [x] Assert `--overwrite-all` still creates the same paths.

### 5. Progress Documentation

#### `docs/progress/selective-overwrite.md`

**Status**: COMPLETED

**Checklist**:

- [x] Add issue 48 status, spec reference, implemented items, remaining work,
      design decisions, and notes.
- [x] Keep unrelated existing worktree changes untouched.

---

## Module Status

| Module | File Path | Status | Tests |
|--------|-----------|--------|-------|
| Generator protected-path decision | `internal/template/generator/generator.go` | COMPLETED | `internal/template/generator/generator_ignored_descendants_test.go` |
| Shared ignore matching boundary | `internal/template/generator/filter.go` | COMPLETED | `internal/template/generator/generator_ignored_descendants_test.go` |
| App manifest integration | `internal/app/manifest.go` | COMPLETED | `internal/app/update_issue48_ignored_descendants_test.go` |
| Regression tests | `internal/template/generator/generator_ignored_descendants_test.go`, `internal/app/update_issue48_ignored_descendants_test.go` | COMPLETED | focused Go tests |
| Progress documentation | `docs/progress/selective-overwrite.md` | COMPLETED | review only |

## Dependencies

| Task | Depends On | Status |
|------|------------|--------|
| Generator protected-path decision | Accepted design | COMPLETED |
| Shared ignore matching boundary | Generator decision | COMPLETED |
| App manifest integration | Generator result semantics | COMPLETED |
| Regression tests | Generator and app behavior decisions | COMPLETED |
| Progress documentation | Final implementation behavior | COMPLETED |

## Task Breakdown

1. Update generator selective-overwrite classification so protected paths are
   skipped before creation or overwrite.
2. Verify shared matcher behavior for ignored directory ancestors and
   descendants.
3. Confirm manifest persistence only tracks actually written generated paths.
4. Add focused generator regression tests for skipped missing descendants and
   dry-run/write agreement.
5. Add app-layer update regression tests proving ignored descendants remain
   absent and absent from `.ign/ign-files.json`.
6. Refresh issue 48 progress documentation.
7. Run focused and full Go verification.

## Parallelization

| Task | Parallelizable | Reason |
|------|----------------|--------|
| Generator implementation | No | It defines the result semantics used by tests and manifest checks. |
| Matcher verification | No | It shares the generator write scope. |
| App manifest tests | No | Depends on generator result semantics. |
| Progress documentation | Yes, after behavior is known | Disjoint write scope from Go files. |

## Completion Criteria

- [x] `ign update --overwrite` skips matched missing files and symlinks.
- [x] Descendants under ignored `src/`, `Sources/`, and `Tests/` directories
      remain absent when absent.
- [x] Existing ignored paths remain unchanged.
- [x] Skipped protected paths are not added to `.ign/ign-files.json`.
- [x] Dry-run and confirmation preview use the same protected-path decision as
      the write path.
- [x] `--overwrite-all` and `--force` behavior is unchanged.
- [x] Regression tests cover generator and app-layer manifest behavior.
- [x] `docs/progress/selective-overwrite.md` records issue 48 progress.
- [x] Verification commands pass.

## Verification

```bash
go test ./internal/template/generator -run 'SelectiveOverwrite|IgnoredDescendant'
go test ./internal/app -run 'SelectiveOverwrite|IgnoredDescendant'
go test ./...
go build -o /dev/null ./...
go vet ./...
git diff --check
```

## Progress Log

### Session: 2026-08-11 21:34 JST

**Tasks Completed**: Created issue 48 implementation plan from accepted design
review.

**Tasks In Progress**: None.

**Blockers**: None.

**Notes**: Existing unrelated worktree changes were present in `.ign/ign.json`,
`scripts/render-homebrew-formula.sh`, and `.riela/sessions/`; implementation
must avoid reverting or incorporating them.

### Session: 2026-08-11 22:03 JST

**Tasks Completed**: Implemented selective-overwrite pre-creation filtering for
regular files and symlinks in `internal/template/generator/generator.go`.
Added focused generator regressions in
`internal/template/generator/generator_ignored_descendants_test.go` and
app-layer manifest regressions in
`internal/app/update_issue48_ignored_descendants_test.go`. Updated
`docs/progress/selective-overwrite.md`.

**Tasks In Progress**: Final verification.

**Blockers**: None.

**Notes**: The accepted design remains aligned: filtering occurs before
filesystem mutation and before manifest persistence, using the existing
gitignore-style matcher and without placeholders, post-generation deletion, or
manual manifest edits.

### Session: 2026-08-11 22:24 JST

**Tasks Completed**: Passed focused generator and app regression tests, the
full uncached Go test suite, compile check, vet, gofmt check, and diff hygiene.
All touched Go files remain below 1000 lines.

**Tasks In Progress**: None.

**Blockers**: None.

**Notes**: The mandatory independent Go verifier completed without modifying
the worktree.
