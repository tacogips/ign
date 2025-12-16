# Template Syntax Reference

This document describes the complete template syntax used in ign templates.

## Overview

Ign uses a custom template syntax with the `@ign-` prefix to avoid conflicts with common programming language syntax. All directives follow the pattern `@ign-DIRECTIVE:ARGS@`.

### Design Rationale

- **Unique markers**: `@ign-` prefix avoids conflicts with existing code
- **No escaping issues**: Unlike `{{}}` or `<%%>`, the syntax rarely needs escaping
- **Explicit and readable**: Clear intent in template files
- **Language agnostic**: Works with any file type (code, config, markdown, etc.)

---

## 1. Variable Substitution

### 1.1 Variable Syntax Overview

Ign supports four syntax variants for variable substitution:

| Syntax | Description | Required/Optional |
|--------|-------------|-------------------|
| `@ign-var:NAME@` | Basic variable | Required |
| `@ign-var:NAME:TYPE@` | Variable with explicit type | Required |
| `@ign-var:NAME=DEFAULT@` | Variable with default value | Optional |
| `@ign-var:NAME:TYPE=DEFAULT@` | Variable with type and default | Optional |

**Key Rule:** Variables WITHOUT a default value are **required** (must be defined in ign-var.json). Variables WITH a default value are **optional** (use default if not defined).

### 1.2 Basic Variable: `@ign-var:NAME@`

Simple variable substitution. The variable must be defined in ign-var.json.

**Syntax:**
```
@ign-var:VARIABLE_NAME@
```

**Example Template:**
```go
package main

const ProjectName = "@ign-var:project_name@"
const Version = "@ign-var:version@"
const Port = @ign-var:port@
```

**With ign-var.json:**
```json
{
  "variables": {
    "project_name": "my-api",
    "version": "1.0.0",
    "port": 8080
  }
}
```

**Generated Output:**
```go
package main

const ProjectName = "my-api"
const Version = "1.0.0"
const Port = 8080
```

### 1.3 Variable with Type: `@ign-var:NAME:TYPE@`

Variable with explicit type validation. The variable must be defined and must match the specified type.

**Syntax:**
```
@ign-var:VARIABLE_NAME:TYPE@
```

**Supported Types:** `string`, `int`, `bool`

**Example Template:**
```go
const Port = @ign-var:port:int@
const Debug = @ign-var:debug:bool@
const Name = "@ign-var:name:string@"
```

**With ign-var.json:**
```json
{
  "variables": {
    "port": 8080,
    "debug": true,
    "name": "my-service"
  }
}
```

**Type Mismatch Error:**
```
Error: variable port: type mismatch, expected int but got string
```

### 1.4 Variable with Default: `@ign-var:NAME=DEFAULT@`

Variable with a default value. If not defined in ign-var.json, the default value is used.

**Syntax:**
```
@ign-var:VARIABLE_NAME=DEFAULT_VALUE@
```

**Default Value Type Inference:**
- `true` or `false` -> bool
- Numeric string (e.g., `8080`) -> int
- Anything else -> string

**Example Template:**
```go
const Host = "@ign-var:host=localhost@"
const Port = @ign-var:port=8080@
const Debug = @ign-var:debug=false@
const Version = "@ign-var:version=1.0.0@"
```

**With ign-var.json (partial):**
```json
{
  "variables": {
    "port": 3000
  }
}
```

**Generated Output:**
```go
const Host = "localhost"      // default used (not in ign-var.json)
const Port = 3000             // from ign-var.json (overrides default)
const Debug = false           // default used
const Version = "1.0.0"       // default used
```

### 1.5 Variable with Type and Default: `@ign-var:NAME:TYPE=DEFAULT@`

Variable with both explicit type validation and a default value.

**Syntax:**
```
@ign-var:VARIABLE_NAME:TYPE=DEFAULT_VALUE@
```

**Example Template:**
```go
const Port = @ign-var:port:int=8080@
const Debug = @ign-var:debug:bool=false@
const Author = "@ign-var:author:string=anonymous@"
```

**Behavior:**
1. If variable exists in ign-var.json, use that value and validate type
2. If variable does not exist, use default value
3. Type validation applies to both provided and default values

### 1.6 Variable Types

Ign supports three primitive types:

