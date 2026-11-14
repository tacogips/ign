# Multi-Template Checkout

Design for supporting multiple checked-out templates in one project while
preserving existing version 1 `.ign` state.

## Overview

`ign checkout` currently records one template in `.ign/ign.json`, one variable
map in `.ign/ign-var.json`, and one flat managed-file list in
`.ign/ign-files.json`. The new design treats missing metadata versions as
version 1, writes new checkout metadata as version 2, and stores an ordered list
of template checkout records.

Version 2 keeps checkout order as the source of precedence. Later templates win
when multiple templates generate the same output path. Removing templates
updates metadata and replays the remaining templates in order so paths that were
hidden by later overwrites are restored.

## Issue Mapping

- Issue title: Support multiple checked-out templates
- Active workflow review issue: Self-review and improve multi-template checkout
  implementation
- Workflow mode: issue-resolution
- Source: `runtimeVariables.workflowInput`
- Remote issue inspection: not attempted because no issue URL, repository, or
  issue number was provided.
- Feature area: ign checkout metadata, variable management, and template removal

## Review-Gate Alignment

The current workflow pass must treat this document and
`impl-plans/multi-template-checkout.md` as accepted references. The remaining
work is review-driven: inspect the current repository diff, identify high or
mid correctness issues, apply targeted code or documentation fixes, and rerun
verification before the implementation self-review and independent review gates
accept the result.

Review should not restart the feature from scratch. It should preserve the
existing v2 metadata direction unless a correctness defect proves that a local
adjustment is required.

Review blockers are high or mid issues that can corrupt project state, delete or
overwrite unmanaged files, collapse v2 metadata to v1, violate output-root
boundaries, make template ID selection ambiguous, or leave documented command
behavior inconsistent with implementation. Remaining test-expansion items may be
tracked as follow-up only when the behavior is already covered by lower-level
tests or explicitly accepted by the review gates.

## Reference Mapping

- `AGENTS.md`: repository rules, Go conventions, path hygiene, and workflow
  expectations.
- `.agents/agents/go-coding.md`: implementation agent contract for later Go
  changes.
- `.agents/agents/go-check-and-test-after-modify.md`: verification agent
  contract after Go changes.
- `../../codex-agent`: unavailable in this worktree. No product behavior is
  copied from it.

## Repository Module Mapping

The design is local to the current ign repository. Step 4 should plan changes
across these module boundaries:

- `internal/template/model/ignconfig.go`: template-side `IgnJson` remains the
  source for template name, template version, variable definitions, settings,
  and template hash. Checkout and update should copy the fetched template name,
  version, and hash from this model into each v2 project checkout record.
- `internal/template/model/ignvar.go`: project-side `.ign/ign.json` and
  `.ign/ign-var.json` models need explicit metadata versioning plus v2 ordered
  template records and per-template variables.
- `internal/template/model/ignmanifest.go`: `.ign/ign-files.json` needs
  versioned per-template file ownership records instead of one flat file list.
- `internal/config/loader.go`: project metadata loaders and savers should
  detect v1 missing-version files, validate v2 cross-file consistency, and
  expose a normalized project-state API to application code.
- `internal/app/checkout.go`: checkout should create first v2 state, add later
  template records to existing state, migrate v1 on second checkout, and apply
  managed-path overwrite precedence.
- `internal/app/manifest.go`: manifest writes should preserve per-template
  ownership and checkout order rather than sorting one global file list.
- `internal/app/variables.go`: variable preparation, validation, and `@file:`
  resolution should run against a selected template record's variable map.
- `internal/app/rewind.go`: full removal remains the backward-compatible all
  templates path; selective removal should reuse managed-path validation.
- `internal/app/update.go`: update should process normalized project state and
  preserve every v2 template record, variables record, and manifest record.
- `internal/app/template_update.go`: no project-state schema ownership, but
  template hash behavior remains relevant to each checked-out template record.
- `internal/cli/checkout.go`: stop treating existing `.ign` as an error when
  not forced; route existing state to add-template checkout.
