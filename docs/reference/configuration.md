# Configuration File Reference

Complete reference for all ign configuration file formats, schemas, and filesystem layouts.

---

## Configuration Files Overview

| File | Location | Purpose | Required |
|------|----------|---------|----------|
| `ign.json` | Template root | Template definition | Yes (in template) |
| `ign.json` | `.ign/` | Template reference and hash | Yes (for init) |
| `ign-var.json` | `.ign/` | User variables | Yes (for init) |
| `config.json` | `~/.config/ign/` | Global ign configuration | No |

**Note:** There are two different `ign.json` files:
1. **Template `ign.json`** - Located in the template repository root, defines template metadata and variable definitions
2. **Project `ign.json`** - Located in `.ign/` directory, stores template source reference and content hash

---

## 1. Template Configuration (`ign.json`)

### 1.1 Purpose

Defines template metadata, required variables, and template-specific settings.

**Location:** Root of template repository (or subdirectory specified as ign root)

**Deployment:** NOT copied to generated project (configuration only)

### 1.2 Schema

```json
{
  "name": "template-name",
  "version": "1.0.0",
  "description": "Template description",
  "author": "Author Name <email@example.com>",
  "repository": "https://github.com/owner/repo",
  "variables": {
    "variable_name": {
      "type": "string|int|bool",
      "description": "Variable description",
      "default": "optional default value",
      "required": true,
      "example": "example value"
    }
  },
  "settings": {
    "preserve_executable": true,
    "ignore_patterns": ["*.bak", "*.tmp"],
    "binary_extensions": [".png", ".jpg", ".pdf"]
  }
}
```

### 1.3 Fields

#### Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Template identifier (lowercase, hyphens) |
| `version` | string | Yes | Semantic version (e.g., "1.0.0") |
| `description` | string | No | Human-readable description |
| `author` | string | No | Author name and email |
| `repository` | string | No | Source repository URL |
| `license` | string | No | License identifier (e.g., "MIT", "Apache-2.0") |
| `tags` | string[] | No | Searchable tags (e.g., ["go", "api", "rest"]) |

#### Variables Section

```json
{
  "variables": {
    "project_name": {
      "type": "string",
      "description": "Name of the project",
      "required": true,
      "example": "my-awesome-project"
    },
    "port": {
      "type": "int",
      "description": "Server port number",
      "default": 8080,
      "required": false
    },
    "enable_tls": {
      "type": "bool",
      "description": "Enable TLS/HTTPS",
      "default": false,
      "required": false
    }
  }
}
```

**Variable Definition Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Variable type: `string`, `int`, or `bool` |
| `description` | string | Yes | Human-readable description |
| `required` | bool | No | If true, must have value in ign-var.json (default: false) |
| `default` | any | No | Default value (type must match) |
| `example` | any | No | Example value for documentation |
| `pattern` | string | No | Regex validation pattern (strings only) |
| `min` | int | No | Minimum value (integers only) |
| `max` | int | No | Maximum value (integers only) |

#### Settings Section

```json
{
  "settings": {
    "preserve_executable": true,
    "ignore_patterns": ["*.log", "*.tmp", ".DS_Store"],
    "binary_extensions": [".png", ".jpg", ".gif", ".pdf"],
    "include_dotfiles": true,
    "template_delimiter": "@ign-",
    "max_include_depth": 10
  }
}
```

**Settings Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `preserve_executable` | bool | `true` | Preserve executable bit from template files |
| `ignore_patterns` | string[] | `[]` | Glob patterns for files to ignore |
| `binary_extensions` | string[] | See below | File extensions to copy without processing |
| `include_dotfiles` | bool | `true` | Include hidden files (starting with `.`) |
| `template_delimiter` | string | `@ign-` | Custom directive prefix (future feature) |
| `max_include_depth` | int | `10` | Maximum nested include depth |

**Default Binary Extensions:**
```json
[".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".pdf",
 ".zip", ".tar", ".gz", ".bz2", ".7z", ".exe", ".dll",
 ".so", ".dylib", ".woff", ".woff2", ".ttf", ".eot"]
```

### 1.4 Complete Example