| Type | Example Value | JSON Type | Usage |
|------|--------------|-----------|-------|
| `string` | `"hello"` | `string` | Text values, names, paths |
| `int` | `8080` | `number` | Ports, counts, numeric IDs |
| `bool` | `true`, `false` | `boolean` | Feature flags, toggles |

**Type Handling:**
- Strings are inserted as-is (no quotes added by ign)
- Integers are converted to string representation
- Booleans are converted to "true" or "false"
- Variables must be explicitly quoted in templates if quotes are needed

**Example:**
```yaml
# Template
name: @ign-var:service_name@
port: @ign-var:port@
debug: @ign-var:debug_mode@

# ign-var.json
{
  "variables": {
    "service_name": "api-gateway",
    "port": 3000,
    "debug_mode": true
  }
}

# Generated
name: api-gateway
port: 3000
debug: true
```

### 1.7 File-based Variables: `@file:PATH`

Load variable value from a file (useful for large content like license headers, code snippets).

**Syntax in ign-var.json:**
```json
{
  "variables": {
    "license_header": "@file:license-header.txt",
    "readme_template": "@file:templates/readme-template.md"
  }
}
```

**Rules:**
- Paths are relative to `.ign-config/` directory
- File content is read as-is (preserves whitespace, newlines)
- Files are read during `ign init` execution
- Missing files cause an error

**Example:**

`.ign-config/license-header.txt`:
```
Copyright (c) 2025 My Company
Licensed under MIT License
```

`.ign-config/ign-var.json`:
```json
{
  "variables": {
    "license": "@file:license-header.txt"
  }
}
```

Template file:
```go
/*
@ign-var:license@
*/

package main
```

Generated:
```go
/*
Copyright (c) 2025 My Company
Licensed under MIT License
*/

package main
```

### 1.8 Filename Variable Substitution

Variables can be used in filenames and directory names, allowing dynamic file naming based on template variables.

**Supported Directives in Filenames:**
- `@ign-var:NAME@` - Variable substitution (all variants: with type, with default, etc.)
- `@ign-raw:CONTENT@` - Raw/escape directive for literal `@` characters

**IMPORTANT:** Only `@ign-var:` and `@ign-raw:` directives are processed in filenames. Other directives (`@ign-if:`, `@ign-comment:`, `@ign-include:`) are kept as-is in the filename and NOT processed. This is a security and simplicity design decision.

**Syntax:**
- Use the same `@ign-var:NAME@` syntax in file and directory names
- All variable syntaxes are supported (with type, with default, etc.)
- Variables are processed before files are written to disk
- Use `@ign-raw:` to escape `@` characters in filenames (e.g., email addresses)

**Template Structure Example:**
```
<ign-root>/
├── ign.json
├── cmd/@ign-var:app_name@/
│   └── main.go
├── @ign-var:project_name@.go
└── config-@ign-var:env@.yaml
```

**With ign-var.json:**
```json
{
  "variables": {
    "app_name": "myapp",
    "project_name": "handler",
    "env": "production"
  }
}
```

**Generated File Structure:**
```
<output-dir>/
├── cmd/myapp/
│   └── main.go
├── handler.go
└── config-production.yaml
```

**Rules:**
- Variables in filenames are processed the same way as content variables
- Type validation applies (use string variables for filenames)
- Default values work in filenames
- Required variables must be provided
- The resulting path must be valid for the target filesystem
- Path traversal (e.g., `../`) in variable values is rejected for security

**Example 1: Project-Specific Files**

Template:
```
templates/
├── @ign-var:project_name@/
│   ├── @ign-var:project_name@.go
│   └── @ign-var:project_name@_test.go
```

With `project_name: "auth"`:
```
output/
├── auth/
│   ├── auth.go
│   └── auth_test.go
```

**Example 2: Environment-Specific Configuration**

Template:
```
config/
├── @ign-var:env=dev@.yaml
└── secrets-@ign-var:env=dev@.yaml
```

With `env: "staging"`:
```
config/
├── staging.yaml
└── secrets-staging.yaml
```

Without `env` variable (uses default):
```
config/
├── dev.yaml
└── secrets-dev.yaml
```

**Example 3: Multiple Variables in Path**

Template:
```
src/@ign-var:module_name@/@ign-var:component_type@/@ign-var:component_name@.go
```

