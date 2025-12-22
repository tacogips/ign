# Implementation Architecture

This document describes the internal architecture, package structure, interfaces, and implementation details for the ign project.

---

## 1. Architecture Overview

### 1.1 High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│  (cobra commands, flags, user interaction)                      │
└────────────────┬────────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────────┐
│                     Application Layer                           │
│  (build init, init workflows, orchestration)                    │
└─┬──────────┬──────────┬──────────┬──────────┬──────────────────┘
  │          │          │          │          │
  ▼          ▼          ▼          ▼          ▼
┌────┐  ┌────────┐  ┌──────┐  ┌──────┐  ┌──────────┐
│Tmpl│  │Template│  │ Var  │  │Cache │  │ Template │
│Prov│  │ Parser │  │ Mgmt │  │ Mgmt │  │Generator │
└────┘  └────────┘  └──────┘  └──────┘  └──────────┘
  │          │          │          │          │
  │          │          │          │          │
┌─▼──────────▼──────────▼──────────▼──────────▼──────────────────┐
│                      Core/Domain Layer                          │
│  (interfaces, domain models, business logic)                    │
└─────────────────────────────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                   Infrastructure Layer                          │
│  (filesystem, HTTP, Git, JSON parsing)                          │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Design Principles

| Principle | Description |
|-----------|-------------|
| **Clean Architecture** | Dependency inversion, interface-based design |
| **Provider Pattern** | Abstraction for template sources (GitHub, future providers) |
| **Single Responsibility** | Each package has one clear purpose |
| **Interface Segregation** | Small, focused interfaces |
| **Testability** | All components mockable, unit testable |
| **Extensibility** | Easy to add new providers, directives, features |

---

## 2. Package Structure

### 2.1 Project Layout

Following Standard Go Project Layout:

```
ign/
├── cmd/
│   └── ign/
│       └── main.go                    # Entry point
│
├── internal/
│   ├── app/
│   │   ├── build.go                   # Build workflow implementation
│   │   ├── init.go                    # Init workflow implementation
│   │   └── workflows.go               # Common workflow logic
│   │
│   ├── cli/
│   │   ├── root.go                    # Root command
│   │   ├── build.go                   # Build command
│   │   ├── init.go                    # Init command
│   │   └── version.go                 # Version command
│   │
│   ├── config/
│   │   ├── config.go                  # Global config management
│   │   ├── loader.go                  # Config file loader
│   │   └── validation.go              # Config validation
│   │
│   ├── template/
│   │   ├── provider/
│   │   │   ├── provider.go            # Provider interface
│   │   │   ├── github.go              # GitHub implementation
│   │   │   └── factory.go             # Provider factory
│   │   │
│   │   ├── parser/
│   │   │   ├── parser.go              # Template parser interface
│   │   │   ├── directive.go           # Directive parsing
│   │   │   ├── variable.go            # Variable substitution
│   │   │   ├── conditional.go         # Conditional blocks
│   │   │   ├── include.go             # File inclusion
│   │   │   └── comment.go             # Comment removal
│   │   │
│   │   ├── generator/
│   │   │   ├── generator.go           # Project generator
│   │   │   ├── processor.go           # File processor
│   │   │   └── writer.go              # File writer
│   │   │
│   │   └── model/
│   │       ├── template.go            # Template domain model
│   │       ├── variable.go            # Variable definitions
│   │       └── directive.go           # Directive models
│   │
│   ├── cache/
│   │   ├── cache.go                   # Cache interface
│   │   ├── filesystem.go              # Filesystem cache impl
│   │   └── manager.go                 # Cache management
│   │
│   ├── vcs/
│   │   ├── git.go                     # Git operations
│   │   └── clone.go                   # Repository cloning
│   │
│   └── util/
│       ├── files.go                   # File utilities
│       ├── json.go                    # JSON helpers
│       └── validation.go              # Common validation
│
├── pkg/
│   └── ignconfig/
│       ├── types.go                   # Public config types
│       └── schema.go                  # JSON schemas
│
├── test/
│   ├── fixtures/                      # Test fixtures
│   ├── integration/                   # Integration tests
│   └── testdata/                      # Test data
│
├── docs/                              # Documentation
├── scripts/                           # Build/dev scripts
├── go.mod
├── go.sum
├── Taskfile.yml                       # Task runner config
└── README.md
```

