# ign - System Design Document

## 1. Overview

### 1.1 Project Summary

| Item | Description |
|------|-------------|
| **Project Name** | ign (ignition) |
| **Purpose** | Download project templates from GitHub and initialize projects |
| **Reference** | Similar to Python's cookiecutter, but simpler and more flexible |
| **Distribution** | Single Go binary |
| **Supported Platforms** | macOS, Linux (Windows NOT supported) |

### 1.2 Design Goals

- Simple and flexible project scaffolding tool
- Unique template syntax that avoids escaping issues
- Single binary distribution (Go)
- GitHub-first with abstraction for future provider support
- DRY management of project meta files (`.envrc`, `.gitignore`, etc.)

### 1.3 Design Policy

| Policy | Description |
|--------|-------------|
| **One-shot generation** | Files are generated once, then fully owned by user |
| **No state tracking** | No lock files, no managed file concept |
| **Explicit overwrite** | Existing files are skipped by default, `--overwrite` required for replacement |
| **No implicit inputs** | Environment variables do not affect output |

---

## 2. Core Concepts

### 2.1 Ign Root (Template)

An **Ign Root** is a GitHub repository (or repository + subdirectory path) that contains:

1. `ign.json` - Configuration file for the template (NOT deployed to output)
2. Template files - All other files/directories that will be deployed with variable substitution

```
<ign-root>/
|-- ign.json             # Template configuration (not deployed)
|-- .envrc               # Template file (deployed)
|-- .claude/
|   +-- setting.json     # Template file (deployed)
|-- CLAUDE.md            # Template file (deployed)
+-- flake.nix            # Template file (deployed)
```

### 2.2 Build Directory (`.ign-build/`)

User-created directory containing build configuration:

```
<working-dir>/
+-- .ign-build/
    |-- ign-var.json           # Template reference + variables
    +-- license-header.txt     # Files for @file: references (optional)
```

### 2.3 Template Source Abstraction

While initially supporting only GitHub, the implementation must use an abstraction layer (interface) to allow future support for:

- GitLab
- Bitbucket
- Local filesystem
- Other Git hosting services

```
+---------------------------------------------+
|           TemplateProvider                  |
|              (Interface)                    |
+---------------------------------------------+
| + Fetch(url, ref) -> TemplateRoot           |
| + List() -> []Template                      |
| + Validate(url) -> bool                     |
+---------------------------------------------+
              ^                ^
              |                |
   +----------+----+    +------+--------+
   |GitHubProvider |    |Future Provider|
   |  (Initial)    |    |   (TBD)       |
   +---------------+    +---------------+
```

---

## 3. Configuration Files

### 3.1 ign.json (Template Configuration)

Located at the ign root. Defines template metadata and variables. **Not deployed to output.**

```json
{
  "name": "go ignition template",
  "description": "Go project with nix flake",
  "variables": {
    "PROJECT_NAME": {
      "type": "string",
      "optional": false,
      "description": "Name of the project"
    },
    "HTTP_PORT": {
      "type": "int",
      "optional": true,
      "default": "8080",
      "description": "HTTP server port"
    },
    "ENABLE_DOCKER": {
      "type": "bool",
      "optional": true,
      "default": "false"
    }
  }
}
```

#### Variable Definition Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Variable type: `string`, `int`, `bool` |
| `optional` | boolean | No | Whether variable can be omitted (default: false) |
| `default` | string | No | Default value if not provided |
| `description` | string | No | Human-readable description for prompts |

**Supported Types:**
- `string` - Text value
- `int` - Integer value
- `bool` - Boolean value (`true` / `false`)

Note: `array` and `object` types are NOT supported.

### 3.2 ign-var.json (User Build Configuration)

Located in `.ign-build/` directory. Contains template reference and user-provided variable values.

```json
{
  "template": {
    "url": "github.com/owner/templates",
    "path": "go-basic",
    "ref": "main"
  },
  "variables": {
    "PROJECT_NAME": "my-awesome-project",
    "MODULE_PATH": "github.com/myuser/my-project",
    "HTTP_PORT": "3000",
    "ENABLE_DOCKER": "true",
    "LICENSE_HEADER": "@file:license-header.txt"
  }
}
```

#### Template Reference Fields

