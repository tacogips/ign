# CLI Commands Reference

Complete reference for all ign command-line interface commands, flags, and behaviors.

---

## Command Overview

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `ign build init` | Create build configuration | `--config`, `--output`, `--ref` |
| `ign init` | Generate project from template | `--output`, `--overwrite`, `--config` |
| `ign version` | Show version information | None |
| `ign help` | Show help information | None |

---

## 1. Build Initialization

### 1.1 `ign build init`

Create a new `.ign/` directory with `ign-var.json` for a template.

**Syntax:**
```bash
ign build init [URL] [flags]
```

**Arguments:**
- `URL`: GitHub repository URL or short form (see URL Formats below)

**Examples:**
```bash
# Full GitHub URL
ign build init https://github.com/owner/repo

# Short form
ign build init github.com/owner/repo

# With subdirectory
ign build init github.com/owner/repo/templates/go-basic

# With specific branch/tag
ign build init github.com/owner/repo --ref v1.2.0

# Specify output location
ign build init github.com/owner/repo --output ./my-config
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `.ign` | Output directory for build config |
| `--ref` | `-r` | string | `main` | Git branch, tag, or commit SHA |
| `--config` | `-c` | string | `~/.config/ign/config.json` | Path to global config file |
| `--force` | `-f` | bool | `false` | Overwrite existing .ign directory |

**Behavior:**

1. **Validate URL**: Check if URL is accessible and contains `ign.json`
2. **Create directory**: Create output directory (default: `.ign/`)
3. **Fetch template**: Clone or download template from GitHub
4. **Read ign.json**: Parse template configuration
5. **Generate ign-var.json**: Create variable file with:
   - Template URL reference
   - All variables from `ign.json` with empty values
   - Variable type information
6. **Success message**: Display next steps

**Example Output:**
```
Initializing build configuration...

✓ Template found: github.com/owner/repo/templates/go-basic@v1.2.0
✓ Created: .ign/ign-var.json

Next steps:
1. Edit .ign/ign-var.json to set variable values
2. Run: ign init --output ./my-project

Variables to configure (5):
  - project_name (string)
  - version (string)
  - port (int)
  - enable_tls (bool)
  - license (string) - Tip: Use @file:license.txt
```

**Error Cases:**

| Error | Cause | Exit Code |
|-------|-------|-----------|
| Template not found | URL doesn't exist or not accessible | 1 |
| Missing ign.json | Repository doesn't have ign.json | 1 |
| Invalid ign.json | Malformed JSON or schema error | 1 |
| Directory exists | `.ign/` exists and `--force` not used | 1 |
| Network error | Cannot reach GitHub | 2 |

### 1.2 URL Formats

**Supported URL formats:**

```bash
# Full HTTPS URL
https://github.com/owner/repo

# Full HTTPS with subdirectory
https://github.com/owner/repo/tree/main/templates/go-basic

# Git SSH URL
git@github.com:owner/repo.git

# Short form (GitHub assumed)
github.com/owner/repo

# Short form with subdirectory
github.com/owner/repo/templates/go-basic

# Owner/repo only (GitHub assumed)
owner/repo

# Owner/repo with subdirectory
owner/repo/templates/go-basic
```

**URL Resolution:**
1. If URL starts with `https://` or `git@`: Use as-is
2. If URL starts with `github.com/`: Prepend `https://`
3. If URL is `owner/repo` format: Prepend `https://github.com/`
4. Extract subdirectory path after repository name

**Reference (--ref) Formats:**
```bash
# Branch name
--ref main
--ref develop
--ref feature/new-template

# Tag (semantic version recommended)
--ref v1.0.0
--ref v2.1.3-beta

# Commit SHA (full or short)
--ref abc123def456
--ref abc123d
```

---

## 2. Project Initialization

### 2.1 `ign init`

Generate project files from template using `.ign/ign-var.json`.

**Syntax:**
```bash
ign init [flags]
```

