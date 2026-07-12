# ign vars

**Status**: Completed

## Spec Reference

- `design-docs/specs/ign-vars.md`
- `impl-plans/ign-vars.md`

## Implemented

- [x] Added read-only app-layer variable inspection (`internal/app/vars.go`)
- [x] Added shared tracked-template fetch helper (`internal/app/template_fetch.go`)
- [x] Added `ign vars`, `--json`, and `--unset` CLI support (`internal/cli/vars.go`, `internal/cli/root.go`)
- [x] Added unit and integration tests (`internal/app/vars_test.go`, `internal/cli/vars_test.go`, `test/integration/vars_test.go`)
- [x] Updated command documentation (`README.md`)

## Remaining

- [ ] None

## Design Decisions

- Treat key absence as unset; present `false`, `0`, empty string, and `nil` values remain current values.
- Keep declaration fetch failures non-fatal and return local-only rows.
- Route declaration warnings to stderr so JSON stdout remains parseable.

## Notes

- Offline fallback cannot identify missing required variables when declarations are unavailable.
