package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tacogips/ign/internal/build"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
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
	// SkipConfigSetup skips .ign creation/backup during preparation.
	SkipConfigSetup bool
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
	// PreparedInputs contains prevalidated variable inputs from PrepareCompleteCheckoutInputs.
	PreparedInputs *PreparedCompleteCheckoutInputs
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

func validateTemplateHash(hash string) error {
	debug.DebugValue("[app] Template hash from ign-template.json", hash)
	if hash == "" {
		debug.Debug("[app] Template hash is missing in ign-template.json")
		return NewCheckoutError(
			"template is missing hash in ign-template.json.\n"+
				"The template author needs to run 'ign template update' to generate the hash.",
			nil,
		)
	}

	if config.IsValidSHA256Hash(hash) {
		return nil
	}

	debug.Debug("[app] Template hash has invalid format")
	return NewCheckoutError(
		"template hash in ign-template.json must be a valid SHA256 string.\n"+
			"The template author needs to run 'ign template update' to regenerate the hash.",
		nil,
	)
}

// PrepareCheckout prepares for checkout by fetching the template and handling config directory.
// Returns template information and variable definitions for interactive prompting.
func PrepareCheckout(ctx context.Context, opts PrepareCheckoutOptions) (*PrepareCheckoutResult, error) {
	configDir := model.IgnConfigDir

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

	if err := validateTemplateHash(template.Config.Hash); err != nil {
		return nil, err
	}

	if !opts.SkipConfigSetup {
		if err := PrepareCheckoutConfigDir(opts.ConfigExists); err != nil {
			return nil, err
		}
	}

	debug.Debug("[app] PrepareCheckout completed successfully")
	return &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   templateRef,
		NormalizedURL: normalizedURL,
	}, nil
}

// PrepareCheckoutConfigDir creates or backs up .ign before writing checkout/init files.
func PrepareCheckoutConfigDir(configExists bool) error {
	configDir := model.IgnConfigDir

	if configExists {
		// Directory exists and Force is true (checked in CLI layer)
		debug.Debug("[app] Config directory exists, Force mode - backing up")

		// Backup existing ign.json if it exists
		ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
		if _, err := os.Stat(ignConfigPath); err == nil {
			backupNum, err := findNextBackupNumber(configDir, model.IgnProjectConfigFile)
			if err != nil {
				return NewCheckoutError(err.Error(), nil)
			}
			backupPath := filepath.Join(configDir, fmt.Sprintf("%s.bk%d", model.IgnProjectConfigFile, backupNum))
			debug.Debug("[app] Backing up existing ign.json to: %s", backupPath)
			if err := os.Rename(ignConfigPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing ign.json: %v", err)
				return NewCheckoutError("failed to backup existing ign.json", err)
			}
			debug.Debug("[app] Existing ign.json backed up successfully")
		}

		// Backup existing ign-var.json if it exists
		ignVarPath := filepath.Join(configDir, model.IgnVarFile)
		if _, err := os.Stat(ignVarPath); err == nil {
			backupNum, err := findNextBackupNumber(configDir, model.IgnVarFile)
			if err != nil {
				return NewCheckoutError(err.Error(), nil)
			}
			backupPath := filepath.Join(configDir, fmt.Sprintf("%s.bk%d", model.IgnVarFile, backupNum))
			debug.Debug("[app] Backing up existing ign-var.json to: %s", backupPath)
			if err := os.Rename(ignVarPath, backupPath); err != nil {
				debug.Debug("[app] Failed to backup existing ign-var.json: %v", err)
				return NewCheckoutError("failed to backup existing ign-var.json", err)
			}
			debug.Debug("[app] Existing ign-var.json backed up successfully")
		}

		if err := backupManifestIfExists(); err != nil {
			debug.Debug("[app] Failed to backup existing ign-files.json: %v", err)
			return NewCheckoutError("failed to backup existing ign-files.json", err)
		}

		return nil
	}

	// Create config directory
	debug.Debug("[app] Creating config directory: %s", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		debug.Debug("[app] Failed to create config directory: %v", err)
		return NewCheckoutError("failed to create config directory", err)
	}
	debug.Debug("[app] Config directory created successfully")

	return nil
}

