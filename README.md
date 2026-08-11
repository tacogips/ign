# ign

A simple project scaffolding tool. Download templates from GitHub and generate projects with variable substitution.

## Why ign?

- **Simple workflow**: Initialize, configure, generate
- **GitHub-first**: Templates live in GitHub repos
- **No escaping headaches**: `@ign-var:NAME@` syntax avoids conflicts with any programming language
- **One-shot generation**: Files are generated once and fully owned by you

## Quick Start

```bash
# 1. Initialize from a template
ign init github.com/owner/templates/go-basic

# 2. Edit variables
vim .ign/ign-var.json

# 3. Generate project
ign checkout .              # Current directory
ign checkout ./my-project   # Specific directory
ign rewind                  # Remove ign-managed files
ign switch ./another-template
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
ign init github.com/owner/repo --var app_name=my-app --var port=8080

# From local path
ign init ./my-local-template
ign init /absolute/path/to/template
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--ref` | `-r` | Git branch, tag, or commit SHA (default: main) |
| `--force` | `-f` | Backup existing config and reinitialize |
| `--var` | `-V` | Set a template variable as `key=value` (repeatable) |

**Behavior:**

| Condition | Action |
|-----------|--------|
| `.ign/` does not exist | Create `.ign/ign-var.json` |
| `.ign/` exists | Return an error |
| `.ign/` exists + `--force` | Backup existing config, then reinitialize |

**Backup naming:** When `--force` is used, existing `ign-var.json` is backed up as `ign-var.json.bk1`, `ign-var.json.bk2`, etc.

Variables supplied with `--var` are written to `.ign/ign-var.json`. Missing
variables are still prompted interactively. If stdin is redirected or otherwise
non-interactive, missing variables fail with an error instead of prompting; pass
all required values with repeatable `--var key=value` or `-V key=value`.

```bash
# Force reinitialize with backup
ign init github.com/owner/repo --force

# Result:
# .ign/
#   ign-var.json       <- New config
#   ign-var.json.bk1   <- Previous config
```

### `ign checkout <url-or-path> [output-path]`

Initialize configuration and generate project files from a template in one step.

```bash
ign checkout github.com/owner/repo              # Generate to current directory
ign checkout github.com/owner/repo ./my-project # Generate to specific directory
ign checkout ./my-local-template ./my-project   # Generate from local template
ign checkout github.com/owner/repo --dry-run    # Preview without writing files
ign checkout github.com/owner/repo --verbose    # Show detailed processing info
ign checkout github.com/owner/repo --var app_name=my-app --var port=8080
```

If `.ign/` already exists, checkout returns an error unless `--force` is used.
Variables supplied with `--var` are used for generation and saved to
`.ign/ign-var.json`. Missing variables are still prompted interactively. If
stdin is redirected or otherwise non-interactive, missing variables fail with an
error instead of prompting; pass all required values with repeatable
`--var key=value` or `-V key=value`.

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Overwrite existing files |
| `--dry-run` | `-d` | Show what would be generated without writing |
| `--verbose` | `-v` | Show detailed processing information |
| `--var` | `-V` | Set a template variable as `key=value` (repeatable, one-shot checkout) |

**File handling:**

| Condition | Action |
|-----------|--------|
| File does not exist | Create |
| File exists | Skip (do not overwrite) |
| File exists + `--force` | Overwrite |

After a successful checkout, ign stores the created file list in `.ign/ign-files.json`.

### `ign vars`

Inspect template variables and the current values stored in `.ign/ign-var.json`.

```bash
ign vars
ign vars --json
ign vars --unset
ign vars --json --unset
```

The table output includes `NAME`, `TYPE`, `REQUIRED`, `DEFAULT`, `CURRENT`, and
`DESCRIPTION`. JSON mode prints the same rows with unset counts for scripts.

`--unset` shows only variables without a current value. If any known required
variable is unset, the command exits with code `1`; otherwise it exits with code
`0`.

If template declarations cannot be fetched, `ign vars` falls back to local
`.ign/ign-var.json` values and prints a warning outside JSON stdout.

### `ign update [output-path]`

Fetch the checked-out template again and regenerate project files when the template hash has changed.

