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
)

// UpdateOptions contains options for the update command.
type UpdateOptions struct {
	// OutputDir is the directory where project files will be generated.
	OutputDir string
	// Overwrite determines whether to overwrite existing files.
	Overwrite bool
	// OverwriteMode determines how existing files are overwritten.
	OverwriteMode generator.OverwriteMode
	// DryRun simulates generation without writing files.
	DryRun bool
	// Verbose enables detailed logging.
	Verbose bool
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
	// TargetRef overrides the stored template ref for this update.
	TargetRef string
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
	// PreviousRef is the ref stored before applying an update ref override.
	PreviousRef string
	// RequestedRef is the requested ref override.
	RequestedRef string
	// EffectiveRef is the ref used to fetch the template.
	EffectiveRef string
	// RefOverrideRequested indicates whether update was called with a ref override.
	RefOverrideRequested bool
	// RefChanged indicates whether the requested ref differs from the stored ref.
	RefChanged bool
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
	// FilesDeleted is the number of previously managed paths removed from disk
	// or pruned from tracking because they no longer exist in the template
	// during an overwrite update.
	FilesDeleted int
	// Errors contains non-fatal errors encountered during generation.
	Errors []error
	// Files contains the paths of all files processed.
	Files []string
	// DeletedFiles contains previously managed paths removed from disk or
	// pruned from tracking because they no longer exist in the template during
	// an overwrite update.
	DeletedFiles []string
	// DryRunFiles contains detailed information for dry-run mode.
	DryRunFiles []DryRunFile
	// UnresolvedTransitionPaths lists managed directories preserved because ign
	// could not prove their directory-to-symlink transition was safe. The
	// matching explanation is in Errors; callers report these separately so the
	// generic "already exists" skip never stands in for a recovery diagnostic.
	UnresolvedTransitionPaths []string
	// Directories contains directories that would be created (dry-run only).
	Directories []string
	// RefChanged indicates whether the stored template ref changed.
	RefChanged bool
	// RefOverrideRequested indicates whether update was called with a ref override.
	RefOverrideRequested bool
	// ExecutionPlan is returned by a preview and must be reused for its
	// confirmed mutation to prevent a changed source tree from being reclassified.
	ExecutionPlan *UpdateExecutionPlan
}

