package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

type cleanupRemovedManagedFilesOptions struct {
	ManifestPath       string
	OutputDir          string
	Template           *model.Template
	GenerateResult     *generator.GenerateResult
	OverwriteMode      generator.OverwriteMode
	Overwrite          bool
	DryRun             bool
	SymlinkTransitions map[string]generator.SymlinkTransition
}

type cleanupRemovedManagedFilesResult struct {
	FilesDeleted          int
	DeletedFiles          []string
	RemovedCanonicalPaths map[string]struct{}
}

func cleanupRemovedManagedFilesForUpdate(ctx context.Context, opts cleanupRemovedManagedFilesOptions) (*cleanupRemovedManagedFilesResult, error) {
	result := &cleanupRemovedManagedFilesResult{
		RemovedCanonicalPaths: map[string]struct{}{},
	}
	overwriteMode := effectiveUpdateOverwriteMode(opts.OverwriteMode, opts.Overwrite)
	if overwriteMode == generator.OverwriteNone || opts.GenerateResult == nil {
		return result, nil
	}

	manifest, err := loadManifestOrEmpty(opts.ManifestPath)
	if err != nil {
		return result, err
	}
	if len(manifest.Files) == 0 {
		return result, nil
	}

	currentFiles, err := canonicalGeneratedFileSet(opts.GenerateResult.Files)
	if err != nil {
		return result, err
	}
	overwriteIgnorePatterns := overwriteIgnorePatternsFromTemplate(opts.Template)
	absOutputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return result, fmt.Errorf("failed to resolve output directory %s: %w", opts.OutputDir, err)
	}

	var cleanupErrors []error
	for _, manifestFile := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		cleanPath, err := validateManagedPath(manifestFile, opts.OutputDir)
		if err != nil {
			if hasPreservedTransition(opts.SymlinkTransitions) {
				// A transition with incomplete manifest ownership must preserve the
				// existing tree. Leave invalid manifest records untouched rather
				// than turning that safety decision into an update failure.
				continue
			}
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		canonicalPath := filepath.Clean(cleanPath)
		if isPreservedTransitionPath(canonicalPath, opts.SymlinkTransitions) {
			// A preserved transition deliberately leaves the former managed tree in
			// place. Retain every prior manifest entry beneath it; in particular, do
			// not prune an absent descendant or inspect it through a later path.
			continue
		}
		if isRetiredTransitionPath(canonicalPath, opts.SymlinkTransitions) {
			// The transition removed this path as part of its validated directory
			// replacement. Never inspect it through the new symlink target.
			result.RemovedCanonicalPaths[canonicalPath] = struct{}{}
			continue
		}
		if _, exists := currentFiles[canonicalPath]; exists {
			continue
		}

		relPath, err := filepath.Rel(absOutputDir, canonicalPath)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to compare removed managed path %s against output directory %s: %w", canonicalPath, opts.OutputDir, err))
			continue
		}

		shouldRemove := shouldRemoveManagedPathDuringUpdate(relPath, overwriteMode, overwriteIgnorePatterns)
		if !shouldRemove {
			if _, statErr := os.Lstat(canonicalPath); os.IsNotExist(statErr) {
				recordRemovedManagedPath(result, opts.OutputDir, relPath, canonicalPath)
			}
			continue
		}

		info, statErr := os.Lstat(canonicalPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				recordRemovedManagedPath(result, opts.OutputDir, relPath, canonicalPath)
				continue
			}
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to stat removed managed path %s: %w", canonicalPath, statErr))
			continue
		}
		if info.IsDir() {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("managed path %s is a directory; refusing to remove", canonicalPath))
			continue
		}

		if opts.DryRun {
			recordRemovedManagedPath(result, opts.OutputDir, relPath, canonicalPath)
			continue
		}

		debug.Debug("[app] Removing managed file no longer present in template: %s", canonicalPath)
		if err := os.Remove(canonicalPath); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to remove managed file no longer present in template %s: %w", canonicalPath, err))
			continue
		}
		recordRemovedManagedPath(result, opts.OutputDir, relPath, canonicalPath)
		removeEmptyParentDirs(canonicalPath, opts.OutputDir)
	}

	sort.Strings(result.DeletedFiles)
	if len(cleanupErrors) > 0 {
		return result, errors.Join(cleanupErrors...)
	}
	return result, nil
}

func recordRemovedManagedPath(result *cleanupRemovedManagedFilesResult, outputDir, relPath, canonicalPath string) {
	result.FilesDeleted++
	result.DeletedFiles = append(result.DeletedFiles, outputPathForManagedRelativePath(outputDir, relPath))
	result.RemovedCanonicalPaths[canonicalPath] = struct{}{}
}

func hasPreservedTransition(transitions map[string]generator.SymlinkTransition) bool {
	for _, transition := range transitions {
		if transition.Disposition == generator.SymlinkTransitionPreserved {
			return true
		}
	}
	return false
}

func isPreservedTransitionPath(path string, transitions map[string]generator.SymlinkTransition) bool {
	for root, transition := range transitions {
		if transition.Disposition != generator.SymlinkTransitionPreserved {
			continue
		}
		if pathWithinOrEqual(path, root) {
			return true
		}
	}
	return false
}

func isRetiredTransitionPath(path string, transitions map[string]generator.SymlinkTransition) bool {
	for _, transition := range transitions {
		if transition.Disposition != generator.SymlinkTransitionEligible {
			continue
		}
		for _, retired := range transition.RetiredManagedPaths {
			if filepath.Clean(retired) == filepath.Clean(path) {
				return true
			}
		}
	}
	return false
}

func outputPathForManagedRelativePath(outputDir string, relPath string) string {
	if outputDir == "" {
		outputDir = "."
	}
	return filepath.Clean(filepath.Join(outputDir, relPath))
}

func effectiveUpdateOverwriteMode(mode generator.OverwriteMode, overwrite bool) generator.OverwriteMode {
	if mode != "" {
		return mode
	}
	if overwrite {
		return generator.OverwriteAll
	}
	return generator.OverwriteNone
}

func canonicalGeneratedFileSet(paths []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		canonical, err := canonicalManagedPathForComparison(path)
		if err != nil {
			return nil, err
		}
		result[canonical] = struct{}{}
	}
	return result, nil
}

func canonicalManagedPathForComparison(path string) (string, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return "", fmt.Errorf("managed path is empty")
	}
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("failed to resolve managed path %s: %w", clean, err)
	}
	return filepath.Clean(absPath), nil
}

func overwriteIgnorePatternsFromTemplate(template *model.Template) []string {
	if template == nil {
		return nil
	}
	for _, file := range template.Files {
		if filepath.ToSlash(file.Path) == model.IgnOverwriteIgnoreFile {
			return generator.ParseIgnoreFilePatterns(file.Content)
		}
	}
	return nil
}

func shouldRemoveManagedPathDuringUpdate(path string, mode generator.OverwriteMode, overwriteIgnorePatterns []string) bool {
	switch mode {
	case generator.OverwriteAll:
		return true
	case generator.OverwriteSelective:
		return !generator.MatchesGitIgnorePattern(path, overwriteIgnorePatterns)
	default:
		return false
	}
}
