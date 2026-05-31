# Selective Overwrite

## Overview

`ign update` supports selective overwrite so template authors can protect
user-owned generated paths while still allowing safe template updates.

The template root may define `.ign-overwrite-ignore`. The file uses
gitignore-style patterns against generated output paths. During
`ign update --overwrite`, existing output paths matched by this file are skipped.

## Command Behavior

- `ign update --overwrite` performs selective overwrite.
- `ign update --overwrite-all` preserves the previous overwrite-all behavior.
- `ign update --force` regenerates even when the template hash is unchanged and
  uses overwrite-all semantics.
- `ign update --overwrite --yes` and `ign update --overwrite-all --yes` skip the
  overwrite confirmation prompt.
- `--yes` affects confirmation only. It does not alter overwrite mode, hash
  comparison, or dry-run behavior.
- Without `--yes`, a writing update in overwrite mode previews planned changes
  before mutation using `A` for new files, `M` for overwritten existing files,
  and `D` for removed managed files.
- In overwrite mode, update removes files recorded in `.ign/ign-files.json` when
  the current template no longer generates them. Selective overwrite preserves
  removed template paths matched by `.ign-overwrite-ignore`.

## Template Metadata

- `.ign-overwrite-ignore` is recognized only at the remote template root.
- Root `.ign-overwrite-ignore` is template metadata and is not emitted into
  generated project output.
- Nested files named `.ign-overwrite-ignore` are normal template files.
- Root `.ign-overwrite-ignore` is included in template hash calculation so
  overwrite-policy changes are detected by `ign update`.
- Update reads overwrite-ignore policy from the fetched remote template, not from
  any local project copy.

## Matching Rules

Selective overwrite matching uses normalized slash-separated generated paths.

The supported gitignore-style behavior includes:

- anchored patterns such as `/config/app.yaml`
- directory patterns such as `config/`
- basename patterns such as `.env`
- glob patterns such as `*.local`
- recursive `**` patterns
- negation ordering such as `config/` followed by `!config/default.yaml`

## Validation

The dry-run/confirmation preview path and final write path must share the same
template rendering, file filtering, overwrite mode, and ignore-pattern decisions
so the displayed plan matches the eventual mutation.
