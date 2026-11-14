# Add `ign update --ref <ref>` Flag

Feature-local design for `ign-update-ref`, adding non-destructive template ref
retargeting to the existing `ign update` flow.

## Feature Contract

- Workflow mode: `issue-resolution`
- Issue reference: `workflowInput.issueBody FEATURE 2: 'ign update --ref <ref>' - retarget the tracked branch/tag`
- Feature id: `ign-update-ref`
- Feature title: Add `ign update --ref <ref>` flag
- Declared design document: `design-docs/specs/ign-update-ref.md`
- Implementation plan target: `impl-plans/ign-update-ref.md`
- Target areas: `internal/cli`, `internal/app`
- Codex agent references:
  - `.agents/agents/go-coding.md`
  - `.agents/agents/go-check-and-test-after-modify.md`

## Summary

`ign update --ref <ref>` lets a checked-out project retarget the template branch,
tag, or commit recorded in `.ign/ign.json` without using the destructive
`ign switch` flow. The command keeps the configured template URL and path,
fetches the template at the requested ref, compares hashes, and then uses the
normal update protections for generation, overwrite handling, dry runs, and
rollback-safe artifact persistence.

Without `--ref`, `ign update` must continue using the stored
`.ign/ign.json.template.ref` with no behavior changes.

## CLI Behavior

Add a string `--ref` flag to `internal/cli/update.go`; use short `-r` for
consistency with `ign checkout --ref` and `ign switch --ref`.

Supported examples:

```bash
ign update --ref v2.0.0
ign update --ref release/2026.07 --dry-run
ign update --ref v2.0.0 --overwrite --yes
ign update ./my-project --ref v2.0.0 --force
```

Design rules:

- `--ref` composes with `--dry-run`, `--overwrite`, `--overwrite-all`, `--yes`,
  and `--force`.
- The CLI validates a supplied ref before calling `app.PrepareUpdate`, so
  invalid refs fail before provider or network access.
- The CLI threads the requested ref through `app.UpdateOptions`.
- Output shows the effective ref when `--ref` is supplied, including `main`
  when the user explicitly requested it.
- If the requested ref resolves to identical template content, output notes that
  content is identical while still allowing the pin update on successful
  non-dry-run completion.
- `--dry-run --ref <ref>` fetches and previews against the requested ref but
  leaves `.ign/ign.json`, `.ign/ign-var.json`, and `.ign/ign-files.json`
  unchanged.

## App-Layer Behavior

Extend `app.UpdateOptions` with an optional ref override field, for example
`TargetRef string`.

`PrepareUpdate` remains responsible for loading current `.ign` state, resolving
the provider, fetching the template, validating the template hash, comparing
hashes, and identifying variable changes.

Design rules:

- Load `.ign/ign.json` and `.ign/ign-var.json` before fetching, matching current
  update behavior and missing-configuration errors.
- Preserve the configured template URL and path; `--ref` changes only
  `.ign/ign.json.template.ref`.
- Validate `TargetRef` in `internal/app` as a defense for non-CLI callers
  without importing `internal/cli`.
- Apply the ref override to the in-memory `model.TemplateRef` before provider
  fetch and hash comparison.
- Keep previous config state available for rollback before mutating the loaded
  config object.
- Add preparation result fields for requested ref, previous ref, effective ref,
  whether a ref override was requested, whether the stored ref changes, and
  whether content hash changes.
- Continue using the fetched template's `ign-template.json` hash as
  `PrepareUpdateResult.NewHash`.

## Persistence And Transaction Safety

`CompleteUpdate` remains the non-dry-run path that commits update artifacts. The
feature must preserve the transaction ordering used by update: generation and
removed-file cleanup happen before `.ign` artifact writes, and artifact writes
use snapshots so failures can restore previous state.

Design rules:

- On successful non-dry-run completion, persist both `.ign/ign.json.hash =
  NewHash` and `.ign/ign.json.template.ref = requested ref`.
- If generation or cleanup fails before artifact persistence, keep the previous
  ref in `.ign/ign.json`.
- If saving `.ign/ign.json`, `.ign/ign-var.json`, or `.ign/ign-files.json`
  fails, rollback restores the previous ref with the previous hash, variables,
  and manifest.
- When the requested ref is different but the content hash is identical, skip
  project-file regeneration and persist the requested ref through a
  configuration-only commit path that still uses rollback-safe artifact
  snapshots.
- Do not require `--overwrite` or `--force` to persist an identical-content ref
  retarget.
- Dry runs never persist the requested ref.

## Regeneration Decision

Existing update regeneration remains based on hash changes, `--force`, or
overwrite mode. `--ref` adds one separate completion reason:
`refChanged && !dryRun`.

When only the ref changed and the hash is identical, completion should update
configuration without rewriting project files or replacing the managed-file
manifest with an empty generation result.

## Test Design

Unit tests:

- `internal/cli/update_test.go`: registers `--ref` and `-r`.
- `internal/cli/update_test.go`: parses `--ref` and forwards it through update
  options.
- `internal/cli/update_test.go`: invalid refs fail before app update execution.
- `internal/app/update_test.go`: `PrepareUpdate` fetches the requested ref and
  preserves stored URL and path.
- `internal/app/update_test.go`: invalid `TargetRef` is rejected in the app
  layer.
- `internal/app/update_test.go`: identical-hash ref retarget persists
  `.ign/ign.json.template.ref` without rewriting project files.
- `internal/app/update_test.go`: dry-run ref retarget leaves `.ign/ign.json`
  unchanged.
- `internal/app/update_test.go`: save failure rolls back the previous ref.
- `internal/app/update_test.go`: absent `TargetRef` preserves existing behavior.

Integration tests:

- `test/integration`: update a checked-out fixture with `--ref` to a newer
  template version and verify generated files plus `.ign/ign.json.template.ref`.
- `test/integration`: update with `--ref` where content hash is unchanged and
  verify the ref is persisted while generated files are not rewritten.
- `test/integration`: update with `--ref --dry-run` and verify preview output
  plus unchanged config.
- `test/integration`: update with `--ref --overwrite --yes` and verify existing
  overwrite protections, including `.ign-overwrite-ignore`, still apply.

Required verification commands:

```bash
gofmt -w <modified-go-files>
go vet ./...
go build ./...
go test ./...
```

## Decisions

- Reuse `ign update` instead of adding a new command because retargeting is an
  update concern and must compose with update protections.
- Keep retargeting non-destructive; it must not call the `ign switch` or
  `ign rewind` flows.
- Persist the requested ref even when content is identical so `.ign/ign.json`
  records the user's requested pin.
- Validate refs in both CLI and app layers, centralizing the rule in a neutral
  package if that avoids drift.
- Keep dry-run strictly preview-only.

## Addressed Feedback

- No upstream review feedback or mailbox payloads were attached to this branch
  execution.
- The design uses the declared fanout feature contract for `ign-update-ref`
  rather than the similarly named historical design documents already present.

## Open Questions

- None.

## Risks

- Identical-content ref retargeting needs a configuration-only commit path; if
  it bypasses artifact snapshots, rollback guarantees can regress.
- Ref validation can drift if duplicated between CLI and app layers.
- Local provider fixtures cannot fully emulate remote git ref resolution, so
  integration tests should separate persisted-config assertions from remote
  ref-switching behavior.
