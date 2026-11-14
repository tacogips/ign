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

// InitOptions contains options for configuration initialization.
type InitOptions struct {
	// URL is the template URL or path.
	URL string
	// Ref is the git branch, tag, or commit SHA.
	Ref string
	// Force backs up and overwrites existing .ign-config directory if true.
	Force bool
	// Config is the path to global config file (optional).
	Config string
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
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

// Init initializes configuration from a template.
// Creates .ign-config/ign-var.json with template metadata and empty variables.
func Init(ctx context.Context, opts InitOptions) error {
	configDir := ".ign-config"

	debug.DebugSection("[app] Init workflow start")
	debug.DebugValue("[app] Template URL", opts.URL)
	debug.DebugValue("[app] ConfigDir", configDir)
	debug.DebugValue("[app] Ref", opts.Ref)
	debug.DebugValue("[app] Force", opts.Force)

	// Validate options
	if err := validateInitOptions(opts); err != nil {
		debug.Debug("[app] Init options validation failed: %v", err)
		return NewValidationError("invalid init options", err)
	}
	debug.Debug("[app] Init options validated successfully")

	// Normalize template URL
	normalizedURL := NormalizeTemplateURL(opts.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)

	// Create provider with token if available
	debug.Debug("[app] Creating template provider")
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return NewInitError("failed to create provider", err)
	}
	debug.Debug("[app] Template provider created successfully")

	// Resolve URL to template reference
	debug.Debug("[app] Resolving template URL")
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return NewInitError("failed to resolve template URL", err)
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
		return NewInitError("template validation failed", err)
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

	// Check if config directory already exists
	if _, err := os.Stat(configDir); err == nil {
		// Directory exists
		debug.Debug("[app] Config directory already exists: %s", configDir)
		if !opts.Force {
			debug.Debug("[app] Force flag not set, skipping initialization")
			return NewInitError(
				fmt.Sprintf("configuration already exists at %s (use --force to reinitialize)", configDir),
				nil,
			)
		}

		// Backup existing ign-var.json if it exists
		ignVarPath := filepath.Join(configDir, "ign-var.json")
		if _, err := os.Stat(ignVarPath); err == nil {
			backupNum := findNextBackupNumber(configDir)
			backupPath := filepath.Join(configDir, fmt.Sprintf("ign-var.json.bk%d", backupNum))
			debug.Debug("[app] Backing up existing config to: %s", backupPath)
			if err := os.Rename(ignVarPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing config: %v", err)
				return NewInitError("failed to backup existing configuration", err)
			}
			debug.Debug("[app] Existing config backed up successfully")
		}
	} else {
		// Create config directory
		debug.Debug("[app] Creating config directory: %s", configDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			debug.Debug("[app] Failed to create config directory: %v", err)
			return NewInitError("failed to create config directory", err)
		}
		debug.Debug("[app] Config directory created successfully")
	}

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
			GeneratedBy:     "ign init",
			TemplateName:    template.Config.Name,
			TemplateVersion: template.Config.Version,
		},
	}

	// Save ign-var.json
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	debug.DebugValue("[app] Saving ign-var.json to", ignVarPath)
	if err := config.SaveIgnVarJson(ignVarPath, ignVarJson); err != nil {
		debug.Debug("[app] Failed to save ign-var.json: %v", err)
		return NewInitError("failed to save ign-var.json", err)
	}
	debug.Debug("[app] ign-var.json saved successfully")

	debug.Debug("[app] Init workflow completed successfully")
	return nil
}

// validateInitOptions validates init options.
func validateInitOptions(opts InitOptions) error {
	if opts.URL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	return nil
}
