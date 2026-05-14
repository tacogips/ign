# Selective Overwrite

**Status**: Completed

## Spec Reference

- User request: add selective overwrite for `ign update` using template-side `.ign-overwrite-ignore`

## Implemented

- [x] Added `.ign-overwrite-ignore` as template metadata (`internal/template/model/types.go`)
- [x] Added selective overwrite mode and gitignore-style matching (`internal/template/generator/`)
- [x] Wired `ign update --overwrite`, `--overwrite-all`, and `--yes` (`internal/cli/update.go`)
- [x] Included `.ign-overwrite-ignore` changes in template hash calculation (`internal/app/template_update.go`)
- [x] Documented update overwrite behavior (`README.md`)

## Remaining

- [ ] None

## Design Decisions

- `--overwrite` performs selective overwrite and respects the remote template's `.ign-overwrite-ignore`.
- `--overwrite-all` preserves the previous overwrite-all behavior.
- `--force` remains the explicit regenerate option and uses overwrite-all semantics.

## Notes

- `.ign-overwrite-ignore` is not generated into project output.
