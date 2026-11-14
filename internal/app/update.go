package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tacogips/ign/internal/build"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
	"github.com/tacogips/ign/internal/template/provider"
)

// UpdateOptions contains options for the update command.
type UpdateOptions struct {
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

// PrepareUpdateResult contains the result of update preparation.
type PrepareUpdateResult struct {
	// Template is the fetched template.
	Template *model.Template
	// IgnJson is the template configuration with variable definitions.
	IgnJson *model.IgnJson
	// ExistingVars contains the existing variable values from ign-var.json.
	ExistingVars map[string]interface{}
	// NewVars contains names of newly added variables that need prompting.
	NewVars []string
	// RemovedVars contains names of variables that no longer exist in template.
	RemovedVars []string
	// CurrentHash is the current hash stored in .ign/ign.json.
	CurrentHash string
	// NewHash is the new hash of the fetched template.
	NewHash string
	// HashChanged indicates whether the template has changed.
	HashChanged bool
	// IgnConfigPath is the path to .ign/ign.json.
	IgnConfigPath string
	// IgnVarPath is the path to .ign/ign-var.json.
	IgnVarPath string
	// IgnConfig is the existing ign.json configuration.
	IgnConfig *model.IgnConfig
}

// UpdateResult contains the results of the update operation.
type UpdateResult struct {
	// HashChanged indicates if the template was updated.
	HashChanged bool
	// NewVariables lists new variables that were added.
	NewVariables []string
	// RemovedVariables lists variables that were removed.
	RemovedVariables []string
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

// PrepareUpdate prepares for update by checking if .ign exists and fetching template.
// Returns information about hash changes and new variables that need prompting.
func PrepareUpdate(ctx context.Context, opts UpdateOptions) (*PrepareUpdateResult, error) {
	configDir := ".ign"
	ignConfigPath := filepath.Join(configDir, "ign.json")
	ignVarPath := filepath.Join(configDir, "ign-var.json")

	debug.DebugSection("[app] PrepareUpdate workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigDir", configDir)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)

	// Step 1: Check if .ign directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		debug.Debug("[app] .ign directory not found")
		return nil, NewValidationError(
			"update requires prior checkout: .ign directory not found.\n"+
				"Run 'ign checkout <template-url>' first.",
			nil,
		)
	}

	// Step 2: Load existing configuration
	debug.Debug("[app] Loading existing ign.json")
	ignConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		debug.Debug("[app] Failed to load ign.json: %v", err)
		return nil, NewCheckoutError(
			"failed to load .ign/ign.json: run 'ign checkout <template-url>' first",
			err,
		)
	}
	debug.DebugValue("[app] Template URL", ignConfig.Template.URL)
	debug.DebugValue("[app] Current hash", ignConfig.Hash)

	// Step 3: Load existing variables
	debug.Debug("[app] Loading existing ign-var.json")
	ignVar, err := config.LoadIgnVarJson(ignVarPath)
	if err != nil {
		debug.Debug("[app] Failed to load ign-var.json: %v", err)
		return nil, NewCheckoutError(
			"failed to load .ign/ign-var.json: run 'ign checkout <template-url>' first",
			err,
		)
	}
	existingVars := ignVar.Variables
	if existingVars == nil {
		existingVars = make(map[string]interface{})
	}
	debug.DebugValue("[app] Existing variables count", len(existingVars))

	// Step 4: Create provider and fetch template
	templateSource := ignConfig.Template
	normalizedURL := NormalizeTemplateURL(templateSource.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)

	debug.Debug("[app] Creating template provider")
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return nil, NewCheckoutError("failed to create provider", err)
	}

	debug.Debug("[app] Resolving template URL")
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}

	// Use ref and path from configuration
	if templateSource.Ref != "" {
		templateRef.Ref = templateSource.Ref
	}
	if templateSource.Path != "" {
		templateRef.Path = templateSource.Path
	}

	debug.Debug("[app] Fetching template from provider")
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}
	debug.Debug("[app] Template fetched successfully")
	debug.DebugValue("[app] Template name", template.Config.Name)
	debug.DebugValue("[app] Template version", template.Config.Version)

	// Step 5: Get hash from template's ign.json and compare
	// The hash is stored in the template's ign.json (calculated by 'ign template update')
	newHash := template.Config.Hash
	debug.DebugValue("[app] Template hash from ign.json", newHash)

	// If template doesn't have a hash yet, calculate it from content
	// This fallback handles templates created before hash support was added.
	// Templates should always include hash in ign.json from 'ign template update' command.
	if newHash == "" {
		debug.Debug("[app] Template has no hash in ign.json, calculating from content")
		newHash = calculateTemplateHash(template)
		debug.DebugValue("[app] Calculated template hash", newHash)
	}

	hashChanged := newHash != ignConfig.Hash
	debug.DebugValue("[app] Hash changed", hashChanged)

	// Step 6: Find new and removed variables
	newVars, removedVars := findVariableChanges(existingVars, template.Config.Variables)
	debug.DebugValue("[app] New variables", newVars)
	debug.DebugValue("[app] Removed variables", removedVars)

	result := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  existingVars,
		NewVars:       newVars,
		RemovedVars:   removedVars,
		CurrentHash:   ignConfig.Hash,
		NewHash:       newHash,
		HashChanged:   hashChanged,
		IgnConfigPath: ignConfigPath,
		IgnVarPath:    ignVarPath,
		IgnConfig:     ignConfig,
	}

	debug.Debug("[app] PrepareUpdate completed successfully")
	return result, nil
}