With variables:
```json
{
  "module_name": "api",
  "component_type": "handlers",
  "component_name": "user"
}
```

Generated:
```
src/api/handlers/user.go
```

**Escaping `@` in Filenames:**

Use `@ign-raw:` to include literal `@` characters in filenames:

Template filename: `support@ign-raw:@@company.com.txt`

Generated: `support@company.com.txt`

**Other Directives Are NOT Processed:**

Filenames do NOT process `@ign-if:`, `@ign-comment:`, or `@ign-include:` directives - they are kept literally:

Template filename: `config@ign-if:prod@-prod@ign-endif@.yaml`

Generated: `config@ign-if:prod@-prod@ign-endif@.yaml` (kept as-is, NOT processed)

This design ensures filenames remain predictable and secure without complex conditional logic.

**Security and Validation:**
- Variable values containing `..` are rejected
- Absolute paths in variable values are rejected
- Invalid filesystem characters cause errors
- Empty variable values in filenames cause errors

**Error Example:**

Template filename: `@ign-var:name@.go`

With malicious variable:
```json
{
  "name": "../etc/passwd"
}
```

Error:
```
Error: Invalid filename variable value: "../etc/passwd" contains path traversal
```

---

## 2. Template Comments

### 2.1 Template Comment: `@ign-comment:TEXT@`

A template-only comment that is removed from the output when `ign init` is executed.

**Purpose:**
- Add notes, TODOs, or explanations visible only in the template source
- Document template logic without affecting generated output
- Leave instructions for template maintainers

**Syntax:**
```
@ign-comment:any text here@
```

**Behavior:**
1. The entire line containing `@ign-comment:TEXT@` is removed from output
2. The text after the colon is free-form content (not a variable reference)
3. Only whitespace is allowed before and after the directive on the same line

**Validation Rules:**
- The directive must be on its own line (only whitespace before/after)
- Non-whitespace characters before or after the directive cause an error

**Example 1: Basic Template Comment**

Template:
```go
package main
@ign-comment:TODO: Add error handling later@

func main() {
    @ign-comment:This function will be customized per project@
    fmt.Println("Hello, @ign-var:project_name@!")
}
```

Output (with `project_name: "myapp"`):
```go
package main

func main() {
    fmt.Println("Hello, myapp!")
}
```

**Example 2: Documentation for Template Maintainers**

Template:
```yaml
@ign-comment:Database configuration section@
@ign-comment:Supported values: postgres, mysql, sqlite@
database:
  type: @ign-var:db_type@
  host: @ign-var:db_host@
```

Output (with `db_type: "postgres"`, `db_host: "localhost"`):
```yaml
database:
  type: postgres
  host: localhost
```

**Example 3: Invalid Usage (Error)**

```
code @ign-comment:this will error@
```
Error: `@ign-comment directive must be on its own line (non-whitespace found before directive)`

```
@ign-comment:comment@ more code
```
Error: `@ign-comment directive must be on its own line (non-whitespace found after directive)`

### 2.2 Note on Variable Extraction

`@ign-comment:` directives are NOT treated as variable references. They do not appear in the list of variables extracted by `ign info` or similar commands.

---

## 3. Raw/Escape Directive

### 3.1 Raw Output: `@ign-raw:CONTENT@`

Output content literally without processing ign directives.

**Purpose:**
- Include literal `@ign-*` text in output
- Escape template syntax
- Document template syntax in generated files

**Syntax:**
```
@ign-raw:LITERAL_CONTENT@
```

**Example 1: Literal Template Syntax**

Template:
```markdown
To use a variable, write: @ign-raw:@ign-var:myvar@@
```

Generated:
```markdown
To use a variable, write: @ign-var:myvar@
```

**Example 2: Documentation**

Template:
```go
// This file was generated from a template.
// Template syntax: @ign-raw:@ign-var:NAME@@ for variables
```

Generated:
```go
// This file was generated from a template.
// Template syntax: @ign-var:NAME@ for variables
```

**Example 3: Nested Directives**

Raw blocks are NOT processed recursively:
```
@ign-raw:This @ign-var:name@ will not be replaced@
```

Output:
```
This @ign-var:name@ will not be replaced
```

---

## 4. Conditional Blocks

