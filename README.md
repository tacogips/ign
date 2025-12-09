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
ign build init github.com/owner/templates/go-basic

# 2. Edit variables
vim .ign-build/ign-var.json

# 3. Generate project
ign init --output ./my-project
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

## Commands

| Command | Description |
|---------|-------------|
| `ign build init <url>` | Create `.ign-build/` config |
| `ign init` | Generate project |
| `ign init --overwrite` | Regenerate and overwrite |

## Template Sources

```bash
# GitHub
ign build init github.com/owner/repo
ign build init github.com/owner/repo/path/to/template
ign build init github.com/owner/repo --ref v1.0.0

# Local
ign build init ./my-local-template
```

## Private Repos

```bash
# Using gh CLI (recommended)
gh auth login
ign build init github.com/private/repo

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
