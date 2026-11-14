//go:build darwin || linux

package app

import (
	"fmt"
	"path/filepath"

	"github.com/tacogips/ign/internal/template/model"
)

// validateTransitionArtifactPaths permits non-default tracking directories but
// requires all three tracking files to remain siblings. The journal stores only
// descriptor-relative artifact names, so accepting split roots would restore
// metadata into the wrong location after an interrupted transition.
func validateTransitionArtifactPaths(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	manifestPath := paths[0]
	if manifestPath == "" {
		return fmt.Errorf("missing transition manifest path")
	}
	manifestDir, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return fmt.Errorf("resolve transition artifact directory: %w", err)
	}
	if filepath.Base(manifestPath) != model.IgnManifestFile {
		return fmt.Errorf("unexpected transition manifest filename")
	}
	expected := []string{"ign.json", "ign-var.json"}
	for i, path := range paths[1:] {
		if path == "" {
			return fmt.Errorf("missing transition tracking artifact path")
		}
		dir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("resolve transition tracking artifact directory: %w", err)
		}
		if dir != manifestDir || i >= len(expected) || filepath.Base(path) != expected[i] {
			return fmt.Errorf("transition tracking artifacts must be sibling ign-files.json, ign.json, and ign-var.json files")
		}
	}
	return nil
}