// findVariableChanges compares existing variables with template variables.
// Returns lists of new variables (in template but not in existing) and
// removed variables (in existing but not in template).
// Results are sorted alphabetically for consistent output.
func findVariableChanges(existing map[string]interface{}, templateVars map[string]model.VarDef) (newVars, removedVars []string) {
	// Find new variables (in template but not in existing)
	for name := range templateVars {
		if _, ok := existing[name]; !ok {
			newVars = append(newVars, name)
		}
	}

	// Find removed variables (in existing but not in template)
	for name := range existing {
		if _, ok := templateVars[name]; !ok {
			removedVars = append(removedVars, name)
		}
	}

	// Sort for deterministic output
	sort.Strings(newVars)
	sort.Strings(removedVars)

	return newVars, removedVars
}

// CompleteUpdateOptions contains options for completing the update.
type CompleteUpdateOptions struct {
	// PrepareResult is the result from PrepareUpdate.
	PrepareResult *PrepareUpdateResult
	// NewVariables contains values for newly added variables.
	NewVariables map[string]interface{}
	// OutputDir is the directory where project files will be generated.
	OutputDir string
	// Overwrite determines whether to overwrite existing files.
	Overwrite bool
	// DryRun simulates generation without writing files.
	DryRun bool
	// Verbose enables detailed logging.
	Verbose bool
}