### 2.2 Package Responsibilities

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `cmd/ign` | Application entry point | `main()` |
| `internal/cli` | CLI commands and flags | Command handlers |
| `internal/app` | Application workflows | `BuildWorkflow`, `InitWorkflow` |
| `internal/config` | Configuration management | `Config`, `Loader` |
| `internal/template/provider` | Template source abstraction | `Provider`, `GitHubProvider` |
| `internal/template/parser` | Template parsing | `Parser`, `Directive` |
| `internal/template/generator` | Project generation | `Generator`, `Processor` |
| `internal/template/model` | Domain models | `Template`, `Variable` |
| `internal/cache` | Template caching | `Cache`, `Manager` |
| `internal/vcs` | Version control operations | `Git`, `Clone` |
| `internal/util` | Shared utilities | Helper functions |
| `pkg/ignconfig` | Public API types | `IgnJson`, `IgnVarJson` |

---

## 3. Core Interfaces

### 3.1 Template Provider

```go
// Provider abstracts template source locations (GitHub, GitLab, local, etc.)
type Provider interface {
    // Fetch downloads a template from the provider
    Fetch(ctx context.Context, ref TemplateRef) (*Template, error)

    // Validate checks if a template reference is valid
    Validate(ctx context.Context, ref TemplateRef) error

    // Resolve converts a URL string to a TemplateRef
    Resolve(url string) (TemplateRef, error)

    // Name returns the provider name (e.g., "github", "gitlab")
    Name() string
}

// TemplateRef represents a reference to a template
type TemplateRef struct {
    Provider string  // Provider name (e.g., "github")
    Owner    string  // Repository owner
    Repo     string  // Repository name
    Path     string  // Subdirectory path (optional)
    Ref      string  // Branch, tag, or commit SHA
}

// Template represents a fetched template
type Template struct {
    Ref      TemplateRef
    Config   IgnJson           // Parsed ign.json
    Files    []TemplateFile    // Template files
    RootPath string            // Local path to template
}

// TemplateFile represents a single file in the template
type TemplateFile struct {
    Path       string    // Relative path from template root
    Content    []byte    // File content
    Mode       os.FileMode
    IsBinary   bool
}
```

**Implementations:**
- `GitHubProvider` - GitHub repository access (initial)
- Future: `GitLabProvider`, `LocalProvider`, `BitbucketProvider`

### 3.2 Template Parser

```go
// Parser processes template files and substitutes variables
type Parser interface {
    // Parse processes a template file with variable substitution
    Parse(ctx context.Context, input []byte, vars Variables) ([]byte, error)

    // Validate validates template syntax without processing
    Validate(ctx context.Context, input []byte) error

    // ExtractVariables finds all variable references in a template
    ExtractVariables(input []byte) ([]string, error)
}

// Directive represents a template directive
type Directive interface {
    // Name returns the directive name (e.g., "var", "if", "include")
    Name() string

    // Process handles the directive and returns processed content
    Process(ctx context.Context, args string, vars Variables) ([]byte, error)

    // Validate checks if directive syntax is valid
    Validate(args string) error
}

// Variables holds template variable values
type Variables interface {
    // Get retrieves a variable value by name
    Get(name string) (interface{}, bool)

    // GetString retrieves a string variable
    GetString(name string) (string, error)

    // GetInt retrieves an integer variable
    GetInt(name string) (int, error)

    // GetBool retrieves a boolean variable
    GetBool(name string) (bool, error)

    // Set sets a variable value
    Set(name string, value interface{}) error

    // All returns all variables
    All() map[string]interface{}
}
```

**Directive Implementations:**
- `VarDirective` - `@ign-var:NAME@`
- `CommentDirective` - `@ign-comment:TEXT@` (template comment, line removed from output)
- `RawDirective` - `@ign-raw:CONTENT@`
- `IfDirective` - `@ign-if:VAR@...@ign-endif@`
- `IncludeDirective` - `@ign-include:PATH@`

### 3.3 Generator