// CompleteCheckout completes checkout by saving configuration and generating files.
func CompleteCheckout(ctx context.Context, opts CompleteCheckoutOptions) (*CheckoutResult, error) {
	configDir := model.IgnConfigDir
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)

	debug.DebugSection("[app] CompleteCheckout workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)
	debug.DebugValue("[app] IgnVarPath", ignVarPath)
	debug.DebugValue("[app] Overwrite", opts.Overwrite)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Verbose", opts.Verbose)

	preparedInputs, err := checkoutInputsForCompletion(opts)
	if err != nil {
		return nil, err
	}
	rawVars := preparedInputs.RawVariables
	vars := preparedInputs.RuntimeVariables
	prep := opts.PrepareResult

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
	var rollback *checkoutGenerationRollback
	if !opts.DryRun {
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
			rollback.rollback(genResult)
		}
		return nil, NewCheckoutError("generation failed", err)
	}
	debug.Debug("[app] Generation completed successfully")
	debug.DebugValue("[app] Files created", genResult.FilesCreated)
	debug.DebugValue("[app] Files skipped", genResult.FilesSkipped)
	debug.DebugValue("[app] Files overwritten", genResult.FilesOverwritten)

	if !opts.DryRun {
		if err := saveCompleteCheckoutArtifacts(configDir, prep, rawVars, genResult, rollback); err != nil {
			return nil, err
		}
	}

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

func saveCompleteCheckoutArtifacts(configDir string, prep *PrepareCheckoutResult, rawVars map[string]interface{}, genResult *generator.GenerateResult, rollback *checkoutGenerationRollback) error {
	templateHash := prep.IgnJson.Hash
	ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)

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
			IgnVersion:      build.Version(),
		},
	}

	debug.Debug("[app] Creating ign-var.json")
	ignVarJson := &model.IgnVarJson{
		Variables: rawVars,
	}

	debug.DebugValue("[app] Saving ign.json to", ignConfigPath)
	if err := config.SaveIgnConfig(ignConfigPath, ignConfig); err != nil {
		debug.Debug("[app] Failed to save ign.json: %v", err)
		debug.Debug("[app] Rolling back generated files due to ign.json save failure")
		rollback.rollback(genResult)
		return NewCheckoutError("failed to save ign.json", err)
	}
	debug.Debug("[app] ign.json saved successfully")

	debug.DebugValue("[app] Saving ign-var.json to", ignVarPath)
	if err := config.SaveIgnVarJson(ignVarPath, ignVarJson); err != nil {
		debug.Debug("[app] Failed to save ign-var.json: %v", err)
		debug.Debug("[app] Rolling back checkout artifacts due to ign-var.json save failure")
		rollback.rollback(genResult)
		if removeErr := os.Remove(ignConfigPath); removeErr != nil {
			debug.Debug("[app] Failed to rollback ign.json: %v (original error: %v)", removeErr, err)
		}
		return NewCheckoutError("failed to save ign-var.json (rolled back ign.json)", err)
	}
	debug.Debug("[app] ign-var.json saved successfully")

	manifestPath := manifestPathFromConfigPath(ignConfigPath)
	manifestSnapshot, err := rollback.snapshotPath(manifestPath)
	if err != nil {
		debug.Debug("[app] Failed to prepare ign-files.json rollback: %v", err)
		rollback.rollback(genResult)
		if removeErr := os.Remove(ignVarPath); removeErr != nil {
			debug.Debug("[app] Failed to rollback ign-var.json: %v (original error: %v)", removeErr, err)
		}
		if removeErr := os.Remove(ignConfigPath); removeErr != nil {
			debug.Debug("[app] Failed to rollback ign.json: %v (original error: %v)", removeErr, err)
		}
		return NewCheckoutError("failed to prepare ign-files.json rollback", err)
	}

	if err := saveManifestFromGenerateResult(manifestPath, genResult); err != nil {
		debug.Debug("[app] Failed to save ign-files.json: %v", err)
		debug.Debug("[app] Rolling back checkout artifacts due to ign-files.json save failure")
		rollback.rollback(genResult)
		if restoreErr := restoreCheckoutRollbackEntry(manifestSnapshot); restoreErr != nil {
			debug.Debug("[app] Failed to restore ign-files.json after save failure: %v", restoreErr)
		}
		if removeErr := os.Remove(ignVarPath); removeErr != nil {
			debug.Debug("[app] Failed to rollback ign-var.json: %v (original error: %v)", removeErr, err)
		}
		if removeErr := os.Remove(ignConfigPath); removeErr != nil {
			debug.Debug("[app] Failed to rollback ign.json: %v (original error: %v)", removeErr, err)
		}
		return NewCheckoutError("failed to save ign-files.json (rolled back ign.json and ign-var.json)", err)
	}
	debug.Debug("[app] ign-files.json saved successfully")

	return nil
}

