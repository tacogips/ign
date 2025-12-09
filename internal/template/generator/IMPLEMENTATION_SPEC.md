# Generator Implementation Specification

## Purpose
Implement the Project Generator layer responsible for generating project files from templates by processing template directives and writing output files.

## Reference Documents
- /g/gits/tacogips/ign/docs/implementation/architecture.md (Section 3.3: Generator)
- /g/gits/tacogips/ign/docs/reference/cli-commands.md (Section 2: Project Initialization, file handling rules)

## Architecture Context

### Dependencies
- **Parser**: `/g/gits/tacogips/ign/internal/template/parser` - For template processing
- **Model**: `/g/gits/tacogips/ign/internal/template/model` - For domain types
- **Standard Library**: `os`, `path/filepath`, `io/fs`, etc.

### Key Interfaces from Architecture

From `parser` package:
```go
type Parser interface {
    Parse(ctx context.Context, input []byte, vars Variables) ([]byte, error)
    ParseWithContext(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error)
}

type Variables interface {
    Get(name string) (interface{}, bool)
    GetString(name string) (string, error)
    // ... other methods
}

type ParseContext struct {
    Variables      Variables
    IncludeDepth   int
    IncludeStack   []string
    TemplateRoot   string
    CurrentFile    string
}
```

From `model` package:
```go
type Template struct {
    Ref      TemplateRef
    Config   IgnJson
    Files    []TemplateFile
    RootPath string
}

type TemplateFile struct {
    Path       string
    Content    []byte
    Mode       os.FileMode
    IsBinary   bool
}

type TemplateSettings struct {
    PreserveExecutable bool
    IgnorePatterns     []string
    BinaryExtensions   []string
    IncludeDotfiles    bool
    MaxIncludeDepth    int
}
```

## Implementation Requirements

### File 1: generator.go

**Purpose**: Main generator interface and orchestration logic

**Types**:
```go
// Generator generates projects from templates
type Generator interface {
    Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)
    DryRun(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)
}

// GenerateOptions configures project generation
type GenerateOptions struct {
    Template    *model.Template
    Variables   parser.Variables
    OutputDir   string
    Overwrite   bool
    Verbose     bool
}

// GenerateResult contains generation statistics
type GenerateResult struct {
    FilesCreated     int
    FilesSkipped     int
    FilesOverwritten int
    Errors          []error
    Files           []string
}

// DefaultGenerator implements Generator
type DefaultGenerator struct {
    parser    parser.Parser
    processor Processor
    writer    Writer
}
```

**Methods**:
- `NewGenerator() Generator` - Create new default generator
- `Generate(ctx, opts) (*GenerateResult, error)` - Generate project files
- `DryRun(ctx, opts) (*GenerateResult, error)` - Simulate without writing

**Logic Flow**:
1. Validate options (Template, Variables, OutputDir not nil/empty)
2. Create output directory if needed
3. For each file in Template.Files:
   - Check if should be ignored (filter.go)
   - Skip ign.json and .ign-build directory
   - Check if output file exists
   - If exists and !overwrite: skip and increment FilesSkipped
   - Determine if binary (processor.go)
   - If binary: copy as-is
   - If text: process with parser
   - Write to output (writer.go)
   - Track result
4. Return GenerateResult

### File 2: processor.go

**Purpose**: File processing logic - determine binary vs text, process templates

**Types**:
```go
// Processor processes individual files during generation
type Processor interface {
    Process(ctx context.Context, file model.TemplateFile, vars parser.Variables, templateRoot string) ([]byte, error)
    ShouldProcess(file model.TemplateFile) bool
}

// FileProcessor implements Processor
type FileProcessor struct {
    parser           parser.Parser
    binaryExtensions []string
}
```

**Methods**:
- `NewFileProcessor(p parser.Parser, binaryExts []string) Processor` - Create processor
- `Process(ctx, file, vars, templateRoot) ([]byte, error)` - Process file content
- `ShouldProcess(file) bool` - Check if file needs template processing
- `isBinary(file) bool` - Determine if file is binary

**Logic**:
- `ShouldProcess`: Returns false if file.IsBinary or matches binary extensions
- `isBinary`: Check extension against binary extensions list, check first 512 bytes for binary markers
- `Process`:
  - If binary or !ShouldProcess: return file.Content unchanged
  - Otherwise: create ParseContext and call parser.ParseWithContext
  - ParseContext should set TemplateRoot and CurrentFile

### File 3: writer.go

**Purpose**: File system operations - write files, create directories

**Types**:
```go
// Writer writes files to the filesystem
type Writer interface {
    WriteFile(path string, content []byte, mode os.FileMode) error
    CreateDir(path string) error
    Exists(path string) bool
}

// FileWriter implements Writer
type FileWriter struct {
    preserveExecutable bool
}
```

**Methods**:
- `NewFileWriter(preserveExec bool) Writer` - Create writer
- `WriteFile(path, content, mode) error` - Write file with permissions
- `CreateDir(path) error` - Create directory (with parents)
- `Exists(path) bool` - Check if file/dir exists

**Logic**:
- `WriteFile`:
  - Create parent directories if needed
  - If preserveExecutable: use provided mode
  - Otherwise: use 0644 for regular files
  - Write content atomically (temp file + rename)