```go
// Generator generates projects from templates
type Generator interface {
    // Generate creates a project from template
    Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)

    // DryRun simulates generation without writing files
    DryRun(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)
}

// GenerateOptions configures project generation
type GenerateOptions struct {
    Template    *Template
    Variables   Variables
    OutputDir   string
    Overwrite   bool
    Verbose     bool
}

// GenerateResult contains generation statistics
type GenerateResult struct {
    FilesCreated    int
    FilesSkipped    int
    FilesOverwritten int
    Errors          []error
    Files           []string
}

// Processor processes individual files during generation
type Processor interface {
    // Process processes a single template file
    Process(ctx context.Context, file TemplateFile, vars Variables) ([]byte, error)

    // ShouldProcess determines if a file should be template-processed
    ShouldProcess(file TemplateFile) bool
}
```

### 3.4 Cache

```go
// Cache stores fetched templates
type Cache interface {
    // Get retrieves a cached template
    Get(ctx context.Context, ref TemplateRef) (*Template, error)

    // Put stores a template in cache
    Put(ctx context.Context, ref TemplateRef, template *Template) error

    // Has checks if a template is cached
    Has(ctx context.Context, ref TemplateRef) bool

    // Delete removes a template from cache
    Delete(ctx context.Context, ref TemplateRef) error

    // Clear removes all cached templates
    Clear(ctx context.Context) error

    // Size returns cache size in bytes
    Size(ctx context.Context) (int64, error)
}

// CacheKey generates a unique key for a template reference
type CacheKey string

// CacheManager handles cache lifecycle
type CacheManager interface {
    // Clean removes expired or oversized cache entries
    Clean(ctx context.Context) error

    // Stats returns cache statistics
    Stats(ctx context.Context) (CacheStats, error)
}

// CacheStats contains cache statistics
type CacheStats struct {
    Entries    int
    SizeBytes  int64
    HitRate    float64
    OldestEntry time.Time
}
```

### 3.5 Config Management

```go
// ConfigLoader loads configuration files
type ConfigLoader interface {
    // Load loads configuration from file
    Load(path string) (*Config, error)

    // LoadOrDefault loads config or returns defaults
    LoadOrDefault(path string) (*Config, error)

    // Validate validates configuration
    Validate(config *Config) error
}

// Config represents global configuration
type Config struct {
    Cache     CacheConfig
    GitHub    GitHubConfig
    Templates TemplateConfig
    Output    OutputConfig
    Defaults  DefaultsConfig
}

// Individual config sections match configuration.md schema
```

---

## 4. Key Workflows

### 4.1 Build Init Workflow

```
User: ign build init github.com/owner/repo/path --ref v1.0.0

1. CLI Layer (internal/cli/build.go)
   ├─ Parse command flags
   ├─ Create BuildInitOptions
   └─ Call app.BuildInit()

2. Application Layer (internal/app/build.go)
   ├─ Resolve URL to TemplateRef
   ├─ Get Provider from factory
   ├─ Fetch template
   │  ├─ Check cache
   │  ├─ If not cached: provider.Fetch()
   │  └─ Cache result
   ├─ Parse ign.json
   ├─ Create .ign directory
   ├─ Generate ign-var.json
   │  ├─ Template reference
   │  ├─ Empty variable values
   │  └─ Metadata
   └─ Print success message

3. Provider Layer (internal/template/provider/github.go)
   ├─ Construct GitHub API URL
   ├─ Download repository archive
   ├─ Extract to temp directory
   ├─ Read ign.json
   ├─ Collect template files
   └─ Return Template

4. Output
   └─ .ign/ign-var.json created
```

### 4.2 Init Workflow

