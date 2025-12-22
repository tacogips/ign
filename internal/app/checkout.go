package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
	"github.com/tacogips/ign/internal/template/provider"
)

// PrepareCheckoutOptions contains options for preparing checkout.
type PrepareCheckoutOptions struct {
	// URL is the template URL or path.
	URL string
	// Ref is the git branch, tag, or commit SHA.
	Ref string
	// Force backs up and overwrites existing .ign directory if true.
	Force bool
	// ConfigExists indicates if .ign already exists.
	ConfigExists bool
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
}

// PrepareCheckoutResult contains the result of checkout preparation.
type PrepareCheckoutResult struct {
	// Template is the fetched template.
	Template *model.Template
	// IgnJson is the template configuration with variable definitions.
	IgnJson *model.IgnJson
	// TemplateRef is the resolved template reference.
	TemplateRef model.TemplateRef
	// NormalizedURL is the normalized template URL.
	NormalizedURL string
}

// CompleteCheckoutOptions contains options for completing checkout.
type CompleteCheckoutOptions struct {
	// PrepareResult is the result from PrepareCheckout.
	PrepareResult *PrepareCheckoutResult
	// Variables contains the user-provided variable values.
	Variables map[string]interface{}
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

// findNextBackupNumber finds the next available backup number for ign-var.json.bkN
func findNextBackupNumber(dir string) int {
	n := 1
	for {
		backupPath := filepath.Join(dir, fmt.Sprintf("ign-var.json.bk%d", n))
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return n
		}
		n++
	}
}

// PrepareCheckout prepares for checkout by fetching the template and handling config directory.
// Returns template information and variable definitions for interactive prompting.
func PrepareCheckout(ctx context.Context, opts PrepareCheckoutOptions) (*PrepareCheckoutResult, error) {
	configDir := ".ign"

	debug.DebugSection("[app] PrepareCheckout workflow start")
	debug.DebugValue("[app] Template URL", opts.URL)
	debug.DebugValue("[app] ConfigDir", configDir)
	debug.DebugValue("[app] Ref", opts.Ref)
	debug.DebugValue("[app] Force", opts.Force)
	debug.DebugValue("[app] ConfigExists", opts.ConfigExists)

	// Validate options
	if opts.URL == "" {
		return nil, NewValidationError("URL cannot be empty", nil)
	}

	// Normalize template URL
	normalizedURL := NormalizeTemplateURL(opts.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)

	// Create provider with token if available
	debug.Debug("[app] Creating template provider")
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return nil, NewCheckoutError("failed to create provider", err)
	}
	debug.Debug("[app] Template provider created successfully")

	// Resolve URL to template reference
	debug.Debug("[app] Resolving template URL")
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}
	debug.Debug("[app] Template URL resolved successfully")

	// Use provided ref or default
	if opts.Ref != "" {
		templateRef.Ref = opts.Ref
		debug.DebugValue("[app] Using provided ref", opts.Ref)
	}

	// Validate template is accessible
	debug.Debug("[app] Validating template accessibility")
	if err := prov.Validate(ctx, templateRef); err != nil {
		debug.Debug("[app] Template validation failed: %v", err)
		return nil, NewCheckoutError("template validation failed", err)
	}
	debug.Debug("[app] Template validated successfully")

	// Fetch template
	debug.Debug("[app] Fetching template from provider")
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}
	debug.Debug("[app] Template fetched successfully")
	debug.DebugValue("[app] Template name", template.Config.Name)
	debug.DebugValue("[app] Template version", template.Config.Version)

	// Handle config directory
	if opts.ConfigExists {
		// Directory exists and Force is true (checked in CLI layer)
		debug.Debug("[app] Config directory exists, Force mode - backing up")

		// Backup existing ign-var.json if it exists
		ignVarPath := filepath.Join(configDir, "ign-var.json")
		if _, err := os.Stat(ignVarPath); err == nil {
			backupNum := findNextBackupNumber(configDir)
			backupPath := filepath.Join(configDir, fmt.Sprintf("ign-var.json.bk%d", backupNum))
			debug.Debug("[app] Backing up existing config to: %s", backupPath)
			if err := os.Rename(ignVarPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing config: %v", err)
				return nil, NewCheckoutError("failed to backup existing configuration", err)
			}
			debug.Debug("[app] Existing config backed up successfully")
		}
	} else {
		// Create config directory
		debug.Debug("[app] Creating config directory: %s", configDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			debug.Debug("[app] Failed to create config directory: %v", err)
			return nil, NewCheckoutError("failed to create config directory", err)
		}
		debug.Debug("[app] Config directory created successfully")
	}

	debug.Debug("[app] PrepareCheckout completed successfully")
	return &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   templateRef,
		NormalizedURL: normalizedURL,
	}, nil
}

