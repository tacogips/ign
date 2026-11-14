# ign

A simple project scaffolding tool. Download templates from GitHub and generate projects with variable substitution.

## Why ign?

- **Simple workflow**: Initialize, configure, generate
- **GitHub-first**: Templates live in GitHub repos
- **No escaping headaches**: `@ign-var:NAME@` syntax avoids conflicts with any programming language
- **One-shot generation**: Files are generated once and fully owned by you

## Quick Start

```bash
# Initialize from a template and generate into the current directory
ign checkout github.com/owner/templates/go-basic .

# Add another template to the same project
ign checkout github.com/owner/templates/lint .

# Remove one checked-out template by ID
ign remove lint-2

# Remove all ign-managed files and metadata
ign rewind

# Replace the checked-out templates with a new template
ign switch github.com/owner/templates/another .
```

## Commands

### Global Flags

These flags apply to all commands:

| Flag | Description |
|------|-------------|
| `--no-color` | Disable colored output |
| `--quiet`, `-q` | Suppress non-error output |
| `--debug` | Enable debug output |

### `ign init <url-or-path>`

Initialize configuration from a template source.

```bash
# From GitHub
ign init github.com/owner/repo
ign init github.com/owner/repo/path/to/template
ign init github.com/owner/repo --ref v1.0.0

# From local path
ign init ./my-local-template
ign init /absolute/path/to/template
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--ref` | `-r` | Git branch, tag, or commit SHA (default: main) |
| `--force` | `-f` | Backup existing config and reinitialize |

**Behavior:**

| Condition | Action |
|-----------|--------|
| `.ign/` does not exist | Create `.ign/ign-var.json` |
| `.ign/` exists | Do nothing (skip) |
| `.ign/` exists + `--force` | Backup existing config, then reinitialize |

**Backup naming:** When `--force` is used, existing `ign-var.json` is backed up as `ign-var.json.bk1`, `ign-var.json.bk2`, etc.

```bash
# Force reinitialize with backup
ign init github.com/owner/repo --force

# Result:
# .ign/
#   ign-var.json       <- New config
#   ign-var.json.bk1   <- Previous config
```

### `ign checkout <url-or-path> [output-path]`

Initialize from a template and generate project files to the specified path.
When `.ign` already exists, `ign checkout` adds another template to the same
project state instead of replacing the existing template.

```bash
ign checkout github.com/owner/repo              # Generate to current directory
ign checkout github.com/owner/repo ./my-project # Generate to specific directory
ign checkout ./local-template sub_dir           # Generate from local template
ign checkout github.com/owner/lint .            # Add a second template to an existing .ign
ign checkout github.com/owner/repo . --dry-run  # Preview without writing files
ign checkout github.com/owner/repo . --verbose  # Show detailed processing info
```

New checkout metadata is written with `"version": 2`. Older `.ign` files without
a version are treated as version 1 and continue to work. Adding a second
template to a version 1 checkout migrates `.ign/ign.json`, `.ign/ign-var.json`,
and `.ign/ign-files.json` to version 2. All template records in one `.ign`
state share the same stored `output_root`; adding a template with an explicit
different output path is rejected unless `--force` is used to replace the
existing state.

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Overwrite existing files |
| `--dry-run` | `-d` | Show what would be generated without writing |
| `--verbose` | `-v` | Show detailed processing information |
| `--ref` | `-r` | Git branch, tag, or commit SHA (default: main) |

**File handling:**

| Condition | Action |
|-----------|--------|
| File does not exist | Create |
| File exists | Skip (do not overwrite) |
| File exists + `--force` | Overwrite |
| File is managed by an earlier checked-out template | Later checkout may overwrite it |

After a successful checkout, ign stores per-template metadata in `.ign/ign.json`,
per-template variables in `.ign/ign-var.json`, and per-template generated file
ownership in `.ign/ign-files.json`. Template IDs are derived from the template
name and checkout order, for example `go-basic-1` and `lint-2`.

### `ign update [output-path]`

Fetch the checked-out template again and regenerate project files when the template hash has changed.

```bash
ign update
ign update ./my-project
ign update --dry-run
ign update --overwrite
ign update --overwrite --yes
ign update --overwrite-all
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--overwrite` | `-o` | Overwrite existing files except paths matched by the remote template's `.ign-overwrite-ignore` |
| `--overwrite-all` | | Overwrite all existing files |
| `--force` | `-f` | Regenerate even if the hash is unchanged and overwrite all existing files |
| `--yes` | `-y` | Skip the overwrite confirmation prompt |
| `--dry-run` | `-d` | Preview what would be generated without writing |
| `--verbose` | `-v` | Show detailed processing information |

When `--overwrite` or `--overwrite-all` is used without `--yes`, `ign update` displays files that will be written before prompting:

```text
A new-file.txt
M existing-file.txt
```

Existing files whose generated content and permissions are unchanged are omitted from the confirmation list.

For version 2 metadata, `ign update` checks every checked-out template in
checkout order, prompts for new variables per template, preserves existing
per-template records, and keeps unchanged managed files in the manifest.

Template authors can add `.ign-overwrite-ignore` to the template root to protect user-owned files during selective overwrite. The file uses gitignore-style patterns and is included in the template hash.