### 4.1 If Directive: `@ign-if:VAR@...@ign-endif@`

Conditionally include or exclude blocks based on boolean variables.

**Syntax:**
```
@ign-if:VARIABLE_NAME@
... content to include if variable is true ...
@ign-endif@
```

**Rules:**
- Variable must be boolean type
- If `true`: block content is included and processed
- If `false`: entire block is removed (including newlines)
- Supports nested conditionals
- Supports `@ign-else@` for alternative content

**Example 1: Basic Conditional**

Template:
```go
type Config struct {
    Host string
    @ign-if:use_tls@
    TLSCert string
    TLSKey  string
    @ign-endif@
}
```

With `use_tls: true`:
```go
type Config struct {
    Host string
    TLSCert string
    TLSKey  string
}
```

With `use_tls: false`:
```go
type Config struct {
    Host string
}
```

**Example 2: If-Else**

Template:
```go
func NewServer() *Server {
    @ign-if:use_cache@
    return &Server{Cache: NewRedisCache()}
    @ign-else@
    return &Server{Cache: NewMemoryCache()}
    @ign-endif@
}
```

With `use_cache: true`:
```go
func NewServer() *Server {
    return &Server{Cache: NewRedisCache()}
}
```

With `use_cache: false`:
```go
func NewServer() *Server {
    return &Server{Cache: NewMemoryCache()}
}
```

**Example 3: Nested Conditionals**

Template:
```yaml
features:
  @ign-if:enable_api@
  api:
    enabled: true
    @ign-if:api_requires_auth@
    auth: jwt
    @ign-endif@
  @ign-endif@
```

**Example 4: Multiple Features**

Template:
```go
import (
    "fmt"
    @ign-if:use_database@
    "database/sql"
    @ign-endif@
    @ign-if:use_http@
    "net/http"
    @ign-endif@
)
```

### 4.2 Else Directive: `@ign-else@`

**Syntax:**
```
@ign-if:VARIABLE@
... content if true ...
@ign-else@
... content if false ...
@ign-endif@
```

**Rules:**
- Must be inside an `@ign-if:` block
- Only one `@ign-else@` per `@ign-if:` block
- Cannot be used independently

### 4.3 Endif Directive: `@ign-endif@`

**Syntax:**
```
@ign-if:VARIABLE@
... content ...
@ign-endif@
```

**Rules:**
- Required to close every `@ign-if:` block
- Must match the nesting level
- Standalone (no arguments)

---

## 5. File Inclusion

### 5.1 Include Directive: `@ign-include:PATH@`

Include content from another file.

**Syntax:**
```
@ign-include:PATH@
```

**Path Resolution:**
- **Relative paths** (no protocol): Relative to current template file
- **Absolute paths within template**: Start from ign root (template repository root)
- **GitHub URLs**: `github:OWNER/REPO/PATH@REF` format (future feature)

**Example 1: Local Include**

Template structure:
```
<ign-root>/
├── ign.json
├── main.go.template
└── common/
    └── license.txt
```

`main.go.template`:
```go
@ign-include:common/license.txt@

package main
```

**Example 2: Absolute Path**

```go
@ign-include:/common/license.txt@
```

**Example 3: Include with Processing**

Included files are processed for ign directives:

`common/header.txt`:
```
// Project: @ign-var:project_name@
// Version: @ign-var:version@
```

Main template:
```go
@ign-include:common/header.txt@

package main
```

Generated:
```go
// Project: my-service
// Version: 1.0.0

package main
```

### 5.2 Include Processing Rules

**Processing Order:**
1. Read include file
2. Process ign directives in included content
3. Insert processed content at include location
4. Continue processing main file

**Recursive Includes:**
- Supported (includes can include other files)
- Circular includes are detected and cause an error
- Maximum include depth: 10 levels (configurable)

**Error Handling:**
- Missing include file: fatal error
- Circular dependency: fatal error with dependency chain
- Invalid path: fatal error

**Example: Circular Include Detection**

`a.txt`:
```
@ign-include:b.txt@
```

`b.txt`:
```
@ign-include:a.txt@
```

Error:
```
Error: Circular include detected: a.txt -> b.txt -> a.txt
```

---

## 6. Directive Combination Examples

### 6.1 Variables with Conditionals

