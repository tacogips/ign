package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
	"github.com/tacogips/ign/internal/template/provider"
	"github.com/tacogips/ign/internal/version"
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

// calculateTemplateHash calculates SHA256 hash of template files content.
// Files are sorted by path to ensure deterministic hash generation.
// Returns empty string if template is nil or has no files.
func calculateTemplateHash(template *model.Template) string {
	// Defensive: handle nil template
	if template == nil {
		return ""
	}

	// Handle empty template (no files)
	if len(template.Files) == 0 {
		return ""
	}

	h := sha256.New()

	// Sort files by path for deterministic hash
	files := make([]model.TemplateFile, len(template.Files))
	copy(files, template.Files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Hash each file's path and content
	for _, file := range files {
		h.Write([]byte(file.Path))
		h.Write(file.Content)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// maxBackups is the maximum number of backup files allowed to prevent infinite loops
const maxBackups = 100

// findNextBackupNumber finds the next available backup number for the given filename.
// Returns an error if the maximum number of backups (100) has been exceeded.
func findNextBackupNumber(dir, filename string) (int, error) {
	for n := 1; n <= maxBackups; n++ {
		backupPath := filepath.Join(dir, fmt.Sprintf("%s.bk%d", filename, n))
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return n, nil
		}
	}
	return 0, fmt.Errorf("too many backup files exist for %s (max %d), please clean up old backups", filename, maxBackups)
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

		// Backup existing ign.json if it exists
		ignConfigPath := filepath.Join(configDir, "ign.json")
		if _, err := os.Stat(ignConfigPath); err == nil {
			backupNum, err := findNextBackupNumber(configDir, "ign.json")
			if err != nil {
				return nil, NewCheckoutError(err.Error(), nil)
			}
			backupPath := filepath.Join(configDir, fmt.Sprintf("ign.json.bk%d", backupNum))
			debug.Debug("[app] Backing up existing ign.json to: %s", backupPath)
			if err := os.Rename(ignConfigPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing ign.json: %v", err)
				return nil, NewCheckoutError("failed to backup existing ign.json", err)
			}
			debug.Debug("[app] Existing ign.json backed up successfully")
		}

		// Backup existing ign-var.json if it exists
		ignVarPath := filepath.Join(configDir, "ign-var.json")
		if _, err := os.Stat(ignVarPath); err == nil {
			backupNum, err := findNextBackupNumber(configDir, "ign-var.json")
			if err != nil {
				return nil, NewCheckoutError(err.Error(), nil)
			}
			backupPath := filepath.Join(configDir, fmt.Sprintf("ign-var.json.bk%d", backupNum))
			debug.Debug("[app] Backing up existing ign-var.json to: %s", backupPath)
			if err := os.Rename(ignVarPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing ign-var.json: %v", err)
				return nil, NewCheckoutError("failed to backup existing ign-var.json", err)
			}
			debug.Debug("[app] Existing ign-var.json backed up successfully")
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
	ignVarPath := filepath.Join(configDir, "ign-var.json")

	debug.DebugSection("[app] CompleteCheckout workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] IgnVarPath", ignVarPath)
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

	// Create and save configuration files (unless dry-run)
	if !opts.DryRun {
		// Calculate template hash
		templateHash := calculateTemplateHash(prep.Template)
		debug.DebugValue("[app] Template hash", templateHash)

		// Validate hash is not empty (should not happen if template is valid)
		if templateHash == "" {
			debug.Debug("[app] Template hash is empty - template may be nil or have no files")
			return nil, NewCheckoutError("failed to calculate template hash: template is empty or invalid", nil)
		}

		// Save ign.json (template source and hash)
		ignConfigPath := filepath.Join(configDir, "ign.json")
		debug.Debug("[app] Creating ign.json")
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{
				URL:  prep.NormalizedURL,
				Path: prep.TemplateRef.Path,
				Ref:  prep.TemplateRef.Ref,
			},
			Hash: templateHash,
			Metadata: &model.FileMetadata{
				GeneratedAt:     time.Now(),
				GeneratedBy:     "ign checkout",
				TemplateName:    prep.IgnJson.Name,
				TemplateVersion: prep.IgnJson.Version,
				IgnVersion:      version.Version,
			},
		}

		// Save ign-var.json (variables only)
		debug.Debug("[app] Creating ign-var.json")
		ignVarJson := &model.IgnVarJson{
			Variables: opts.Variables,
			Metadata: &model.FileMetadata{
				GeneratedAt:     time.Now(),
				GeneratedBy:     "ign checkout",
				TemplateName:    prep.IgnJson.Name,
				TemplateVersion: prep.IgnJson.Version,
				IgnVersion:      version.Version,
			},
		}

		// Save both configuration files with rollback on failure.
		// Write ign.json first, then ign-var.json.
		// If ign-var.json save fails, remove ign.json to maintain consistent state.
		debug.DebugValue("[app] Saving ign.json to", ignConfigPath)
		if err := config.SaveIgnConfig(ignConfigPath, ignConfig); err != nil {
			debug.Debug("[app] Failed to save ign.json: %v", err)
			return nil, NewCheckoutError("failed to save ign.json", err)
		}
		debug.Debug("[app] ign.json saved successfully")

		debug.DebugValue("[app] Saving ign-var.json to", ignVarPath)
		if err := config.SaveIgnVarJson(ignVarPath, ignVarJson); err != nil {
			debug.Debug("[app] Failed to save ign-var.json: %v", err)
			// Rollback: remove ign.json to avoid inconsistent state
			debug.Debug("[app] Rolling back ign.json due to ign-var.json save failure")
			if removeErr := os.Remove(ignConfigPath); removeErr != nil {
				debug.Debug("[app] Failed to rollback ign.json: %v (original error: %v)", removeErr, err)
			}
			return nil, NewCheckoutError("failed to save ign-var.json (rolled back ign.json)", err)
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
// Loads .ign/ign.json and .ign/ign-var.json, fetches template, and generates project files.
// Supports backward compatibility with old ign-var.json format that includes template info.
// Deprecated: Use PrepareCheckout and CompleteCheckout for new code.
func Checkout(ctx context.Context, opts CheckoutOptions) (*CheckoutResult, error) {
	configDir := ".ign"
	ignConfigPath := filepath.Join(configDir, "ign.json")
	ignVarPath := filepath.Join(configDir, "ign-var.json")

	debug.DebugSection("[app] Checkout workflow start (backward-compatible)")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigDir", configDir)
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

	// Load configuration - try new format first, fallback to old format
	var templateSource model.TemplateSource
	var variables map[string]interface{}

	// Try loading new format (ign.json + ign-var.json)
	ignConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err == nil {
		// New format found - load template from ign.json
		debug.Debug("[app] Loaded template config from ign.json")
		templateSource = ignConfig.Template

		// Load variables from ign-var.json
		ignVar, err := config.LoadIgnVarJson(ignVarPath)
		if err != nil {
			debug.Debug("[app] Failed to load ign-var.json: %v", err)
			return nil, NewCheckoutError("failed to load ign-var.json", err)
		}
		variables = ignVar.Variables
		debug.Debug("[app] Loaded variables from ign-var.json")
	} else {
		// New format not found - check if old format exists and provide helpful error
		debug.Debug("[app] ign.json not found")

		// Check if old ign-var.json format exists (with template metadata)
		if _, err := os.Stat(ignVarPath); err == nil {
			// Old format file exists - provide migration instructions
			debug.Debug("[app] Found old format ign-var.json - migration required")
			return nil, NewCheckoutError(
				"old configuration format detected: .ign/ign-var.json exists but .ign/ign.json is missing.\n"+
					"The configuration format has been split into two files:\n"+
					"  - .ign/ign.json (template source and hash)\n"+
					"  - .ign/ign-var.json (variable values only)\n"+
					"To migrate, please delete the .ign directory and run 'ign checkout <template-url>' again.",
				nil,
			)
		}

		// Neither new nor old format found
		debug.Debug("[app] No configuration found")
		return nil, NewCheckoutError("configuration not found: .ign/ign.json does not exist. Run 'ign checkout <template-url>' first.", nil)
	}

	// Validate template source
	if templateSource.URL == "" {
		debug.Debug("[app] Template URL is empty in configuration")
		return nil, NewCheckoutError("template URL is empty in configuration", nil)
	}
	debug.DebugValue("[app] Template URL", templateSource.URL)
	debug.DebugValue("[app] Template Ref", templateSource.Ref)
	debug.DebugValue("[app] Template Path", templateSource.Path)

	// Convert variables map to parser.Variables and resolve @file: references
	debug.Debug("[app] Loading variables")
	vars, err := LoadVariablesFromMap(variables, configDir)
	if err != nil {
		debug.Debug("[app] Failed to load variables: %v", err)
		return nil, err
	}
	debug.Debug("[app] Variables loaded successfully")

	// Create provider from template URL
	normalizedURL := NormalizeTemplateURL(templateSource.URL)
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
	if templateSource.Ref != "" {
		templateRef.Ref = templateSource.Ref
	}

	// Use path from configuration if available
	if templateSource.Path != "" {
		templateRef.Path = templateSource.Path
	}

	// Fetch template
	debug.Debug("[app] Fetching template from provider")
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}
	debug.Debug("[app] Template fetched successfully")

	// Calculate and update template hash in ign.json (unless dry-run)
	if !opts.DryRun {
		debug.Debug("[app] Calculating template hash")
		templateHash := calculateTemplateHash(template)
		debug.DebugValue("[app] Template hash", templateHash)

		// Load existing ign.json
		existingConfig, err := config.LoadIgnConfig(ignConfigPath)
		if err != nil {
			debug.Debug("[app] Could not load existing ign.json (will skip hash update): %v", err)
		} else {
			// Update hash in ign.json
			existingConfig.Hash = templateHash
			debug.Debug("[app] Updating template hash in ign.json")
			if err := config.SaveIgnConfig(ignConfigPath, existingConfig); err != nil {
				debug.Debug("[app] Failed to update hash in ign.json: %v", err)
				return nil, NewCheckoutError("failed to update template hash in ign.json", err)
			}
			debug.Debug("[app] Template hash updated in ign.json")
		}
	}

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