**Examples:**
```bash
# Generate to current directory
ign init

# Generate to specific directory
ign init --output ./my-project

# Overwrite existing files
ign init --output ./my-project --overwrite

# Use custom build config location
ign init --config ./custom-build/ign-var.json --output ./output

# Dry run (show what would be generated)
ign init --dry-run
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `.` | Output directory for generated project |
| `--overwrite` | `-w` | bool | `false` | Overwrite existing files |
| `--config` | `-c` | string | `.ign/ign-var.json` | Path to variable config file |
| `--dry-run` | `-d` | bool | `false` | Show what would be generated without writing files |
| `--verbose` | `-v` | bool | `false` | Show detailed processing information |

**Process Flow:**

```
1. Read Configuration
   ├─ Load ign.json (template source and hash)
   ├─ Load ign-var.json (variables)
   ├─ Validate all required variables are set
   └─ Resolve @file: references

2. Fetch Template
   ├─ Download/clone template from URL
   ├─ Read ign.json
   └─ Validate template structure

3. Process Files
   ├─ For each file in template (except ign.json):
   │  ├─ Check if output file exists
   │  ├─ Skip if exists and --overwrite not set
   │  ├─ Process template directives:
   │  │  ├─ @ign-var: substitution
   │  │  ├─ @ign-comment: line removal (template comments)
   │  │  ├─ @ign-if: conditional blocks
   │  │  ├─ @ign-include: file inclusion
   │  │  └─ @ign-raw: literal output
   │  └─ Write processed content to output
   └─ Preserve directory structure

4. Report Results
   ├─ Files created: X
   ├─ Files skipped: Y
   └─ Files overwritten: Z
```

**Example Output:**
```
Generating project from template...

Template: github.com/owner/templates/go-basic@v1.2.0
Output: ./my-project

Processing files:
  ✓ .envrc
  ✓ .gitignore
  ✓ flake.nix
  ✓ go.mod
  ✓ main.go
  ✓ README.md
  ⊘ LICENSE (exists, use --overwrite to replace)

Summary:
  Created: 6 files
  Skipped: 1 file (already exists)

Project ready at: ./my-project
```

**Verbose Output Example:**
```bash
$ ign init --output ./project --verbose

[INFO] Reading config: .ign/ign-var.json
[INFO] Variables loaded: 5
[INFO] Fetching template: github.com/owner/repo@main
[INFO] Template cached at: ~/.cache/ign/github.com/owner/repo/main
[INFO] Processing: .envrc
[DEBUG]   - Substituted: @ign-var:project_name@ -> "my-api"
[DEBUG]   - Substituted: @ign-var:go_version@ -> "1.21"
[INFO]   ✓ Written: ./project/.envrc (142 bytes)
[INFO] Processing: main.go
[DEBUG]   - Conditional: @ign-if:enable_server@ -> true (included)
[DEBUG]   - Substituted: @ign-var:port@ -> 8080
[INFO]   ✓ Written: ./project/main.go (456 bytes)
...
```

**Dry Run Output:**
```bash
$ ign init --dry-run

[DRY RUN] Would generate:

Files to create:
  - .envrc
  - .gitignore
  - flake.nix
  - go.mod
  - main.go
  - README.md

Files that would be skipped (already exist):
  - LICENSE

Directories to create:
  - cmd/
  - pkg/
  - internal/

Variables used:
  - project_name: "my-api"
  - version: "1.0.0"
  - port: 8080
  - enable_tls: true
  - license: @file:license-header.txt

No files written (dry run).
```

### 2.2 File Handling Rules

**Default Behavior (without --overwrite):**
- Existing files: **Skip** (do not modify)
- New files: **Create**
- Empty directories: **Create** (if needed for file paths)

**With --overwrite:**
- Existing files: **Replace** with generated content
- New files: **Create**
- Backup: No automatic backup (user responsibility)

**File Permissions:**
- Executable bit is preserved from template
- Other permissions use system defaults

**Special Files:**
- `.ign/`: Never touched during `ign init`
- `ign.json`: Never copied to output (template config only)
- Hidden files (`.file`): Processed normally
- Binary files: Copied as-is (no template processing)

**Directory Handling:**
```
Template:
  dir1/
    file1.txt
    file2.txt

