# ign - System Specification

> This is the main specification document. Detailed references are in separate files.

## Documentation Structure

```
docs/
├── spec.md                          # This file - Core specification
├── reference/
│   ├── template-syntax.md           # Template directive reference
│   ├── cli-commands.md              # CLI command reference
│   └── configuration.md             # Configuration file formats
└── implementation/
    └── architecture.md              # Package structure, interfaces, error handling
```

---

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

### 2.3 Template Sources

Templates can be sourced from:

**Currently Supported:**
- GitHub repositories: `github.com/owner/repo`
- Local filesystem: Relative paths only (e.g., `./template`, `templates/go-basic`)
  - `..` is NOT allowed in paths (security restriction)
  - Absolute paths are NOT allowed (portability)

**Future Support:**
- GitLab
- Bitbucket
- Other Git hosting services

The implementation uses an abstraction layer (interface) via the TemplateProvider pattern

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

## 3. User Workflow

### 3.1 Three-Step Process

```
Step 1: Initialize build configuration
+------------------------------------------------------------------+
$ ign build init github.com/owner/templates/go-basic
  -> Creates .ign-build/ign-var.json

Step 2: Edit variables
+------------------------------------------------------------------+
$ vim .ign-build/ign-var.json
$ vim .ign-build/license-header.txt  (if using @file:)

Step 3: Generate project
+------------------------------------------------------------------+
$ ign init --output ./my-project
  -> Generates project from template
```

### 3.2 Basic Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `ign build init <source>` | Create `.ign-build/` with `ign-var.json` | `ign build init github.com/owner/repo`<br>`ign build init ./my-template` |
| `ign init` | Generate project from `.ign-build/ign-var.json` | `ign init --output ./my-project` |
| `ign init --overwrite` | Regenerate, overwriting existing files | `ign init --overwrite` |

**Template sources:**
- GitHub: `github.com/owner/repo` or `github.com/owner/repo/path/to/template`
- Local: `./relative/path` only (no `..`, no absolute paths)

See [CLI Reference](reference/cli-commands.md) for complete command documentation.

---

## 4. Template Syntax Overview

| Directive | Syntax | Description |
|-----------|--------|-------------|
| Variable | `@ign-var:NAME@` | Simple variable substitution |
| Comment-style | `@ign-comment:NAME@` | Variable with comment markers removed |
| Raw/Escape | `@ign-raw:CONTENT@` | Output literally without processing |
| Conditional | `@ign-if:VAR@...@ign-endif@` | Conditional block |
| Include | `@ign-include:PATH@` | Include another file |

See [Template Syntax Reference](reference/template-syntax.md) for detailed syntax documentation.

---

## 5. Configuration Files

### Quick Reference

| File | Location | Purpose |
|------|----------|---------|
| `ign.json` | Template root | Template definition (not deployed) |
| `ign-var.json` | `.ign-build/` | User variables and template reference |
| `ign-list.json` | Any | Collection of template references |
| `config.json` | `~/.config/ign/` | Global ign configuration |

See [Configuration Reference](reference/configuration.md) for complete file format documentation.

---

## 6. Design Decisions Summary

### Key Decisions

| Question | Decision |
|----------|----------|
| Template syntax | `@ign-var:VAR@` with custom directives |
| Template sources | GitHub URLs, local relative paths (no `..`, no absolute) |
| Variable types | `string`, `int`, `bool` only |
| File-based variables | `@file:` prefix, paths relative to `.ign-build/` |
| Lock file | **Not used** (one-shot generation) |
| Merge strategy | **None** (skip existing or explicit overwrite) |
| Config override | `--config` flag |

---

## 7. Implementation

See [Architecture Documentation](implementation/architecture.md) for:
- Package structure
- Key interfaces
- Error handling strategy

---

## 8. TODO Items

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

- [ ] **Additional providers** - GitLab, Bitbucket support
- [ ] **Template versioning** - Support for template versions/tags
- [ ] **Template inheritance** - Extend/override existing templates
- [ ] **Post-init hooks** - Run scripts after project generation
- [ ] **Loops support** - `@ign-for:` directive if needed
- [ ] **Filters/Transforms** - `@ign-var:NAME|filter@` syntax if needed

---

## 9. Revision History

| Version | Date | Changes |
|---------|------|---------|
| 0.1 | 2025-12-09 | Initial draft |
| 0.2 | 2025-12-09 | Added template directives, resolved core syntax questions |
| 0.3 | 2025-12-09 | Simplified design: removed lock file, added build workflow |
| 0.4 | 2025-12-09 | Split documentation into multiple focused files, removed "Removed Features" section, added local filesystem support as current feature |
