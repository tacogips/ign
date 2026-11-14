package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tacogips/ign/internal/build"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
	"github.com/tacogips/ign/internal/template/provider"
)

func loadProjectStateIfExists(configDir string) (*config.ProjectState, bool, error) {
	if _, err := os.Stat(filepath.Join(configDir, model.IgnProjectConfigFile)); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	state, err := config.LoadProjectState(configDir)
	if err != nil {
		return nil, true, err
	}
	return state, true, nil
}

func validateOutputRootMatches(state *config.ProjectState, outputDir string) error {
	return validateOutputRootForState(state, outputDir)
}

func allowedManagedOverwritePaths(state *config.ProjectState) map[string]struct{} {
	return managedOverwritePaths(state)
}

func normalizeOutputRoot(outputDir string) string {
	return normalizeCommandOutputRoot(outputDir)
}

// ResolveCheckoutOutputDir applies existing checkout metadata defaults for add-template checkout.
func ResolveCheckoutOutputDir(requested string, explicit bool, force bool) (string, error) {
	if requested == "" {
		requested = "."
	}
	if force {
		return requested, nil
	}
	state, exists, err := loadProjectStateIfExists(model.IgnConfigDir)
	if err != nil {
		return "", NewCheckoutError("failed to load existing .ign metadata", err)
	}
	if !exists {
		return requested, nil
	}
	if explicit {
		if err := validateOutputRootMatches(state, requested); err != nil {
			return "", err
		}
		return requested, nil
	}
	return state.OutputRoot, nil
}

func newProjectTemplateRecord(prep *PrepareCheckoutResult, variables map[string]interface{}, files []string, existing []config.ProjectTemplateState, generatedBy string) config.ProjectTemplateState {
	state := &config.ProjectState{Templates: existing}
	order := nextTemplateOrder(state)
	record := checkoutRecordFromPrepare(prep, variables, files, order)
	assignTemplateID(state, &record, prep.IgnJson.Name)
	if record.Metadata != nil && generatedBy != "" {
		record.Metadata.GeneratedBy = generatedBy
	}
	return record
}

func checkoutRecordFromPrepare(prep *PrepareCheckoutResult, variables map[string]interface{}, files []string, order int) config.ProjectTemplateState {
	metadata := &model.FileMetadata{
		GeneratedAt:     time.Now(),
		GeneratedBy:     "ign checkout",
		TemplateName:    prep.IgnJson.Name,
		TemplateVersion: prep.IgnJson.Version,
		IgnVersion:      build.Version(),
	}
	return config.ProjectTemplateState{
		ID:    nextTemplateID(nil, metadata.TemplateName, order),
		Order: order,
		Template: model.TemplateSource{
			URL:  prep.NormalizedURL,
			Path: prep.TemplateRef.Path,
			Ref:  prep.TemplateRef.Ref,
		},
		Hash:      prep.IgnJson.Hash,
		Metadata:  metadata,
		Variables: variables,
		Files:     cleanGeneratedFiles(files),
	}
}

func nextTemplateOrder(state *config.ProjectState) int {
	next := 1
	if state == nil {
		return next
	}
	for _, tmpl := range state.Templates {
		if tmpl.Order >= next {
			next = tmpl.Order + 1
		}
	}
	return next
}

func assignTemplateID(state *config.ProjectState, record *config.ProjectTemplateState, templateName string) {
	if record == nil {
		return
	}
	record.ID = nextTemplateID(state, templateName, record.Order)
}

