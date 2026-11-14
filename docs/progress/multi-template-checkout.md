# Multi-Template Checkout

**Status**: Completed

## Spec Reference

- `design-docs/specs/design-multi-template-checkout.md`
- `impl-plans/multi-template-checkout.md`

## Implemented

- [x] Versioned project metadata fields and v2 template, variable, and manifest
      records (`internal/template/model/ignvar.go`,
      `internal/template/model/ignmanifest.go`).
- [x] Normalized v1/v2 project state loading and v2 saving
      (`internal/config/project_state.go`, `internal/config/loader.go`,
      `internal/config/validation.go`).
- [x] Atomic-enough v2 metadata save staging with rollback on partial replace
      failure (`internal/config/project_state.go`,
      `internal/config/project_state_test.go`).
- [x] Load-time v2 `output_root` validation rejects absolute, traversal, and
      non-normalized roots (`internal/config/project_state.go`,
      `internal/config/project_state_test.go`).
- [x] Checkout-created v2 metadata and add-template checkout with v1 migration
      (`internal/app/checkout.go`, `internal/cli/checkout.go`).
- [x] Managed overwrite allowance for paths already owned by earlier templates
      (`internal/template/generator/generator.go`).
- [x] Per-template manifest ownership and v2-aware rewind file loading
      (`internal/app/rewind.go`, `internal/config/project_state.go`).
- [x] Selective template removal, all-template removal, and ordered replay of
      remaining templates (`internal/app/remove.go`, `internal/cli/remove.go`,
      `internal/cli/root.go`).
- [x] V2-safe update save path that preserves template records rather than
      replacing v2 metadata with v1 files (`internal/app/update.go`).
- [x] V2 update processing across all template records in checkout order
      (`internal/app/update.go`).
- [x] V2 update manifest preservation for unchanged managed files during
      `SkipUnchanged` generation (`internal/app/update.go`,
      `internal/app/multi_template_checkout_test.go`).
- [x] Per-template update variable-change tracking and CLI prompting for later
      template records (`internal/app/update.go`, `internal/cli/update.go`).
- [x] V2 output-root validation for rewind and switch full-removal flows
      (`internal/app/rewind.go`,
      `internal/app/multi_template_checkout_test.go`).
- [x] Remove replay protects unmanaged skipped files while preserving managed
      overwrite precedence (`internal/app/remove.go`,
      `internal/app/multi_template_checkout_test.go`).
- [x] Add-template checkout without `[output-path]` reuses stored `output_root`
      and rejects explicit mismatched output roots
      (`internal/app/project_state.go`, `internal/app/checkout.go`,
      `internal/cli/checkout.go`, `internal/app/multi_template_checkout_test.go`).
- [x] Regression tests for add-template checkout, v1 migration, and restoring a
      hidden file when removing a later template
      (`internal/app/multi_template_checkout_test.go`).
- [x] Regression tests for skipped unmanaged files not becoming managed and for
      all-record v2 update (`internal/app/multi_template_checkout_test.go`).
- [x] README command examples and behavior notes for multi-template checkout and
      remove (`README.md`).
- [x] Step 8 documentation refresh aligned README and examples with accepted
      multi-template checkout behavior (`README.md`, `examples/README.md`).

## Remaining

- [ ] Expand tests for dry-run migration reporting, unknown/multiple/all remove
      IDs, and direct switch-from-v2 CLI coverage.

## Design Decisions

- New checkout writes version 2 metadata even for the first template.
- Missing metadata version is normalized as version 1.
- V2 metadata uses stable IDs derived from template name plus monotonic checkout
  order, such as `alpha-1`.
- V2 save keeps top-level v1-compatible fields populated from the first template
  record while storing authoritative v2 data under `templates`.
- Output roots are stored as normalized relative paths from the project state
  root; absolute command paths inside the project root are converted to
  relative paths before saving.
- Removing selected templates deletes their managed files first, then replays
  remaining templates in order to restore files hidden by later overwrites.
- Replay uses the managed-path set from remaining template records as the only
  allowed overwrite set, so files skipped during checkout remain user-owned.
- Add-template checkout treats omitted `[output-path]` as the stored
  `output_root` when `.ign` already exists; explicit mismatches still fail.
- V2 update keeps unchanged managed paths in the per-template manifest when
  generation skips writing identical content, while still avoiding ownership of
  skipped unmanaged files.
- Rewind validates the requested output path against the stored v2 output root
  before deleting generated files or `.ign`.

## Notes

- Commands run:
  - `gofmt -w internal/app internal/config internal/cli internal/template/model internal/template/generator`
  - `gofmt -w internal test`
  - `GOCACHE=/private/tmp/ign-go-cache go test ./...`
  - `GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign`
  - `GOCACHE=/private/tmp/ign-go-cache go vet ./...`
  - `GOCACHE=/private/tmp/ign-go-cache task test`
  - `GOCACHE=/private/tmp/ign-go-cache task build`
  - `git diff --check`
- Step 8 documentation refresh:
  - `README.md` now documents shared v2 `output_root` behavior and the
    per-template `ign-files.json` shape.
  - `examples/README.md` now notes that version 2 project metadata stores
    variables and generated files per checked-out template.
- Step 6 review found and fixed two correctness issues:
  - v2 update could drop unchanged managed files from the manifest when at
    least one sibling file was written during the same template update.
  - v2 rewind did not reject an explicit output path that differed from the
    stored output root before full removal.
- Step 7 review found and Step 6 rerun fixed one correctness issue:
  - `SaveProjectState` previously replaced the three metadata files one at a
    time; the rerun now stages all files, backs up the existing set, restores
    backups on replace failure, and covers the failure with a regression test.
- Step 7 rerun found and this Step 6 rerun fixed one correctness issue:
  - v2 load-time `output_root` validation accepted absolute, traversal, or
    non-normalized roots when all metadata files agreed on the same invalid
    value; the rerun now rejects those roots and covers them with regression
    tests.
