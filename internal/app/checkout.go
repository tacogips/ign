package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/provider"
)

// CheckoutOptions contains options for project checkout.
type CheckoutOptions struct {
	// OutputDir is the directory where project files will be generated.
	OutputDir string
	// Overwrite determines whether to overwrite existing files.
	Overwrite bool
	// DryRun simulates generation without writing files.
	DryRun bool
	// Verbose enables detailed logging.
	Verbose bool
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
}

// DryRunFile contains information about a file that would be created in dry-run mode.
type DryRunFile struct {
	// Path is the output file path.
	Path string
	// Content is the processed file content.
	Content []byte
	// Exists indicates if the file already exists.
	Exists bool
	// WouldOverwrite indicates if the file would be overwritten.
	WouldOverwrite bool
	// WouldSkip indicates if the file would be skipped.
	WouldSkip bool
}

// CheckoutResult contains the results of project checkout.
type CheckoutResult struct {
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
	// DryRunFiles contains detailed information for dry-run mode.
	DryRunFiles []DryRunFile
	// Directories contains directories that would be created (dry-run only).
	Directories []string
}

// Checkout generates project files from template using configuration.
// Loads .ign-config/ign-var.json, fetches template, and generates project files.
func Checkout(ctx context.Context, opts CheckoutOptions) (*CheckoutResult, error) {
	configPath := ".ign-config/ign-var.json"

	debug.DebugSection("[app] Checkout workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigPath", configPath)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)

	// Validate options
	if err := validateCheckoutOptions(opts); err != nil {
		debug.Debug("[app] Checkout options validation failed: %v", err)
		return nil, NewValidationError("invalid checkout options", err)
	}
	debug.Debug("[app] Checkout options validated successfully")

	// Load ign-var.json
	debug.Debug("[app] Loading configuration from: %s", configPath)
	ignVar, err := config.LoadIgnVarJson(configPath)
	if err != nil {
		debug.Debug("[app] Failed to load configuration: %v", err)
		return nil, NewCheckoutError("failed to load configuration", err)
	}
	debug.Debug("[app] Configuration loaded successfully")

	// Validate template source
	if ignVar.Template.URL == "" {
		debug.Debug("[app] Template URL is empty in configuration")
		return nil, NewCheckoutError("template URL is empty in configuration", nil)
	}
	debug.DebugValue("[app] Template URL", ignVar.Template.URL)
	debug.DebugValue("[app] Template Ref", ignVar.Template.Ref)
	debug.DebugValue("[app] Template Path", ignVar.Template.Path)

	// Get config directory from config path
	configDir := filepath.Dir(configPath)
	debug.DebugValue("[app] Config directory", configDir)

	// Load and process variables (resolve @file: references)
	debug.Debug("[app] Loading variables")
	vars, err := LoadVariables(ignVar, configDir)
	if err != nil {
		debug.Debug("[app] Failed to load variables: %v", err)
		return nil, err
	}
	debug.Debug("[app] Variables loaded successfully")

	// Create provider from template URL
	normalizedURL := NormalizeTemplateURL(ignVar.Template.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)
	debug.Debug("[app] Creating template provider")
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return nil, NewCheckoutError("failed to create provider", err)
	}
	debug.Debug("[app] Template provider created successfully")

	// Resolve template reference
	debug.Debug("[app] Resolving template URL")
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}
	debug.Debug("[app] Template URL resolved successfully")

	// Use ref from configuration if available
	if ignVar.Template.Ref != "" {
		templateRef.Ref = ignVar.Template.Ref
	}

	// Use path from configuration if available
	if ignVar.Template.Path != "" {
		templateRef.Path = ignVar.Template.Path
	}

	// Fetch template
	debug.Debug("[app] Fetching template from provider")
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}
	debug.Debug("[app] Template fetched successfully")

	// Validate that all required variables are set
	debug.Debug("[app] Validating required variables")
	if err := ValidateVariables(&template.Config, vars); err != nil {
		debug.Debug("[app] Variable validation failed: %v", err)
		return nil, err
	}
	debug.Debug("[app] Variables validated successfully")

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
		debug.Debug("[app] Starting dry run generation")
		genResult, err = gen.DryRun(ctx, genOpts)
	} else {
		debug.Debug("[app] Starting project generation")
		genResult, err = gen.Generate(ctx, genOpts)
	}

	if err != nil {
		debug.Debug("[app] Generation failed: %v", err)
		return nil, NewCheckoutError("generation failed", err)
	}
	debug.Debug("[app] Generation completed successfully")
	debug.DebugValue("[app] Files created", genResult.FilesCreated)
	debug.DebugValue("[app] Files skipped", genResult.FilesSkipped)
	debug.DebugValue("[app] Files overwritten", genResult.FilesOverwritten)

	// Convert generator result to checkout result
	result := &CheckoutResult{
		FilesCreated:     genResult.FilesCreated,
		FilesSkipped:     genResult.FilesSkipped,
		FilesOverwritten: genResult.FilesOverwritten,
		Errors:           genResult.Errors,
		Files:            genResult.Files,
		Directories:      genResult.Directories,
	}

	// Convert dry-run files
	if opts.DryRun && len(genResult.DryRunFiles) > 0 {
		result.DryRunFiles = make([]DryRunFile, len(genResult.DryRunFiles))
		for i, f := range genResult.DryRunFiles {
			result.DryRunFiles[i] = DryRunFile{
				Path:           f.Path,
				Content:        f.Content,
				Exists:         f.Exists,
				WouldOverwrite: f.WouldOverwrite,
				WouldSkip:      f.WouldSkip,
			}
		}
	}

	debug.Debug("[app] Checkout workflow completed successfully")
	return result, nil
}

// validateCheckoutOptions validates checkout options.
func validateCheckoutOptions(opts CheckoutOptions) error {
	if opts.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return err
	}

	return nil
}