func nextTemplateID(state *config.ProjectState, templateName string, order int) string {
	base := sanitizeTemplateIDBase(templateName)
	if base == "" {
		base = "template"
	}
	candidate := fmt.Sprintf("%s-%d", base, order)
	if state == nil {
		return candidate
	}
	used := make(map[string]struct{}, len(state.Templates))
	for _, tmpl := range state.Templates {
		used[tmpl.ID] = struct{}{}
	}
	if _, ok := used[candidate]; !ok {
		return candidate
	}
	for suffix := 2; ; suffix++ {
		candidate = fmt.Sprintf("%s-%d-%d", base, order, suffix)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
}

func sanitizeTemplateIDBase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func cleanGeneratedFiles(paths []string) []string {
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
	sort.Strings(result)
	return result
}

func generatedFilesFromResult(result *generator.GenerateResult) []string {
	if result == nil {
		return nil
	}
	if len(result.WrittenFiles) > 0 {
		return result.WrittenFiles
	}
	return result.CreatedFiles
}

func managedOverwritePaths(state *config.ProjectState) map[string]struct{} {
	if state == nil {
		return nil
	}
	allowed := make(map[string]struct{})
	for _, tmpl := range state.Templates {
		for _, path := range tmpl.Files {
			clean := filepath.Clean(path)
			if clean == "" || clean == "." {
				continue
			}
			allowed[clean] = struct{}{}
		}
	}
	return allowed
}

func validateOutputRootForState(state *config.ProjectState, outputDir string) error {
	if state == nil || state.OutputRoot == "" {
		return nil
	}
	requested := normalizeCommandOutputRoot(outputDir)
	stored := normalizeCommandOutputRoot(state.OutputRoot)
	if requested != stored {
		return NewValidationError(
			fmt.Sprintf("output path %s does not match checked-out output root %s", filepath.Clean(outputDir), stored),
			nil,
		)
	}
	return nil
}

func normalizeCommandOutputRoot(path string) string {
	if path == "" {
		return "."
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return "."
	}
	if filepath.IsAbs(clean) {
		if cwd, err := os.Getwd(); err == nil {
			if rel, relErr := filepath.Rel(cwd, clean); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				clean = filepath.Clean(rel)
			} else if realCWD, cwdErr := filepath.EvalSymlinks(cwd); cwdErr == nil {
				if realClean, cleanErr := filepath.EvalSymlinks(filepath.Dir(clean)); cleanErr == nil {
					realPath := filepath.Join(realClean, filepath.Base(clean))
					if rel, relErr := filepath.Rel(realCWD, realPath); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
						clean = filepath.Clean(rel)
					}
				}
			}
		}
	}
	return clean
}

func fetchTemplateForRecord(ctx context.Context, record config.ProjectTemplateState, token string) (*model.Template, model.TemplateRef, error) {
	normalizedURL := NormalizeTemplateURL(record.Template.URL)
	prov, err := provider.NewProviderWithToken(normalizedURL, token)
	if err != nil {
		return nil, model.TemplateRef{}, NewCheckoutError("failed to create provider", err)
	}
	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		return nil, model.TemplateRef{}, NewCheckoutError("failed to resolve template URL", err)
	}
	if record.Template.Ref != "" {
		templateRef.Ref = record.Template.Ref
	}
	if record.Template.Path != "" {
		templateRef.Path = record.Template.Path
	}
	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		return nil, model.TemplateRef{}, NewTemplateFetchError("failed to fetch template", err)
	}
	return template, templateRef, nil
}

func generateTemplateRecord(ctx context.Context, record config.ProjectTemplateState, template *model.Template, outputDir string, overwrite bool, overwriteMode generator.OverwriteMode, verbose bool, skipUnchanged bool, allowed map[string]struct{}, dryRun bool) (*generator.GenerateResult, map[string]interface{}, parser.Variables, error) {
	rawVars, vars, err := prepareVariablesForGeneration(template.Config.Variables, record.Variables, model.IgnConfigDir, outputDir)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := ValidateVariables(&template.Config, vars); err != nil {
		return nil, nil, nil, err
	}
	gen := generator.NewGenerator()
	genOpts := generator.GenerateOptions{
		Template:              template,
		Variables:             vars,
		OutputDir:             outputDir,
		Overwrite:             overwrite,
		OverwriteMode:         overwriteMode,
		Verbose:               verbose,
		SkipUnchanged:         skipUnchanged,
		AllowedOverwritePaths: allowed,
	}
	debug.Debug("[app] Generating template record %s", record.ID)
	var genResult *generator.GenerateResult
	if dryRun {
		genResult, err = gen.DryRun(ctx, genOpts)
	} else {
		genResult, err = gen.Generate(ctx, genOpts)
	}
	if err != nil {
		return nil, nil, nil, NewCheckoutError("generation failed", err)
	}
	return genResult, rawVars, vars, nil
}

func removeGeneratedFiles(ctx context.Context, files []string, outputDir string) (*RewindResult, error) {
	files = cleanGeneratedFiles(files)
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
		cleanPath, err := validateManagedPath(path, outputDir)
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
		for _, dir := range removeEmptyParentDirs(cleanPath, outputDir) {
			removedDirs[dir] = struct{}{}
		}
	}
	result.DirectoriesRemoved = len(removedDirs)
	if len(result.Errors) > 0 {
		return result, NewCheckoutError("failed to remove some ign-managed files; .ign was preserved", fmt.Errorf("%v", result.Errors))
	}
	return result, nil
}
