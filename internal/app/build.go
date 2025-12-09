package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/provider"
)

// BuildInitOptions contains options for build initialization.
type BuildInitOptions struct {
	// URL is the template URL or path.
	URL string
	// OutputDir is the directory where .ign-build will be created.
	OutputDir string
	// Ref is the git branch, tag, or commit SHA.
	Ref string
	// Force overwrites existing .ign-build directory if true.
	Force bool
	// Config is the path to global config file (optional).
	Config string
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
	// IgnVersion is the version of ign CLI (for metadata).
	IgnVersion string
}

// BuildInit initializes a build configuration from a template.
// Creates .ign-build/ign-var.json with template metadata and empty variables.
func BuildInit(ctx context.Context, opts BuildInitOptions) error {
	debug.DebugSection("[app] BuildInit workflow start")
	debug.DebugValue("[app] Template URL", opts.URL)
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] Ref", opts.Ref)
	debug.DebugValue("[app] Force", opts.Force)

	// Validate options
	if err := validateBuildInitOptions(opts); err != nil {
		debug.Debug("[app] Build init options validation failed: %v", err)
		return NewValidationError("invalid build init options", err)
	}
	debug.Debug("[app] Build init options validated successfully")

	// Normalize template URL
	normalizedURL := NormalizeTemplateURL(opts.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)

	// Create provider with token if available
	debug.Debug("[app] Creating template provider")
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return NewBuildInitError("failed to create provider", err)
	}
	debug.Debug("[app] Template provider created successfully")

	// Resolve URL to template reference
	debug.Debug("[app] Resolving template URL")
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return NewBuildInitError("failed to resolve template URL", err)
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
		return NewBuildInitError("template validation failed", err)
	}
	debug.Debug("[app] Template validated successfully")

	// Fetch template
	debug.Debug("[app] Fetching template from provider")
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return NewTemplateFetchError("failed to fetch template", err)
	}
	debug.Debug("[app] Template fetched successfully")
	debug.DebugValue("[app] Template name", template.Config.Name)
	debug.DebugValue("[app] Template version", template.Config.Version)

	// Check if output directory already exists
	buildDir := opts.OutputDir
	if _, err := os.Stat(buildDir); err == nil {
		// Directory exists
		debug.Debug("[app] Build directory already exists: %s", buildDir)
		if !opts.Force {
			debug.Debug("[app] Force flag not set, aborting")
			return NewBuildInitError(
				fmt.Sprintf("build directory already exists: %s (use --force to overwrite)", buildDir),
				nil,
			)
		}
		// Remove existing directory
		debug.Debug("[app] Removing existing build directory")
		if err := os.RemoveAll(buildDir); err != nil {
			debug.Debug("[app] Failed to remove existing build directory: %v", err)
			return NewBuildInitError("failed to remove existing build directory", err)
		}
		debug.Debug("[app] Existing build directory removed")
	}

	// Create build directory
	debug.Debug("[app] Creating build directory: %s", buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		debug.Debug("[app] Failed to create build directory: %v", err)
		return NewBuildInitError("failed to create build directory", err)
	}
	debug.Debug("[app] Build directory created successfully")

	// Create ign-var.json with empty variables
	debug.Debug("[app] Creating ign-var.json")
	ignVarJson := &model.IgnVarJson{
		Template: model.TemplateSource{
			URL:  normalizedURL,
			Path: templateRef.Path,
			Ref:  templateRef.Ref,
		},
		Variables: CreateEmptyVariablesMap(&template.Config),
		Metadata: &model.VarMetadata{
			GeneratedAt:     time.Now(),
			GeneratedBy:     "ign build init",
			TemplateName:    template.Config.Name,
			TemplateVersion: template.Config.Version,
			IgnVersion:      opts.IgnVersion,
		},
	}

	// Save ign-var.json
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
	debug.DebugValue("[app] Saving ign-var.json to", ignVarPath)
	if err := config.SaveIgnVarJson(ignVarPath, ignVarJson); err != nil {
		debug.Debug("[app] Failed to save ign-var.json: %v", err)
		return NewBuildInitError("failed to save ign-var.json", err)
	}
	debug.Debug("[app] ign-var.json saved successfully")

	debug.Debug("[app] BuildInit workflow completed successfully")
	return nil
}

// validateBuildInitOptions validates build init options.
func validateBuildInitOptions(opts BuildInitOptions) error {
	if opts.URL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if opts.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return err
	}

	return nil
}