```json
{
  "name": "go-rest-api",
  "version": "2.1.0",
  "description": "Production-ready Go REST API template with PostgreSQL",
  "author": "John Doe <john@example.com>",
  "repository": "https://github.com/johndoe/templates",
  "license": "MIT",
  "tags": ["go", "api", "rest", "postgresql", "docker"],

  "variables": {
    "project_name": {
      "type": "string",
      "description": "Project name (lowercase, hyphens)",
      "required": true,
      "example": "my-api-service",
      "pattern": "^[a-z][a-z0-9-]*$"
    },
    "go_module_path": {
      "type": "string",
      "description": "Go module import path",
      "required": true,
      "example": "github.com/username/my-api"
    },
    "port": {
      "type": "int",
      "description": "HTTP server port",
      "default": 8080,
      "min": 1024,
      "max": 65535
    },
    "postgres_port": {
      "type": "int",
      "description": "PostgreSQL port",
      "default": 5432
    },
    "enable_swagger": {
      "type": "bool",
      "description": "Include Swagger/OpenAPI documentation",
      "default": true
    },
    "enable_docker": {
      "type": "bool",
      "description": "Include Docker and docker-compose files",
      "default": true
    },
    "license_header": {
      "type": "string",
      "description": "License header for source files (use @file:)",
      "example": "@file:license-header.txt"
    }
  },

  "settings": {
    "preserve_executable": true,
    "ignore_patterns": ["*.swp", "*.swo", "*~", ".DS_Store"],
    "binary_extensions": [".png", ".jpg", ".pdf"],
    "include_dotfiles": true,
    "max_include_depth": 5
  }
}
```

### 1.5 Validation

**JSON Schema Validation:**
- Valid JSON syntax
- All required fields present
- Type constraints enforced
- Pattern validation for variable names

**Variable Name Rules:**
- Must start with letter (a-z, A-Z)
- Can contain letters, numbers, underscores, hyphens
- Case-sensitive
- No spaces or special characters

**Valid:**
- `project_name`
- `HTTP_PORT`
- `enable-feature-x`
- `version2`

**Invalid:**
- `123project` (starts with number)
- `project name` (contains space)
- `project.name` (contains dot)
- `_project` (starts with underscore)

---

## 2. Project Configuration Files (`.ign/`)

The `.ign/` directory contains two configuration files that work together:

1. **`ign.json`** - Template source reference and content hash
2. **`ign-var.json`** - User-provided variable values

This separation allows:
- Variables to be edited independently without affecting template reference
- Template hash verification for integrity checking
- Cleaner separation of concerns

---

## 2.1 Project Template Reference (`ign.json`)

### Purpose

Stores the template source reference and a hash of the downloaded template content for verification.

**Location:** `.ign/ign.json`

**Created by:** `ign checkout` command

### Schema

```json
{
  "template": {
    "url": "github.com/owner/repo",
    "path": "templates/subdir",
    "ref": "v1.0.0"
  },
  "hash": "sha256:a1b2c3d4e5f6...",
  "metadata": {
    "generated_at": "2025-12-09T10:30:00Z",
    "generated_by": "ign checkout",
    "template_name": "go-rest-api",
    "template_version": "2.1.0"
  }
}
```

### Fields

#### Template Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Template repository URL |
| `path` | string | No | Subdirectory path within repository |
| `ref` | string | No | Git branch, tag, or commit (default: `main`) |

**URL Formats:**
```json
{
  "url": "github.com/owner/repo",
  "url": "https://github.com/owner/repo",
  "url": "git@github.com:owner/repo.git"
}
```

#### Hash Field

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `hash` | string | Yes | SHA256 hash of template content |

The hash is calculated from all template file paths and contents, sorted deterministically. This allows verification that the template hasn't changed since checkout.

#### Metadata Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `generated_at` | string | No | ISO 8601 timestamp |
| `generated_by` | string | No | Command that generated the file |
| `template_name` | string | No | Template name from template's ign.json |
| `template_version` | string | No | Template version |

### Complete Example

**.ign/ign.json:**
```json
{
  "template": {
    "url": "github.com/myorg/templates",
    "path": "go/rest-api",
    "ref": "v2.1.0"
  },
  "hash": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "metadata": {
    "generated_at": "2025-12-09T10:30:00Z",
    "generated_by": "ign checkout",
    "template_name": "go-rest-api",
    "template_version": "2.1.0"
  }
}
```

---

## 2.2 User Variables (`ign-var.json`)

### Purpose

Stores user-provided variable values for template generation.

**Location:** `.ign/ign-var.json`

**Created by:** `ign checkout` command

