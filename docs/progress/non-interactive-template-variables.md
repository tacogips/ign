# Non-Interactive Template Variables

**Status**: Completed

## Spec Reference

- `design-docs/specs/command.md` Section "Non-Interactive Template Variables"

## Implemented

- [x] Added repeatable `--var` / `-V` parsing for `key=value` template variables (`internal/cli/variables.go`)
- [x] Added prompt fallback that skips variables already supplied by option (`internal/cli/prompt.go`)
- [x] Added `--var` support to one-shot checkout (`internal/cli/checkout.go`)
- [x] Added registered `ign init` command with `--var` support (`internal/cli/init.go`, `internal/cli/root.go`)
- [x] Added app-layer init completion that stores provided variables without generation (`internal/app/config_init.go`, `internal/app/workflows.go`)
- [x] Added unit coverage for parsing, malformed assignments, no-prompt supplied values, and init variable map overlay (`internal/cli/variables_test.go`, `internal/app/app_test.go`)
- [x] Deferred `.ign` creation/backups until after variable parsing and prompting succeed (`internal/app/checkout.go`, `internal/cli/init.go`, `internal/cli/checkout.go`)
- [x] Added checkout preflight validation for runtime variable loading and template hashes before `.ign` is created or backed up, and reused the validated runtime inputs after force backups (`internal/app/variables.go`, `internal/app/app_test.go`, `internal/cli/checkout.go`, `internal/cli/checkout_test.go`)
- [x] Added integration coverage for side-effect-free template preparation (`test/integration/config_init_test.go`)
- [x] Documented user-facing usage (`README.md`)
- [x] Added non-TTY prompt preflight in the shared variable prompt boundary so
  unresolved variables fail before calling `survey/v2` (`internal/cli/prompt.go`)
- [x] Added regression coverage for non-interactive missing variables and fully
  supplied `--var`/`-V` flows across `checkout`, `init`, and `switch`
  (`internal/cli/prompt_test.go`, `internal/cli/checkout_test.go`,
  `internal/cli/init_test.go`, `internal/cli/switch_test.go`)
- [x] Documented that scripted `checkout`, `init`, and `switch` runs must
  provide required variables with `--var key=value` / `-V key=value`
  (`README.md`)

## Remaining

- [ ] None

## Design Decisions

- Use repeatable `--var key=value` instead of JSON input to match common CLI
  conventions and keep simple shell usage readable.
- Reject unknown variables after template preparation so typos fail before any
  generation happens.
- Preserve interactive prompts for missing variables to avoid breaking the
  existing guided workflow.
- Do not consume redirected empty lines as default answers; non-interactive
  missing-variable runs return a controlled error and the supported scripted path
  is explicit `--var`/`-V` assignment.

## Notes

- `ign init` is now registered in the CLI because the app layer and README
  already described it, but the command was not wired into `rootCmd`.
- Issue #41 remediation verifies `checkout`, `init`, and `switch` fail before
  `.ign` creation, backup/replacement, or output generation when unresolved
  variables require prompts without a TTY.
- Verification run during issue #41 implementation:
  `go test ./internal/cli -run 'TestPrompt|TestCheckout|TestSwitch|TestInit'`,
  `go test ./...`, `go build -o /dev/null ./...`, and `go vet ./...`.