// CompleteUpdate completes the update operation by merging variables and regenerating files.
func CompleteUpdate(ctx context.Context, opts CompleteUpdateOptions) (*UpdateResult, error) {
	debug.DebugSection("[app] CompleteUpdate workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)

	prep := opts.PrepareResult
	if prep == nil {
		return nil, NewValidationError("update preparation result cannot be nil", nil)
	}
	if prep.IgnJson == nil {
		return nil, NewValidationError("template configuration cannot be nil", nil)
	}

	// Validate output directory
	if opts.OutputDir == "" {
		return nil, NewValidationError("update output directory cannot be empty", nil)
	}
	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return nil, NewValidationError("invalid output directory", err)
	}

	// Merge existing variables with new variables
	mergedVars := make(map[string]interface{})
	// Check if template has variable definitions
	if prep.IgnJson.Variables != nil {
		for name, value := range prep.ExistingVars {
			// Skip removed variables
			if _, exists := prep.IgnJson.Variables[name]; exists {
				mergedVars[name] = value
			}
		}
	}
	// If no variables defined in template, just use new variables
	for name, value := range opts.NewVariables {
		mergedVars[name] = value
	}
	debug.DebugValue("[app] Merged variables count", len(mergedVars))

	// Convert variables to parser.Variables
	vars := parser.NewMapVariables(mergedVars)

	// Validate that all required variables are set
	debug.Debug("[app] Validating required variables")
	if err := ValidateVariables(prep.IgnJson, vars); err != nil {
		debug.Debug("[app] Variable validation failed: %v", err)
		return nil, err
	}

	// Update configuration files (unless dry-run)
	if !opts.DryRun {
		// Update ign.json with new hash
		debug.Debug("[app] Updating ign.json with new hash")
		prep.IgnConfig.Hash = prep.NewHash
		prep.IgnConfig.Metadata = &model.FileMetadata{
			GeneratedAt:     time.Now(),
			GeneratedBy:     "ign update",
			TemplateName:    prep.IgnJson.Name,
			TemplateVersion: prep.IgnJson.Version,
			IgnVersion:      build.Version(),
		}

		if err := config.SaveIgnConfig(prep.IgnConfigPath, prep.IgnConfig); err != nil {
			debug.Debug("[app] Failed to save ign.json: %v", err)
			return nil, NewCheckoutError("failed to save ign.json", err)
		}
		debug.Debug("[app] ign.json updated successfully")

		// Update ign-var.json with merged variables (no metadata as it's already in ign.json)
		debug.Debug("[app] Updating ign-var.json with merged variables")
		ignVarJson := &model.IgnVarJson{
			Variables: mergedVars,
		}

		if err := config.SaveIgnVarJson(prep.IgnVarPath, ignVarJson); err != nil {
			debug.Debug("[app] Failed to save ign-var.json: %v", err)
			return nil, NewCheckoutError("failed to save ign-var.json", err)
		}
		debug.Debug("[app] ign-var.json updated successfully")
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

	// Build result
	result := &UpdateResult{
		HashChanged:      prep.HashChanged,
		NewVariables:     prep.NewVars,
		RemovedVariables: prep.RemovedVars,
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

	debug.Debug("[app] CompleteUpdate workflow completed successfully")
	return result, nil
}

// GetNewVariableDefinitions returns VarDef for new variables that need prompting.
func GetNewVariableDefinitions(prep *PrepareUpdateResult) map[string]model.VarDef {
	result := make(map[string]model.VarDef)
	if prep == nil || prep.IgnJson == nil || prep.IgnJson.Variables == nil {
		return result
	}
	for _, name := range prep.NewVars {
		if varDef, ok := prep.IgnJson.Variables[name]; ok {
			result[name] = varDef
		}
	}
	return result
}

// FilterVariablesForPrompt returns only the variables that need to be prompted.
// Variables with defaults and not required are excluded.
func FilterVariablesForPrompt(newVarDefs map[string]model.VarDef) map[string]model.VarDef {
	result := make(map[string]model.VarDef)
	for name, varDef := range newVarDefs {
		// Prompt if variable is required OR has no default
		if varDef.Required || varDef.Default == nil {
			result[name] = varDef
		}
	}
	return result
}

// ApplyDefaults applies default values from newVarDefs to variables.
// If providedVars is nil, it is treated as an empty map and only defaults are applied.
// Returns a new map containing provided variables plus defaults for any missing variables.
func ApplyDefaults(newVarDefs map[string]model.VarDef, providedVars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	// Copy provided variables (nil map iteration is safe in Go - no-op)
	for name, value := range providedVars {
		result[name] = value
	}
	// Add defaults for variables not provided
	for name, varDef := range newVarDefs {
		if _, provided := result[name]; !provided && varDef.Default != nil {
			result[name] = varDef.Default
		}
	}
	return result
}

// FormatVariableChanges returns a formatted string describing variable changes.
func FormatVariableChanges(prep *PrepareUpdateResult) string {
	if len(prep.NewVars) == 0 && len(prep.RemovedVars) == 0 {
		return ""
	}

	var msg string
	if len(prep.NewVars) > 0 {
		msg += fmt.Sprintf("New variables: %v\n", prep.NewVars)
	}
	if len(prep.RemovedVars) > 0 {
		msg += fmt.Sprintf("Removed variables: %v\n", prep.RemovedVars)
	}
	return msg
}