**Edited by:** User (to customize variable values)

### Schema

```json
{
  "variables": {
    "variable_name": "value",
    "another_var": 42,
    "feature_flag": true,
    "file_content": "@file:relative/path.txt"
  }
}
```

### Fields

#### Variables Section

Contains all user-provided variable values:

```json
{
  "variables": {
    "project_name": "my-awesome-api",
    "version": "1.0.0",
    "port": 8080,
    "enable_tls": true,
    "enable_metrics": false,
    "license_header": "@file:license-header.txt",
    "readme_intro": "@file:templates/intro.md"
  }
}
```

**Type Examples:**

```json
{
  "variables": {
    // Strings
    "name": "value",
    "description": "A longer text value with spaces",
    "multiline": "Line 1\nLine 2\nLine 3",

    // Integers
    "port": 8080,
    "timeout": 30,
    "max_connections": 100,

    // Booleans
    "enabled": true,
    "debug_mode": false,

    // File references
    "license": "@file:license-header.txt",
    "config": "@file:config/template.yaml"
  }
}
```

**File Reference Rules:**

- Prefix: `@file:`
- Path: Relative to `.ign/` directory
- Example: `@file:license.txt` resolves to `.ign/license.txt`
- Can use subdirectories: `@file:templates/readme.md`
- File must exist when `ign init` runs
- Content is read as-is (including newlines)

### Complete Example

**.ign/ign-var.json:**
```json
{
  "variables": {
    "project_name": "user-service",
    "go_module_path": "github.com/mycompany/user-service",
    "port": 8080,
    "postgres_port": 5432,
    "enable_swagger": true,
    "enable_docker": true,
    "license_header": "@file:license-header.txt"
  }
}
```

**.ign/license-header.txt:**
```
Copyright (c) 2025 My Company, Inc.
Licensed under the Apache License, Version 2.0
```

---

## 2.3 Generation Process

**Step 1: `ign checkout github.com/myorg/templates/go/rest-api`**

Creates two files in `.ign/`:

**.ign/ign.json:**
```json
{
  "template": {
    "url": "github.com/myorg/templates",
    "path": "go/rest-api",
    "ref": "main"
  },
  "hash": "sha256:...",
  "metadata": {
    "generated_at": "2025-12-09T10:30:00Z",
    "generated_by": "ign checkout",
    "template_name": "go-rest-api",
    "template_version": "2.1.0"
  }
}
```

**.ign/ign-var.json:**
```json
{
  "variables": {
    "project_name": "",
    "go_module_path": "",
    "port": 8080,
    "postgres_port": 5432,
    "enable_swagger": true,
    "enable_docker": true,
    "license_header": ""
  }
}
```

**Step 2: User edits ign-var.json**

User fills in required values and creates referenced files.

**Step 3: `ign init --output ./my-project`**

Reads both configuration files and generates project.

---

## 3. Global Configuration (`config.json`)

### 3.1 Purpose

Global settings for ign behavior across all projects.

**Location:**
- Linux/macOS: `~/.config/ign/config.json`
- Custom: Override with `IGN_CONFIG` environment variable or `--config` flag

**Optional:** Not required (defaults used if missing)

### 3.2 Schema

```json
{
  "cache": {
    "enabled": true,
    "directory": "~/.cache/ign",
    "ttl": 3600,
    "max_size_mb": 500
  },
  "github": {
    "token": "",
    "default_ref": "main",
    "api_url": "https://api.github.com"
  },
  "templates": {
    "max_include_depth": 10,
    "preserve_executable": true,
    "ignore_patterns": [".DS_Store", "*.swp"]
  },
  "output": {
    "color": true,
    "progress": true,
    "verbose": false
  },
  "defaults": {
    "build_dir": ".ign",
  }
}
```

### 3.3 Fields

#### Cache Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable template caching |
| `directory` | string | `~/.cache/ign` | Cache directory path |
| `ttl` | int | `3600` | Cache time-to-live in seconds (0 = no expiration) |
| `max_size_mb` | int | `500` | Maximum cache size in megabytes |
| `auto_clean` | bool | `true` | Automatically clean old cache entries |

#### GitHub Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `token` | string | `""` | GitHub personal access token (for private repos) |
| `default_ref` | string | `"main"` | Default branch/ref if not specified |
| `api_url` | string | `https://api.github.com` | GitHub API URL (for enterprise) |
| `timeout` | int | `30` | Request timeout in seconds |

