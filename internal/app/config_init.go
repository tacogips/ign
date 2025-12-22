package app

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
)

// InitOptions contains options for configuration initialization.
// Deprecated: Use PrepareCheckout and CompleteCheckout instead.
type InitOptions struct {
	// URL is the template URL or path.
	URL string
	// Ref is the git branch, tag, or commit SHA.
	Ref string
	// Force backs up and overwrites existing .ign directory if true.
	Force bool
	// Config is the path to global config file (optional).
	Config string
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
}

// Init initializes configuration from a template.
// Creates two configuration files in .ign directory:
//   - .ign/ign.json (template source and hash)
//   - .ign/ign-var.json (variable values with defaults)
//
// Deprecated: Use PrepareCheckout and CompleteCheckout instead.
func Init(ctx context.Context, opts InitOptions) error {
	configDir := ".ign"

	debug.DebugSection("[app] Init workflow start (deprecated)")
	debug.DebugValue("[app] Template URL", opts.URL)
	debug.DebugValue("[app] ConfigDir", configDir)
	debug.DebugValue("[app] Ref", opts.Ref)
	debug.DebugValue("[app] Force", opts.Force)

	// Check if config already exists
	configExists := false
	if _, err := checkConfigDir(configDir); err == nil {
		configExists = true
		if !opts.Force {
			return NewInitError(
				"configuration already exists at "+configDir+" (use --force to reinitialize)",
				nil,
			)
		}
	}

	// Use PrepareCheckout to handle template fetching and config directory setup
	prepResult, err := PrepareCheckout(ctx, PrepareCheckoutOptions{
		URL:          opts.URL,
		Ref:          opts.Ref,
		Force:        opts.Force,
		ConfigExists: configExists,
		GitHubToken:  opts.GitHubToken,
	})
	if err != nil {
		return err
	}

	// Calculate template hash
	templateHash := calculateTemplateHash(prepResult.Template)
	debug.DebugValue("[app] Template hash", templateHash)

	// Save ign.json (template source and hash)
	ignConfigPath := filepath.Join(configDir, "ign.json")
	debug.Debug("[app] Creating ign.json")
	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL:  prepResult.NormalizedURL,
			Path: prepResult.TemplateRef.Path,
			Ref:  prepResult.TemplateRef.Ref,
		},
		Hash: templateHash,
		Metadata: &model.FileMetadata{
			GeneratedAt:     time.Now(),
			GeneratedBy:     "ign init",
			TemplateName:    prepResult.IgnJson.Name,
			TemplateVersion: prepResult.IgnJson.Version,
		},
	}

	debug.DebugValue("[app] Saving ign.json to", ignConfigPath)
	if err := config.SaveIgnConfig(ignConfigPath, ignConfig); err != nil {
		debug.Debug("[app] Failed to save ign.json: %v", err)
		return NewInitError("failed to save ign.json", err)
	}
	debug.Debug("[app] ign.json saved successfully")

	// Create ign-var.json with empty/default variables (not generating files)
	debug.Debug("[app] Creating ign-var.json with default variables")
	ignVarJson := &model.IgnVarJson{
		Variables: CreateEmptyVariablesMap(prepResult.IgnJson),
		Metadata: &model.FileMetadata{
			GeneratedAt:     time.Now(),
			GeneratedBy:     "ign init",
			TemplateName:    prepResult.IgnJson.Name,
			TemplateVersion: prepResult.IgnJson.Version,
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

// checkConfigDir checks if the config directory exists.
func checkConfigDir(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, NewInitError(dir+" exists but is not a directory", nil)
	}
	return true, nil
}