```go
type Server struct {
    Name string
    @ign-if:enable_metrics@
    MetricsPort int
    @ign-endif@
}

func NewServer() *Server {
    return &Server{
        Name: "@ign-var:server_name@",
        @ign-if:enable_metrics@
        MetricsPort: @ign-var:metrics_port@,
        @ign-endif@
    }
}
```

### 6.2 Include with Conditionals

```yaml
@ign-include:common/base-config.yaml@

features:
  @ign-if:enable_experimental@
  @ign-include:configs/experimental-features.yaml@
  @ign-endif@
```

### 6.3 Template Comments in Conditionals

```go
func init() {
    @ign-comment:Setup logger based on configuration@
    @ign-if:use_custom_logger@
    logger = NewCustomLogger()
    @ign-else@
    logger = log.Default()
    @ign-endif@
}
```

---

## 7. Special Cases and Edge Cases

### 7.1 Empty Values

**String Variables:**
- Empty string `""` is valid
- Substitutes to empty string (not an error)

**Boolean Variables:**
- Must be `true` or `false`
- Missing boolean variable: error
- Empty string as boolean: error

**Integer Variables:**
- Zero `0` is valid
- Missing integer: error
- Non-numeric value: error

### 7.2 Whitespace Handling

**Directives:**
- Leading/trailing whitespace in directive names is trimmed
- `@ign-var: name @` same as `@ign-var:name@`

**Content:**
- Whitespace in variable values is preserved
- Newlines in file-based variables are preserved
- Block directive indentation is preserved

**Example:**
```yaml
# Template
description: "@ign-var:description@"

# Variable with newlines
{
  "description": "Line 1\nLine 2\nLine 3"
}

# Generated
description: "Line 1
Line 2
Line 3"
```

### 7.3 Multiline Blocks

**Conditionals:**
```go
@ign-if:feature_enabled@
// This is a multi-line block
// with several lines
// that will all be included or excluded
@ign-endif@
```

**Includes:**
- Included files can contain multiple lines
- Line breaks are preserved

### 7.4 Special Characters

**In Variable Names:**
- Allowed: `a-z`, `A-Z`, `0-9`, `_`, `-`
- Must start with letter
- Case-sensitive

**In Variable Values:**
- All characters allowed (including newlines, quotes, special chars)
- No escaping needed (value is used as-is)

### 7.5 Directive in Comments

Template directives work inside comment blocks:

```go
/*
 * Generated configuration
 * Project: @ign-var:project_name@
 * Author: @ign-var:author@
 */
```

---

## 8. Error Cases

### 8.1 Common Errors

| Error | Cause | Example |
|-------|-------|---------|
| Unknown directive | Invalid `@ign-*` prefix | `@ign-loop:@` |
| Missing variable | Variable not in ign-var.json | `@ign-var:undefined@` |
| Type mismatch | Boolean expected, got string | `@ign-if:not_a_bool@` |
| Unclosed block | Missing `@ign-endif@` | See below |
| Include not found | File doesn't exist | `@ign-include:missing.txt@` |
| Circular include | A includes B, B includes A | See section 5.2 |

**Example: Unclosed Block**
```go
@ign-if:feature@
  some content
// Missing @ign-endif@
```

Error:
```
Error: Unclosed @ign-if: block at line 1
```

### 8.2 Error Reporting

Errors include:
- File path
- Line number
- Directive type
- Detailed message

Example:
```
Error in template: main.go.template:15
  Unknown variable: @ign-var:undefined_var@
  Available variables: project_name, version, port
```

---

## 9. Best Practices

### 9.1 Variable Naming

**DO:**
- Use descriptive names: `database_connection_string`
- Use snake_case: `project_name`
- Group related vars: `api_host`, `api_port`, `api_timeout`

**DON'T:**
- Single letters: `@ign-var:a@`
- Ambiguous: `@ign-var:temp@`
- Mixed case: `@ign-var:ProjectName@`

### 9.2 Conditional Organization

**Keep conditionals focused:**
```go
// GOOD: Clear, single-purpose
@ign-if:enable_cache@
cache := NewCache()
@ign-endif@

// AVOID: Too much in one conditional
@ign-if:enable_everything@
// 50 lines of mixed features
@ign-endif@
```

