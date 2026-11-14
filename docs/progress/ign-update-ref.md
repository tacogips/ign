# ign update --ref

**Status**: Completed

## Spec Reference

- `design-docs/specs/ign-update-ref.md`
- `impl-plans/ign-update-ref.md`

## Implemented

- [x] Added shared git ref validation (`internal/app/ref_validation.go`)
- [x] Added `ign update --ref` / `-r` CLI flag (`internal/cli/update.go`)
- [x] Threaded target refs through update preparation (`internal/app/update.go`)
- [x] Persisted requested refs on successful non-dry-run completion, including identical-content retargets (`internal/app/update.go`)
- [x] Added unit and integration tests (`internal/app/update_test.go`, `internal/cli/update_test.go`, `test/integration/update_ref_test.go`)
- [x] Updated command documentation (`README.md`)

## Remaining

- [ ] None

## Design Decisions

- Centralize ref validation in `internal/app` and keep the CLI wrapper delegated to it.
- Use a config-only completion path for identical-content retargets when neither overwrite nor force requests regeneration.
- Preserve dry-run semantics by leaving `.ign/ign.json` unchanged.

## Notes

- Local integration fixtures verify persisted refs and dry-run behavior. Remote provider ref resolution remains covered by provider-level behavior.