```
User: ign init --output ./my-project

1. CLI Layer (internal/cli/init.go)
   ├─ Parse command flags
   ├─ Create InitOptions
   └─ Call app.Init()

2. Application Layer (internal/app/init.go)
   ├─ Load .ign/ign-var.json
   ├─ Validate variables
   ├─ Resolve @file: references
   ├─ Get Provider and fetch template
   ├─ Create Generator
   └─ Call generator.Generate()

3. Generator Layer (internal/template/generator/)
   ├─ For each template file:
   │  ├─ Check if output exists
   │  ├─ Skip if exists and !overwrite
   │  ├─ Determine if binary
   │  ├─ If text: Process with Parser
   │  ├─ If binary: Copy as-is
   │  └─ Write to output
   └─ Return GenerateResult

4. Parser Layer (internal/template/parser/)
   ├─ Scan for directives
   ├─ Process directives in order:
   │  ├─ @ign-raw: (skip processing)
   │  ├─ @ign-include: (recursive parse)
   │  ├─ @ign-if: (evaluate condition)
   │  ├─ @ign-comment: (remove entire line)
   │  └─ @ign-var: (substitute value)
   └─ Return processed content

5. Output
   └─ Project files created in output directory
```

---

## 5. Data Models

### 5.1 Core Domain Models

```go
// IgnJson represents the ign.json configuration
type IgnJson struct {
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Description string                 `json:"description,omitempty"`
    Author      string                 `json:"author,omitempty"`
    Repository  string                 `json:"repository,omitempty"`
    License     string                 `json:"license,omitempty"`
    Tags        []string               `json:"tags,omitempty"`
    Variables   map[string]VarDef      `json:"variables"`
    Settings    *TemplateSettings      `json:"settings,omitempty"`
}

// VarDef defines a template variable
type VarDef struct {
    Type        VarType     `json:"type"`
    Description string      `json:"description"`
    Required    bool        `json:"required,omitempty"`
    Default     interface{} `json:"default,omitempty"`
    Example     interface{} `json:"example,omitempty"`
    Pattern     string      `json:"pattern,omitempty"`    // For strings
    Min         *int        `json:"min,omitempty"`        // For ints
    Max         *int        `json:"max,omitempty"`        // For ints
}

// VarType represents variable type
type VarType string

const (
    VarTypeString VarType = "string"
    VarTypeInt    VarType = "int"
    VarTypeBool   VarType = "bool"
)

// TemplateSettings contains template-specific settings
type TemplateSettings struct {
    PreserveExecutable bool     `json:"preserve_executable,omitempty"`
    IgnorePatterns     []string `json:"ignore_patterns,omitempty"`
    BinaryExtensions   []string `json:"binary_extensions,omitempty"`
    IncludeDotfiles    bool     `json:"include_dotfiles,omitempty"`
    MaxIncludeDepth    int      `json:"max_include_depth,omitempty"`
}

// IgnConfig represents the .ign/ign.json file (template reference + hash)
type IgnConfig struct {
    Template  TemplateSource   `json:"template"`
    Hash      string           `json:"hash"`
    Metadata  *ConfigMetadata  `json:"metadata,omitempty"`
}

// IgnVarJson represents the .ign/ign-var.json file (user variables only)
type IgnVarJson struct {
    Variables map[string]interface{} `json:"variables"`
}

// TemplateSource identifies the template
type TemplateSource struct {
    URL  string `json:"url"`
    Path string `json:"path,omitempty"`
    Ref  string `json:"ref,omitempty"`
}

// ConfigMetadata contains generation metadata for IgnConfig
type ConfigMetadata struct {
    GeneratedAt     time.Time `json:"generated_at,omitempty"`
    GeneratedBy     string    `json:"generated_by,omitempty"`
    TemplateName    string    `json:"template_name,omitempty"`
    TemplateVersion string    `json:"template_version,omitempty"`
    IgnVersion      string    `json:"ign_version,omitempty"`
}
```

### 5.2 Parser Models

```go
// ParseContext holds state during parsing
type ParseContext struct {
    Variables      Variables
    IncludeDepth   int
    IncludeStack   []string  // For circular include detection
    TemplateRoot   string
    CurrentFile    string
}

// DirectiveMatch represents a matched directive in text
type DirectiveMatch struct {
    Type      DirectiveType
    Start     int
    End       int
    Name      string
    Args      string
    RawText   string
}

// DirectiveType identifies directive types
type DirectiveType int

const (
    DirectiveVar DirectiveType = iota
    DirectiveComment
    DirectiveRaw
    DirectiveIf
    DirectiveElse
    DirectiveEndif
    DirectiveInclude
)
```

---

