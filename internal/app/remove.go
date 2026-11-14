package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/provider"
)

// RemoveOptions contains options for removing one or more checked-out templates.
type RemoveOptions struct {
	// TemplateIDs selects template records to remove.
	TemplateIDs []string
	// All removes every checked-out template and deletes .ign.
	All bool
	// OutputDir must match the stored output root.
	OutputDir string
	// GitHubToken is used while replaying remaining templates.
	GitHubToken string
	// Verbose enables detailed logging during replay.
	Verbose bool
}

// RemoveResult contains the result of selective template removal.
type RemoveResult struct {
	TemplatesRemoved   int
	FilesRemoved       int
	FilesMissing       int
	DirectoriesRemoved int
	FilesReplayed      int
	FilesOverwritten   int
	RemainingTemplates []string
	RemovedTemplateIDs []string
	Errors             []error
}

// Remove removes selected template records and replays remaining templates.
func Remove(ctx context.Context, opts RemoveOptions) (*RemoveResult, error) {
	debug.DebugSection("[app] Remove workflow start")
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if err := ValidateOutputDir(opts.OutputDir); err != nil {
		return nil, NewValidationError("invalid output directory", err)
	}

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		return nil, NewCheckoutError("failed to load .ign metadata", err)
	}
	if err := validateOutputRootMatches(state, opts.OutputDir); err != nil {
		return nil, err
	}

	removeIDs, err := resolveRemoveIDs(state, opts)
	if err != nil {
		return nil, err
	}

	removeSet := make(map[string]struct{}, len(removeIDs))
	for _, id := range removeIDs {
		removeSet[id] = struct{}{}
	}

	removed, remaining := partitionTemplates(state.Templates, removeSet)
	result := &RemoveResult{
		TemplatesRemoved:   len(removed),
		RemovedTemplateIDs: removeIDs,
		Errors:             []error{},
	}

	filesToRemove := []string{}
	for _, tmpl := range removed {
		filesToRemove = append(filesToRemove, tmpl.Files...)
	}
	removedFiles, missingFiles, removedDirs, removeErrs := removeManagedFiles(ctx, dedupePaths(filesToRemove), opts.OutputDir)
	result.FilesRemoved = removedFiles
	result.FilesMissing = missingFiles
	result.DirectoriesRemoved = removedDirs
	result.Errors = append(result.Errors, removeErrs...)
	if len(result.Errors) > 0 {
		return result, NewCheckoutError("failed to remove selected template files; metadata was preserved", errors.Join(result.Errors...))
	}

	if len(remaining) == 0 {
		if err := os.RemoveAll(model.IgnConfigDir); err != nil {
			return result, NewCheckoutError("failed to remove .ign directory", err)
		}
		return result, nil
	}

	allowedReplayOverwrites := managedOverwritePaths(&config.ProjectState{Templates: remaining})
	for _, tmpl := range remaining {
		genResult, err := replayTemplate(ctx, tmpl, opts, allowedReplayOverwrites)
		if err != nil {
			return result, err
		}
		result.FilesReplayed += genResult.FilesCreated
		result.FilesOverwritten += genResult.FilesOverwritten
		result.RemainingTemplates = append(result.RemainingTemplates, tmpl.ID)
	}

	state.Templates = remaining
	state.Version = model.IgnProjectMetadataVersionV2
	if err := config.SaveProjectState(model.IgnConfigDir, state); err != nil {
		return result, NewCheckoutError("failed to save .ign metadata after removal", err)
	}

	debug.Debug("[app] Remove workflow completed successfully")
	return result, nil
}

func resolveRemoveIDs(state *config.ProjectState, opts RemoveOptions) ([]string, error) {
	if opts.All {
		ids := make([]string, 0, len(state.Templates))
		for _, tmpl := range state.Templates {
			ids = append(ids, tmpl.ID)
		}
		return ids, nil
	}
	if len(opts.TemplateIDs) == 0 {
		return nil, NewValidationError("remove requires at least one template id or --all", nil)
	}
	known := make(map[string]struct{}, len(state.Templates))
	for _, tmpl := range state.Templates {
		known[tmpl.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(opts.TemplateIDs))
	for _, id := range opts.TemplateIDs {
		if _, ok := known[id]; !ok {
			return nil, NewValidationError(fmt.Sprintf("unknown template id %q", id), nil)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func partitionTemplates(templates []config.ProjectTemplateState, removeSet map[string]struct{}) ([]config.ProjectTemplateState, []config.ProjectTemplateState) {
	removed := []config.ProjectTemplateState{}
	remaining := []config.ProjectTemplateState{}
	for _, tmpl := range templates {
		if _, ok := removeSet[tmpl.ID]; ok {
			removed = append(removed, tmpl)
		} else {
			remaining = append(remaining, tmpl)
		}
	}
	sort.SliceStable(remaining, func(i, j int) bool {
		return remaining[i].Order < remaining[j].Order
	})
	return removed, remaining
}

func removeManagedFiles(ctx context.Context, files []string, outputDir string) (int, int, int, []error) {
	sort.Slice(files, func(i, j int) bool {
		iDepth := managedPathDepth(files[i])
		jDepth := managedPathDepth(files[j])
		if iDepth != jDepth {
			return iDepth > jDepth
		}
		return files[i] > files[j]
	})
	removed := 0
	missing := 0
	removedDirs := map[string]struct{}{}
	errs := []error{}
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		cleanPath, err := validateManagedPath(path, outputDir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		info, statErr := os.Lstat(cleanPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				missing++
				continue
			}
			errs = append(errs, fmt.Errorf("failed to stat %s: %w", cleanPath, statErr))
			continue
		}
		if info.IsDir() {
			errs = append(errs, fmt.Errorf("managed path %s is a directory; refusing to remove", cleanPath))
			continue
		}
		if err := os.Remove(cleanPath); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", cleanPath, err))
			continue
		}
		removed++
		for _, dir := range removeEmptyParentDirs(cleanPath, outputDir) {
			removedDirs[dir] = struct{}{}
		}
	}
	return removed, missing, len(removedDirs), errs
}

func replayTemplate(ctx context.Context, tmpl config.ProjectTemplateState, opts RemoveOptions, allowedOverwritePaths map[string]struct{}) (*generator.GenerateResult, error) {
	normalizedURL := NormalizeTemplateURL(tmpl.Template.URL)
	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		return nil, NewCheckoutError("failed to create provider", err)
	}
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}
	if tmpl.Template.Ref != "" {
		templateRef.Ref = tmpl.Template.Ref
	}
	if tmpl.Template.Path != "" {
		templateRef.Path = tmpl.Template.Path
	}
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}

	_, vars, err := prepareVariablesForGeneration(template.Config.Variables, tmpl.Variables, model.IgnConfigDir, opts.OutputDir)
	if err != nil {
		return nil, err
	}
	if err := ValidateVariables(&template.Config, vars); err != nil {
		return nil, err
	}
	gen := generator.NewGenerator()
	result, err := gen.Generate(ctx, generator.GenerateOptions{
		Template:              template,
		Variables:             vars,
		OutputDir:             opts.OutputDir,
		Overwrite:             false,
		OverwriteMode:         generator.OverwriteNone,
		Verbose:               opts.Verbose,
		SkipUnchanged:         true,
		AllowedOverwritePaths: allowedOverwritePaths,
	})
	if err != nil {
		return nil, NewCheckoutError("failed to replay remaining template "+tmpl.ID, err)
	}
	return result, nil
}
