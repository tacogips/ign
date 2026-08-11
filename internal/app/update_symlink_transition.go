package app

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

// UpdateExecutionPlan binds preview and mutation to the same validated
// directory-to-symlink decisions. Its fields are deliberately private so a
// caller can only reuse, not modify, the plan returned by CompleteUpdate.
type UpdateExecutionPlan struct {
	outputDir     string
	manifestPath  string
	templateHash  string
	overwriteMode generator.OverwriteMode
	transitions   map[string]generator.SymlinkTransition
	sourcePaths   []string
	fingerprint   string
}

// afterOwnedDirectoryTree is a test seam for the narrow classifier interval
// between manifest ownership inspection and source fingerprint capture.
// Production leaves it as a no-op; the renamed-tree ownership check is the
// mutation boundary that rejects any intervening addition.
var afterOwnedDirectoryTree = func() {}

func classifyUpdateSymlinkTransitions(outputDir, manifestPath string, template *model.Template, preview *generator.GenerateResult, mode generator.OverwriteMode) (*UpdateExecutionPlan, error) {
	canonicalOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve update output directory: %w", err)
	}
	canonicalManifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve update manifest path: %w", err)
	}
	plan := &UpdateExecutionPlan{
		outputDir: filepath.Clean(canonicalOutputDir), manifestPath: filepath.Clean(canonicalManifestPath),
		templateHash: template.Config.Hash, overwriteMode: mode,
		transitions: map[string]generator.SymlinkTransition{},
	}
	if mode == generator.OverwriteNone || preview == nil {
		return plan, nil
	}
	manifest, err := loadManifestOrEmpty(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load manifest for symlink transitions: %w", err)
	}
	managed, managedPathsValid := managedCanonicalSet(manifest.Files, outputDir)
	patterns := overwriteIgnorePatternsFromTemplate(template)
	for _, file := range preview.DryRunFiles {
		if file.SymlinkTarget == "" {
			continue
		}
		path, err := canonicalManagedPathForComparison(file.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve symlink destination %s: %w", file.Path, err)
		}
		if !file.Exists {
			// A missing destination is an ordinary generation path, not a
			// directory-to-symlink transition. Keep it in the fingerprint so a
			// confirmation still rejects source-state divergence.
			plan.sourcePaths = append(plan.sourcePaths, path)
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			// Existing regular files and symlinks retain the generator's normal
			// overwrite behavior. They are not ownership-validated directory
			// transitions.
			plan.sourcePaths = append(plan.sourcePaths, path)
			continue
		}
		// Do not inspect, fingerprint, or later mutate a directory candidate
		// reached through an ancestor symlink. Preserve it so generation cannot
		// turn a refused replacement into a misleading metadata write.
		if !transitionPathIsSafe(canonicalOutputDir, path) {
			plan.transitions[path] = generator.SymlinkTransition{Disposition: generator.SymlinkTransitionPreserved, Target: file.SymlinkTarget}
			continue
		}
		plan.sourcePaths = append(plan.sourcePaths, path)
		transition := generator.SymlinkTransition{Disposition: generator.SymlinkTransitionPreserved, Target: file.SymlinkTarget}
		if managedPathsValid && shouldOverwriteTransitionPath(path, canonicalOutputDir, mode, patterns) {
			if retired, ok := ownedDirectoryTree(path, canonicalOutputDir, managed, mode, patterns); ok {
				afterOwnedDirectoryTree()
				sourceFingerprint, err := fingerprintTransitionSource(canonicalOutputDir, path)
				if err == nil {
					retired = managedPathsAtOrBelow(managed, path)
					transition = generator.SymlinkTransition{
						Disposition:         generator.SymlinkTransitionEligible,
						RetiredManagedPaths: retired,
						SourceFingerprint:   sourceFingerprint,
						Target:              file.SymlinkTarget,
					}
				}
			}
		}
		plan.transitions[path] = transition
	}
	sort.Strings(plan.sourcePaths)
	plan.sourcePaths = compactSortedPaths(plan.sourcePaths)
	fingerprint, err := updatePlanFingerprint(plan.sourcePaths, plan.manifestPath, canonicalOutputDir)
	if err != nil {
		return nil, err
	}
	plan.fingerprint = fingerprint
	return plan, nil
}

func managedPathsAtOrBelow(managed map[string]struct{}, root string) []string {
	retired := make([]string, 0)
	for path := range managed {
		if pathWithinOrEqual(path, root) {
			retired = append(retired, path)
		}
	}
	sort.Strings(retired)
	return retired
}

func managedCanonicalSet(paths []string, outputDir string) (map[string]struct{}, bool) {
	managed := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		canonical, err := validateManagedPath(path, outputDir)
		if err != nil {
			// An invalid manifest entry cannot prove that a directory tree is
			// owned. Preserve every candidate transition rather than failing an
			// otherwise safe overwrite update.
			return managed, false
		}
		managed[filepath.Clean(canonical)] = struct{}{}
	}
	return managed, true
}