type checkoutGenerationRollback struct {
	outputDir                      string
	outputDirExistedBeforeGenerate bool
	overwritten                    []checkoutRollbackEntry
	backupDir                      string
}

type checkoutRollbackEntry struct {
	path       string
	existed    bool
	mode       os.FileMode
	backupPath string
	linkTarget string
	isSymlink  bool
	isDir      bool
}

func prepareCheckoutGenerationRollback(ctx context.Context, gen generator.Generator, genOpts generator.GenerateOptions) (*checkoutGenerationRollback, error) {
	rollback := &checkoutGenerationRollback{
		outputDir:                      genOpts.OutputDir,
		outputDirExistedBeforeGenerate: pathExists(genOpts.OutputDir),
	}

	dryRunResult, err := gen.DryRun(ctx, genOpts)
	if err != nil {
		return nil, NewCheckoutError("generation failed", err)
	}

	if err := rollback.captureOverwrittenFiles(dryRunResult); err != nil {
		rollback.cleanup()
		return nil, NewCheckoutError("failed to prepare checkout rollback", err)
	}

	return rollback, nil
}

func (r *checkoutGenerationRollback) captureOverwrittenFiles(genResult *generator.GenerateResult) error {
	if r == nil || genResult == nil {
		return nil
	}
	for _, file := range genResult.DryRunFiles {
		if !file.WouldOverwrite {
			continue
		}
		entry, err := r.snapshotPath(file.Path)
		if err != nil {
			return err
		}
		if entry.existed {
			r.overwritten = append(r.overwritten, entry)
		}
	}
	return nil
}

func (r *checkoutGenerationRollback) snapshotPath(path string) (checkoutRollbackEntry, error) {
	entry := checkoutRollbackEntry{path: path}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return entry, nil
		}
		return entry, err
	}

	entry.existed = true
	entry.mode = info.Mode()
	entry.isSymlink = info.Mode()&os.ModeSymlink != 0
	entry.isDir = info.IsDir()

	switch {
	case entry.isSymlink:
		target, err := os.Readlink(path)
		if err != nil {
			return entry, err
		}
		entry.linkTarget = target
	case entry.isDir:
		return entry, nil
	default:
		backupPath, err := r.backupFile(path, info.Mode().Perm())
		if err != nil {
			return entry, err
		}
		entry.backupPath = backupPath
	}

	return entry, nil
}

func (r *checkoutGenerationRollback) backupFile(path string, mode os.FileMode) (string, error) {
	if r.backupDir == "" {
		dir, err := os.MkdirTemp("", "ign-checkout-rollback-*")
		if err != nil {
			return "", err
		}
		r.backupDir = dir
	}

	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.CreateTemp(r.backupDir, "file-*")
	if err != nil {
		return "", err
	}
	dstPath := dst.Name()
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}
	if err := os.Chmod(dstPath, mode); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}

	return dstPath, nil
}

func (r *checkoutGenerationRollback) rollback(genResult *generator.GenerateResult) {
	if r == nil {
		return
	}
	r.rollbackCreatedFiles(genResult)
	for i := len(r.overwritten) - 1; i >= 0; i-- {
		if err := restoreCheckoutRollbackEntry(r.overwritten[i]); err != nil {
			debug.Debug("[app] Failed to restore overwritten path %s: %v", r.overwritten[i].path, err)
		}
	}
}

func (r *checkoutGenerationRollback) rollbackCreatedFiles(genResult *generator.GenerateResult) {
	if genResult == nil {
		return
	}
	for _, path := range genResult.CreatedFiles {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			debug.Debug("[app] Failed to rollback generated file %s: %v", path, err)
		}
	}

	rollbackEmptyGeneratedDirs(genResult.CreatedFiles, r.outputDir, !r.outputDirExistedBeforeGenerate)
}

func (r *checkoutGenerationRollback) cleanup() {
	if r == nil || r.backupDir == "" {
		return
	}
	if err := os.RemoveAll(r.backupDir); err != nil {
		debug.Debug("[app] Failed to remove checkout rollback backup directory %s: %v", r.backupDir, err)
	}
}