Output (if dir1/ doesn't exist):
  Creates: dir1/
  Creates: dir1/file1.txt
  Creates: dir1/file2.txt

Output (if dir1/ exists with file1.txt):
  Skips:   dir1/file1.txt
  Creates: dir1/file2.txt
```

### 2.3 Error Handling

**Fatal Errors (exit immediately):**
- Missing `.ign/ign.json` or `.ign/ign-var.json`
- Invalid JSON in config file
- Missing required variables
- Template fetch failure
- Invalid template directive syntax
- Circular include dependency

**Warnings (continue processing):**
- File skipped (exists, no --overwrite)
- Unknown variable in template (if not required)
- Template uses deprecated syntax

**Example Error Output:**
```
Error: Missing required variable

File: .ign/ign-var.json
Variable: "database_url" (string)

Template requires this variable but no value was provided.
Please edit .ign/ign-var.json and set a value.

Exit code: 1
```

---

## 3. Utility Commands

### 3.1 `ign version`

Display version information.

**Syntax:**
```bash
ign version
```

**Example Output:**
```
ign version 1.0.0
Built with: Go 1.21.5
Commit: abc123def456
Build date: 2025-12-09T10:30:00Z
OS/Arch: linux/amd64
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--short` | Show version number only |
| `--json` | Output as JSON |

**Short Output:**
```bash
$ ign version --short
1.0.0
```

**JSON Output:**
```bash
$ ign version --json
{
  "version": "1.0.0",
  "go_version": "1.21.5",
  "commit": "abc123def456",
  "build_date": "2025-12-09T10:30:00Z",
  "os": "linux",
  "arch": "amd64"
}
```

### 3.2 `ign help`

Display help information.

**Syntax:**
```bash
ign help [command]
```

**Examples:**
```bash
# General help
ign help

# Command-specific help
ign help build
ign help init

# Alternative syntax
ign build --help
ign init --help
```

**Help Output Format:**
```
ign - Project template initialization tool

USAGE:
  ign [command] [flags]

COMMANDS:
  build init    Create build configuration from template
  init          Generate project from build configuration
  version       Show version information
  help          Show help information

FLAGS:
  -h, --help     Show help
  -v, --version  Show version

Run 'ign help [command]' for more information about a command.
```

---

## 4. Global Flags

These flags work with all commands:

| Flag | Description | Example |
|------|-------------|---------|
| `--help`, `-h` | Show command help | `ign init --help` |
| `--version`, `-v` | Show version | `ign --version` |
| `--no-color` | Disable colored output | `ign init --no-color` |
| `--quiet`, `-q` | Suppress non-error output | `ign init --quiet` |

---

## 5. Configuration Files

### 5.1 Global Config (`~/.config/ign/config.json`)

Override default behaviors globally.

**Location:**
- Linux: `~/.config/ign/config.json`
- macOS: `~/.config/ign/config.json`

**Example:**
```json
{
  "cache_dir": "~/.cache/ign",
  "default_ref": "main",
  "github_token": "",
  "cache_ttl": 3600,
  "max_include_depth": 10,
  "color_output": true
}
```

**Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `cache_dir` | string | `~/.cache/ign` | Template cache location |
| `default_ref` | string | `main` | Default git ref if not specified |
| `github_token` | string | `""` | GitHub personal access token (for private repos) |
| `cache_ttl` | int | 3600 | Cache time-to-live in seconds |
| `max_include_depth` | int | 10 | Max nested include depth |
| `color_output` | bool | `true` | Enable colored terminal output |

**Override with --config:**
```bash
ign init --config ./custom-config.json
```

### 5.2 Environment Variables

Some settings can be configured via environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `IGN_CONFIG` | Path to global config | `export IGN_CONFIG=~/my-config.json` |
| `IGN_CACHE_DIR` | Cache directory | `export IGN_CACHE_DIR=/tmp/ign-cache` |
| `GITHUB_TOKEN` | GitHub access token | `export GITHUB_TOKEN=ghp_xxx` |
| `GH_TOKEN` | GitHub access token (alternative) | `export GH_TOKEN=ghp_xxx` |
| `IGN_NO_COLOR` | Disable colors | `export IGN_NO_COLOR=1` |

**Priority Order:**
1. Command-line flags (highest priority)
2. Environment variables (`GITHUB_TOKEN`, `GH_TOKEN`)
3. gh CLI authentication (`gh auth token`)
4. Built-in defaults (lowest priority)

---

## 6. Exit Codes

| Code | Meaning | Example Cause |
|------|---------|---------------|
| 0 | Success | Operation completed successfully |
| 1 | General error | Invalid config, missing file, validation error |
| 2 | Network error | Cannot reach GitHub, timeout |
| 3 | Template error | Invalid ign.json, missing required file |
| 4 | User error | Invalid command syntax, unknown flag |
| 130 | Interrupted | User pressed Ctrl+C |

**Usage in Scripts:**
```bash
#!/bin/bash

ign init --output ./project
if [ $? -eq 0 ]; then
  echo "Project generated successfully"
else
  echo "Failed to generate project"
  exit 1
fi
```

---

## 7. Advanced Usage

### 7.1 Batch Operations

Generate multiple projects from a list:

```bash
#!/bin/bash
# batch-generate.sh

TEMPLATES=(
  "github.com/owner/templates/go-basic"
  "github.com/owner/templates/go-api"
  "github.com/owner/templates/python-app"
)

for template in "${TEMPLATES[@]}"; do
  name=$(basename $template)
  ign build init $template --output .ign-$name
  # Edit variables...
  ign init --config .ign-$name/ign-var.json --output ./$name
done
```

### 7.2 CI/CD Integration

**GitHub Actions Example:**
```yaml
name: Generate Project
on: [push]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Download ign
        run: |
          curl -L https://github.com/owner/ign/releases/download/v1.0.0/ign-linux-amd64 -o ign
          chmod +x ign

      - name: Generate project
        run: |
          ./ign init --output ./generated

      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: generated-project
          path: ./generated
```

### 7.3 Private Repository Access

**Using gh CLI (Recommended):**
```bash
# If you have gh CLI installed and authenticated, ign will use it automatically
gh auth login

# Then use private repo - no additional setup required
ign build init github.com/private-owner/private-repo
```

**Using Environment Variable:**
```bash
# Set token in environment
export GITHUB_TOKEN=ghp_your_token_here
# or
export GH_TOKEN=ghp_your_token_here

# Then use private repo
ign build init github.com/private-owner/private-repo
```

**Token Priority Order:**
1. `GITHUB_TOKEN` environment variable
2. `GH_TOKEN` environment variable
3. `gh auth token` (gh CLI's secure credential storage)

**SSH Key Authentication:**
```bash
# Use git@ URL format
ign build init git@github.com:private-owner/private-repo.git

# Requires SSH key configured with GitHub
```

---

## 8. Troubleshooting

### 8.1 Common Issues

**Issue: "Template not found"**
```
Error: Template not found
URL: github.com/owner/repo
```

Solutions:
- Check URL is correct
- Verify repository exists and is public (or token is set for private)
- Check internet connection
- Try full URL: `https://github.com/owner/repo`

**Issue: "Missing ign.json"**
```
Error: Template missing ign.json
Path: github.com/owner/repo
```

Solutions:
- Verify template has `ign.json` in root or specified subdirectory
- Check `--ref` points to correct branch/tag
- If using subdirectory, include in URL: `owner/repo/subdir`

**Issue: "Variables not set"**
```
Error: Required variable not set
Variable: project_name (string)
File: .ign/ign-var.json
```

Solutions:
- Edit `.ign/ign-var.json`
- Set value for the variable
- Remove the variable if optional (check template ign.json)

**Issue: "File exists"**
```
Warning: File exists, skipping
File: ./my-project/main.go
```

Solutions:
- Use `--overwrite` to replace: `ign init --overwrite`
- Delete existing file manually
- Change output directory: `ign init --output ./new-project`

### 8.2 Debug Mode

Enable verbose logging to troubleshoot issues:

```bash
# Verbose output
ign init --verbose

# With debug-level logging (future feature)
ign init --log-level debug

# Output to file
ign init --verbose 2>&1 | tee ign-debug.log
```

### 8.3 Cache Issues

**Clear template cache:**
```bash
# Remove all cached templates
rm -rf ~/.cache/ign

# Remove specific template cache
rm -rf ~/.cache/ign/github.com/owner/repo
```

**Disable cache (future feature):**
```bash
ign init --no-cache
```

---

## 9. Shell Completion

### 9.1 Generate Completion Scripts (Future Feature)

```bash
# Bash
ign completion bash > /etc/bash_completion.d/ign

# Zsh
ign completion zsh > /usr/local/share/zsh/site-functions/_ign

# Fish
ign completion fish > ~/.config/fish/completions/ign.fish
```

### 9.2 Completion Features

- Command names
- Flag names
- Flag value suggestions
- File path completion
- Template URL completion (from history/cache)

---

## 10. Command Reference Summary

### Quick Command Reference

```bash
# Initialize build config
ign build init <url> [--output DIR] [--ref REF] [--force]

# Generate project
ign init [--output DIR] [--config FILE] [--overwrite] [--dry-run] [--verbose]

# Show version
ign version [--short] [--json]

# Show help
ign help [command]

# Global flags
--help, -h        Show help
--version, -v     Show version
--no-color        Disable colors
--quiet, -q       Quiet mode
```

### Common Workflows

**New Project:**
```bash
ign build init github.com/owner/template
vim .ign/ign-var.json
ign init --output ./my-project
```

**Update Existing:**
```bash
cd my-project
ign init --overwrite
```

**Different Template Version:**
```bash
ign build init github.com/owner/template --ref v2.0.0 --force
vim .ign/ign-var.json
ign init --output ./my-project-v2
```

---

## Appendix A: Command Tree

```
ign
├── build
│   └── init [URL] [flags]
│       ├── --output, -o      (string)
│       ├── --ref, -r         (string)
│       ├── --config, -c      (string)
│       └── --force, -f       (bool)
├── init [flags]
│   ├── --output, -o          (string)
│   ├── --overwrite, -w       (bool)
│   ├── --config, -c          (string)
│   ├── --dry-run, -d         (bool)
│   └── --verbose, -v         (bool)
├── version [flags]
│   ├── --short               (bool)
│   └── --json                (bool)
├── help [command]
└── [global flags]
    ├── --help, -h
    ├── --version, -v
    ├── --no-color
    └── --quiet, -q
```

## Appendix B: Flag Reference

| Flag | Commands | Type | Description |
|------|----------|------|-------------|
| `--config`, `-c` | `build init`, `init` | string | Config file path |
| `--dry-run`, `-d` | `init` | bool | Show actions without execution |
| `--force`, `-f` | `build init` | bool | Overwrite existing build config |
| `--help`, `-h` | All | bool | Show help |
| `--json` | `version` | bool | JSON output format |
| `--no-color` | All | bool | Disable colored output |
| `--output`, `-o` | `build init`, `init` | string | Output directory |
| `--overwrite`, `-w` | `init` | bool | Overwrite existing files |
| `--quiet`, `-q` | All | bool | Suppress output |
| `--ref`, `-r` | `build init` | string | Git reference (branch/tag/commit) |
| `--short` | `version` | bool | Short version output |
| `--verbose`, `-v` | `init` | bool | Verbose output |
| `--version`, `-v` | All | bool | Show version |