- `internal/cli/rewind.go`: keep current full-removal CLI behavior.
- `internal/cli/root.go`: register the new selective removal command.
- `internal/cli/flags.go`: keep token/output flag handling consistent with
  checkout, update, and rewind.
- `internal/cli/update.go`: expose whichever multi-template update selector or
  all-record behavior Step 4 chooses, without collapsing state.

## Project Root and Output Root

A `.ign` directory defines one project state root and one output root for all
template records in that state. Multi-template checkout does not support mixing
template records from different output roots in one `.ign` directory.

The output root should be stored in version 2 metadata as a normalized relative
path from the `.ign` project root. For the existing default workflow this value
is `"."`.

When adding a later template:

- `ign checkout <url-or-path>` uses the stored output root when `.ign` exists.
- `ign checkout <url-or-path> [output-path]` must validate that `[output-path]`
  resolves to the stored output root before mutation.
- A different output path is rejected with a clear message directing the user to
  run checkout from the intended project root or use a separate project state.
- `--force` remains the only checkout mode that may replace existing `.ign`
  state and establish a new output root.

For removal, `ign remove --output PATH` is a compatibility convenience for the
same root check: `PATH` must resolve to the stored output root. It does not
select a different project state. `ign rewind [output-path]` follows the same
root validation for full removal.

## Versioned Metadata

All project-side `.ign` files written by new checkout operations use explicit
`"version": 2`. Loaders treat a missing version as version 1.

Version 1 remains the current single-template shape:

- `.ign/ign.json` contains `template`, `hash`, and optional `metadata`.
- `.ign/ign-var.json` contains a single `variables` map.
- `.ign/ign-files.json` contains a flat `files` array.

Version 2 uses ordered template records:

```json
{
  "version": 2,
  "output_root": ".",
  "templates": [
    {
      "id": "go-basic-1",
      "order": 1,
      "template": {
        "url": "github.com/owner/templates",
        "path": "go-basic",
        "ref": "main"
      },
      "hash": "64-hex-sha256",
      "metadata": {
        "generated_by": "ign checkout",
        "template_name": "go-basic",
        "template_version": "1.0.0",
        "ign_version": "..."
      }
    }
  ]
}
```

`.ign/ign-var.json` mirrors the template order and scopes variables by checkout
record:

```json
{
  "version": 2,
  "output_root": ".",
  "templates": [
    {
      "id": "go-basic-1",
      "variables": {
        "app_name": "demo"
      }
    }
  ]
}
```

`.ign/ign-files.json` records generated paths per template record and preserves
checkout order:

```json
{
  "version": 2,
  "output_root": ".",
  "templates": [
    {
      "id": "go-basic-1",
      "order": 1,
      "files": [
        "README.md",
        "cmd/app/main.go"
      ]
    }
  ]
}
```

The `id` is stable after checkout and is the user-facing selector for removal.
It should be derived from the template name when possible and suffixed with a
numeric checkout ordinal to avoid collisions. The `order` field is monotonic and
must not be renumbered when intermediate templates are removed.

## Loading and Migration

Loaders should expose a normalized project-state API that returns a version 2
view to the application layer. A version 1 load wraps the single `template`,
`hash`, variables map, and flat manifest as one template record.

Migration rules:

- Missing `version` means version 1.
- A version 1 checkout continues to work unchanged for update, checkout replay,
  and rewind paths when no second template is added.
- When `ign checkout` adds a template to a version 1 project, migrate all three
  `.ign` files to version 2 before saving the added template record.
- A migrated v1 project stores `output_root` as `"."` unless the loader can
  infer a stricter existing output root from command context.
- Migration must write the complete metadata set atomically enough that a failed
  write does not leave mixed v1/v2 files. Prefer temp-file writes plus rename,
  with backups only for user-visible recovery.
- Dry-run checkout should report that migration would occur but must not mutate
  `.ign`.

## Checkout Flow

When `.ign` does not exist, `ign checkout <url-or-path> [output-path]` creates
version 2 metadata with one template record and the normalized output root.