## 6. Error Handling

### 6.1 Error Types

```go
// Error types follow the pattern: <Domain>Error

// TemplateError represents template-related errors
type TemplateError struct {
    Type    TemplateErrorType
    Message string
    File    string
    Line    int
    Cause   error
}

type TemplateErrorType int

const (
    TemplateNotFound TemplateErrorType = iota
    TemplateInvalid
    TemplateParseError
    TemplateVariableMissing
    TemplateDirectiveInvalid
)

// ConfigError represents configuration errors
type ConfigError struct {
    Type    ConfigErrorType
    Message string
    File    string
    Field   string
    Cause   error
}

type ConfigErrorType int

const (
    ConfigNotFound ConfigErrorType = iota
    ConfigInvalid
    ConfigValidationFailed
)

// ProviderError represents provider-related errors
type ProviderError struct {
    Type     ProviderErrorType
    Message  string
    Provider string
    URL      string
    Cause    error
}

type ProviderErrorType int

const (
    ProviderFetchFailed ProviderErrorType = iota
    ProviderNotFound
    ProviderAuthFailed
    ProviderTimeout
)

// GeneratorError represents generation errors
type GeneratorError struct {
    Type    GeneratorErrorType
    Message string
    File    string
    Cause   error
}

type GeneratorErrorType int

const (
    GeneratorWriteFailed GeneratorErrorType = iota
    GeneratorProcessFailed
)
```

### 6.2 Error Handling Strategy

**Principles:**
1. **Fail Fast**: Return errors immediately, don't continue with invalid state
2. **Context**: Include file names, line numbers, directive details
3. **User-Friendly**: Clear messages with actionable suggestions
4. **Wrapped**: Use `fmt.Errorf` with `%w` for error chains
5. **Typed**: Use custom error types for programmatic handling

**Example Error Messages:**

```
Error: Template variable not defined

File: main.go.template:15
Variable: @ign-var:database_url@

This variable is required but not found in .ign/ign-var.json
Available variables: project_name, version, port

Please edit .ign/ign-var.json and add:
  "database_url": "your-value-here"
```

```
Error: Circular include detected

File: header.txt
Include chain: main.go.template -> header.txt -> footer.txt -> header.txt

Include directives cannot create circular dependencies.
Please restructure your template files.
```

### 6.3 Error Exit Codes

```go
const (
    ExitSuccess       = 0
    ExitGeneralError  = 1
    ExitNetworkError  = 2
    ExitTemplateError = 3
    ExitUserError     = 4
    ExitInterrupted   = 130
)
```

---

## 7. Testing Strategy

### 7.1 Test Organization

```
test/
├── unit/
│   ├── parser/          # Parser unit tests
│   ├── provider/        # Provider unit tests
│   └── generator/       # Generator unit tests
│
├── integration/
│   ├── build_test.go    # Build workflow integration tests
│   ├── init_test.go     # Init workflow integration tests
│   └── e2e_test.go      # End-to-end tests
│
├── fixtures/
│   ├── templates/       # Test templates
│   ├── configs/         # Test configurations
│   └── expected/        # Expected outputs
│
└── testdata/
    ├── valid/           # Valid test data
    └── invalid/         # Invalid test data
```

### 7.2 Testing Approach

| Layer | Test Type | Focus | Tools |
|-------|-----------|-------|-------|
| CLI | Integration | Command execution, flag parsing | `testing`, subprocess |
| App | Integration | Workflow orchestration | `testing`, mocks |
| Provider | Unit + Integration | Template fetching | `testing`, mocks, test server |
| Parser | Unit | Directive processing | `testing`, table-driven |
| Generator | Unit + Integration | File generation | `testing`, temp dirs |
| Cache | Unit | Cache operations | `testing`, temp dirs |
| VCS | Unit + Integration | Git operations | `testing`, mocks, test repos |

### 7.3 Test Utilities

```go
// TestTemplate creates a test template
func TestTemplate(t *testing.T, files map[string]string) *Template

// TestVariables creates test variables
func TestVariables(vars map[string]interface{}) Variables

// TempDir creates a temporary directory for testing
func TempDir(t *testing.T) string

// AssertFileContent checks file content matches expected
func AssertFileContent(t *testing.T, path string, expected string)

// MockProvider creates a mock template provider
func MockProvider(templates map[string]*Template) Provider
```

