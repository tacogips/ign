package generator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

// Generator generates projects from templates.
type Generator interface {
	// Generate creates a project from a template with the given options.
	// Writes files to the output directory, respecting the overwrite flag.
	Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)

	// DryRun simulates project generation without writing files.
	// Returns what would be generated without actually creating files.
	DryRun(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)
}

// GenerateOptions configures project generation.
type GenerateOptions struct {
	// Template is the template to generate from.
	Template *model.Template

	// Variables holds the variable values for template substitution.
	Variables parser.Variables

	// OutputDir is the directory where files will be generated.
	OutputDir string

	// Overwrite determines whether to overwrite existing files.
	// If false, existing files are skipped.
	Overwrite bool

	// Verbose enables detailed logging during generation.
	Verbose bool
}

// DryRunFile contains information about a file that would be created in dry-run mode.
type DryRunFile struct {
	// Path is the output file path.
	Path string
	// Content is the processed file content.
	Content []byte
	// Exists indicates if the file already exists.
	Exists bool
	// WouldOverwrite indicates if the file would be overwritten (Exists && Overwrite option).
	WouldOverwrite bool
	// WouldSkip indicates if the file would be skipped (Exists && !Overwrite option).
	WouldSkip bool
}

// GenerateResult contains generation statistics.
type GenerateResult struct {
	// FilesCreated is the number of new files created.
	FilesCreated int

	// FilesSkipped is the number of files skipped (already exist).
	FilesSkipped int

	// FilesOverwritten is the number of existing files overwritten.
	FilesOverwritten int

	// Errors contains non-fatal errors encountered during generation.
	Errors []error

	// Files contains the paths of all files processed (created, skipped, or overwritten).
	Files []string

	// DryRunFiles contains detailed information for dry-run mode (only populated in dry-run).
	DryRunFiles []DryRunFile

	// Directories contains directories that would be created (only populated in dry-run).
	Directories []string
}

// DefaultGenerator implements Generator.
type DefaultGenerator struct {
	parser    parser.Parser
	processor Processor
	writer    Writer
}

// NewGenerator creates a new DefaultGenerator.
// If parser is nil, creates a default parser.
func NewGenerator() Generator {
	p := parser.NewParser()
	return &DefaultGenerator{
		parser:    p,
		processor: nil, // Will be created per-generation with template settings
		writer:    nil, // Will be created per-generation with template settings
	}
}

// Generate creates a project from a template with the given options.
func (g *DefaultGenerator) Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
	return g.generate(ctx, opts, false)
}

// DryRun simulates project generation without writing files.
func (g *DefaultGenerator) DryRun(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
	return g.generate(ctx, opts, true)
}

