package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/provider"
)

// InitOptions contains options for project initialization.
type InitOptions struct {
	// OutputDir is the directory where project files will be generated.
	OutputDir string
	// ConfigPath is the path to ign-var.json.
	ConfigPath string
	// Overwrite determines whether to overwrite existing files.
	Overwrite bool
	// DryRun simulates generation without writing files.
	DryRun bool
	// Verbose enables detailed logging.
	Verbose bool
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
}

// InitResult contains the results of project initialization.
type InitResult struct {
	// FilesCreated is the number of new files created.
	FilesCreated int
	// FilesSkipped is the number of files skipped (already exist).
	FilesSkipped int
	// FilesOverwritten is the number of existing files overwritten.
	FilesOverwritten int
	// Errors contains non-fatal errors encountered during generation.
	Errors []error
	// Files contains the paths of all files processed.
	Files []string
}

// Init initializes a project from a template using build configuration.
// Loads ign-var.json, fetches template, and generates project files.
func Init(ctx context.Context, opts InitOptions) (*InitResult, error) {
	// Validate options
	if err := validateInitOptions(opts); err != nil {
		return nil, NewValidationError("invalid init options", err)
	}

	// Load ign-var.json
	ignVar, err := config.LoadIgnVarJson(opts.ConfigPath)
	if err != nil {
		return nil, NewInitError("failed to load configuration", err)
	}

	// Validate template source
	if ignVar.Template.URL == "" {
		return nil, NewInitError("template URL is empty in configuration", nil)
	}

	// Get build directory from config path
	buildDir := filepath.Dir(opts.ConfigPath)

	// Load and process variables (resolve @file: references)
	vars, err := LoadVariables(ignVar, buildDir)
	if err != nil {
		return nil, err
	}

	// Create provider from template URL
	normalizedURL := NormalizeTemplateURL(ignVar.Template.URL)
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		return nil, NewInitError("failed to create provider", err)
	}

	// Resolve template reference
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		return nil, NewInitError("failed to resolve template URL", err)
	}

	// Use ref from configuration if available
	if ignVar.Template.Ref != "" {
		templateRef.Ref = ignVar.Template.Ref
	}

	// Use path from configuration if available
	if ignVar.Template.Path != "" {
		templateRef.Path = ignVar.Template.Path
	}

	// Fetch template
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}

	// Validate that all required variables are set
	if err := ValidateVariables(&template.Config, vars); err != nil {
		return nil, err
	}

	// Create generator
	gen := generator.NewGenerator()

	// Prepare generate options
	genOpts := generator.GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: opts.OutputDir,
		Overwrite: opts.Overwrite,
		Verbose:   opts.Verbose,
	}

	// Generate or dry run
	var genResult *generator.GenerateResult
	if opts.DryRun {
		genResult, err = gen.DryRun(ctx, genOpts)
	} else {
		genResult, err = gen.Generate(ctx, genOpts)
	}

	if err != nil {
		return nil, NewInitError("generation failed", err)
	}

	// Convert generator result to init result
	result := &InitResult{
		FilesCreated:     genResult.FilesCreated,
		FilesSkipped:     genResult.FilesSkipped,
		FilesOverwritten: genResult.FilesOverwritten,
		Errors:           genResult.Errors,
		Files:            genResult.Files,
	}

	return result, nil
}

// validateInitOptions validates init options.
func validateInitOptions(opts InitOptions) error {
	if opts.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return err
	}

	if opts.ConfigPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	return nil
}