| Field | Required | Description |
|-------|----------|-------------|
| `url` | Yes | GitHub repository URL |
| `path` | No | Subdirectory path within repository |
| `ref` | No | Branch, tag, or commit hash (default: repository's default branch) |

#### Reading Variable Values from Files

Variables can reference external files using the `@file:` prefix.
Paths are **relative to `.ign-build/` directory**.

```json
{
  "variables": {
    "LICENSE_HEADER": "@file:license-header.txt",
    "SSH_KEY": "@file:keys/deploy.pub"
  }
}
```

**File Path Rules:**
- Paths are relative to `.ign-build/` directory
- `..` is NOT allowed in paths (security restriction)
- Absolute paths are NOT allowed

### 3.3 ign-list.json (Template Collection)

Aggregates multiple ign.json locations for convenience.

```json
{
  "templates": [
    {
      "name": "go-basic",
      "url": "github.com/owner/repo",
      "path": "templates/go-basic"
    },
    {
      "name": "go-cli",
      "url": "github.com/owner/repo",
      "path": "templates/go-cli"
    },
    {
      "name": "go-web",
      "url": "github.com/other-owner/templates",
      "path": "go-web"
    }
  ]
}
```

### 3.4 config.json (Global Configuration)

Located at `~/.config/ign/config.json`. Global settings for ign.

```json
{
  "data_dir": "~/.local/ign",
  "cache": {
    "validity_days": 90
  }
}
```

Can be overridden with `--config` flag.

---

## 4. Template Syntax

### 4.1 Syntax Overview

| Directive | Syntax | Description |
|-----------|--------|-------------|
| Variable | `@ign-var:NAME@` | Simple variable substitution |
| Comment-style Variable | `@ign-comment:NAME@` | Variable with surrounding comment markers removed |
| Raw (Escape) | `@ign-raw:CONTENT@` | Output CONTENT literally without processing |
| Conditional | `@ign-if:VAR@...@ign-endif@` | Conditional block |
| Include | `@ign-include:PATH@` | Include another file |

### 4.2 Variable Substitution (`@ign-var:`)

**Syntax:** `@ign-var:VARIABLE_NAME@`

Replaces the directive with the variable value.

#### Example Usage

**flake.nix:**
```nix
{
  description = "@ign-var:PROJECT_DESCRIPTION@";

  outputs = { self, nixpkgs }:
    let
      projectName = "@ign-var:PROJECT_NAME@";
    in {
      # ...
    };
}
```

**go.mod:**
```
module @ign-var:MODULE_PATH@

go 1.21
```

### 4.3 Comment-style Variable (`@ign-comment:`)

**Syntax:** `@ign-comment:VARIABLE_NAME@`

Similar to `@ign-var:`, but intended for use within comments. The entire line containing the directive is replaced with just the variable value (comment markers are removed).

#### Example

**Before (template):**
```go
// @ign-comment:PACKAGE_HEADER@
package main
```

**After (if PACKAGE_HEADER = "// Code generated by ign. DO NOT EDIT."):**
```go
// Code generated by ign. DO NOT EDIT.
package main
```

### 4.4 Raw/Escape (`@ign-raw:`)

**Syntax:** `@ign-raw:CONTENT@`

Outputs CONTENT literally without any ign processing. Use this when you need to include literal `@ign-` text in your output.

#### Example

**Template:**
```markdown
To use ign variables, write @ign-raw:@ign-var:VARIABLE_NAME@@
```

**Output:**
```markdown
To use ign variables, write @ign-var:VARIABLE_NAME@
```

### 4.5 Conditional (`@ign-if:` / `@ign-endif@`)

**Syntax:**
```
@ign-if:VARIABLE_NAME@
... content when VARIABLE_NAME is truthy ...
@ign-endif@
```

Includes the content block only if the variable evaluates to truthy.

**Truthy values:** `true`, `"true"`, non-zero integers, non-empty strings
**Falsy values:** `false`, `"false"`, `0`, `""`, undefined variables

#### Example

**Template (Dockerfile):**
```dockerfile
FROM golang:1.21

@ign-if:ENABLE_DOCKER_COMPOSE@
COPY docker-compose.yml /app/
@ign-endif@

WORKDIR /app
COPY . .
```

**Output (if ENABLE_DOCKER_COMPOSE = false):**
```dockerfile
FROM golang:1.21

WORKDIR /app
COPY . .
```

### 4.6 Include (`@ign-include:`)

**Syntax:** `@ign-include:PATH@`

Includes the content of another file at this location. The included file is also processed for ign directives.

**Path Types:**
1. **Relative path** - Relative to current template file (no `..` allowed)
2. **GitHub URL path** - Full path to a file in a GitHub repository

#### Example

**Template structure:**
```
template/
|-- ign.json
|-- main.go
+-- partials/
    +-- header.txt
```

**main.go:**
```go
@ign-include:partials/header.txt@

package main

func main() {
    // ...
}
```

**partials/header.txt:**
```
// Project: @ign-var:PROJECT_NAME@
// Author: @ign-var:AUTHOR@
// Generated by ign
```

**Output:**
```go
// Project: my-project
// Author: John Doe
// Generated by ign

package main

func main() {
    // ...
}
```

### 4.7 Supported Substitution Locations

| Location | Supported |
|----------|-----------|
| File content | Yes |
| File names | **No** |
| Directory names | **No** |

---

## 5. File System Layout

### 5.1 Directory Structure

```
~/.config/ign/
+-- config.json              # Global configuration

~/.local/ign/
|-- http__github__com__owner__repo__path/    # Cached repository
|   +-- <git checkout>
|-- http__github__com__other__repo/          # Another cached repo
|   +-- <git checkout>
+-- cache_meta/
    +-- <metadata files with timestamps>
```

### 5.2 Cache URL Encoding

URL to directory name conversion:
- `/` -> `__`
- `:` -> `__`

Example:
```
https://github.com/owner/repo/templates/go
-> https__github__com__owner__repo__templates__go
```

### 5.3 Cache Strategy

| Aspect | Specification |
|--------|---------------|
| Storage method | Git checkout |
| Validity period | Configurable (default: 90 days) |
| Metadata storage | `~/.local/ign/cache_meta/` |
| Fault tolerance | Auto-recovery on corruption (delete and re-fetch) |

---

## 6. CLI Commands

### 6.1 Command Overview

```
ign [global flags]
|-- init                        # Generate project from .ign-build/ign-var.json
|-- build
|   +-- init <github_url>       # Create .ign-build/ with ign-var.json
|-- template
|   |-- init                    # Create ign template locally (Author)
|   +-- list
|       +-- create              # Create ign-list.json (Author)
|-- list <path>                 # Use ign-list.json (User)
+-- cache
    |-- update                  # Git fetch/update cache
    +-- clear [url]             # Delete cache
```

### 6.2 Global Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to config.json (default: `~/.config/ign/config.json`) |
| `--no-cache` | Bypass cache, fetch fresh from remote |

### 6.3 Command Details

#### `ign build init <github_url>`

Initialize build configuration by creating `.ign-build/` directory with `ign-var.json`.

```bash
# Create build configuration from template
ign build init github.com/owner/templates/go-basic

# With subdirectory path
ign build init github.com/owner/repo --path templates/go-basic

# With specific ref
ign build init github.com/owner/templates/go-basic --ref v1.0.0
```

**Actions:**
1. Fetch template's `ign.json`
2. Create `.ign-build/` directory
3. Generate `ign-var.json` with:
   - Template reference (url, path, ref)
   - All variables from `ign.json` with default/example values

**Generated ign-var.json example:**
```json
{
  "template": {
    "url": "github.com/owner/templates",
    "path": "go-basic",
    "ref": "main"
  },
  "variables": {
    "PROJECT_NAME": "",
    "MODULE_PATH": "",
    "HTTP_PORT": "8080",
    "ENABLE_DOCKER": "false"
  }
}
```

#### `ign init`

Generate project from `.ign-build/ign-var.json`.

```bash
# Generate in current directory
ign init

# Generate in specific directory
ign init --output ./my-project

# Overwrite existing files
ign init --overwrite

# Combine flags
ign init --output ./my-project --overwrite
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--output <dir>` | Output directory (default: current directory) |
| `--overwrite` | Overwrite existing files (default: skip with warning) |

**Behavior:**

| Scenario | Default | With `--overwrite` |
|----------|---------|-------------------|
| File doesn't exist | Create | Create |
| File exists | Skip with warning | Overwrite |
| Directory doesn't exist | Create | Create |

**Process:**
1. Read `.ign-build/ign-var.json`
2. Fetch template (using cache if valid)
3. For each template file (except `ign.json`):
   - Apply variable substitution
   - Apply conditional processing
   - Apply includes
   - Write to output directory

Note: `.ign-build/` directory is NOT copied to output.

#### `ign template init`

Initialize a new ign template repository.

```bash
# In current directory
ign template init

# Specify directory
ign template init --path ./my-template
```

**Actions:**
- Creates `ign.json` with default structure
- Scans all files for `@ign-var:VAR@` patterns
- Extracts variable names and populates `ign.json`

#### `ign template list create`

Create an `ign-list.json` file.

```bash
ign template list create
ign template list create --output ./ign-list.json
```

#### `ign list <path>`

Use a template from an ign-list.json.

```bash
ign list github.com/owner/templates-repo
```

**Actions:**
- Fetches `ign-list.json` from the specified location
- Displays available templates
- Allows user to select and run `ign build init`

#### `ign cache update`

Update all cached repositories.

```bash
ign cache update
```

**Actions:**
- Performs `git fetch` and `git pull` on all cached repositories
- Updates cache metadata timestamps

#### `ign cache clear`

Clear cached repositories.

```bash
# Clear all cache
ign cache clear

# Clear specific repository cache
ign cache clear github.com/owner/repo
```

---

## 7. Process Flows

### 7.1 Complete Workflow

```
+------------------------------------------------------------------+
|                    User Workflow                                  |
+------------------------------------------------------------------+

Step 1: Initialize build configuration
+------------------------------------------------------------------+
$ ign build init github.com/owner/templates/go-basic
                            |
                            v
              +-------------------------+
              |  Fetch template         |
              |  ign.json               |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Create .ign-build/     |
              |  ign-var.json           |
              +-------------------------+

Step 2: Edit variables
+------------------------------------------------------------------+
$ vim .ign-build/ign-var.json
$ vim .ign-build/license-header.txt  (if using @file:)

Step 3: Generate project
+------------------------------------------------------------------+
$ ign init --output ./my-project
                            |
                            v
              +-------------------------+
              |  Read ign-var.json      |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Fetch template         |
              |  (use cache if valid)   |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Process template files |
              |  - @ign-var:            |
              |  - @ign-if:             |
              |  - @ign-include:        |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Write to output dir    |
              |  (skip existing files)  |
              +-------------------------+
                            |
                            v
                        [ Done ]
```

### 7.2 Template Creation Flow (`ign template init`)

```
+-------------------------------------------------------------+
|                   ign template init                         |
+-------------------------------------------------------------+
                            |
                            v
              +-------------------------+
              |  Scan all files in      |
              |  current directory      |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Extract @ign-var:VAR@  |
              |  patterns from files    |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Generate ign.json      |
              |  with discovered vars   |
              +-------------------------+
                            |
                            v
              +-------------------------+
              |  Prompt for variable    |
              |  metadata (type, desc)  |
              +-------------------------+
                            |
                            v
                        [ Done ]
```

---

## 8. Architecture

### 8.1 Package Structure (Proposed)

```
ign/
|-- cmd/
|   +-- ign/
|       +-- main.go              # Entry point
|-- internal/
|   |-- cli/                     # CLI command implementations
|   |   |-- init.go              # ign init
|   |   |-- build.go             # ign build init
|   |   |-- template.go          # ign template
|   |   |-- list.go              # ign list
|   |   +-- cache.go             # ign cache
|   |-- config/                  # Configuration handling
|   |   |-- config.go            # Global config
|   |   |-- ign_json.go          # Template ign.json
|   |   +-- ign_var.go           # User ign-var.json
|   |-- provider/                # Template source providers
|   |   |-- provider.go          # Interface definition
|   |   +-- github/
|   |       +-- github.go
|   |-- template/                # Template processing
|   |   |-- parser.go            # Directive extraction
|   |   |-- renderer.go          # Variable substitution
|   |   |-- conditional.go       # @ign-if: processing
|   |   +-- include.go           # @ign-include: processing
|   |-- cache/                   # Cache management
|   |   +-- cache.go
|   +-- fs/                      # File system utilities
|       +-- fs.go
|-- go.mod
|-- go.sum
+-- Taskfile.yml
```

### 8.2 Key Interfaces

```go
// Provider interface for template sources
type Provider interface {
    Fetch(ctx context.Context, url string, ref string) (*TemplateRoot, error)
    Validate(url string) bool
}

// TemplateRoot represents a fetched template
type TemplateRoot struct {
    Path      string      // Local path to template root
    Config    *IgnConfig  // Parsed ign.json
    Files     []string    // List of template files (excluding ign.json)
}

// IgnConfig represents ign.json (template definition)
type IgnConfig struct {
    Name        string               `json:"name"`
    Description string               `json:"description"`
    Variables   map[string]Variable  `json:"variables"`
}

// Variable represents a template variable definition
type Variable struct {
    Type        string `json:"type"`      // "string", "int", "bool"
    Optional    bool   `json:"optional"`
    Default     string `json:"default"`
    Description string `json:"description"`
}

// IgnVar represents ign-var.json (user build configuration)
type IgnVar struct {
    Template  TemplateRef       `json:"template"`
    Variables map[string]string `json:"variables"`
}

// TemplateRef represents template reference in ign-var.json
type TemplateRef struct {
    URL  string `json:"url"`
    Path string `json:"path,omitempty"`
    Ref  string `json:"ref,omitempty"`
}
```

---

## 9. Error Handling

### 9.1 Error Categories

| Category | Examples | Recovery |
|----------|----------|----------|
| Network errors | GitHub unreachable, timeout | Retry with backoff, use cache if available |
| Cache corruption | Invalid git state, missing files | Delete cache and re-fetch |
| Invalid template | Missing ign.json, invalid syntax | Display clear error message |
| Variable errors | Missing required variable, type mismatch | Prompt user for correction |
| File system errors | Permission denied, disk full | Display error and abort |
| Include errors | File not found, `..` in path | Display clear error message |
| Build config errors | Missing .ign-build/, invalid ign-var.json | Display clear error message |

### 9.2 Fault Tolerance

The cache system must be fault-tolerant:

1. **Validation on read:** Verify cache integrity before use
2. **Auto-recovery:** Delete corrupted cache and re-fetch
3. **Graceful degradation:** Continue with network fetch if cache is unusable

---

## 10. Design Decisions Summary

### Resolved Questions

| Question | Decision |
|----------|----------|
| Template syntax base | `@ign-var:VAR@` |
| Comment-style syntax | `@ign-comment:VAR@` (removes comment markers) |
| Escaping mechanism | `@ign-raw:CONTENT@` |
| Conditionals | Supported: `@ign-if:VAR@...@ign-endif@` |
| Loops | **Not supported** |
| Filters/Transforms | **Not supported** |
| Include files | Supported: `@ign-include:PATH@` (relative or GitHub URL, no `..`) |
| Variable types | `string`, `int`, `bool` only (no array/object) |
| File-based variables | `@file:` prefix, paths relative to `.ign-build/` |
| `.ign-build/` policy | No restrictions on `@file:` paths within directory |
| Lock file | **Not used** (one-shot generation, no state tracking) |
| Merge strategy | **None** (skip existing or overwrite) |
| `ref` default | Repository's default branch (main/master) |
| Config override | `--config` flag |

### Removed Features (from earlier drafts)

| Feature | Reason |
|---------|--------|
| `ign.lock.json` | One-shot generation, no state tracking needed |
| `ign init --update` | Replaced by `--overwrite` |
| `ign init --locked` | No lock file |
| Managed files concept | All files are user-owned after generation |
| 3-way merge | Explicit overwrite instead |

---

## 11. TODO Items

### High Priority

- [ ] **Define "simpler than cookiecutter"** - What specific features are intentionally excluded?
- [ ] **Define "more flexible"** - What specific flexibility features differentiate ign?
- [ ] **Library investigation** - Evaluate koanf (https://github.com/knadh/koanf) and other candidates

### Medium Priority

- [ ] **config.json full schema** - Define complete configuration options
- [ ] **@ign-comment: behavior details** - Exact rules for comment marker detection and removal
- [ ] **@ign-include: GitHub URL format** - Exact URL format for including from remote repositories
- [ ] **Interactive mode** - `ign init` with prompts when ign-var.json has empty values

### Low Priority (Future)

- [ ] **Additional providers** - GitLab, Bitbucket, local filesystem support
- [ ] **Template versioning** - Support for template versions/tags
- [ ] **Template inheritance** - Extend/override existing templates
- [ ] **Post-init hooks** - Run scripts after project generation
- [ ] **Loops support** - `@ign-for:` directive if needed
- [ ] **Filters/Transforms** - `@ign-var:NAME|filter@` syntax if needed

---

## 12. Revision History

| Version | Date | Changes |
|---------|------|---------|
| 0.1 | 2025-12-09 | Initial draft |
| 0.2 | 2025-12-09 | Resolved Q12-Q15, renamed @ign: to @ign-var:, added directives |
| 0.3 | 2025-12-09 | Simplified design: removed lock file, added ign build init, renamed ign-build.json to ign-var.json |