#### Templates Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_include_depth` | int | `10` | Maximum nested include depth |
| `preserve_executable` | bool | `true` | Preserve executable permissions |
| `ignore_patterns` | string[] | `[]` | Global ignore patterns (glob) |
| `binary_extensions` | string[] | See below | File extensions to skip processing |

#### Output Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `color` | bool | `true` | Enable colored output |
| `progress` | bool | `true` | Show progress indicators |
| `verbose` | bool | `false` | Enable verbose logging |
| `quiet` | bool | `false` | Suppress non-error output |

#### Defaults Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `build_dir` | string | `".ign"` | Default build directory name |
| `output_dir` | string | `"."` | Default output directory for `ign init` |

### 3.4 Complete Example

```json
{
  "cache": {
    "enabled": true,
    "directory": "/tmp/ign-cache",
    "ttl": 7200,
    "max_size_mb": 1000,
    "auto_clean": true
  },

  "github": {
    "token": "ghp_xxxxxxxxxxxxxxxxxxxx",
    "default_ref": "main",
    "api_url": "https://api.github.com",
    "timeout": 60
  },

  "templates": {
    "max_include_depth": 15,
    "preserve_executable": true,
    "ignore_patterns": [
      ".DS_Store",
      "*.swp",
      "*.swo",
      "*~",
      "Thumbs.db"
    ],
    "binary_extensions": [
      ".png", ".jpg", ".jpeg", ".gif",
      ".pdf", ".zip", ".tar.gz"
    ]
  },

  "output": {
    "color": true,
    "progress": true,
    "verbose": false,
    "quiet": false
  },

  "defaults": {
    "build_dir": ".ign",
    "output_dir": "."
  }
}
```

### 3.5 Priority and Overrides

Configuration is resolved in this order (highest to lowest priority):

1. **Command-line flags** (e.g., `--verbose`, `--no-color`)
2. **Environment variables** (e.g., `IGN_CACHE_DIR`, `GITHUB_TOKEN`)
3. **gh CLI authentication** (`gh auth token`)
4. **Global config file** (`~/.config/ign/config.json`)
5. **Built-in defaults**

**Example:**
```bash
# All of these override the config file
export GITHUB_TOKEN=ghp_xyz
export IGN_CACHE_DIR=/custom/cache

ign init --verbose --no-color
```

### 3.6 Environment Variables

| Variable | Config Path | Description |
|----------|-------------|-------------|
| `IGN_CONFIG` | (file path) | Path to config.json |
| `IGN_CACHE_DIR` | `cache.directory` | Cache directory |
| `GITHUB_TOKEN` | `github.token` | GitHub access token |
| `GH_TOKEN` | `github.token` | GitHub access token (alternative) |
| `IGN_NO_COLOR` | `output.color` | Disable colors (set to "1") |
| `IGN_VERBOSE` | `output.verbose` | Enable verbose (set to "1") |
| `IGN_BUILD_DIR` | `defaults.build_dir` | Build directory name |

**GitHub Token Resolution:**