func restoreCheckoutRollbackEntry(entry checkoutRollbackEntry) error {
	if !entry.existed {
		if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(entry.path), 0755); err != nil {
		return err
	}

	switch {
	case entry.isSymlink:
		return os.Symlink(entry.linkTarget, entry.path)
	case entry.isDir:
		if err := os.Mkdir(entry.path, entry.mode.Perm()); err != nil && !os.IsExist(err) {
			return err
		}
		return os.Chmod(entry.path, entry.mode.Perm())
	default:
		return restoreCheckoutRollbackFile(entry.backupPath, entry.path, entry.mode.Perm())
	}
}

func restoreCheckoutRollbackFile(srcPath, dstPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Chmod(dstPath, mode)
}

func rollbackEmptyGeneratedDirs(createdFiles []string, outputDir string, removeOutputDir bool) {
	outputDir = filepath.Clean(outputDir)
	for _, path := range createdFiles {
		dir := filepath.Dir(filepath.Clean(path))
		for dir != "." && dir != string(filepath.Separator) {
			if !pathWithinOrEqual(dir, outputDir) {
				break
			}
			if dir == outputDir && !removeOutputDir {
				break
			}
			if err := os.Remove(dir); err != nil {
				if !os.IsNotExist(err) && !os.IsExist(err) {
					debug.Debug("[app] Failed to rollback generated directory %s: %v", dir, err)
				}
				break
			}
			if dir == outputDir {
				break
			}
			dir = filepath.Dir(dir)
		}
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func pathWithinOrEqual(path, base string) bool {
	path = filepath.Clean(path)
	base = filepath.Clean(base)
	if path == base {
		return true
	}
	rel, err := filepath.Rel(base, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
	configDir := model.IgnConfigDir
	ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)

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

	_, vars, err := prepareVariablesForGeneration(template.Config.Variables, variables, configDir, opts.OutputDir)
	if err != nil {
		debug.Debug("[app] Failed to load variables: %v", err)
		return nil, err
	}
	debug.Debug("[app] Variables loaded successfully")

	if err := validateTemplateHash(template.Config.Hash); err != nil {
		return nil, err
	}

	// Validate required variables before mutating ign.json. A validation failure
	// should leave the existing checkout configuration unchanged.
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
	var rollback *checkoutGenerationRollback
	if !opts.DryRun {
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
			rollback.rollback(genResult)
		}
		return nil, NewCheckoutError("generation failed", err)
	}
	debug.Debug("[app] Generation completed successfully")
	debug.DebugValue("[app] Files created", genResult.FilesCreated)
	debug.DebugValue("[app] Files skipped", genResult.FilesSkipped)
	debug.DebugValue("[app] Files overwritten", genResult.FilesOverwritten)

	if !opts.DryRun {
		manifestPath := manifestPathFromConfigPath(ignConfigPath)
		manifestSnapshot, err := rollback.snapshotPath(manifestPath)
		if err != nil {
			return nil, NewCheckoutError("failed to prepare manifest rollback", err)
		}

		if err := saveManifestFromGenerateResult(manifestPath, genResult); err != nil {
			debug.Debug("[app] Failed to save ign-files.json: %v", err)
			rollback.rollback(genResult)
			if restoreErr := restoreCheckoutRollbackEntry(manifestSnapshot); restoreErr != nil {
				debug.Debug("[app] Failed to restore ign-files.json after save failure: %v", restoreErr)
			}
			return nil, NewCheckoutError("failed to save ign-files.json", err)
		}

		templateHash := template.Config.Hash

		// Load existing ign.json
		existingConfig, err := config.LoadIgnConfig(ignConfigPath)
		if err != nil {
			debug.Debug("[app] Could not load existing ign.json (will skip hash update): %v", err)
		} else {
			// Update hash in ign.json only after generation and manifest save succeed.
			existingConfig.Hash = templateHash
			debug.Debug("[app] Updating template hash in ign.json")
			if err := config.SaveIgnConfig(ignConfigPath, existingConfig); err != nil {
				debug.Debug("[app] Failed to update hash in ign.json: %v", err)
				rollback.rollback(genResult)
				if restoreErr := restoreCheckoutRollbackEntry(manifestSnapshot); restoreErr != nil {
					debug.Debug("[app] Failed to restore ign-files.json after ign.json update failure: %v", restoreErr)
				}
				return nil, NewCheckoutError("failed to update template hash in ign.json", err)
			}
			debug.Debug("[app] Template hash updated in ign.json")
		}
	}

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
