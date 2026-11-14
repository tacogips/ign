package app

import (
	"bytes"
	"context"
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
	"github.com/tacogips/ign/internal/template/parser"
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

type symlinkTransitionBlockerKind string

const (
	symlinkTransitionBlockerLostManifestOwnership symlinkTransitionBlockerKind = "lost_manifest_ownership"
	symlinkTransitionBlockerProtectedPath         symlinkTransitionBlockerKind = "protected_path"
	symlinkTransitionBlockerUnsafePath            symlinkTransitionBlockerKind = "unsafe_path"
	symlinkTransitionBlockerUnreadablePath        symlinkTransitionBlockerKind = "unreadable_path"
	symlinkTransitionBlockerUnsupportedNode       symlinkTransitionBlockerKind = "unsupported_node"
)

type symlinkTransitionBlockers struct {
	Kind  symlinkTransitionBlockerKind
	Paths []string
}

// UpdateSymlinkTransitionDiagnostic is returned when a project appears to be in
// the issue-45 partial state but content-equivalence recovery is not provable.
type UpdateSymlinkTransitionDiagnostic struct {
	Path          string
	Target        string
	BlockingPaths []string
	RecoverySteps []string
}

func (d UpdateSymlinkTransitionDiagnostic) Error() string {
	steps := d.RecoverySteps
	if len(steps) == 0 {
		steps = []string{
			fmt.Sprintf("mv %s %s.backup", d.Path, d.Path),
			"ign update --overwrite-all --yes",
			fmt.Sprintf("After verifying the generated %s symlink and backup contents, delete the backup.", d.Path),
		}
	}
	message := fmt.Sprintf("issue-45 partial-state recovery required for %s -> %s: ign cannot prove the existing directory is template-owned or content-equivalent", d.Path, d.Target)
	if len(d.BlockingPaths) > 0 {
		message += fmt.Sprintf("; blocking paths: %s", strings.Join(d.BlockingPaths, ", "))
	}
	message += "; --overwrite-all and --force cannot remove unproven directory contents. Remove or move the stale directory, then rerun update: " + strings.Join(steps, "; ")
	return message
}

// afterOwnedDirectoryTree is a test seam for the narrow classifier interval
// between manifest ownership inspection and source fingerprint capture.
// Production leaves it as a no-op; the renamed-tree ownership check is the
// mutation boundary that rejects any intervening addition.
var afterOwnedDirectoryTree = func() {}

func classifyUpdateSymlinkTransitions(outputDir, manifestPath string, template *model.Template, preview *generator.GenerateResult, mode generator.OverwriteMode) (*UpdateExecutionPlan, error) {
	return classifyUpdateSymlinkTransitionsWithVariables(context.Background(), outputDir, manifestPath, template, nil, preview, mode)
}

func classifyUpdateSymlinkTransitionsWithVariables(ctx context.Context, outputDir, manifestPath string, template *model.Template, variables parser.Variables, preview *generator.GenerateResult, mode generator.OverwriteMode) (*UpdateExecutionPlan, error) {
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
			} else {
				blockers := inspectSymlinkTransitionBlockers(path, canonicalOutputDir, managed, mode, patterns)
				if blockers.Kind == symlinkTransitionBlockerLostManifestOwnership {
					equivalent, blocking, err := proveRenderedTemplateTreeEquivalent(ctx, canonicalOutputDir, path, file.SymlinkTarget, template, variables)
					if err != nil {
						return nil, err
					}
					if equivalent {
						sourceFingerprint, err := fingerprintTransitionSource(canonicalOutputDir, path)
						if err == nil {
							transition = generator.SymlinkTransition{
								Disposition:       generator.SymlinkTransitionEligible,
								SourceFingerprint: sourceFingerprint,
								Target:            file.SymlinkTarget,
								RecoveryReason:    generator.SymlinkTransitionRecoveredByContentEquivalence,
							}
						}
					} else {
						transition.Diagnostic = UpdateSymlinkTransitionDiagnostic{
							Path:          path,
							Target:        file.SymlinkTarget,
							BlockingPaths: capDiagnosticPaths(append(blockers.Paths, blocking...), 8),
						}
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

// collectUpdateSymlinkTransitionDiagnostics reports every managed directory
// whose symlink transition could not be proven safe. An unproven transition
// preserves the directory and its contents but must not abort the update:
// aborting would leave a project unable to apply any template change merely
// because one directory holds a file ign cannot attribute. Callers surface
// these and exit non-zero so a preserved transition is never mistaken for a
// completed one.
func collectUpdateSymlinkTransitionDiagnostics(plan *UpdateExecutionPlan) ([]string, []error) {
	if plan == nil {
		return nil, nil
	}
	var paths []string
	var diagnostics []error
	for _, path := range sortedTransitionPaths(plan.transitions) {
		if diagnostic := plan.transitions[path].Diagnostic; diagnostic != nil {
			paths = append(paths, path)
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return paths, diagnostics
}

func sortedTransitionPaths(transitions map[string]generator.SymlinkTransition) []string {
	paths := make([]string, 0, len(transitions))
	for path := range transitions {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
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

func inspectSymlinkTransitionBlockers(root, outputDir string, managed map[string]struct{}, mode generator.OverwriteMode, patterns []string) symlinkTransitionBlockers {
	root = filepath.Clean(root)
	blockers := symlinkTransitionBlockers{Kind: symlinkTransitionBlockerLostManifestOwnership}
	var inspect func(string)
	inspect = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			blockers.Kind = symlinkTransitionBlockerUnreadablePath
			blockers.Paths = append(blockers.Paths, dir)
			return
		}
		if len(entries) == 0 {
			blockers.Paths = append(blockers.Paths, dir)
			return
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if !pathWithinOrEqual(path, root) || !pathWithinOrEqual(path, outputDir) {
				blockers.Kind = symlinkTransitionBlockerUnsafePath
				blockers.Paths = append(blockers.Paths, path)
				return
			}
			if !shouldOverwriteTransitionPath(path, outputDir, mode, patterns) {
				blockers.Kind = symlinkTransitionBlockerProtectedPath
				blockers.Paths = append(blockers.Paths, path)
				return
			}
			info, err := os.Lstat(path)
			if err != nil {
				blockers.Kind = symlinkTransitionBlockerUnreadablePath
				blockers.Paths = append(blockers.Paths, path)
				return
			}
			if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				inspect(path)
				if blockers.Kind != symlinkTransitionBlockerLostManifestOwnership {
					return
				}
				continue
			}
			if _, ok := managed[filepath.Clean(path)]; !ok {
				blockers.Paths = append(blockers.Paths, path)
			}
		}
	}
	inspect(root)
	blockers.Paths = capDiagnosticPaths(blockers.Paths, 8)
	return blockers
}

type renderedTemplateEntryKind string

const (
	renderedTemplateEntryFile    renderedTemplateEntryKind = "file"
	renderedTemplateEntrySymlink renderedTemplateEntryKind = "symlink"
)

type renderedTemplateTreeEntry struct {
	RelPath       string
	Kind          renderedTemplateEntryKind
	Content       []byte
	SymlinkTarget string
}

func proveRenderedTemplateTreeEquivalent(ctx context.Context, outputDir string, staleRoot string, symlinkTarget string, template *model.Template, variables parser.Variables) (bool, []string, error) {
	if template == nil {
		return false, []string{staleRoot}, nil
	}
	expected, err := renderedTemplateTreeEntries(ctx, template, variables, symlinkTarget)
	if err != nil {
		return false, nil, err
	}
	actual, err := existingTreeEntries(staleRoot, outputDir)
	if err != nil {
		return false, []string{staleRoot}, nil
	}
	if len(expected) == 0 || len(actual) == 0 {
		return false, []string{staleRoot}, nil
	}
	blocking := compareRenderedTemplateEntries(expected, actual)
	return len(blocking) == 0, blocking, nil
}

func renderedTemplateTreeEntries(ctx context.Context, template *model.Template, variables parser.Variables, targetRoot string) (map[string]renderedTemplateTreeEntry, error) {
	entries := make(map[string]renderedTemplateTreeEntry)
	if variables == nil {
		variables = parser.NewMapVariables(nil)
	}
	targetRoot = filepath.ToSlash(filepath.Clean(targetRoot))
	if targetRoot == "." || targetRoot == "/" || strings.HasPrefix(targetRoot, "..") || filepath.IsAbs(targetRoot) {
		return entries, nil
	}
	for _, file := range template.Files {
		processedPath, err := generator.ProcessFilename(ctx, file.Path, variables, parser.NewParser())
		if err != nil {
			return nil, fmt.Errorf("process template path %s for symlink recovery: %w", file.Path, err)
		}
		processedSlash := filepath.ToSlash(filepath.Clean(processedPath))
		if processedSlash == "." || strings.HasPrefix(processedSlash, "../") || filepath.IsAbs(processedSlash) {
			return nil, fmt.Errorf("template path %s escapes output tree", processedPath)
		}
		if processedSlash == targetRoot {
			continue
		}
		prefix := targetRoot + "/"
		if !strings.HasPrefix(processedSlash, prefix) {
			continue
		}
		rel := strings.TrimPrefix(processedSlash, prefix)
		if rel == "" || strings.HasPrefix(rel, "../") {
			continue
		}
		entry := renderedTemplateTreeEntry{RelPath: rel}
		if file.SymlinkTarget != "" {
			entry.Kind = renderedTemplateEntrySymlink
			entry.SymlinkTarget = file.SymlinkTarget
		} else {
			content, err := generator.RenderTemplateFileContent(ctx, template, file, variables)
			if err != nil {
				return nil, fmt.Errorf("render template path %s for symlink recovery: %w", file.Path, err)
			}
			entry.Kind = renderedTemplateEntryFile
			entry.Content = content
		}
		entries[filepath.ToSlash(filepath.Clean(rel))] = entry
	}
	return entries, nil
}

func existingTreeEntries(root, outputDir string) (map[string]renderedTemplateTreeEntry, error) {
	entries := make(map[string]renderedTemplateTreeEntry)
	root = filepath.Clean(root)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if !pathWithinOrEqual(path, root) || !pathWithinOrEqual(path, outputDir) {
			return fmt.Errorf("path %s escapes transition root", path)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("path %s escapes transition root", path)
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries[rel] = renderedTemplateTreeEntry{RelPath: rel, Kind: renderedTemplateEntrySymlink, SymlinkTarget: target}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported node %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries[rel] = renderedTemplateTreeEntry{RelPath: rel, Kind: renderedTemplateEntryFile, Content: content}
		return nil
	})
	return entries, err
}

func compareRenderedTemplateEntries(expected, actual map[string]renderedTemplateTreeEntry) []string {
	blocking := make([]string, 0)
	for path, want := range expected {
		got, ok := actual[path]
		if !ok {
			blocking = append(blocking, path)
			continue
		}
		if want.Kind != got.Kind || want.SymlinkTarget != got.SymlinkTarget || !bytes.Equal(want.Content, got.Content) {
			blocking = append(blocking, path)
		}
	}
	for path := range actual {
		if _, ok := expected[path]; !ok {
			blocking = append(blocking, path)
		}
	}
	sort.Strings(blocking)
	return capDiagnosticPaths(blocking, 8)
}

func capDiagnosticPaths(paths []string, limit int) []string {
	if limit <= 0 || len(paths) == 0 {
		return nil
	}
	cleaned := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		cleaned = append(cleaned, clean)
	}
	sort.Strings(cleaned)
	if len(cleaned) > limit {
		return cleaned[:limit]
	}
	return cleaned
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