// CompleteCheckout completes checkout by saving configuration and generating files.
func CompleteCheckout(ctx context.Context, opts CompleteCheckoutOptions) (*CheckoutResult, error) {
	configDir := ".ign"
	configPath := filepath.Join(configDir, "ign-var.json")

	debug.DebugSection("[app] CompleteCheckout workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigPath", configPath)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)

	// Validate options
	if opts.OutputDir == "" {
		return nil, NewValidationError("output directory cannot be empty", nil)
	}
	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return nil, NewValidationError("invalid output directory", err)
	}

	prep := opts.PrepareResult
	if prep == nil {
		return nil, NewValidationError("prepare result cannot be nil", nil)
	}

	// Convert variables to parser.Variables
	vars := parser.NewMapVariables(opts.Variables)

	// Validate that all required variables are set
	debug.Debug("[app] Validating required variables")
	if err := ValidateVariables(prep.IgnJson, vars); err != nil {
		debug.Debug("[app] Variable validation failed: %v", err)
		return nil, err
	}
	debug.Debug("[app] Variables validated successfully")

	// Create and save ign-var.json (unless dry-run)
	if !opts.DryRun {
		debug.Debug("[app] Creating ign-var.json")
		ignVarJson := &model.IgnVarJson{
			Template: model.TemplateSource{
				URL:  prep.NormalizedURL,
				Path: prep.TemplateRef.Path,
				Ref:  prep.TemplateRef.Ref,
			},
			Variables: opts.Variables,
			Metadata: &model.VarMetadata{
				GeneratedAt:     time.Now(),
				GeneratedBy:     "ign checkout",
				TemplateName:    prep.IgnJson.Name,
				TemplateVersion: prep.IgnJson.Version,
			},
		}

		debug.DebugValue("[app] Saving ign-var.json to", configPath)
		if err := config.SaveIgnVarJson(configPath, ignVarJson); err != nil {
			debug.Debug("[app] Failed to save ign-var.json: %v", err)
			return nil, NewCheckoutError("failed to save ign-var.json", err)
		}
		debug.Debug("[app] ign-var.json saved successfully")
	}

	// Create generator
	gen := generator.NewGenerator()

	// Prepare generate options
	genOpts := generator.GenerateOptions{
		Template:  prep.Template,
		Variables: vars,
		OutputDir: opts.OutputDir,
		Overwrite: opts.Overwrite,
		Verbose:   opts.Verbose,
	}

	// Generate or dry run
	var genResult *generator.GenerateResult
	var err error
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

	debug.Debug("[app] CompleteCheckout workflow completed successfully")
	return result, nil
}

// CheckoutOptions contains options for project checkout (backward-compatible).
// This is used when .ign/ign-var.json already exists.
// Deprecated: Use PrepareCheckout and CompleteCheckout for new code.
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

// Checkout generates project files from template using existing configuration.
// Loads .ign/ign-var.json, fetches template, and generates project files.
// Deprecated: Use PrepareCheckout and CompleteCheckout for new code.
func Checkout(ctx context.Context, opts CheckoutOptions) (*CheckoutResult, error) {
	configPath := ".ign/ign-var.json"

	debug.DebugSection("[app] Checkout workflow start (backward-compatible)")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigPath", configPath)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)

	// Validate options
	if opts.OutputDir == "" {
		return nil, NewValidationError("output directory cannot be empty", nil)
	}
	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return nil, NewValidationError("invalid output directory", err)
	}

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
