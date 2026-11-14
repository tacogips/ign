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
- [x] Omitted unchanged existing files from update overwrite confirmation and overwrite counts (`internal/template/generator/generator.go`)

## Remaining

- [ ] None

## Design Decisions

- `--overwrite` performs selective overwrite and respects the remote template's `.ign-overwrite-ignore`.
- `--overwrite-all` preserves the previous overwrite-all behavior.
- `--force` remains the explicit regenerate option and uses overwrite-all semantics.
- Existing files are only reported as overwrite targets when generated content or permissions differ.

## Notes

- `.ign-overwrite-ignore` is not generated into project output.