### 9.3 File Organization

**Use includes for:**
- Repeated content (license headers, common configs)
- Large blocks (entire function/class definitions)
- Shared configurations

**Don't use includes for:**
- Single-line content
- Content that varies per file
- Over-abstraction

### 9.4 Template Comments

**Use `@ign-comment:` when:**
- Adding notes or TODOs visible only in template source
- Documenting template logic for maintainers
- Leaving instructions about variable usage

**Example:**
```go
@ign-comment:This template generates a basic Go service@
@ign-comment:Required variables: project_name, author@
package main

func main() {
    @ign-comment:TODO: Add proper error handling@
    fmt.Println("Hello, @ign-var:project_name@!")
}
```

The comment lines are completely removed from the generated output.

---

## 10. Template Validation

### 10.1 Syntax Validation

Ign validates templates before processing:

1. **Directive syntax**: All `@ign-*:*@` directives are well-formed
2. **Block matching**: All `@ign-if:` have matching `@ign-endif@`
3. **Variable references**: All `@ign-var:` and `@ign-if:` reference defined variables
4. **Include paths**: All `@ign-include:` paths exist

### 10.2 Validation Command

```bash
# Validate template without generating (future feature)
ign validate

# Validate shows:
# - Missing variables
# - Unclosed blocks
# - Invalid includes
# - Type mismatches
```

---

## 11. Future Extensions (TODO)

### 11.1 Planned Features

- **Loops**: `@ign-for:ITEM in LIST@` ... `@ign-endfor@`
- **Filters**: `@ign-var:name|uppercase@`
- **Arithmetic**: `@ign-var:port+1@`
- **String interpolation**: `@ign-var:protocol@://host`
- **Negation**: `@ign-if-not:VAR@`
- **Elif**: `@ign-elif:VAR@`

### 11.2 Under Consideration

- **Remote includes**: `@ign-include:github:owner/repo/path@ref@`
- **Custom delimiters**: Override `@ign-*@` markers per template

---

## Appendix A: Complete Directive Reference

| Directive | Syntax | In Content | In Filename | Description |
|-----------|--------|------------|-------------|-------------|
| `@ign-var:` | `@ign-var:NAME@` | ✓ | ✓ | Variable substitution (required) |
| `@ign-var:` | `@ign-var:NAME:TYPE@` | ✓ | ✓ | Variable with type validation (required) |
| `@ign-var:` | `@ign-var:NAME=DEFAULT@` | ✓ | ✓ | Variable with default (optional) |
| `@ign-var:` | `@ign-var:NAME:TYPE=DEFAULT@` | ✓ | ✓ | Variable with type and default (optional) |
| `@ign-raw:` | `@ign-raw:CONTENT@` | ✓ | ✓ | Literal output (escape `@` characters) |
| `@ign-comment:` | `@ign-comment:TEXT@` | ✓ | ✗ | Template comment (line removed from output) |
| `@ign-if:` | `@ign-if:VAR@...@ign-endif@` | ✓ | ✗ | Conditional block |
| `@ign-else@` | `@ign-else@` | ✓ | ✗ | Alternative block for if |
| `@ign-endif@` | `@ign-endif@` | ✓ | ✗ | End conditional block |
| `@ign-include:` | `@ign-include:PATH@` | ✓ | ✗ | Include file content |

**Note:** Only `@ign-var:` and `@ign-raw:` are processed in filenames. Other directives appear literally in the filename without processing.

## Appendix B: Variable Type Reference

| Type | JSON Type | Example | Notes |
|------|-----------|---------|-------|
| `string` | `string` | `"hello"` | Any text, preserved as-is |
| `int` | `number` | `8080` | Integer values only |
| `bool` | `boolean` | `true`, `false` | Used in conditionals |

## Appendix C: Variable Syntax Quick Reference

| Syntax | Required | Type | Default | Example |
|--------|----------|------|---------|---------|
| `@ign-var:name@` | Yes | inferred | - | `@ign-var:project_name@` |
| `@ign-var:name:type@` | Yes | explicit | - | `@ign-var:port:int@` |
| `@ign-var:name=value@` | No | inferred | value | `@ign-var:host=localhost@` |
| `@ign-var:name:type=value@` | No | explicit | value | `@ign-var:port:int=8080@` |
