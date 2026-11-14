# Add `ign vars` Command

Feature-local design for issue-resolution work on FEATURE 1 from
`Add ign vars command and ign update --ref flag`.

## Feature Contract

- Workflow mode: `issue-resolution`
- Issue reference: `workflowInput.issueBody FEATURE 1: 'ign vars' - inspect
  template variables and current values`
- Feature id: `ign-vars`
- Feature title: Add `ign vars` command
- Design document path: `design-docs/specs/ign-vars.md`
- Implementation plan path: `impl-plans/ign-vars.md`
- Implementation target: `internal/cli`, `internal/app`, `internal/config`,
  `internal/template`
- Codex agent references: `.agents/agents/go-coding.md`,
  `.agents/agents/go-check-and-test-after-modify.md`

## Overview

`ign vars` gives project users a direct way to inspect the variables declared by
the tracked template and the current values stored in `.ign/ign-var.json`.
The command is read-only and follows the existing layered structure:
`internal/cli` handles flags and output formatting, `internal/app` loads project
state and prepares display data, and `internal/config` plus
`internal/template` continue to own JSON loading and template fetching.

## Command Surface

```bash
ign vars
ign vars --json
ign vars --unset
ign vars --json --unset
```

Flags:

- `--json`: print machine-readable variable rows as JSON.
- `--unset`: include only variables with no current value.

Output columns for table mode:

- `NAME`
- `TYPE`
- `REQUIRED`
- `DEFAULT`
- `CURRENT`
- `DESCRIPTION`

Global behavior:

- `--quiet` suppresses normal table, JSON, and warning output but must not
  suppress errors.
- `--no-color` is respected by using existing CLI output helpers and avoiding
  new color-specific formatting.
- When `.ign` configuration is missing or invalid, return an error consistent
  with `ign update` and `ign rewind` setup failures.

## App-Layer Contract

Add `internal/app/vars.go` with an app-facing query API that returns a stable
result independent of CLI formatting.

The app result should include:

- declaration availability, so the CLI can distinguish full template metadata
  from local-only fallback output;
- sorted variable rows for deterministic table, JSON, and test output;
- unset counts split by all known unset variables and required unset variables;
- a non-fatal fetch error when declarations are unavailable.

Each variable row should include:

- name;
- declared type, required flag, default, and description when a declaration is
  available;
- current value from `.ign/ign-var.json`;
- unset state computed from current value presence, not from default presence.

Unset semantics:

- A variable is unset when its key is absent from `.ign/ign-var.json`.
- Explicit `null`, `false`, `0`, or empty string values are still current values
  if they are stored.
- `--unset` filters to unset rows after declarations and current values are
  merged.
- `ign vars --unset` exits with code `1` only when at least one required known
  variable is unset; otherwise it exits `0`.

## Data Flow

1. Load `.ign/ign.json` from `.ign/ign.json`.
2. Load `.ign/ign-var.json` from `.ign/ign-var.json`.
3. Resolve and fetch the configured template source using the same provider path
   as update/checkout, including stored `template.url`, `template.ref`, and
   `template.path`.
4. Merge fetched `IgnJson.Variables` declarations with current values.
5. Include local-only rows for values present in `.ign/ign-var.json` but absent
   from template declarations.
6. If template fetch fails, return rows for local values only and include a
   declaration-unavailable warning in the result.

The local-only fallback is intentionally best-effort:

- it exits `0` for normal listing because no known declarations can prove a
  required value is missing;
- `--unset` can only report missing required variables when declarations are
  available;
- the CLI should print a concise warning unless `--quiet` is set.
- JSON output keeps warning text off stdout so scripts receive valid JSON.

## CLI Responsibilities

Add `internal/cli/vars.go` and register it from `internal/cli/root.go`.

The CLI layer should:

- parse `--json` and `--unset`;
- call the app-layer vars query;
- format table output with deterministic row order;
- format JSON output with the same fields represented by table mode plus unset
  state, current-value presence, and result counts;
- map app errors directly to Cobra `RunE` returns;
- return a sentinel or typed app result error for the required-unset condition
  so `Execute` exits non-zero without duplicate error printing.
- keep declaration-unavailable warnings on stderr or the existing warning
  channel so `--json` stdout remains valid JSON.

Recommended JSON shape:

```json
{
  "declarations_available": true,
  "variables": [
    {
      "name": "project_name",
      "type": "string",
      "required": true,
      "default": null,
      "current": "my-app",
      "has_current": true,
      "description": "Project name",
      "unset": false
    }
  ],
  "unset_count": 0,
  "required_unset_count": 0
}
```

## Boundaries And Reuse

- Do not add third-party dependencies.
- Reuse `config.LoadIgnConfig` and `config.LoadIgnVarJson`.
- Reuse provider resolution/fetch behavior already used by
  `app.PrepareUpdate`; avoid duplicating URL normalization semantics.
- Reuse existing variable default and validation concepts where they apply, but
  do not prompt, mutate `.ign`, generate files, or validate checkout readiness.
- Keep formatting helpers in `internal/cli/output.go` as the source of truth for
  quiet/error behavior.

## Tests

Unit coverage:

- `internal/app/vars_test.go`: successful declaration/current merge, sorted
  rows, missing current values, required-unset exit condition data, local-only
  fallback when template fetch fails, and missing `.ign` errors.
- `internal/cli/vars_test.go`: flag registration, table output, JSON output,
  `--unset` filtering, quiet suppression, and required-unset non-zero behavior.

Integration coverage:

- `test/integration`: checkout or seed a local template, run `ign vars`, run
  `ign vars --json`, and run `ign vars --unset` with both satisfied and missing
  required variables.

Documentation:

- Refresh `README.md` command documentation with `ign vars`, `--json`, and
  `--unset` examples.

Verification commands:

```bash
gofmt
go vet ./...
go build ./...
go test ./...
```

## Decisions

- `ign vars` is read-only and must not update `.ign` files or template hashes.
- Current values are only values explicitly stored in `.ign/ign-var.json`;
  defaults are displayed separately and do not make a variable "set".
- Offline or failed template fetch degrades to local value listing instead of a
  hard failure, because the command still provides useful inspection of local
  state.
- Missing declarations prevent required-unset failure because requiredness is
  unknown.
- Deterministic sorting by variable name is required for stable CLI output and
  tests.
- Warnings for unavailable declarations should not be emitted on JSON stdout.

## Open Questions

- None for this feature-local design; the issue body defines the command
  surface, fallback behavior, test scope, and documentation scope.

## Risks

- Provider fetch reuse may currently be embedded in `PrepareUpdate`; extracting
  a small shared helper may be needed to avoid duplicating ref/path override
  behavior.
- CLI JSON output must remain script-friendly; avoid colored prefixes or warning
  text on stdout when `--json` is selected.
- The required-unset exit path needs care so Cobra prints no noisy duplicate
  error while still returning exit code `1`.
