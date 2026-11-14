package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/provider"
)

// RewindOptions contains options for removing files previously created by ign.
type RewindOptions struct {
	// OutputDir is used only as a fallback when no manifest exists yet.
	OutputDir string
	// GitHubToken is used when manifest fallback needs to fetch the template.
	GitHubToken string
}

// RewindResult contains the result of removing generated files.
type RewindResult struct {
	FilesRemoved       int
	FilesMissing       int
	DirectoriesRemoved int
	Errors             []error
	Files              []string
}

// Rewind removes files previously created by ign and then deletes .ign.
func Rewind(ctx context.Context, opts RewindOptions) (*RewindResult, error) {
	debug.DebugSection("[app] Rewind workflow start")
	debug.DebugValue("[app] OutputDir", opts.OutputDir)

	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return nil, NewValidationError("invalid output directory", err)
	}

	if _, err := os.Stat(model.IgnConfigDir); os.IsNotExist(err) {
		return nil, NewValidationError(
			"rewind requires prior checkout: .ign directory not found.\n"+
				"Run 'ign checkout <template-url>' first.",
			nil,
		)
	}

	files, err := loadManagedFilesForRewind(ctx, opts)
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		iDepth := managedPathDepth(files[i])
		jDepth := managedPathDepth(files[j])
		if iDepth != jDepth {
			return iDepth > jDepth
		}
		return files[i] > files[j]
	})

	result := &RewindResult{
		Errors: []error{},
		Files:  append([]string(nil), files...),
	}

	removedDirs := make(map[string]struct{})
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		cleanPath, err := validateManagedPath(path, opts.OutputDir)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		info, statErr := os.Lstat(cleanPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				result.FilesMissing++
				continue
			}
			result.Errors = append(result.Errors, fmt.Errorf("failed to stat %s: %w", cleanPath, statErr))
			continue
		}

		if info.IsDir() {
			result.Errors = append(result.Errors, fmt.Errorf("managed path %s is a directory; refusing to remove", cleanPath))
			continue
		}

		if err := os.Remove(cleanPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove %s: %w", cleanPath, err))
			continue
		}
		result.FilesRemoved++

		for _, dir := range removeEmptyParentDirs(cleanPath, opts.OutputDir) {
			removedDirs[dir] = struct{}{}
		}
	}

	result.DirectoriesRemoved = len(removedDirs)

	if len(result.Errors) > 0 {
		debug.Debug("[app] Rewind completed with %d errors; preserving .ign for retry", len(result.Errors))
		return result, NewCheckoutError(
			"failed to remove some ign-managed files; .ign was preserved",
			errors.Join(result.Errors...),
		)
	}

	if err := os.RemoveAll(model.IgnConfigDir); err != nil {
		return result, NewCheckoutError("failed to remove .ign directory", err)
	}

	debug.Debug("[app] Rewind workflow completed successfully")
	return result, nil
}

func loadManagedFilesForRewind(ctx context.Context, opts RewindOptions) ([]string, error) {
	manifest, err := config.LoadIgnManifest(manifestPath())
	if err == nil {
		return dedupePaths(manifest.Files), nil
	}

	if cfgErr, ok := err.(*config.ConfigError); !ok || cfgErr.Type != config.ConfigNotFound {
		return nil, NewCheckoutError("failed to load ign-files.json", err)
	}

	debug.Debug("[app] ign-files.json not found; falling back to current template dry-run")
	return buildManagedFilesFromCurrentTemplate(ctx, opts)
}

func buildManagedFilesFromCurrentTemplate(ctx context.Context, opts RewindOptions) ([]string, error) {
	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)

	ignConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		return nil, NewCheckoutError("failed to load .ign/ign.json: run 'ign checkout <template-url>' first", err)
	}

	ignVar, err := config.LoadIgnVarJson(ignVarPath)
	if err != nil {
		return nil, NewCheckoutError("failed to load .ign/ign-var.json: run 'ign checkout <template-url>' first", err)
	}

	normalizedURL := NormalizeTemplateURL(ignConfig.Template.URL)
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		return nil, NewCheckoutError("failed to create provider", err)
	}

	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}
	if ignConfig.Template.Ref != "" {
		templateRef.Ref = ignConfig.Template.Ref
	}
	if ignConfig.Template.Path != "" {
		templateRef.Path = ignConfig.Template.Path
	}

	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}

	_, vars, err := prepareVariablesForGeneration(template.Config.Variables, ignVar.Variables, model.IgnConfigDir, opts.OutputDir)
	if err != nil {
		return nil, err
	}

	gen := generator.NewGenerator()
	genResult, err := gen.DryRun(ctx, generator.GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: opts.OutputDir,
		Overwrite: true,
	})
	if err != nil {
		return nil, NewCheckoutError("failed to enumerate generated files", err)
	}

	return dedupePaths(genResult.Files), nil
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if clean == "" || clean == "." {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func validateManagedPath(path string, outputDir string) (string, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return "", fmt.Errorf("managed path is empty")
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output directory %s: %w", outputDir, err)
	}

	absPath := clean
	if !filepath.IsAbs(clean) {
		absPath, err = filepath.Abs(clean)
		if err != nil {
			return "", fmt.Errorf("failed to resolve managed path %s: %w", clean, err)
		}
	}

	outputRel, err := filepath.Rel(absOutputDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to compare managed path %s against output directory %s: %w", clean, outputDir, err)
	}
	if outputRel == ".." || strings.HasPrefix(outputRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("managed path %s is outside output directory %s; refusing to remove", clean, filepath.Clean(outputDir))
	}

	absIgnDir, err := filepath.Abs(model.IgnConfigDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s directory: %w", model.IgnConfigDir, err)
	}

	ignRel, err := filepath.Rel(absIgnDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to compare managed path %s against %s: %w", clean, model.IgnConfigDir, err)
	}
	if ignRel == "." || !(ignRel == ".." || strings.HasPrefix(ignRel, ".."+string(filepath.Separator))) {
		return "", fmt.Errorf("managed path %s points into .ign; refusing to remove", clean)
	}

	return absPath, nil
}

func removeEmptyParentDirs(path string, outputDir string) []string {
	removed := []string{}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return removed
	}

	dir := filepath.Dir(path)
	for dir != "." && dir != string(filepath.Separator) {
		relToOutputDir, relErr := filepath.Rel(absOutputDir, dir)
		if relErr != nil || relToOutputDir == "." || relToOutputDir == ".." || strings.HasPrefix(relToOutputDir, ".."+string(filepath.Separator)) {
			break
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			break
		}
		removed = append(removed, dir)
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return removed
}

func managedPathDepth(path string) int {
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return 0
	}

	depth := 0
	for _, c := range clean {
		if c == filepath.Separator {
			depth++
		}
	}
	return depth
}
