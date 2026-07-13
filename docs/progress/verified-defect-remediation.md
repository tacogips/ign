# Verified Defect Remediation

**Status**: Completed

## Spec Reference

- `design-docs/specs/architecture.md` Section: Verified Defect Remediation
- `design-docs/specs/command.md` Section: Defect-Fix CLI Semantics
- `impl-plans/active/verified-defect-remediation.md`

## Implemented

- [x] HIGH-1 update transaction ordering and partial-cleanup manifest persistence (`internal/app/update.go`, `internal/app/update_test.go`)
- [x] HIGH-2 atomic JSON persistence helper for config, manifest, variable, and template JSON writes (`internal/config/loader.go`, `internal/app/template_update.go`)
- [x] HIGH-3 rewind missing-manifest fallback content ownership check (`internal/app/rewind.go`, `internal/app/rewind_test.go`)
- [x] HIGH-4 quiet mode keeps error output visible (`internal/cli/root.go`, `internal/cli/cli_test.go`)
- [x] HIGH-5 checkout existing `.ign` without `--force` returns an error (`internal/cli/checkout.go`, `internal/cli/checkout_test.go`)
- [x] HIGH-6 relative include resolution from the including template file (`internal/template/generator/processor.go`, `internal/template/parser/include.go`)
- [x] MID-1 same-line raw directives parse independently (`internal/template/parser/directive.go`, `internal/template/parser/parser_test.go`)
- [x] MID-2 GitHub `/tree/<branch>` and trailing `.git` URL parsing (`internal/template/provider/url.go`, `internal/template/provider/provider_test.go`)
- [x] MID-3 include and GitHub archive extraction containment hardening (`internal/template/parser/include.go`, `internal/template/provider/github.go`)
- [x] MID-4 single root error-reporting path and no `os.Exit` inside `RunE` (`internal/cli/*.go`)
- [x] MID-5 switch delays `.ign` setup and supports `--var/-V` (`internal/cli/switch.go`, `internal/cli/switch_test.go`)
- [x] MID-6 update rejects malformed template hashes (`internal/app/update.go`, `internal/app/update_test.go`)
- [x] MID-7 invalid interactive regex patterns fail before prompting (`internal/cli/prompt.go`, `internal/cli/prompt_test.go`)

## Remaining

- None.

## Design Decisions

- Update uses the existing checkout rollback snapshot machinery for generated writes and artifact-save failures.
- Cleanup partial failures are treated as applied updates after successful generation; the manifest is saved before returning the cleanup error so successful writes and deletions are not lost.
- Rewind fallback deletes only files whose current bytes match generated dry-run bytes.
- CLI handlers return errors to the root command; template validation details remain printed before returning a single validation error.
- GitHub archive extraction rejects target paths and symlink targets that resolve outside the extraction directory.

## Notes

- No new third-party dependencies were added.
- Focused package verification passed with `go test ./internal/config ./internal/app ./internal/cli ./internal/template/generator ./internal/template/parser ./internal/template/provider`.
- Final verification passed with `gofmt`, `go vet ./...`, `go build ./...`, and `go test ./...`.
