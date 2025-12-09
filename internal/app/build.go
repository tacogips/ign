package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tacogips/ign/internal/config"
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
	// Validate options
	if err := validateBuildInitOptions(opts); err != nil {
		return NewValidationError("invalid build init options", err)
	}

	// Normalize template URL
	normalizedURL := NormalizeTemplateURL(opts.URL)

	// Create provider with token if available
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		return NewBuildInitError("failed to create provider", err)
	}

	// Resolve URL to template reference
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		return NewBuildInitError("failed to resolve template URL", err)
	}

	// Use provided ref or default
	if opts.Ref != "" {
		templateRef.Ref = opts.Ref
	}

	// Validate template is accessible
	if err := prov.Validate(ctx, templateRef); err != nil {
		return NewBuildInitError("template validation failed", err)
	}

	// Fetch template
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		return NewTemplateFetchError("failed to fetch template", err)
	}

	// Check if output directory already exists
	buildDir := opts.OutputDir
	if _, err := os.Stat(buildDir); err == nil {
		// Directory exists
		if !opts.Force {
			return NewBuildInitError(
				fmt.Sprintf("build directory already exists: %s (use --force to overwrite)", buildDir),
				nil,
			)
		}
		// Remove existing directory
		if err := os.RemoveAll(buildDir); err != nil {
			return NewBuildInitError("failed to remove existing build directory", err)
		}
	}

	// Create build directory
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return NewBuildInitError("failed to create build directory", err)
	}

	// Create ign-var.json with empty variables
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
	if err := config.SaveIgnVarJson(ignVarPath, ignVarJson); err != nil {
		return NewBuildInitError("failed to save ign-var.json", err)
	}

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
