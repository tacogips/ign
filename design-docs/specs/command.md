# Command Interface

## Non-Interactive Template Variables

`ign init` and one-shot `ign checkout <url-or-path> [output-path]` support a
repeatable variable option:

```bash
ign init ./template --var project_name=my-app --var port=8080
ign checkout ./template ./out --var project_name=my-app --var port=8080
```

The short form is `-V`:

```bash
ign checkout ./template ./out -V project_name=my-app -V enable_feature=true
```

### Design

- The option surface is `--var key=value`, repeatable with last value winning.
- Values are parsed after template preparation, so assignments can be validated
  against `ign-template.json` variable definitions.
- Unknown variable names fail early to catch typos.
- `.ign` creation and force-mode backup are deferred until after variable
  parsing and prompting succeed, so invalid non-interactive variables do not
  modify existing configuration.
- `ign checkout --dry-run` does not create or back up `.ign`.
- String values preserve everything after the first `=`.
- `int`, `number`, and `bool` values are converted to their configured Go types.
- String patterns and numeric min/max constraints are enforced for supplied
  values.
- Provided variables are passed into the prompt layer before interactive
  collection. Variables supplied by option are not prompted; missing variables
  keep the existing interactive fallback.
- The shared prompt layer must preflight missing variables before invoking
  `survey/v2`. When stdin is not an interactive terminal and any variable would
  still require prompting, the command fails with a controlled actionable error
  instead of calling survey.
- Fully supplied non-interactive runs remain valid: if `--var` values resolve
  every template variable that requires input, `ign init`, one-shot
  `ign checkout`, and `ign switch` must continue without prompting even when
  stdin is redirected.
- The non-TTY failure must occur before `.ign` creation, force-mode backup, or
  switch replacement so redirected prompt failures leave the workspace
  unchanged.
- `ign init` saves supplied values into `.ign/ign-var.json` without generating
  project files.
- `ign checkout` uses supplied values for both saved `.ign/ign-var.json` and
  project generation.

### Redirected Stdin Prompt Failures

Issue [#41](https://github.com/tacogips/ign/issues/41) records a crash where
`ign checkout` panics inside `survey/v2` when unresolved template variables are
prompted while stdin is redirected. The command contract is:

- `checkout`, `init`, and `switch` share one prompt boundary for unresolved
  template variables.
- Prompt code must classify unresolved variables before any survey prompt is
  constructed or executed.
- If unresolved variables exist and stdin is not a TTY, the command returns a
  normal error that explains interactive prompts require a TTY and that missing
  values can be supplied with repeatable `--var key=value` / `-V key=value`.
- Empty lines from redirected stdin are not treated as answers in this defect
  fix. Declared defaults remain interactive prompt defaults, and scripted
  execution should use explicit `--var` assignments.
- Existing TTY prompt behavior, validation, defaults, and help text remain
  unchanged.
- Regression coverage must exercise the shared prompt boundary directly and at
  least the command paths for `checkout` and `switch`; `init` must be covered
  either directly or through the same shared prompt helper.

### Current Behavior Finding

Before this design, the CLI had no registered `ign init` command and one-shot
`ign checkout` always called the interactive prompt helper for all template
variables. There was no CLI option for non-interactive variable assignment.

## Defect-Fix CLI Semantics

This section records CLI behavior required by the verified defect remediation
work in `design-docs/specs/architecture.md`.

### Error Output

- `--quiet` suppresses non-error output only.
- Cobra-level and command-returned errors must always print to stderr.
- Each failure should be reported once. Command `RunE` handlers should return
  errors and avoid printing the same error that `Execute` will print.
- `RunE` handlers must not call `os.Exit`; they should return errors so deferred
  cleanup and the root error path remain consistent.

### Checkout Existing Configuration

- `ign checkout <url-or-path> [output-path]` must return a non-zero exit when
  `.ign` already exists and `--force` is absent.
- The behavior should match `ign init`: report that configuration already exists
  and return an error instead of succeeding after informational output.

### Switch Variables And Setup Timing

- `ign switch <url-or-path> [output-path]` supports repeatable
  `--var key=value` and short `-V key=value` with the same parsing and
  validation semantics as `ign checkout`.
- `ign switch` must prepare and validate the new template without creating or
  backing up `.ign` before variable parsing and prompting complete.
- If variable parsing, validation, or prompting fails, switch leaves the current
  `.ign` state untouched.

### GitHub Tree URL Ambiguity

- `https://github.com/owner/repo/tree/<branch>` with no subdirectory resolves to
  `owner/repo` at ref `<branch>`.
- Branch names containing slashes are ambiguous in `/tree/` URLs. Users must use
  `--ref` for those refs instead of relying on path parsing.
