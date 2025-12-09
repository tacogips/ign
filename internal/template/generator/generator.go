package generator

import (
	"context"
	"fmt"
	"path/filepath"

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

	// Initialize result
	result := &GenerateResult{
		Errors: []error{},
		Files:  []string{},
	}

	// Get template settings
	settings := getTemplateSettings(opts.Template)

	// Create processor and writer based on template settings
	processor := NewFileProcessor(g.parser, settings.BinaryExtensions)
	writer := NewFileWriter(settings.PreserveExecutable)

	// Create output directory if it doesn't exist (unless dry run)
	if !dryRun && !writer.Exists(opts.OutputDir) {
		if err := writer.CreateDir(opts.OutputDir); err != nil {
			return nil, err
		}
	}

	// Process each file in the template
	for _, file := range opts.Template.Files {
		if err := ctx.Err(); err != nil {
			// Context cancelled
			return result, err
		}

		// Check if file should be ignored
		if ShouldIgnoreFile(file.Path, settings.IgnorePatterns) {
			continue
		}

		// Construct output path
		outputPath := filepath.Join(opts.OutputDir, file.Path)

		// Add to processed files list
		result.Files = append(result.Files, outputPath)

		// Check if file exists
		fileExists := writer.Exists(outputPath)

		// Determine action
		if fileExists && !opts.Overwrite {
			// Skip existing file
			result.FilesSkipped++
			continue
		}

		// Process file content
		processed, err := processor.Process(ctx, file, opts.Variables, opts.Template.RootPath)
		if err != nil {
			// Record error but continue processing
			result.Errors = append(result.Errors, fmt.Errorf("failed to process %s: %w", file.Path, err))
			continue
		}

		// Write file (unless dry run)
		if !dryRun {
			if err := writer.WriteFile(outputPath, processed, file.Mode); err != nil {
				// Record error but continue processing
				result.Errors = append(result.Errors, fmt.Errorf("failed to write %s: %w", file.Path, err))
				continue
			}
		}

		// Update statistics
		if fileExists {
			result.FilesOverwritten++
		} else {
			result.FilesCreated++
		}
	}

	return result, nil
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