When `.ign` already exists and `--force` is not set, checkout adds another
template record instead of returning the current "configuration already exists"
message. It first validates that the requested output path matches the stored
output root, prompts for variables for only the new template, and then generates
that template's outputs.

Overwrite rules:

- Later checked-out templates overwrite files managed by earlier template
  records when they generate the same path.
- Files that are not managed by any existing template record remain protected
  unless the user explicitly uses the existing force behavior.
- `--force` keeps the existing replacement behavior: backup existing `.ign`
  files and reinitialize from the new template.

The checkout result should update `.ign/ign-files.json` with the new template
record's generated path list. The list is per-template ownership, not only the
current final winning paths.

## Variable Management

Variables are scoped by template checkout ID. Prompts, update-time new-variable
collection, `@file:` resolution, and validation operate against one template
record at a time.

Users can manage values directly in `.ign/ign-var.json` by locating the template
record ID. Later CLI improvements may add a dedicated variable command, but this
issue only requires the schema and application flows to support multiple
template variable maps.

## Removal Flow

Add `ign remove <template-id>... [--output PATH]` and
`ign remove --all [--output PATH]`.

`ign rewind [output-path]` remains backward-compatible full removal of all
checked-out templates and deletion of `.ign`.

Removal behavior:

1. Load and normalize project state.
2. Resolve requested template IDs. Unknown IDs fail before mutation.
3. Delete output paths generated by the removed template records, validating
   paths with the same managed-path protections used by rewind.
4. Remove the deleted template records from `ign.json`, `ign-var.json`, and
   `ign-files.json`.
5. Replay remaining templates in checkout order so the final workspace matches a
   fresh checkout of those remaining templates in that order.
6. If no templates remain, delete `.ign`.

Replay uses the stored template source, ref, path, hash, and per-template
variables. Later remaining templates retain precedence over earlier remaining
templates. Unmanaged files are not deleted or overwritten.

## Update and Switch Constraints

`ign update` must not silently collapse version 2 metadata to a single template.
The implementation should either update all template records in order or require
an explicit template ID selector. The first implementation may choose the all
records behavior to keep the command simple, but it must preserve each record's
variables, source, order, and manifest ownership.

`ign switch` remains equivalent to full removal followed by checkout. For version
2 state, that means removing all template records and then creating one new
version 2 record.

## Validation Rules

- `version` must be `1` or `2` when present.
- Version 2 `output_root` must be present, normalized, relative to the `.ign`
  project root, and consistent across `ign.json`, `ign-var.json`, and
  `ign-files.json`.
- Version 2 files must contain a non-empty `templates` array except during the
  final step of full removal, where `.ign` is removed instead of saved empty.
- Template IDs must be unique across `ign.json`, `ign-var.json`, and
  `ign-files.json`.
- Template records must be processed by ascending `order`.
- Every variable and manifest record must reference an existing template ID.
- Managed paths must stay inside the requested output directory and outside
  `.ign`.
- Hash values keep the existing SHA-256 validation rule.

## Rollout and Tests

Implementation should add focused tests for:

- Loading missing-version v1 metadata as a normalized single-template state.
- First checkout writing version 2 metadata.
- Adding a second template to v1 metadata and migrating all `.ign` files.
- Ordered overwrite precedence where later checkout wins.
- Per-template variable storage and loading.
- Removing a later template and restoring an earlier hidden file.
- Removing an earlier template while keeping a later overwrite.
- `ign remove --all` and `ign rewind` full removal behavior.
- `ign update` preserving multi-template metadata.

Verification commands for the implementation phase:

```bash
gofmt -w internal test
GOCACHE=/private/tmp/ign-go-cache go test ./...
GOCACHE=/private/tmp/ign-go-cache go build ./cmd/ign
GOCACHE=/private/tmp/ign-go-cache go vet ./...
GOCACHE=/private/tmp/ign-go-cache task test
GOCACHE=/private/tmp/ign-go-cache task build
git diff --check
```
