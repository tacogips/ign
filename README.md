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
vim .ign-config/ign-var.json

# 3. Generate project
ign checkout .              # Current directory
ign checkout ./my-project   # Specific directory
```

## Commands

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
ign init file://./relative-path
```

**Behavior:**

| Condition | Action |
|-----------|--------|
| `.ign-config/` does not exist | Create `.ign-config/ign-var.json` |
| `.ign-config/` exists | Do nothing (skip) |
| `.ign-config/` exists + `--force` | Backup existing config, then reinitialize |

**Backup naming:** When `--force` is used, existing `ign-var.json` is backed up as `ign-var.json.bk1`, `ign-var.json.bk2`, etc.

```bash
# Force reinitialize with backup
ign init github.com/owner/repo --force

# Result:
# .ign-config/
#   ign-var.json       <- New config
#   ign-var.json.bk1   <- Previous config
```

### `ign checkout <path>`

Generate project files to the specified path using existing `.ign-config/`.

```bash
ign checkout .              # Generate to current directory
ign checkout ./my-project   # Generate to specific directory
ign checkout sub_dir        # Generate to subdirectory
```

**Requires:** `.ign-config/ign-var.json` must exist (run `ign init` first).

**File handling:**

| Condition | Action |
|-----------|--------|
| File does not exist | Create |
| File exists | Skip (do not overwrite) |
| File exists + `--force` | Overwrite |

## Configuration Directory

`.ign-config/` contains:

```
.ign-config/
  ign-var.json         # Template reference and variable values
  license-header.txt   # Optional files for @file: references
```

### ign-var.json

```json
{
  "template": {
    "url": "github.com/owner/templates/go-basic",
    "ref": "main"
  },
  "variables": {
    "app_name": "my-app",
    "port": 8080,
    "debug": false
  }
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
ign init github.com/private/repo

# Or via environment variable
export GITHUB_TOKEN=ghp_xxx
```

## Installation

```bash
# Using Nix
nix run github:tacogips/ign

# From source
go install github.com/tacogips/ign@latest
```

## License

MIT