// generate is the internal implementation for both Generate and DryRun.
func (g *DefaultGenerator) generate(ctx context.Context, opts GenerateOptions, dryRun bool) (*GenerateResult, error) {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	// Log generation start
	templateName := fmt.Sprintf("%s/%s", opts.Template.Ref.Owner, opts.Template.Ref.Repo)
	if opts.Template.Ref.Path != "" {
		templateName = fmt.Sprintf("%s/%s", templateName, opts.Template.Ref.Path)
	}
	debug.Debug("[generator] Starting generation: template=%s, outputDir=%s, dryRun=%v, overwrite=%v",
		templateName, opts.OutputDir, dryRun, opts.Overwrite)

	// Initialize result
	result := &GenerateResult{
		Errors:      []error{},
		Files:       []string{},
		DryRunFiles: []DryRunFile{},
		Directories: []string{},
	}

	// Track directories for dry-run mode
	dirsToCreate := make(map[string]bool)

	// Get template settings
	settings := getTemplateSettings(opts.Template)
	debug.Debug("[generator] Template settings: preserveExecutable=%v, ignorePatterns=%v, binaryExtensions=%d",
		settings.PreserveExecutable, settings.IgnorePatterns, len(settings.BinaryExtensions))

	// Create processor and writer based on template settings
	processor := NewFileProcessor(g.parser, settings.BinaryExtensions)
	writer := NewFileWriter(settings.PreserveExecutable)

	// Create output directory if it doesn't exist (unless dry run)
	if !dryRun && !writer.Exists(opts.OutputDir) {
		debug.Debug("[generator] Creating output directory: %s", opts.OutputDir)
		if err := writer.CreateDir(opts.OutputDir); err != nil {
			return nil, err
		}
	}

	// Process each file in the template
	debug.Debug("[generator] Processing %d files from template", len(opts.Template.Files))
	for _, file := range opts.Template.Files {
		if err := ctx.Err(); err != nil {
			// Context cancelled
			return result, err
		}

		// Check if file should be ignored
		if ShouldIgnoreFile(file.Path, settings.IgnorePatterns) {
			debug.Debug("[generator] Ignoring file: %s", file.Path)
			continue
		}

		// Process filename for variable substitution
		processedFilePath, err := ProcessFilename(ctx, file.Path, opts.Variables, g.parser)
		if err != nil {
			// Record error but continue processing
			result.Errors = append(result.Errors, fmt.Errorf("failed to process filename %s: %w", file.Path, err))
			continue
		}

		// Construct output path
		outputPath := filepath.Join(opts.OutputDir, processedFilePath)
		debug.Debug("[generator] Processing file: %s -> %s (processed: %s, size: %d bytes)",
			file.Path, outputPath, processedFilePath, len(file.Content))

		// Track parent directories for dry-run
		if dryRun {
			dir := filepath.Dir(outputPath)
			for dir != "." && dir != "/" && dir != opts.OutputDir {
				if !dirsToCreate[dir] {
					dirsToCreate[dir] = true
				}
				dir = filepath.Dir(dir)
			}
			// Also track the output directory itself
			if !dirsToCreate[opts.OutputDir] {
				dirsToCreate[opts.OutputDir] = true
			}
		}

		// Add to processed files list
		result.Files = append(result.Files, outputPath)

		// Check if file exists
		fileExists := writer.Exists(outputPath)

		// Determine action
		if fileExists && !opts.Overwrite {
			// Skip existing file
			debug.Debug("[generator] Skipping existing file: %s", outputPath)
			result.FilesSkipped++
			if dryRun {
				result.DryRunFiles = append(result.DryRunFiles, DryRunFile{
					Path:      outputPath,
					Content:   nil,
					Exists:    true,
					WouldSkip: true,
				})
			}
			continue
		}

		// Process file content
		debug.Debug("[generator] Processing content for: %s", file.Path)
		processed, err := processor.Process(ctx, file, opts.Variables, opts.Template.RootPath)
		if err != nil {
			// Record error but continue processing
			result.Errors = append(result.Errors, fmt.Errorf("failed to process %s: %w", file.Path, err))
			continue
		}

		// Write file (unless dry run)
		if !dryRun {
			if fileExists {
				debug.Debug("[generator] Overwriting file: %s (size: %d bytes)", outputPath, len(processed))
			} else {
				debug.Debug("[generator] Creating new file: %s (size: %d bytes)", outputPath, len(processed))
			}
			if err := writer.WriteFile(outputPath, processed, file.Mode); err != nil {
				// Record error but continue processing
				result.Errors = append(result.Errors, fmt.Errorf("failed to write %s: %w", file.Path, err))
				continue
			}
		} else {
			debug.Debug("[generator] Dry run: would write %s (size: %d bytes)", outputPath, len(processed))
			result.DryRunFiles = append(result.DryRunFiles, DryRunFile{
				Path:           outputPath,
				Content:        processed,
				Exists:         fileExists,
				WouldOverwrite: fileExists,
				WouldSkip:      false,
			})
		}

		// Update statistics
		if fileExists {
			result.FilesOverwritten++
		} else {
			result.FilesCreated++
		}
	}

	// Collect directories for dry-run result
	if dryRun {
		for dir := range dirsToCreate {
			result.Directories = append(result.Directories, dir)
		}
		// Sort directories for consistent output
		sortPaths(result.Directories)
	}

	// Log final statistics
	debug.Debug("[generator] Generation complete: created=%d, overwritten=%d, skipped=%d, errors=%d",
		result.FilesCreated, result.FilesOverwritten, result.FilesSkipped, len(result.Errors))
	if dryRun {
		debug.Debug("[generator] Dry run mode: no files were actually written")
		debug.Debug("[generator] Directories that would be created: %d", len(result.Directories))
	}

	return result, nil
}

// sortPaths sorts paths in a way that parent directories come before children.
func sortPaths(paths []string) {
	// Simple sort by path depth then lexicographically
	for i := 0; i < len(paths)-1; i++ {
		for j := i + 1; j < len(paths); j++ {
			iDepth := pathDepth(paths[i])
			jDepth := pathDepth(paths[j])
			if iDepth > jDepth || (iDepth == jDepth && paths[i] > paths[j]) {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}

// pathDepth returns the depth of a path (number of path separators).
func pathDepth(path string) int {
	clean := filepath.Clean(path)
	if clean == "." || clean == "/" {
		return 0
	}
	depth := 0
	for _, c := range clean {
		if c == filepath.Separator {
			depth++
		}
	}
	return depth
}

// validateOptions validates GenerateOptions.
func validateOptions(opts GenerateOptions) error {
	if opts.Template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if opts.Variables == nil {
		return fmt.Errorf("variables cannot be nil")
	}

	if opts.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if len(opts.Template.Files) == 0 {
		return fmt.Errorf("template has no files")
	}

	return nil
}

// getTemplateSettings returns template settings with defaults.
func getTemplateSettings(template *model.Template) model.TemplateSettings {
	if template.Config.Settings == nil {
		return model.TemplateSettings{
			PreserveExecutable: true,
			IgnorePatterns:     []string{},
			BinaryExtensions:   defaultBinaryExtensions(),
			IncludeDotfiles:    true,
			MaxIncludeDepth:    10,
		}
	}

	settings := *template.Config.Settings

	// Set defaults for zero values
	if settings.BinaryExtensions == nil {
		settings.BinaryExtensions = defaultBinaryExtensions()
	}
	if settings.MaxIncludeDepth == 0 {
		settings.MaxIncludeDepth = 10
	}

	return settings
}