### 7.4 Example Test

```go
func TestVarDirective(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        vars     map[string]interface{}
        expected string
        wantErr  bool
    }{
        {
            name:     "simple string substitution",
            input:    "name: @ign-var:project_name@",
            vars:     map[string]interface{}{"project_name": "my-app"},
            expected: "name: my-app",
            wantErr:  false,
        },
        {
            name:     "integer substitution",
            input:    "port: @ign-var:port@",
            vars:     map[string]interface{}{"port": 8080},
            expected: "port: 8080",
            wantErr:  false,
        },
        {
            name:     "missing variable",
            input:    "@ign-var:missing@",
            vars:     map[string]interface{}{},
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewParser()
            vars := TestVariables(tt.vars)

            result, err := parser.Parse(context.Background(), []byte(tt.input), vars)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, string(result))
        })
    }
}
```

---

## 8. Dependencies

### 8.1 Core Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| `github.com/spf13/cobra` | CLI framework | Latest |
| `github.com/spf13/viper` | Configuration | Latest |
| `github.com/go-git/go-git/v5` | Git operations | v5 |
| Standard library | Core functionality | - |

### 8.2 Development Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/stretchr/testify` | Testing assertions |
| `github.com/google/go-cmp` | Deep comparison |
| `github.com/golangci/golangci-lint` | Linting |

### 8.3 Dependency Principles

1. **Minimize external dependencies**: Use standard library when possible
2. **Well-maintained**: Choose actively maintained libraries
3. **License compatible**: MIT, Apache 2.0, BSD
4. **No indirect dependencies on non-free software**
5. **Reproducible builds**: Use `go.mod` and `go.sum`

---

## 9. Build and Release

### 9.1 Build Process

**Using go-task:**

```yaml
# Taskfile.yml
version: '3'

tasks:
  build:
    desc: Build ign binary
    cmds:
      - go build -o bin/ign ./cmd/ign

  test:
    desc: Run tests
    cmds:
      - go test -v ./...

  lint:
    desc: Run linters
    cmds:
      - golangci-lint run

  install:
    desc: Install ign
    cmds:
      - go install ./cmd/ign
```

**Commands:**
```bash
task build    # Build binary
task test     # Run tests
task lint     # Run linters
task install  # Install to $GOPATH/bin
```

### 9.2 Release Process

**Versioning:** Semantic Versioning (semver)

**Release workflow:**
1. Update version in code
2. Update CHANGELOG.md
3. Create git tag: `git tag v1.0.0`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions builds release binaries
6. Upload binaries to GitHub Releases

**Supported platforms:**
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

---

## 10. Performance Considerations

### 10.1 Optimization Strategies

| Area | Strategy |
|------|----------|
| **Template caching** | Cache fetched templates, validate with TTL |
| **Parallel processing** | Process independent files concurrently |
| **Lazy loading** | Load template files on demand |
| **Streaming** | Stream large files instead of loading into memory |
| **Binary detection** | Quick binary check (first 512 bytes) |
| **Regex compilation** | Compile directive patterns once, reuse |

### 10.2 Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| `build init` | < 2s | Including network fetch |
| `init` (cached) | < 1s | For typical template (50 files) |
| `init` (uncached) | < 5s | Including download |
| Directive parsing | > 1MB/s | Per file |
| Memory usage | < 100MB | During typical generation |

### 10.3 Benchmarking

```go
// Benchmark directive parsing
func BenchmarkVarDirective(b *testing.B) {
    input := []byte("@ign-var:name@ " * 1000)
    vars := TestVariables(map[string]interface{}{"name": "value"})
    parser := NewParser()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(context.Background(), input, vars)
    }
}
```

---

## 11. Security Considerations

### 11.1 Threat Model

| Threat | Mitigation |
|--------|------------|
| **Malicious templates** | Sandboxed execution, no arbitrary code execution |
| **Path traversal** | Validate all paths, reject `..` |
| **Resource exhaustion** | Limits on file size, include depth |
| **Credential leakage** | Warn on committing secrets, never log tokens |
| **Supply chain** | Pin dependencies, verify checksums |

