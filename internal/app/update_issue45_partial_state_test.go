package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestCompleteUpdateRecoversIssue45PartialStateByContentEquivalence(t *testing.T) {
	root, prep, staleDir, _, agentsPath, manifestPath := setupIssue45PartialState(t, "current settings")

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
		Overwrite:     true,
		OverwriteMode: generator.OverwriteAll,
	})
	if err != nil {
		t.Fatalf("CompleteUpdate returned error: %v", err)
	}
	if result.FilesOverwritten == 0 {
		t.Fatalf("FilesOverwritten = %d, want recovered symlink overwrite", result.FilesOverwritten)
	}
	assertIssue45RecoveredSymlink(t, staleDir)

	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if containsPath(manifest.Files, filepath.Join(staleDir, "settings.json")) {
		t.Fatalf("manifest retained stale descendant: %v", manifest.Files)
	}
	if !containsPath(manifest.Files, staleDir) || !containsPath(manifest.Files, agentsPath) {
		t.Fatalf("manifest = %v, want replacement symlink and target file", manifest.Files)
	}
}

func TestCompleteUpdateRecoversIssue45PartialStateWithForceMode(t *testing.T) {
	root, prep, staleDir, _, _, _ := setupIssue45PartialState(t, "current settings")

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
		OverwriteMode: generator.OverwriteAll,
	}); err != nil {
		t.Fatalf("CompleteUpdate returned error: %v", err)
	}

	assertIssue45RecoveredSymlink(t, staleDir)
}

func TestCompleteUpdateDryRunReportsIssue45RecoveryWithoutMutation(t *testing.T) {
	root, prep, staleDir, staleFile, _, _ := setupIssue45PartialState(t, "current settings")

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
		Overwrite:     true,
		OverwriteMode: generator.OverwriteAll,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("CompleteUpdate dry run returned error: %v", err)
	}
	if result.ExecutionPlan == nil {
		t.Fatal("dry-run did not return execution plan")
	}
	if result.FilesOverwritten == 0 {
		t.Fatalf("FilesOverwritten = %d, want recovery preview", result.FilesOverwritten)
	}
	info, err := os.Lstat(staleDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("stale dir after dry-run = %v, %v; want unchanged directory", info, err)
	}
	content, err := os.ReadFile(staleFile)
	if err != nil || string(content) != "current settings" {
		t.Fatalf("stale file after dry-run = %q, %v; want unchanged", content, err)
	}
}

func TestCompleteUpdateDiagnosesDivergentIssue45PartialState(t *testing.T) {
	root, prep, staleDir, staleFile, _, _ := setupIssue45PartialState(t, "local changes")

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
		Overwrite:     true,
		OverwriteMode: generator.OverwriteAll,
		DryRun:        true,
	})
	// A divergent partial state is reported, not fatal: the rest of the update
	// still applies and the caller exits non-zero on the reported diagnostic.
	if err != nil {
		t.Fatalf("CompleteUpdate returned error: %v", err)
	}
	if !slices.Contains(result.UnresolvedTransitionPaths, staleDir) {
		t.Fatalf("UnresolvedTransitionPaths = %v, want %s", result.UnresolvedTransitionPaths, staleDir)
	}
	errText := joinErrorText(result.Errors)
	for _, want := range []string{"issue-45 partial-state", ".claude", ".agents", "--overwrite-all and --force cannot remove unproven"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("diagnostics = %q, want substring %q", errText, want)
		}
	}
	if strings.Contains(errText, "file exists, use --overwrite or --force") {
		t.Fatalf("diagnostics included generic skip text: %q", errText)
	}
	info, statErr := os.Lstat(staleDir)
	if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("stale dir after diagnostic = %v, %v; want preserved directory", info, statErr)
	}
	content, readErr := os.ReadFile(staleFile)
	if readErr != nil || string(content) != "local changes" {
		t.Fatalf("stale file after diagnostic = %q, %v; want preserved", content, readErr)
	}
}

func setupIssue45PartialState(t *testing.T, staleContent string) (root string, prep *PrepareUpdateResult, staleDir string, staleFile string, agentsPath string, manifestPath string) {
	t.Helper()
	root = t.TempDir()
	setupTestTemplate(t, root, testHash1)
	staleDir = filepath.Join(root, ".claude")
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatal(err)
	}
	staleFile = filepath.Join(staleDir, "settings.json")
	if err := os.WriteFile(staleFile, []byte(staleContent), 0644); err != nil {
		t.Fatal(err)
	}
	agentsPath = filepath.Join(root, ".agents", "settings.json")
	if err := os.MkdirAll(filepath.Dir(agentsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentsPath, []byte("current settings"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath = filepath.Join(root, ".ign", model.IgnManifestFile)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{agentsPath}}); err != nil {
		t.Fatal(err)
	}
	return root, symlinkTransitionPrepareResult(root, symlinkTransitionTemplate(nil)), staleDir, staleFile, agentsPath, manifestPath
}

func assertIssue45RecoveredSymlink(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s mode = %v, want symlink", path, info.Mode())
	}
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if target != ".agents" {
		t.Fatalf("symlink target = %q, want .agents", target)
	}
}

func containsPath(paths []string, want string) bool {
	want = filepath.Clean(want)
	for _, path := range paths {
		if filepath.Clean(path) == want {
			return true
		}
	}
	return false
}

// joinErrorText flattens reported diagnostics for substring assertions.
func joinErrorText(errs []error) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "\n")
}