func shouldOverwriteTransitionPath(path, outputDir string, mode generator.OverwriteMode, patterns []string) bool {
	if mode == generator.OverwriteAll {
		return true
	}
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !generator.MatchesGitIgnorePattern(filepath.ToSlash(rel), patterns)
}

// ownedDirectoryTree proves that every terminal entry is manifest-managed and
// every directory has managed content. os.ReadDir and Lstat keep symlinks as
// terminal nodes instead of following their targets.
func ownedDirectoryTree(root, outputDir string, managed map[string]struct{}, mode generator.OverwriteMode, patterns []string) ([]string, bool) {
	root = filepath.Clean(root)
	retired := make([]string, 0)
	var inspect func(string) bool
	inspect = func(dir string) bool {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			return false
		}
		hasManagedContent := false
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if !pathWithinOrEqual(path, root) || !pathWithinOrEqual(path, outputDir) || !shouldOverwriteTransitionPath(path, outputDir, mode, patterns) {
				return false
			}
			info, err := os.Lstat(path)
			if err != nil {
				return false
			}
			if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				if !inspect(path) {
					return false
				}
				hasManagedContent = true
				continue
			}
			canonical := filepath.Clean(path)
			if _, ok := managed[canonical]; !ok {
				return false
			}
			retired = append(retired, canonical)
			hasManagedContent = true
		}
		return hasManagedContent
	}
	if !inspect(root) {
		return nil, false
	}
	sort.Strings(retired)
	return retired, true
}

func updatePlanFingerprint(sourcePaths []string, manifestPath, outputDir string) (string, error) {
	h := sha256.New()
	for _, path := range sourcePaths {
		if err := writeFingerprintField(h, "path", []byte(path)); err != nil {
			return "", err
		}
		if err := fingerprintNode(h, path); err != nil {
			return "", err
		}
	}
	manifest, err := loadManifestOrEmpty(manifestPath)
	if err != nil {
		return "", fmt.Errorf("load manifest for execution plan fingerprint: %w", err)
	}
	manifestPaths := append([]string(nil), manifest.Files...)
	sort.Strings(manifestPaths)
	if err := writeFingerprintField(h, "manifest", nil); err != nil {
		return "", err
	}
	for _, path := range manifestPaths {
		if err := writeFingerprintField(h, "manifest-path", []byte(path)); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeFingerprintField writes a typed, length-prefixed value. Delimiters are
// insufficient here because managed file content may contain arbitrary bytes.
func writeFingerprintField(w io.Writer, kind string, value []byte) error {
	for _, field := range [][]byte{[]byte(kind), value} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		if _, err := w.Write(length[:]); err != nil {
			return err
		}
		if _, err := w.Write(field); err != nil {
			return err
		}
	}
	return nil
}

func fingerprintNode(h io.Writer, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return writeFingerprintField(h, "node", []byte("absent"))
		}
		return writeFingerprintUnavailableState(h, "lstat")
	}
	if err := writeFingerprintField(h, "mode", []byte(info.Mode().String())); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return writeFingerprintUnavailableState(h, "readlink")
		}
		return writeFingerprintField(h, "symlink-target", []byte(target))
	}
	if !info.IsDir() {
		file, err := os.Open(path)
		if err != nil {
			return writeFingerprintUnavailableState(h, "open")
		}
		defer file.Close()
		contents, err := io.ReadAll(file)
		if err != nil {
			return writeFingerprintUnavailableState(h, "read")
		}
		return writeFingerprintField(h, "file-content", contents)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return writeFingerprintUnavailableState(h, "readdir")
	}
	for _, entry := range entries {
		if err := writeFingerprintField(h, "directory-entry", []byte(entry.Name())); err != nil {
			return err
		}
		if err := fingerprintNode(h, filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return writeFingerprintField(h, "directory-end", nil)
}

func writeFingerprintUnavailableState(h io.Writer, operation string) error {
	return writeFingerprintField(h, "unavailable", []byte(operation))
}

func (p *UpdateExecutionPlan) validate(outputDir, manifestPath, templateHash string, mode generator.OverwriteMode) error {
	if p == nil {
		return nil
	}
	canonicalOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve update output directory: %w", err)
	}
	canonicalManifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return fmt.Errorf("resolve update manifest path: %w", err)
	}
	if p.outputDir != filepath.Clean(canonicalOutputDir) || p.manifestPath != filepath.Clean(canonicalManifestPath) || p.templateHash != templateHash || p.overwriteMode != mode {
		return fmt.Errorf("update execution plan does not match this update invocation")
	}
	fingerprint, err := updatePlanFingerprint(p.sourcePaths, p.manifestPath, p.outputDir)
	if err != nil {
		return fmt.Errorf("validate update transition source state: %w", err)
	}
	if fingerprint != p.fingerprint {
		return fmt.Errorf("update transition source state changed after preview")
	}
	return nil
}

func compactSortedPaths(paths []string) []string {
	if len(paths) < 2 {
		return paths
	}
	result := paths[:1]
	for _, path := range paths[1:] {
		if path != result[len(result)-1] {
			result = append(result, path)
		}
	}
	return result
}