```bash
ign update
ign update ./my-project
ign update --dry-run
ign update --overwrite
ign update --overwrite --yes
ign update --overwrite-all
ign update --ref v2.0.0
ign update --ref v2.0.0 --dry-run
ign update --ref v2.0.0 --overwrite --yes
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
| `--ref` | `-r` | Retarget the tracked template branch, tag, or commit SHA |

`ign update --ref <ref>` fetches the stored template URL and path at the
requested ref, then uses the normal update flow and overwrite protections. On
successful non-dry-run completion, `.ign/ign.json` is updated to pin the
requested ref. If the new ref has identical template content, ign still persists
the requested ref and leaves generated files unchanged unless overwrite or force
options request regeneration. Dry runs never change the stored ref.

When `--overwrite` or `--overwrite-all` is used, `ign update` also removes project files recorded in `.ign/ign-files.json` when the current template no longer generates them. The manifest is pruned after removal, and stale manifest entries for files that are already missing are pruned without reporting a deletion. Selective overwrite does not remove paths matched by `.ign-overwrite-ignore`.

When `--overwrite` or `--overwrite-all` is used without `--yes`, `ign update` displays files that will change before prompting:

```text
A new-file.txt
M existing-file.txt
D obsolete-file.txt
```

Existing files whose generated content and permissions are unchanged are omitted from the confirmation list.

Template authors can add `.ign-overwrite-ignore` to the template root to protect user-owned files during selective overwrite. The file uses gitignore-style patterns and is included in the template hash.

```gitignore
config/
.env
*.local
!config/default.yaml
```

### `ign rewind [output-path]`

Remove files previously created by ign and delete `.ign/`.

```bash
ign rewind
ign rewind ./my-project
```

If `.ign/ign-files.json` exists, ign uses it directly. Otherwise it falls back to the
currently checked-out template and variables to infer the managed files. During
that fallback, ign removes only files whose current content matches what the
template would generate and skips files with different user-owned content.

### `ign switch <url-or-path> [output-path]`

Replace the current checked-out template with a new one.

```bash
ign switch github.com/owner/new-template
ign switch ./new-local-template ./my-project
ign switch github.com/owner/new-template --var app_name=my-app --var port=8080
```

`ign switch` is equivalent to `ign rewind` followed by `ign checkout`.
It supports the same repeatable `--var`/`-V` variable assignment syntax as
`ign checkout`. Template preparation, variable validation, and prompting happen
before the current `.ign/` directory is replaced, so failed input does not leave
a partial replacement configuration. When stdin is non-interactive, unresolved
variables fail before replacement; provide every required value with
`--var key=value` or `-V key=value` for scripted use.

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--ref` | `-r` | Git branch, tag, or commit SHA (default: main) |
| `--force` | `-f` | Overwrite existing files when applying the new template |
| `--verbose` | `-v` | Show detailed processing information |
| `--var` | `-V` | Set a template variable as `key=value` (repeatable) |

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
  ign.json             # Template reference and content hash
  ign-files.json       # Files created by ign
  ign-var.json         # User variable values
  license-header.txt   # Optional files for @file: references
```

### ign.json (Template Reference)

```json
{
  "template": {
    "url": "github.com/owner/templates/go-basic",
    "ref": "main"
  },
  "hash": "sha256:e3b0c44298fc1c149..."
}
```

### ign-var.json (User Variables)

```json
{
  "variables": {
    "app_name": "my-app",
    "port": 8080,
    "debug": false
  }
}
```

## Template Settings

Template authors can tune generation through the optional `settings` block in the
template's `ign-template.json`:

```json
{
  "settings": {
    "preserve_executable": true,
    "ignore_patterns": [".git", ".DS_Store"],
    "binary_extensions": [".png", ".jpg"]
  }
}
```

| Setting | Default | Description |
|---------|---------|-------------|
| `preserve_executable` | `true` | Keep the executable bit of template files in generated projects |
| `ignore_patterns` | none | Glob patterns for files excluded from generation |
| `binary_extensions` | built-in list | Extensions copied without variable substitution |

Omitting a setting (or the whole `settings` block) applies its default. Set
`"preserve_executable": false` explicitly to write every generated file as `0644`.
Projects generated before this default was fixed can be repaired with
`ign update --overwrite --yes`.

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
| `@ign-include:PATH@` | Include another file, resolved relative to the including file and contained within the template root |
| `@ign-raw:CONTENT@` | Output literally (escape) |
| `@ign-comment:TEXT@` | Template-only comment (removed) |

## GitHub Template URLs

ign accepts GitHub URLs in shorthand, HTTPS, SSH, and `.git` forms. URLs such
as `https://github.com/owner/repo/tree/branch-name` select `branch-name` as the
template ref when no subdirectory is present. Branch names that contain `/` are
ambiguous in `/tree/` URLs; use `--ref` for those refs.

## Private Repos

```bash
# Using gh CLI (recommended)
gh auth login
ign init github.com/private/repo

# Or via environment variable
export GITHUB_TOKEN=ghp_xxx
```

## Installation

```bash
# Using Homebrew
brew tap tacogips/tap
brew install ign
ign version

# From source
go install github.com/tacogips/ign@latest
```

The Homebrew formula is maintained in `tacogips/homebrew-tap` and installs the
latest published GitHub Release artifact for the current platform.

## License

MIT
