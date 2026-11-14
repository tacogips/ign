# Architecture

## Verified Defect Remediation

This section records the design contract for fixing the verified defects from
the full-codebase review of `ign` confirmed at commit `b4a6b19`.

### Scope

The workflow mode is `issue-resolution`. The change set targets:

- `internal/app`: update transaction ordering, rewind fallback ownership
  checks, update hash validation, and template metadata persistence.
- `internal/config`: crash-safe writes for `.ign` JSON files.
- `internal/cli`: error output, checkout exit status, switch setup timing, and
  prompt validation.
- `internal/template`: include resolution, raw directive parsing, GitHub URL
  parsing, archive extraction, and path containment.

All HIGH findings are mandatory. MID findings should be included when they fit
the same defect-fix change set without adding unrelated product behavior.

### Update Transaction Safety

`internal/app/update.go` must treat generation and removed-managed-file cleanup
as the mutation phase, and `.ign` persistence as the commit phase.

Design rules:

- `PrepareUpdate` must reject missing or malformed template hashes with the same
  SHA-256 validation contract used by checkout.
- `CompleteUpdate` must not write `.ign/ign.json` or `.ign/ign-var.json` before
  generation succeeds.
- Non-dry-run update must prepare rollback state before generation, using the
  existing checkout rollback machinery or an equivalent app-layer helper.
- If generation fails, generated files and overwritten files must be rolled back
  and the previous `.ign` state must remain unchanged.
- If stale managed-file cleanup partially fails, successfully generated files
  and successful deletions must still be reflected in `.ign/ign-files.json`.
  The returned error should describe cleanup failure without causing the next
  `ign update` to falsely report that the template is already up to date.
- If final `.ign` artifact persistence fails, file mutations from the update
  must roll back where rollback data is available.

### Crash-Safe Config Writes

All writes for `.ign/ign-var.json`, `.ign/ign-files.json`, `.ign/ign.json`, and
template-side `ign-template.json` must use one shared atomic write helper.

Design rules:

- The helper writes JSON bytes to a temporary file in the target directory.
- The helper fsyncs the temporary file before rename.
- The helper uses `os.Rename` to replace the target path atomically on the same
  filesystem.
- The helper cleans up temporary files on failure.
- Callers keep their existing validation, parent-directory creation, JSON
  marshaling, and error wrapping behavior.
- No environment variable values or machine-local paths may be persisted into
  config files, docs, or commit messages.

### Rewind Ownership Safety

`internal/app/rewind.go` may use current-template dry-run output only as a
fallback when `.ign/ign-files.json` is missing. That fallback does not prove that
all generated paths were actually written by `ign`.

Design rules:

- Manifest-backed rewind remains authoritative and unchanged.
- In the fallback path, `ign rewind` may delete a file only when the current
  on-disk content exactly matches the dry-run content generated for that path.
- If a candidate file exists but content differs from dry-run content, rewind
  must skip it and report it as not removed.
- Directory removal remains limited to empty parent directories after accepted
  file removals.
- Paths still pass the existing managed-path containment checks before any
  deletion.

### Template Processing And Provider Safety

Template processing must preserve documented template behavior while hardening
path handling.

Design rules:

- `internal/template/generator/processor.go` must set parser `CurrentFile` to
  the full template-root-relative filesystem path so relative
  `@ign-include:file@` directives resolve against the including file.
- `internal/template/parser/include.go` must use `filepath.Rel` style
  containment, equivalent to the provider local path helper, rather than a weak
  string prefix check.
- `internal/template/parser/directive.go` must parse multiple raw directives on
  one line independently. A raw directive closes at the nearest valid `@@`
  terminator for that raw directive.
- `internal/template/provider/url.go` must parse branch-only
  `https://github.com/owner/repo/tree/<branch>` URLs as `Ref=<branch>` with an
  empty template path, and must trim a trailing `.git` suffix for all GitHub URL
  forms.
- Slashed branch names in `/tree/` URLs remain ambiguous by URL shape and should
  require explicit `--ref`; this is documented behavior, not a parser guess.
- `internal/template/provider/github.go` must reject archive entries whose
  cleaned target escapes the extraction root.
- GitHub archive symlink entries must be rejected when their link target is
  absolute or resolves outside the extraction root.

### CLI Error And Prompt Behavior

CLI error behavior is specified in `design-docs/specs/command.md`.

Prompt validation design rules:

- Invalid variable regex patterns must be compiled before interactive prompting
  begins.
- A template variable with an invalid regex pattern fails immediately with an
  error identifying the variable, rather than entering a re-prompt loop.
- Non-interactive `--var` validation keeps the existing fail-fast behavior.

### Regression Coverage

Each implemented finding must receive regression coverage in the package closest
to the behavior:

- `internal/app` tests for transactional update order, manifest persistence
  after cleanup failure, rewind fallback content matching, and update hash
  validation.
- `internal/config` tests for atomic-write success and failure behavior where it
  can be exercised without process-kill tests.
- `internal/cli` tests for quiet error output, checkout existing-config error,
  switch delayed `.ign` setup, switch `--var`, single error reporting, and
  invalid prompt regex fail-fast behavior.
- `internal/template` tests for relative include resolution, same-line raw
  directives, GitHub URL parsing, archive path containment, symlink rejection,
  and include path containment.

Required verification commands:

```bash
gofmt
go vet ./...
go build ./...
go test ./...
```

### Constraints

- Follow existing Go conventions and keep the layered structure:
  `cli -> app -> config/template`.
- Do not add third-party dependencies.
- Do not change public CLI behavior except where the verified findings require
  it: checkout error exit and `ign switch --var/-V`.
- Preserve existing behavior except where a finding identifies a defect.