The GitHub token for private repository access is resolved in this order:
1. `GITHUB_TOKEN` environment variable
2. `GH_TOKEN` environment variable
3. `gh auth token` command (gh CLI's secure credential storage)

If you have `gh` CLI installed and authenticated, no additional token configuration is needed.

**Example:**
```bash
export IGN_CONFIG=~/my-ign-config.json
export IGN_CACHE_DIR=/tmp/cache
export GITHUB_TOKEN=ghp_mytoken
export IGN_NO_COLOR=1

ign init
```

---

## 4. Filesystem Layout

### 4.1 Template Repository Structure

**Minimal Template:**
```
template-repo/
└── ign.json                  # Required: Template configuration
└── README.md                 # Template files (deployed)
└── main.go                   # Template files (deployed)
```

**Complete Template:**
```
template-repo/
├── ign.json                  # Required: Template configuration
├── README.md                 # Template documentation (deployed)
├── .envrc                    # Template file (deployed)
├── .gitignore                # Template file (deployed)
├── main.go                   # Template file (deployed)
├── go.mod                    # Template file (deployed)
├── cmd/
│   └── server/
│       └── main.go           # Template files in subdirectories
├── pkg/
│   ├── config/
│   │   └── config.go
│   └── api/
│       └── handler.go
├── common/
│   ├── license-header.txt    # For @ign-include: directives
│   └── makefile-template     # Reusable template fragments
└── docs/
    └── usage.md              # Template documentation
```

**With Subdirectories (Multi-template repo):**
```
templates-repo/
├── README.md                 # Repository documentation
├── go/
│   ├── basic/
│   │   ├── ign.json          # Template: go-basic
│   │   ├── main.go
│   │   └── go.mod
│   ├── rest-api/
│   │   ├── ign.json          # Template: go-rest-api
│   │   ├── main.go
│   │   └── ... (more files)
│   └── grpc/
│       ├── ign.json          # Template: go-grpc
│       └── ... (more files)
└── python/
    ├── basic/
    │   ├── ign.json          # Template: python-basic
    │   └── ... (more files)
    └── fastapi/
        ├── ign.json          # Template: python-fastapi
        └── ... (more files)
```

### 4.2 User Project Structure

**Before `ign build init`:**
```
my-workspace/
└── (empty or existing project)
```

**After `ign build init`:**
```
my-workspace/
└── .ign/
    └── ign-var.json          # Generated build configuration
```

**User adds files:**
```
my-workspace/
└── .ign/
    ├── ign-var.json          # Build configuration
    ├── license-header.txt    # For @file: references
    └── templates/
        └── intro.md          # For @file: references
```

**After `ign init --output ./my-project`:**
```
my-workspace/
├── .ign/               # Build config (untouched)
│   ├── ign-var.json
│   ├── license-header.txt
│   └── templates/
│       └── intro.md
└── my-project/               # Generated project
    ├── .envrc
    ├── .gitignore
    ├── README.md
    ├── go.mod
    ├── main.go
    ├── cmd/
    │   └── server/
    │       └── main.go
    └── pkg/
        ├── config/
        │   └── config.go
        └── api/
            └── handler.go
```

### 4.3 Cache Directory Structure

**Default cache location:** `~/.cache/ign/`

```
~/.cache/ign/
├── github.com/
│   ├── owner1/
│   │   ├── repo1/
│   │   │   ├── main/
│   │   │   │   ├── ign.json
│   │   │   │   └── ... (template files)
│   │   │   ├── v1.0.0/
│   │   │   │   └── ... (template files)
│   │   │   └── abc123def/
│   │   │       └── ... (template files)
│   │   └── repo2/
│   │       └── main/
│   │           └── ... (template files)
│   └── owner2/
│       └── templates/
│           ├── main/
│           │   ├── go/
│           │   │   └── basic/
│           │   │       ├── ign.json
│           │   │       └── ...
│           │   └── python/
│           │       └── basic/
│           │           └── ...
│           └── v2.0.0/
│               └── ... (same structure)
└── metadata.json             # Cache metadata (timestamps, sizes)
```

**Cache key format:**
```
<host>/<owner>/<repo>/<ref>/<path>
```

**Examples:**
- `github.com/myorg/templates/main/`
- `github.com/myorg/templates/v1.0.0/go/basic/`
- `github.com/myorg/repo/abc123def/`

### 4.4 Global Configuration Directory

**Default location:** `~/.config/ign/`

```
~/.config/ign/
└── config.json               # Global configuration
```

---

## 5. File Ignore Patterns

### 5.1 Default Ignore Patterns

Ign automatically ignores these patterns (cannot be overridden):

```
ign.json                      # Template config (not deployed)
.git/                         # Git directory
.ign/                   # Build configuration
```

### 5.2 Template-Specific Ignores

In `ign.json`:
```json
{
  "settings": {
    "ignore_patterns": [
      "*.log",
      "*.tmp",
      ".DS_Store",
      "Thumbs.db",
      "node_modules/",
      "*.swp",
      "*~"
    ]
  }
}
```

### 5.3 Global Ignores

In `~/.config/ign/config.json`:
```json
{
  "templates": {
    "ignore_patterns": [
      ".DS_Store",
      "Thumbs.db",
      "*.swp",
      "*.swo"
    ]
  }
}
```

### 5.4 Pattern Syntax

Uses standard glob patterns:

| Pattern | Matches |
|---------|---------|
| `*.log` | All files ending in `.log` |
| `temp*` | All files starting with `temp` |
| `*.{log,tmp}` | Files ending in `.log` or `.tmp` |
| `**/node_modules/` | `node_modules/` in any directory |
| `test/**/*.tmp` | `.tmp` files under `test/` |
| `!important.log` | Negation (include despite other rules) |

---

## 6. Binary File Detection

### 6.1 Default Binary Extensions

Files with these extensions are copied without template processing:

```
.png, .jpg, .jpeg, .gif, .bmp, .ico, .svg
.pdf, .doc, .docx, .xls, .xlsx, .ppt, .pptx
.zip, .tar, .gz, .bz2, .7z, .rar
.exe, .dll, .so, .dylib, .a
.woff, .woff2, .ttf, .eot, .otf
.mp3, .mp4, .avi, .mov, .wav
.db, .sqlite, .sqlite3
```

### 6.2 Custom Binary Extensions

In `ign.json`:
```json
{
  "settings": {
    "binary_extensions": [
      ".png", ".jpg", ".pdf",
      ".custom", ".proprietary"
    ]
  }
}
```

### 6.3 Binary Detection Algorithm

1. Check file extension against binary list
2. If not in list, read first 512 bytes
3. If contains null bytes (`\0`), treat as binary
4. Otherwise, treat as text (process templates)

---

## 7. Validation and Schema

### 7.1 JSON Schema Validation

All configuration files are validated against JSON schemas:

**ign.json schema:** `https://ign-tool.org/schema/ign.json.schema.json`
**ign-var.json schema:** `https://ign-tool.org/schema/ign-var.json.schema.json`
**config.json schema:** `https://ign-tool.org/schema/config.json.schema.json`

### 7.2 Common Validation Errors

| Error | File | Fix |
|-------|------|-----|
| `Missing required field: name` | ign.json | Add `"name": "template-name"` |
| `Invalid variable type: xyz` | ign.json | Use `string`, `int`, or `bool` |
| `Variable name invalid: 123abc` | ign.json | Start with letter |
| `Missing template.url` | ign-var.json | Add template URL |
| `Type mismatch: expected int` | ign-var.json | Use number, not string |
| `Invalid JSON syntax` | Any | Fix JSON formatting |

### 7.3 Validation Command (Future Feature)

```bash
# Validate config files
ign validate --config .ign/ign-var.json
ign validate --template ./template/ign.json

# Validate before generation
ign init --validate-only
```

---

## Appendix A: Complete Schema Definitions

### ign.json Schema

```typescript
interface IgnJson {
  name: string;                  // Required
  version: string;               // Required (semver)
  description?: string;
  author?: string;
  repository?: string;
  license?: string;
  tags?: string[];

  variables: {
    [key: string]: {
      type: "string" | "int" | "bool";  // Required
      description: string;               // Required
      required?: boolean;
      default?: string | number | boolean;
      example?: string | number | boolean;
      pattern?: string;                  // For strings
      min?: number;                      // For ints
      max?: number;                      // For ints
    }
  };

  settings?: {
    preserve_executable?: boolean;
    ignore_patterns?: string[];
    binary_extensions?: string[];
    include_dotfiles?: boolean;
    max_include_depth?: number;
  };
}
```

### Project ign.json Schema (`.ign/ign.json`)

```typescript
interface IgnConfig {
  template: {
    url: string;                 // Required
    path?: string;
    ref?: string;
  };

  hash: string;                  // Required - SHA256 hash of template

  metadata?: {
    generated_at?: string;
    generated_by?: string;
    template_name?: string;
    template_version?: string;
  };
}
```

### ign-var.json Schema

```typescript
interface IgnVarJson {
  variables: {
    [key: string]: string | number | boolean;
  };
}
```

### config.json Schema

```typescript
interface ConfigJson {
  cache?: {
    enabled?: boolean;
    directory?: string;
    ttl?: number;
    max_size_mb?: number;
    auto_clean?: boolean;
  };

  github?: {
    token?: string;
    default_ref?: string;
    api_url?: string;
    timeout?: number;
  };

  templates?: {
    max_include_depth?: number;
    preserve_executable?: boolean;
    ignore_patterns?: string[];
    binary_extensions?: string[];
  };

  output?: {
    color?: boolean;
    progress?: boolean;
    verbose?: boolean;
    quiet?: boolean;
  };

  defaults?: {
    build_dir?: string;
    output_dir?: string;
  };
}
```

## Appendix B: Configuration Examples

See individual sections for complete examples of each configuration file type.