- `CreateDir`: Use `os.MkdirAll` with 0755
- `Exists`: Use `os.Stat` and check error

### File 4: filter.go

**Purpose**: File filtering logic - ignore patterns, special files

**Functions**:
```go
// ShouldIgnoreFile checks if a file should be ignored during generation
func ShouldIgnoreFile(path string, ignorePatterns []string) bool

// IsSpecialFile checks if a file is a special file that should be excluded
func IsSpecialFile(path string) bool

// MatchesPattern checks if path matches a glob pattern
func MatchesPattern(path, pattern string) bool
```

**Logic**:
- `IsSpecialFile`: Return true for:
  - "ign.json" (exact match)
  - Paths starting with ".ign-build/" or ".ign-build"
- `ShouldIgnoreFile`:
  - Check IsSpecialFile first
  - Then check against each ignore pattern using glob matching
  - Return true if any pattern matches
- `MatchesPattern`: Use `filepath.Match` for glob matching

### File 5: errors.go

**Purpose**: Generator-specific error types

**Types**:
```go
// GeneratorError represents generator-specific errors
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
    GeneratorPathError
)
```

**Methods**:
- `Error() string` - Implement error interface
- `Unwrap() error` - Return Cause for error unwrapping
- `newGeneratorError(typ, msg, file, cause) *GeneratorError` - Constructor

### File 6: generator_test.go

**Purpose**: Unit tests for generator functionality

**Test Cases**:
- `TestGenerator_Generate` - Basic generation
- `TestGenerator_GenerateWithOverwrite` - Overwrite behavior
- `TestGenerator_DryRun` - Dry run doesn't write files
- `TestProcessor_ShouldProcess` - Binary detection
- `TestProcessor_Process` - Template processing
- `TestWriter_WriteFile` - File writing with permissions
- `TestWriter_CreateDir` - Directory creation
- `TestFilter_ShouldIgnoreFile` - Ignore pattern matching
- `TestFilter_IsSpecialFile` - Special file detection

**Test Utilities**:
- Use `t.TempDir()` for temporary directories
- Create test templates with `model.Template`
- Create test variables with `parser.NewMapVariables`
- Mock file systems where appropriate

## File Handling Rules (from cli-commands.md Section 2.2)

**Default Behavior (without --overwrite)**:
- Existing files: **Skip** (do not modify) → increment FilesSkipped
- New files: **Create** → increment FilesCreated
- Empty directories: **Create** (if needed for file paths)

**With --overwrite**:
- Existing files: **Replace** with generated content → increment FilesOverwritten
- New files: **Create** → increment FilesCreated

**File Permissions**:
- Executable bit is preserved from template if PreserveExecutable is true
- Other permissions use system defaults (0644 for regular files)

**Special Files**:
- `.ign-build/`: Never touched during generation
- `ign.json`: Never copied to output (template config only)
- Hidden files (`.file`): Processed normally (unless filtered)
- Binary files: Copied as-is (no template processing)

## Binary File Detection

A file is considered binary if:
1. `TemplateFile.IsBinary` is true, OR
2. File extension matches BinaryExtensions list, OR
3. First 512 bytes contain binary markers (null bytes)

Common binary extensions (default if not in settings):
- Images: .png, .jpg, .jpeg, .gif, .bmp, .ico, .svg
- Archives: .zip, .tar, .gz, .bz2, .xz, .rar, .7z
- Executables: .exe, .dll, .so, .dylib, .bin
- Media: .mp3, .mp4, .avi, .mov, .wav
- Documents: .pdf, .doc, .docx, .xls, .xlsx
- Fonts: .ttf, .otf, .woff, .woff2

## Integration with Parser

When processing text files:
```go
pctx := &parser.ParseContext{
    Variables:      vars,
    IncludeDepth:   0,
    IncludeStack:   []string{},
    TemplateRoot:   templateRoot,
    CurrentFile:    file.Path,
}

processed, err := p.parser.ParseWithContext(ctx, file.Content, pctx)
```

This ensures:
- @ign-include: directives can resolve relative paths
- Circular include detection works
- Include depth limits are enforced

## Error Handling

All errors should:
1. Wrap underlying errors with context
2. Include file paths when relevant
3. Use GeneratorError types for categorization
4. Return immediately on critical errors
5. Continue processing on non-critical errors (accumulate in GenerateResult.Errors)

Critical errors (return immediately):
- Invalid options (nil Template, Variables, empty OutputDir)
- Cannot create output directory
- Template has no files

Non-critical errors (accumulate and continue):
- Individual file processing failures
- Individual file write failures

## Completion Criteria

✓ Generator interface with Generate and DryRun methods
✓ File processor correctly distinguishes binary from text files
✓ Template processing via parser for text files
✓ Binary files copied as-is
✓ Overwrite flag respected
✓ Executable permissions preserved (if PreserveExecutable=true)
✓ Ignore patterns applied
✓ ign.json excluded from output
✓ .ign-build directory excluded from output
✓ Unit tests pass
✓ `go build ./...` succeeds
✓ `go test ./internal/template/generator/...` passes
