# ign Template Examples

This directory contains example templates demonstrating various ign features.

## Examples

### 01-no-variables
Simple static template with no variables. Demonstrates basic file scaffolding.

**Features demonstrated:**
- `@ign-comment:` directive (removed from output)
- Static file generation

### 02-with-variables
Go web server template with configurable variables.

**Features demonstrated:**
- `@ign-var:` directive for variable substitution
- Variables in `ign.json` definition
- `ign-var.json` user configuration
- String, int variable types

### 03-conditionals
Go service template with feature flags using conditionals.

**Features demonstrated:**
- `@ign-if:` / `@ign-endif:` conditional blocks
- `@ign-else@` alternative blocks
- Boolean variables for feature toggles
- Conditional imports and code sections

### 04-includes
Go project template with shared reusable content.

**Features demonstrated:**
- `@ign-include:` directive for file inclusion
- Absolute paths (from template root): `@ign-include:/_includes/file.txt@`
- Shared license headers across files
- `@ign-raw:` to show literal directive syntax

## Directive Reference

### Variable Syntax

| Syntax | Required | Example |
|--------|----------|---------|
| `@ign-var:NAME@` | Yes | `@ign-var:app_name@` |
| `@ign-var:NAME:TYPE@` | Yes | `@ign-var:port:int@` |
| `@ign-var:NAME=DEFAULT@` | No | `@ign-var:host=localhost@` |
| `@ign-var:NAME:TYPE=DEFAULT@` | No | `@ign-var:port:int=8080@` |

Variables WITHOUT default are **required** (must be in ign-var.json).
Variables WITH default are **optional** (use default if not provided).

### Other Directives

| Directive | Purpose | Example |
|-----------|---------|---------|
| `@ign-comment:TEXT@` | Template comment (removed) | `@ign-comment:TODO: fix this@` |
| `@ign-if:VAR@` | Conditional start | `@ign-if:enable_auth@` |
| `@ign-else@` | Conditional else | `@ign-else@` |
| `@ign-endif@` | Conditional end | `@ign-endif@` |
| `@ign-include:PATH@` | Include file content | `@ign-include:/_includes/header.txt@` |
| `@ign-raw:CONTENT@` | Output literal content | `@ign-raw:@ign-var:x@@` |

## Variable Types

| Type | JSON | Example | Usage |
|------|------|---------|-------|
| `string` | `"value"` | `"my-app"` | Text values |
| `int` | `123` | `8080` | Numeric values |
| `bool` | `true/false` | `true` | Feature flags (conditionals) |

## File Naming Convention

Template files that contain ign directives use the `.tmpl` suffix:
- `main.go.tmpl` - Go template file (becomes `main.go` after generation)
- `go.mod.tmpl` - Go mod template file

This convention prevents syntax errors from linters and IDEs that expect valid source code.

## Usage

```bash
# Initialize from a template
ign init github.com/example/ign-templates/02-with-variables

# Edit variables
vim .ign-config/ign-var.json

# Generate files
ign checkout .
```