// PrepareUpdate prepares for update by checking if .ign exists and fetching template.
// Returns information about hash changes and new variables that need prompting.
func PrepareUpdate(ctx context.Context, opts UpdateOptions) (*PrepareUpdateResult, error) {
	configDir := filepath.Join(opts.OutputDir, model.IgnConfigDir)
	ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)

	debug.DebugSection("[app] PrepareUpdate workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] ConfigDir", configDir)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)
	debug.DebugValue("[app] TargetRef", opts.TargetRef)

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

	previousRef := ignConfig.Template.Ref
	requestedRef := opts.TargetRef
	refOverrideRequested := requestedRef != ""
	if refOverrideRequested {
		if err := ValidateGitRef(requestedRef); err != nil {
			return nil, NewValidationError("invalid update ref", err)
		}
	}

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
	if refOverrideRequested {
		templateSource.Ref = requestedRef
	}

	debug.Debug("[app] Fetching template from provider")
	fetched, err := fetchTrackedTemplate(ctx, trackedTemplateFetchOptions{
		Source:      templateSource,
		GitHubToken: opts.GitHubToken,
	})
	if err != nil {
		return nil, err
	}
	template := fetched.Template
	effectiveRef := fetched.TemplateRef.Ref
	debug.Debug("[app] Template fetched successfully")
	debug.DebugValue("[app] Template name", template.Config.Name)
	debug.DebugValue("[app] Template version", template.Config.Version)

	// Step 5: Get hash from template's ign-template.json and compare
	// The hash must be present (calculated by 'ign template update' on the template side)
	newHash := template.Config.Hash
	debug.DebugValue("[app] Template hash from ign-template.json", newHash)

	if err := validateTemplateHash(newHash); err != nil {
		return nil, err
	}

	hashChanged := newHash != ignConfig.Hash
	refChanged := refOverrideRequested && requestedRef != previousRef
	debug.DebugValue("[app] Hash changed", hashChanged)
	debug.DebugValue("[app] Ref changed", refChanged)

	// Step 6: Find new and removed variables
	newVars, removedVars := findVariableChanges(existingVars, template.Config.Variables)
	debug.DebugValue("[app] New variables", newVars)
	debug.DebugValue("[app] Removed variables", removedVars)

	result := &PrepareUpdateResult{
		Template:             template,
		IgnJson:              &template.Config,
		ExistingVars:         existingVars,
		NewVars:              newVars,
		RemovedVars:          removedVars,
		CurrentHash:          ignConfig.Hash,
		NewHash:              newHash,
		HashChanged:          hashChanged,
		IgnConfigPath:        ignConfigPath,
		IgnVarPath:           ignVarPath,
		IgnConfig:            ignConfig,
		PreviousRef:          previousRef,
		RequestedRef:         requestedRef,
		EffectiveRef:         effectiveRef,
		RefOverrideRequested: refOverrideRequested,
		RefChanged:           refChanged,
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
	// OverwriteMode determines how existing files are overwritten.
	OverwriteMode generator.OverwriteMode
	// DryRun simulates generation without writing files.
	DryRun bool
	// Verbose enables detailed logging.
	Verbose bool
	// ExecutionPlan is the opaque plan returned by a preceding dry-run preview.
	ExecutionPlan *UpdateExecutionPlan
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

	configDir := filepath.Dir(prep.IgnConfigPath)
	rawVars, vars, err := prepareVariablesForGeneration(prep.IgnJson.Variables, mergedVars, configDir, opts.OutputDir)
	if err != nil {
		return nil, err
	}

	// Validate that all required variables are set
	debug.Debug("[app] Validating required variables")
	if err := ValidateVariables(prep.IgnJson, vars); err != nil {
		debug.Debug("[app] Variable validation failed: %v", err)
		return nil, err
	}

	manifestPath := manifestPathFromConfigPath(prep.IgnConfigPath)
	if opts.DryRun {
		pending, err := symlinkTransitionJournalPending(opts.OutputDir)
		if err != nil {
			return nil, NewCheckoutError("inspect interrupted managed directory-to-symlink transition", err)
		}
		if pending {
			return nil, NewValidationError("interrupted managed directory-to-symlink transition requires recovery before dry run", nil)
		}
	} else if err := recoverSymlinkTransitionJournal(opts.OutputDir, manifestPath, prep.IgnConfigPath, prep.IgnVarPath); err != nil {
		return nil, NewCheckoutError("recover interrupted managed directory-to-symlink transition", err)
	}

	if shouldCompleteUpdateConfigOnly(prep, opts) {
		if !opts.DryRun {
			if err := saveCompleteUpdateConfigOnlyArtifacts(prep, rawVars); err != nil {
				return nil, err
			}
		}
		return &UpdateResult{
			HashChanged:          prep.HashChanged,
			NewVariables:         prep.NewVars,
			RemovedVariables:     prep.RemovedVars,
			RefChanged:           prep.RefChanged,
			RefOverrideRequested: prep.RefOverrideRequested,
		}, nil
	}

	// Create generator
	gen := generator.NewGenerator()

	// Prepare generate options
	genOpts := generator.GenerateOptions{
		Template:      prep.Template,
		Variables:     vars,
		OutputDir:     opts.OutputDir,
		Overwrite:     opts.Overwrite,
		OverwriteMode: opts.OverwriteMode,
		Verbose:       opts.Verbose,
		SkipUnchanged: true,
	}

	plan := opts.ExecutionPlan
	if plan != nil {
		if err := plan.validate(opts.OutputDir, manifestPath, prep.NewHash, effectiveUpdateOverwriteMode(opts.OverwriteMode, opts.Overwrite)); err != nil {
			return nil, NewValidationError("invalid update execution plan", err)
		}
	} else if effectiveUpdateOverwriteMode(opts.OverwriteMode, opts.Overwrite) != generator.OverwriteNone {
		preview, err := gen.DryRun(ctx, genOpts)
		if err != nil {
			return nil, NewCheckoutError("failed to classify symlink transitions", err)
		}
		plan, err = classifyUpdateSymlinkTransitionsWithVariables(ctx, opts.OutputDir, manifestPath, prep.Template, vars, preview, effectiveUpdateOverwriteMode(opts.OverwriteMode, opts.Overwrite))
		if err != nil {
			return nil, NewCheckoutError("failed to classify symlink transitions", err)
		}
	}
	unresolvedTransitionPaths, transitionDiagnostics := collectUpdateSymlinkTransitionDiagnostics(plan)
	if plan != nil {
		genOpts.SymlinkTransitions = plan.transitions
	}

	// Generate or dry run
	var genResult *generator.GenerateResult
	var rollback *checkoutGenerationRollback
	var transitionTransactions *symlinkTransitionTransactions
	if !opts.DryRun {
		transitionTransactions, err = prepareSymlinkTransitionTransactions(opts.OutputDir, genOpts.SymlinkTransitions, manifestPath, prep.IgnConfigPath, prep.IgnVarPath)
		if err != nil {
			return nil, NewCheckoutError("prepare managed directory-to-symlink transition", err)
		}
		defer func() {
			if rollbackErr := transitionTransactions.rollback(); rollbackErr != nil {
				debug.Debug("[app] Failed to restore managed symlink transition: %v", rollbackErr)
			}
		}()
		rollback, err = prepareCheckoutGenerationRollback(ctx, gen, genOpts)
		if err != nil {
			return nil, err
		}
		defer rollback.cleanup()
	}
	if opts.DryRun {
		debug.Debug("[app] Starting dry run generation")
		genResult, err = gen.DryRun(ctx, genOpts)
	} else {
		debug.Debug("[app] Starting project generation")
		genResult, err = gen.Generate(ctx, genOpts)
	}

	if err != nil {
		debug.Debug("[app] Generation failed: %v", err)
		if rollback != nil {
			rollback.captureGeneratedNodes(genResult)
			if rollbackErr := transitionTransactions.rollback(); rollbackErr != nil {
				debug.Debug("[app] Failed to restore managed symlink transition: %v", rollbackErr)
			}
			rollback.rollback(genResult)
		}
		return nil, NewCheckoutError("generation failed", err)
	}
	debug.Debug("[app] Generation completed successfully")
	if rollback != nil {
		rollback.captureGeneratedNodes(genResult)
	}

	removedManagedFiles, cleanupErr := cleanupRemovedManagedFilesForUpdate(ctx, cleanupRemovedManagedFilesOptions{
		ManifestPath:       manifestPath,
		OutputDir:          opts.OutputDir,
		Template:           prep.Template,
		GenerateResult:     genResult,
		OverwriteMode:      opts.OverwriteMode,
		Overwrite:          opts.Overwrite,
		DryRun:             opts.DryRun,
		SymlinkTransitions: genOpts.SymlinkTransitions,
	})
	if cleanupErr != nil {
		debug.Debug("[app] Failed to remove stale managed files: %v", cleanupErr)
	}

	if !opts.DryRun {
		if err := saveCompleteUpdateArtifacts(prep, rawVars, genResult, removedManagedFiles, rollback, transitionTransactions); err != nil {
			return nil, err
		}
		if err := transitionTransactions.markArtifactsCommitted(); err != nil {
			rollbackUpdateGeneration(rollback, genResult, transitionTransactions)
			return nil, NewCheckoutError("record committed managed directory-to-symlink transition", err)
		}
		if err := transitionTransactions.commit(); err != nil {
			return nil, NewCheckoutError("remove managed directory-to-symlink transaction backup", err)
		}
	}

	// Build result
	result := &UpdateResult{
		HashChanged:          prep.HashChanged,
		NewVariables:         prep.NewVars,
		RemovedVariables:     prep.RemovedVars,
		FilesCreated:         genResult.FilesCreated,
		FilesSkipped:         genResult.FilesSkipped,
		FilesOverwritten:     genResult.FilesOverwritten,
		FilesDeleted:         removedManagedFiles.FilesDeleted,
		Errors:               append(append([]error(nil), genResult.Errors...), transitionDiagnostics...),
		Files:                genResult.Files,
		DeletedFiles:         removedManagedFiles.DeletedFiles,
		Directories:          genResult.Directories,
		RefChanged:           prep.RefChanged,
		RefOverrideRequested: prep.RefOverrideRequested,
		ExecutionPlan:        plan,

		UnresolvedTransitionPaths: unresolvedTransitionPaths,
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
	if cleanupErr != nil {
		return result, NewCheckoutError("failed to remove files no longer present in template", cleanupErr)
	}
	return result, nil
}

func shouldCompleteUpdateConfigOnly(prep *PrepareUpdateResult, opts CompleteUpdateOptions) bool {
	return prep.RefChanged && !prep.HashChanged && !opts.Overwrite
}

func saveCompleteUpdateArtifacts(prep *PrepareUpdateResult, rawVars map[string]interface{}, genResult *generator.GenerateResult, removedManagedFiles *cleanupRemovedManagedFilesResult, rollback *checkoutGenerationRollback, transitions *symlinkTransitionTransactions) error {
	if removedManagedFiles == nil {
		removedManagedFiles = &cleanupRemovedManagedFilesResult{}
	}

	ignConfigSnapshot, err := rollback.snapshotPath(prep.IgnConfigPath)
	if err != nil {
		rollbackUpdateGeneration(rollback, genResult, transitions)
		return NewCheckoutError("failed to prepare ign.json rollback", err)
	}
	ignVarSnapshot, err := rollback.snapshotPath(prep.IgnVarPath)
	if err != nil {
		rollbackUpdateGeneration(rollback, genResult, transitions)
		return NewCheckoutError("failed to prepare ign-var.json rollback", err)
	}
	manifestPath := manifestPathFromConfigPath(prep.IgnConfigPath)
	manifestSnapshot, err := rollback.snapshotPath(manifestPath)
	if err != nil {
		rollbackUpdateGeneration(rollback, genResult, transitions)
		return NewCheckoutError("failed to prepare ign-files.json rollback", err)
	}

	manifestWrite, err := saveManifestFromGenerateResultWithResult(manifestPath, genResult, removedManagedFiles.RemovedCanonicalPaths)
	bindCheckoutArtifactWrite(rollback, &manifestSnapshot, manifestWrite)
	if err != nil {
		debug.Debug("[app] Failed to save ign-files.json: %v", err)
		rollbackUpdateGeneration(rollback, genResult, transitions)
		restoreUpdateArtifactSnapshots(rollback, ignConfigSnapshot, ignVarSnapshot, manifestSnapshot)
		return NewCheckoutError("failed to save ign-files.json", err)
	}

	applyCompleteUpdateConfig(prep)
	configWrite, err := saveIgnConfigWithResult(prep.IgnConfigPath, prep.IgnConfig)
	bindCheckoutArtifactWrite(rollback, &ignConfigSnapshot, configWrite)
	if err != nil {
		debug.Debug("[app] Failed to save ign.json: %v", err)
		rollbackUpdateGeneration(rollback, genResult, transitions)
		restoreUpdateArtifactSnapshots(rollback, ignConfigSnapshot, ignVarSnapshot, manifestSnapshot)
		return NewCheckoutError("failed to save ign.json", err)
	}
	ignVarJson := &model.IgnVarJson{Variables: rawVars}
	varWrite, err := saveIgnVarJsonWithResult(prep.IgnVarPath, ignVarJson)
	bindCheckoutArtifactWrite(rollback, &ignVarSnapshot, varWrite)
	if err != nil {
		debug.Debug("[app] Failed to save ign-var.json: %v", err)
		rollbackUpdateGeneration(rollback, genResult, transitions)
		restoreUpdateArtifactSnapshots(rollback, ignConfigSnapshot, ignVarSnapshot, manifestSnapshot)
		return NewCheckoutError("failed to save ign-var.json", err)
	}

	return nil
}

func rollbackUpdateGeneration(rollback *checkoutGenerationRollback, genResult *generator.GenerateResult, transitions *symlinkTransitionTransactions) {
	if err := transitions.rollback(); err != nil {
		debug.Debug("[app] Failed to restore managed symlink transition: %v", err)
	}
	rollback.rollback(genResult)
}

func saveCompleteUpdateConfigOnlyArtifacts(prep *PrepareUpdateResult, rawVars map[string]interface{}) error {
	rollback := &checkoutGenerationRollback{outputDir: filepath.Dir(prep.IgnConfigPath)}
	defer rollback.cleanup()

	ignConfigSnapshot, err := rollback.snapshotPath(prep.IgnConfigPath)
	if err != nil {
		return NewCheckoutError("failed to prepare ign.json rollback", err)
	}
	ignVarSnapshot, err := rollback.snapshotPath(prep.IgnVarPath)
	if err != nil {
		return NewCheckoutError("failed to prepare ign-var.json rollback", err)
	}

	applyCompleteUpdateConfig(prep)
	configWrite, err := saveIgnConfigWithResult(prep.IgnConfigPath, prep.IgnConfig)
	bindCheckoutArtifactWrite(rollback, &ignConfigSnapshot, configWrite)
	if err != nil {
		restoreUpdateArtifactSnapshots(rollback, ignConfigSnapshot, ignVarSnapshot)
		return NewCheckoutError("failed to save ign.json", err)
	}
	ignVarJson := &model.IgnVarJson{Variables: rawVars}
	varWrite, err := saveIgnVarJsonWithResult(prep.IgnVarPath, ignVarJson)
	bindCheckoutArtifactWrite(rollback, &ignVarSnapshot, varWrite)
	if err != nil {
		restoreUpdateArtifactSnapshots(rollback, ignConfigSnapshot, ignVarSnapshot)
		return NewCheckoutError("failed to save ign-var.json", err)
	}

	return nil
}

func applyCompleteUpdateConfig(prep *PrepareUpdateResult) {
	prep.IgnConfig.Hash = prep.NewHash
	if prep.RefOverrideRequested {
		prep.IgnConfig.Template.Ref = prep.RequestedRef
	}
	prep.IgnConfig.Metadata = &model.FileMetadata{
		GeneratedAt:     time.Now(),
		GeneratedBy:     "ign update",
		TemplateName:    prep.IgnJson.Name,
		TemplateVersion: prep.IgnJson.Version,
		IgnVersion:      build.Version(),
	}
}

func restoreUpdateArtifactSnapshots(rollback *checkoutGenerationRollback, entries ...checkoutRollbackEntry) {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].expectedInfo == nil && entries[i].expectedFingerprint == "" {
			continue
		}
		if err := rollback.restoreCheckoutRollbackEntry(entries[i]); err != nil {
			debug.Debug("[app] Failed to restore update artifact %s: %v", entries[i].path, err)
		}
	}
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
	return mergeVariableDefaults(newVarDefs, providedVars)
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