### 11.2 Security Best Practices

1. **Input validation**: Validate all user input (URLs, paths, variable names)
2. **No arbitrary code execution**: Template system is declarative only
3. **Filesystem isolation**: Only write to specified output directory
4. **Token handling**: Never log or print GitHub tokens
5. **HTTPS only**: Use HTTPS for all network requests
6. **Dependency scanning**: Regular dependency updates and security scans

### 11.3 Safe Filesystem Operations

```go
// ValidatePath ensures path is safe (no traversal)
func ValidatePath(path string) error {
    if strings.Contains(path, "..") {
        return fmt.Errorf("path contains '..': %s", path)
    }
    if filepath.IsAbs(path) && !isAllowedAbsPath(path) {
        return fmt.Errorf("absolute path not allowed: %s", path)
    }
    return nil
}

// SafeJoin joins paths safely
func SafeJoin(base, path string) (string, error) {
    if err := ValidatePath(path); err != nil {
        return "", err
    }
    joined := filepath.Join(base, path)
    if !strings.HasPrefix(joined, base) {
        return "", fmt.Errorf("path escapes base directory")
    }
    return joined, nil
}
```

---

## 12. Future Architecture Considerations

### 12.1 Plugin System (Future)

Allow custom directives and providers via plugins:

```go
// Plugin interface for custom directives
type DirectivePlugin interface {
    Name() string
    Process(ctx context.Context, args string, vars Variables) ([]byte, error)
}

// Plugin interface for custom providers
type ProviderPlugin interface {
    Name() string
    Fetch(ctx context.Context, ref TemplateRef) (*Template, error)
}

// Plugin loader
type PluginLoader interface {
    Load(path string) (Plugin, error)
    LoadAll(dir string) ([]Plugin, error)
}
```

### 12.2 Template Registry (Future)

Central registry for discovering and sharing templates:

```go
// Registry client
type RegistryClient interface {
    Search(query string) ([]TemplateInfo, error)
    Get(name string) (*Template, error)
    Publish(template *Template) error
}
```

### 12.3 Interactive Mode (Future)

Prompt for variables if not set:

```go
// Interactive variable collector
type InteractiveCollector interface {
    Collect(varDefs map[string]VarDef) (Variables, error)
    PromptString(name, description string) (string, error)
    PromptInt(name, description string, min, max *int) (int, error)
    PromptBool(name, description string) (bool, error)
}
```

---

## Appendix A: Complete Interface Listing

### Core Interfaces

```go
// Template Management
type Provider interface { ... }
type Cache interface { ... }
type CacheManager interface { ... }

// Template Processing
type Parser interface { ... }
type Directive interface { ... }
type Variables interface { ... }

// Project Generation
type Generator interface { ... }
type Processor interface { ... }

// Configuration
type ConfigLoader interface { ... }

// Future Extensions
type DirectivePlugin interface { ... }
type ProviderPlugin interface { ... }
type RegistryClient interface { ... }
type InteractiveCollector interface { ... }
```

## Appendix B: Coding Standards

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Comments on all exported symbols
- Package documentation in `doc.go`

### Naming Conventions

- Interfaces: Noun or verb-er (e.g., `Parser`, `Generator`, `Provider`)
- Implementations: Concrete names (e.g., `GitHubProvider`, `FileCache`)
- Methods: Verb phrases (e.g., `Fetch`, `Parse`, `Generate`)
- Variables: Clear, descriptive names (avoid single letters except in short scopes)

### Error Handling

```go
// GOOD: Wrapped error with context
if err != nil {
    return fmt.Errorf("failed to parse template %s: %w", filename, err)
}

// BAD: Generic error
if err != nil {
    return err
}
```

### Testing

- Table-driven tests for multiple cases
- Descriptive test names: `TestFunctionName_Scenario`
- Use `testify/require` for fatal assertions
- Use `testify/assert` for non-fatal assertions
- Clean up resources with `t.Cleanup()`

---

This architecture document provides a comprehensive blueprint for implementing the ign project. It should be updated as the implementation evolves and new design decisions are made.