```gitignore
config/
.env
*.local
!config/default.yaml
```

### `ign remove <template-id>...`

Remove one or more checked-out templates from the local project.

```bash
ign remove go-basic-1
ign remove go-basic-1 lint-2
ign remove --all
```

Removing a template deletes files owned by that template, removes its metadata,
and replays the remaining templates in checkout order. If a later template had
overwritten a file from an earlier template, removing the later template restores
the earlier file during replay.

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | | Remove all checked-out templates and delete `.ign` |
| `--output` | `-o` | Project output path; must match the stored `.ign` output root |
| `--verbose` | `-v` | Show detailed replay information |

### `ign rewind [output-path]`

Remove files previously created by ign and delete `.ign/`.

```bash
ign rewind
ign rewind ./my-project
```

If `.ign/ign-files.json` exists, ign uses it directly. Otherwise it falls back to the
currently checked-out template and variables to infer the managed files.
For version 2 metadata, the requested output path must match the stored
`output_root`.

### `ign switch <url-or-path> [output-path]`

Replace the current checked-out template with a new one.

```bash
ign switch github.com/owner/new-template
ign switch ./new-local-template ./my-project
```

`ign switch` is equivalent to `ign rewind` followed by `ign checkout`, so it
removes the current checked-out templates before saving the replacement
template.

### `ign template check [PATH]`

Validate template files for syntax errors.

```bash
ign template check              # Check current directory
ign template check ./templates  # Check specific directory
ign template check -r           # Recursive check
ign template check -r -v        # Recursive with verbose output
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--recursive` | `-r` | Recursively check subdirectories |
| `--verbose` | `-v` | Show detailed validation info |

### `ign version`

Show version information.

```bash
ign version          # Full version info
ign version --short  # Version number only
ign version --json   # JSON format output
```

## Configuration Directory

`.ign/` contains:

```
.ign/
  ign.json             # Template reference, content hash, and checkout records
  ign-files.json       # Files created by ign, grouped by template in version 2
  ign-var.json         # User variable values, grouped by template in version 2
  license-header.txt   # Optional files for @file: references
```

New metadata files include `"version": 2`. Missing version fields are loaded as
legacy version 1 metadata for backward compatibility.

### ign.json (Template Reference)

```json
{
  "version": 2,
  "output_root": ".",
  "template": {
    "url": "github.com/owner/templates/go-basic",
    "ref": "main"
  },
  "hash": "sha256:e3b0c44298fc1c149...",
  "templates": [
    {
      "id": "go-basic-1",
      "order": 1,
      "template": {
        "url": "github.com/owner/templates/go-basic",
        "ref": "main"
      },
      "hash": "sha256:e3b0c44298fc1c149..."
    }
  ]
}
```

### ign-var.json (User Variables)

```json
{
  "version": 2,
  "output_root": ".",
  "variables": {
    "app_name": "my-app",
    "port": 8080,
    "debug": false
  },
  "templates": [
    {
      "id": "go-basic-1",
      "variables": {
        "app_name": "my-app",
        "port": 8080,
        "debug": false
      }
    }
  ]
}
```

### ign-files.json (Generated Files)

```json
{
  "version": 2,
  "output_root": ".",
  "files": [
    "README.md",
    "cmd/app/main.go"
  ],
  "templates": [
    {
      "id": "go-basic-1",
      "order": 1,
      "files": [
        "README.md",
        "cmd/app/main.go"
      ]
    }
  ]
}
```

## Template Syntax

```go
package main

const AppName = "@ign-var:app_name@"
const Port = @ign-var:port:int=8080@       // optional, default 8080
const Debug = @ign-var:debug:bool=false@   // optional, default false

func main() {
    @ign-if:enable_logging@
    log.Println("Starting...")
    @ign-endif@
}
```

### Variable Syntax

| Syntax | Required | Description |
|--------|----------|-------------|
| `@ign-var:NAME@` | Yes | Basic variable |
| `@ign-var:NAME:TYPE@` | Yes | With type validation |
| `@ign-var:NAME=DEFAULT@` | No | With default value |
| `@ign-var:NAME:TYPE=DEFAULT@` | No | With type and default |

**Types:** `string`, `int`, `bool`

String default values support `{current_dir}` as a placeholder for the output directory name.

```json
{
  "variables": {
    "project_name": {
      "type": "string",
      "default": "{current_dir}"
    },
    "module_path": {
      "type": "string",
      "default": "github.com/acme/{current_dir}"
    }
  }
}
```

### Other Directives

| Directive | Usage |
|-----------|-------|
| `@ign-if:VAR@...@ign-endif@` | Conditional block (bool) |
| `@ign-include:PATH@` | Include another file |
| `@ign-raw:CONTENT@` | Output literally (escape) |
| `@ign-comment:TEXT@` | Template-only comment (removed) |

## Private Repos

```bash
# Using gh CLI (recommended)
gh auth login
ign checkout github.com/private/repo

# Or via environment variable
export GITHUB_TOKEN=<github-token>
```

## Installation

```bash
# From source
go install github.com/tacogips/ign@latest
```

## License

MIT
