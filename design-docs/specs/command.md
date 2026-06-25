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
- `ign init` saves supplied values into `.ign/ign-var.json` without generating
  project files.
- `ign checkout` uses supplied values for both saved `.ign/ign-var.json` and
  project generation.

### Current Behavior Finding

Before this design, the CLI had no registered `ign init` command and one-shot
`ign checkout` always called the interactive prompt helper for all template
variables. There was no CLI option for non-interactive variable assignment.
