//go:build darwin || linux

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestCompleteUpdate_OverwriteSymlinkManifestPersistenceFailureRestoresPriorState(t *testing.T) {
	tempDir, prep, managedDir, retiredPath, _, manifestPath := setupSymlinkTransitionUpdate(t)
	ignConfigBefore, err := os.ReadFile(prep.IgnConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	ignVarsBefore, err := os.ReadFile(prep.IgnVarPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	originalSave := saveIgnManifestWithResult
	saveIgnManifestWithResult = func(string, *model.IgnManifest) (config.AtomicWriteResult, error) {
		return config.AtomicWriteResult{}, errors.New("injected manifest persistence failure")
	}
	t.Cleanup(func() { saveIgnManifestWithResult = originalSave })

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true,
	}); err == nil {
		t.Fatal("update succeeded despite manifest persistence failure")
	}

	info, err := os.Lstat(managedDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want restored directory", info, err)
	}
	if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
		t.Fatalf("retired path content = %q, %v; want restored content", content, err)
	}
	assertFileContentUnchanged(t, prep.IgnConfigPath, ignConfigBefore)
	assertFileContentUnchanged(t, prep.IgnVarPath, ignVarsBefore)
	assertFileContentUnchanged(t, manifestPath, manifestBefore)
	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(manifest.Files, managedDir) {
		t.Fatalf("manifest paths = %v; unwritten symlink was persisted", manifest.Files)
	}
}

func assertFileContentUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != string(want) {
		t.Fatalf("contents of %s changed: got %q, want %q", path, got, want)
	}
}

func TestCompleteUpdate_OverwriteSymlinkManagedDirectory(t *testing.T) {
	tempDir, prep, managedDir, retiredPath, agentsPath, manifestPath := setupSymlinkTransitionUpdate(t)
	ignConfigBefore, err := os.ReadFile(prep.IgnConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	ignVarsBefore, err := os.ReadFile(prep.IgnVarPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir,
		Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatalf("preview update: %v", err)
	}
	if preview.ExecutionPlan == nil {
		t.Fatal("preview did not return an execution plan")
	}
	assertDryRunOverwrite(t, preview, managedDir)
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory after preview = %v, %v; want unchanged directory", info, err)
	}
	if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
		t.Fatalf("retired path after preview = %q, %v; want unchanged content", content, err)
	}
	assertFileContentUnchanged(t, prep.IgnConfigPath, ignConfigBefore)
	assertFileContentUnchanged(t, prep.IgnVarPath, ignVarsBefore)
	assertFileContentUnchanged(t, manifestPath, manifestBefore)

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir,
		Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	})
	if err != nil {
		t.Fatalf("complete update: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("generation errors = %v, want none", result.Errors)
	}
	info, err := os.Lstat(managedDir)
	if err != nil {
		t.Fatalf("lstat replacement: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("managed directory mode = %v, want symlink", info.Mode())
	}
	if target, err := os.Readlink(managedDir); err != nil || target != ".agents" {
		t.Fatalf("replacement target = %q, %v; want .agents", target, err)
	}
	content, err := os.ReadFile(agentsPath)
	if err != nil || string(content) != "current settings" {
		t.Fatalf("symlink target content = %q, %v; stale cleanup followed new symlink", content, err)
	}
	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	wantManifest := []string{agentsPath, managedDir}
	if !slices.Equal(manifest.Files, wantManifest) {
		t.Fatalf("manifest paths = %v, want exactly %v", manifest.Files, wantManifest)
	}
	symlinkEntries := 0
	for _, path := range manifest.Files {
		if path == managedDir {
			symlinkEntries++
		}
	}
	if symlinkEntries != 1 {
		t.Fatalf("manifest symlink entries for %s = %d, want 1", managedDir, symlinkEntries)
	}
}

func TestCompleteUpdate_OverwriteCreatesAbsentSymlink(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)
	prep := symlinkTransitionPrepareResult(tempDir, symlinkTransitionTemplate(nil))
	path := filepath.Join(tempDir, ".claude")
	manifestPath := filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile)

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true,
	})
	if err != nil {
		t.Fatalf("complete update: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("generation errors = %v, want none", result.Errors)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("created path = %v, %v; want symlink", info, err)
	}
	if target, err := os.Readlink(path); err != nil || target != ".agents" {
		t.Fatalf("created symlink target = %q, %v; want .agents", target, err)
	}
	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if !slices.Contains(manifest.Files, path) {
		t.Fatalf("manifest paths = %v; want created symlink %s", manifest.Files, path)
	}
}

func assertDryRunOverwrite(t *testing.T, result *UpdateResult, path string) {
	t.Helper()
	for _, file := range result.DryRunFiles {
		if filepath.Clean(file.Path) != filepath.Clean(path) {
			continue
		}
		if !file.WouldOverwrite || file.WouldSkip {
			t.Fatalf("dry-run transition = %#v, want one overwrite", file)
		}
		return
	}
	t.Fatalf("dry-run files = %#v, missing transition %s", result.DryRunFiles, path)
}

func TestCompleteUpdate_OverwriteSymlinkPreservesAndReportsUnownedOrEmptyDirectory(t *testing.T) {
	for _, tc := range []struct {
		name     string
		add      func(t *testing.T, dir string)
		manifest func(retiredPath string) []string
	}{
		{
			name: "untracked descendant",
			add: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "user.txt"), []byte("keep"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			manifest: func(retiredPath string) []string { return []string{retiredPath} },
		},
		{
			name:     "empty directory",
			add:      func(*testing.T, string) {},
			manifest: func(string) []string { return nil },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			setupTestTemplate(t, tempDir, testHash1)
			managedDir := filepath.Join(tempDir, ".claude")
			if err := os.MkdirAll(managedDir, 0755); err != nil {
				t.Fatal(err)
			}
			retiredPath := filepath.Join(managedDir, "settings.json")
			if tc.name != "empty directory" {
				if err := os.WriteFile(retiredPath, []byte("old settings"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			agentsPath := filepath.Join(tempDir, ".agents", "settings.json")
			if err := os.MkdirAll(filepath.Dir(agentsPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(agentsPath, []byte("current settings"), 0644); err != nil {
				t.Fatal(err)
			}
			tc.add(t, managedDir)
			manifestPath := filepath.Join(tempDir, ".ign", model.IgnManifestFile)
			if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: tc.manifest(retiredPath)}); err != nil {
				t.Fatal(err)
			}

			prep := symlinkTransitionPrepareResult(tempDir, symlinkTransitionTemplate(nil))
			result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
				PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true,
			})
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			assertUnresolvedTransitionReported(t, result, managedDir)
			info, err := os.Lstat(managedDir)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("managed directory = %v, %v; want preserved directory", info, err)
			}
			manifest, err := config.LoadIgnManifest(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(manifest.Files, tc.manifest(retiredPath)) {
				t.Fatalf("manifest paths = %v, want %v", manifest.Files, tc.manifest(retiredPath))
			}
		})
	}
}

func TestCompleteUpdate_OverwriteSymlinkPreservesUnreadableOrUncontainedDirectory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, tempDir, managedDir, manifestPath string)
	}{
		{
			name: "unreadable directory",
			setup: func(t *testing.T, _ string, managedDir, _ string) {
				t.Helper()
				if err := os.Chmod(managedDir, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(managedDir, 0755); err != nil && !os.IsNotExist(err) {
						t.Errorf("restore directory permissions: %v", err)
					}
				})
			},
		},
		{
			name: "uncontained manifest path",
			setup: func(t *testing.T, tempDir, _, manifestPath string) {
				t.Helper()
				manifest, err := config.LoadIgnManifest(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				manifest.Files = append(manifest.Files, filepath.Join(tempDir, "..", "outside-project"))
				if err := config.SaveIgnManifest(manifestPath, manifest); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, prep, managedDir, retiredPath, _, manifestPath := setupSymlinkTransitionUpdate(t)
			tc.setup(t, tempDir, managedDir, manifestPath)

			result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
				PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true,
			})
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			if len(result.Errors) != 0 {
				t.Fatalf("generation errors = %v", result.Errors)
			}
			if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("managed directory = %v, %v; want preserved directory", info, err)
			}
			if tc.name == "unreadable directory" {
				if err := os.Chmod(managedDir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
				t.Fatalf("retired path content = %q, %v; want preserved content", content, err)
			}
			manifest, err := config.LoadIgnManifest(manifestPath)
			if err != nil || !slices.Contains(manifest.Files, retiredPath) || slices.Contains(manifest.Files, managedDir) {
				t.Fatalf("manifest = %#v, %v; want retained prior path without symlink", manifest, err)
			}
		})
	}
}

func TestCompleteUpdate_SymlinkTransitionNoOverwriteAndProtectedPreserveDirectory(t *testing.T) {
	for _, tc := range []struct {
		name          string
		overwrite     bool
		overwriteMode generator.OverwriteMode
		template      *model.Template
	}{
		{name: "no overwrite", template: symlinkTransitionTemplate(nil)},
		{
			name:          "protected",
			overwrite:     true,
			overwriteMode: generator.OverwriteSelective,
			template:      symlinkTransitionTemplate([]byte(".claude/\n")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, _, managedDir, retiredPath, _, manifestPath := setupSymlinkTransitionUpdate(t)
			prep := symlinkTransitionPrepareResult(tempDir, tc.template)
			result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
				PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir,
				Overwrite: tc.overwrite, OverwriteMode: tc.overwriteMode,
			})
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			if len(result.Errors) != 0 {
				t.Fatalf("generation errors = %v", result.Errors)
			}
			if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("managed directory = %v, %v; want preserved directory", info, err)
			}
			manifest, err := config.LoadIgnManifest(manifestPath)
			if err != nil || !slices.Contains(manifest.Files, retiredPath) || slices.Contains(manifest.Files, managedDir) {
				t.Fatalf("manifest = %#v, %v; want retained retired path", manifest, err)
			}
		})
	}
}

func TestCompleteUpdate_OverwriteSymlinkPreservedTreeReportsAndSkipsCleanup(t *testing.T) {
	tempDir, prep, managedDir, retiredPath, agentsPath, manifestPath := setupSymlinkTransitionUpdate(t)
	if err := os.WriteFile(filepath.Join(managedDir, "user.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true,
	})
	if err != nil {
		t.Fatalf("preserved update: %v", err)
	}
	assertUnresolvedTransitionReported(t, result, managedDir)
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want preserved directory", info, err)
	}
	if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
		t.Fatalf("retired path content = %q, %v; want preserved file", content, err)
	}
	if content, err := os.ReadFile(agentsPath); err != nil || string(content) != "current settings" {
		t.Fatalf("agents path content = %q, %v", content, err)
	}
	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil || !slices.Contains(manifest.Files, retiredPath) || slices.Contains(manifest.Files, managedDir) {
		t.Fatalf("manifest = %#v, %v; want retired path retained without symlink", manifest, err)
	}
}

func TestCompleteUpdate_OverwriteSymlinkPlanRejectsSourceAndManifestDivergence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, tempDir, retiredPath, manifestPath string)
	}{
		{
			name: "managed file content",
			mutate: func(t *testing.T, _ string, retiredPath, _ string) {
				t.Helper()
				if err := os.WriteFile(retiredPath, []byte("changed"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "manifest ownership",
			mutate: func(t *testing.T, tempDir, _ string, manifestPath string) {
				t.Helper()
				if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{filepath.Join(tempDir, ".claude", "settings.json"), filepath.Join(tempDir, "untracked.txt")}}); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, prep, managedDir, retiredPath, _, manifestPath := setupSymlinkTransitionUpdate(t)
			preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true})
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(t, tempDir, retiredPath, manifestPath)
			if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, ExecutionPlan: preview.ExecutionPlan}); err == nil {
				t.Fatal("update succeeded after preview source divergence")
			}
			if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("managed directory = %v, %v; mutation ran despite divergence", info, err)
			}
		})
	}
}

func TestUpdateExecutionPlanFingerprintsAbsentAndRegularSymlinkDestinations(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)
	path := filepath.Join(tempDir, ".claude")
	if err := os.WriteFile(path, []byte("before"), 0644); err != nil {
		t.Fatal(err)
	}
	prep := symlinkTransitionPrepareResult(tempDir, symlinkTransitionTemplate(nil))
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, ExecutionPlan: preview.ExecutionPlan}); err == nil {
		t.Fatal("update succeeded after regular-file destination changed")
	}
}

func TestPrepareSymlinkTransitionTransactionsRefusesSymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(external, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(external, ".claude", "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "linked-output")); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "linked-output", ".claude")
	_, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		path: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err == nil {
		t.Fatal("prepared transition through symlinked ancestor")
	}
	if content, err := os.ReadFile(sentinel); err != nil || string(content) != "keep" {
		t.Fatalf("external sentinel = %q, %v; want unchanged", content, err)
	}
}

func TestSymlinkTransitionTransactionRollbackRestoresNestedSymlinkWithoutTargetTraversal(t *testing.T) {
	root := t.TempDir()
	managedDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	retired := filepath.Join(managedDir, "settings.json")
	if err := os.WriteFile(retired, []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "keep.txt")
	if err := os.WriteFile(targetFile, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	nestedLink := filepath.Join(managedDir, "nested", "outside-link")
	if err := os.MkdirAll(filepath.Dir(nestedLink), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, nestedLink); err != nil {
		t.Fatal(err)
	}

	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatalf("prepare transition: %v", err)
	}
	info, err := os.Lstat(managedDir)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("replacement = %v, %v; want symlink", info, err)
	}
	if err := transactions.rollback(); err != nil {
		t.Fatalf("rollback transition: %v", err)
	}
	info, err = os.Lstat(managedDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("restored source = %v, %v; want directory", info, err)
	}
	if content, err := os.ReadFile(retired); err != nil || string(content) != "old settings" {
		t.Fatalf("restored content = %q, %v", content, err)
	}
	if info, err := os.Lstat(nestedLink); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("restored nested link = %v, %v; want symlink", info, err)
	}
	if target, err := os.Readlink(nestedLink); err != nil || target != targetDir {
		t.Fatalf("restored nested link target = %q, %v; want %q", target, err, targetDir)
	}
	if content, err := os.ReadFile(targetFile); err != nil || string(content) != "keep" {
		t.Fatalf("external target content = %q, %v; transaction rollback traversed nested symlink target", content, err)
	}
}

func TestRecoverSymlinkTransitionJournalRestoresPreCommitDirectory(t *testing.T) {
	root := t.TempDir()
	managedDir := filepath.Join(root, ".claude")
	retired := filepath.Join(managedDir, "settings.json")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retired, []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{retired}}); err != nil {
		t.Fatal(err)
	}

	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatalf("prepare transition: %v", err)
	}
	t.Cleanup(transactions.close)
	if err := recoverSymlinkTransitionJournal(root, manifestPath); err != nil {
		t.Fatalf("recover interrupted transition: %v", err)
	}
	info, err := os.Lstat(managedDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("recovered source = %v, %v; want directory", info, err)
	}
	if content, err := os.ReadFile(retired); err != nil || string(content) != "old settings" {
		t.Fatalf("recovered content = %q, %v; want old settings", content, err)
	}
	if _, err := os.Lstat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("recovery journal = %v; want removed", err)
	}
}

func TestRecoverSymlinkTransitionJournalFinalizesCommittedBackup(t *testing.T) {
	root := t.TempDir()
	managedDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(managedDir, "settings.json"), []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ign.json", "ign-var.json"} {
		if err := os.WriteFile(filepath.Join(root, model.IgnConfigDir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatalf("prepare transition: %v", err)
	}
	t.Cleanup(transactions.close)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{managedDir}}); err != nil {
		t.Fatal(err)
	}
	if err := transactions.markArtifactsCommitted(); err != nil {
		t.Fatalf("mark artifacts committed: %v", err)
	}
	backup := filepath.Join(root, transactions.entries[0].backup)
	if _, err := os.Lstat(backup); err != nil {
		t.Fatalf("transaction backup: %v", err)
	}
	if err := recoverSymlinkTransitionJournal(root, manifestPath); err != nil {
		t.Fatalf("finalize interrupted transition: %v", err)
	}
	info, err := os.Lstat(managedDir)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("committed replacement = %v, %v; want symlink", info, err)
	}
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		t.Fatalf("transaction backup = %v; want moved to archive", err)
	}
	archive := findTransitionArchive(t, root)
	if content, err := os.ReadFile(filepath.Join(archive, "settings.json")); err != nil || string(content) != "old settings" {
		t.Fatalf("archived backup content = %q, %v; want old settings", content, err)
	}
	if _, err := os.Lstat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("recovery journal = %v; want removed after archival", err)
	}
}

func TestRecoverSymlinkTransitionJournalRestoresArtifactsAfterManifestOnlyPersistence(t *testing.T) {
	root := t.TempDir()
	managedDir := filepath.Join(root, ".claude")
	retired := filepath.Join(managedDir, "settings.json")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retired, []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{retired}}); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatalf("prepare transition: %v", err)
	}
	t.Cleanup(transactions.close)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{managedDir}}); err != nil {
		t.Fatal(err)
	}
	if err := recoverSymlinkTransitionJournal(root, manifestPath); err != nil {
		t.Fatalf("recover manifest-only interruption: %v", err)
	}
	info, err := os.Lstat(managedDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("recovered source = %v, %v; want directory", info, err)
	}
	assertFileContentUnchanged(t, manifestPath, manifestBefore)
}

func TestCompleteUpdateDryRunRefusesPendingSymlinkTransitionRecovery(t *testing.T) {
	tempDir, prep, managedDir, _, _, manifestPath := setupSymlinkTransitionUpdate(t)
	transactions, err := prepareSymlinkTransitionTransactions(tempDir, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatalf("prepare transition: %v", err)
	}
	t.Cleanup(transactions.close)
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true,
	})
	if err == nil {
		t.Fatal("dry run succeeded with an interrupted symlink transition")
	}
	info, err := os.Lstat(managedDir)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("dry-run replacement = %v, %v; want unchanged symlink", info, err)
	}
	backup := filepath.Join(tempDir, transactions.entries[0].backup)
	if _, err := os.Lstat(backup); err != nil {
		t.Fatalf("dry-run backup = %v; want unchanged transaction backup", err)
	}
	assertFileContentUnchanged(t, manifestPath, manifestBefore)
}

func TestRecoverSymlinkTransitionJournalRejectsInvalidArtifactDocumentWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		artifacts func(sentinel string) []symlinkTransitionJournalArtifact
	}{
		{
			name: "absolute artifact name",
			artifacts: func(sentinel string) []symlinkTransitionJournalArtifact {
				return []symlinkTransitionJournalArtifact{{Name: sentinel}, {Name: "ign.json", Exists: true, Data: []byte("rollback")}, {Name: "ign-var.json", Exists: true, Data: []byte("rollback")}}
			},
		},
		{
			name: "parent relative artifact name",
			artifacts: func(_ string) []symlinkTransitionJournalArtifact {
				return []symlinkTransitionJournalArtifact{{Name: "../outside"}, {Name: "ign.json", Exists: true, Data: []byte("rollback")}, {Name: "ign-var.json", Exists: true, Data: []byte("rollback")}}
			},
		},
		{
			name: "duplicate artifact name",
			artifacts: func(_ string) []symlinkTransitionJournalArtifact {
				return []symlinkTransitionJournalArtifact{{Name: "ign.json", Exists: true, Data: []byte("rollback")}, {Name: "ign.json"}, {Name: "ign-var.json", Exists: true, Data: []byte("rollback")}}
			},
		},
		{
			name: "unknown artifact name",
			artifacts: func(_ string) []symlinkTransitionJournalArtifact {
				return []symlinkTransitionJournalArtifact{{Name: "unknown.json"}, {Name: "ign.json", Exists: true, Data: []byte("rollback")}, {Name: "ign-var.json", Exists: true, Data: []byte("rollback")}}
			},
		},
		{
			name: "missing artifact name",
			artifacts: func(_ string) []symlinkTransitionJournalArtifact {
				return []symlinkTransitionJournalArtifact{{Name: "ign.json", Exists: true, Data: []byte("rollback")}, {Name: "ign-var.json", Exists: true, Data: []byte("rollback")}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			setupTestTemplate(t, root, testHash1)
			ignConfigPath := filepath.Join(root, model.IgnConfigDir, model.IgnProjectConfigFile)
			ignConfigBefore, err := os.ReadFile(ignConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(root, "outside")
			if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			journal, err := openSymlinkTransitionJournal(root)
			if err != nil {
				t.Fatal(err)
			}
			journal.artifacts = tc.artifacts(sentinel)
			if err := journal.persist(); err != nil {
				journal.close()
				t.Fatal(err)
			}
			journal.close()

			if err := recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); err == nil {
				t.Fatal("recovered invalid artifact document")
			}
			assertFileContentUnchanged(t, ignConfigPath, ignConfigBefore)
			assertFileContentUnchanged(t, sentinel, []byte("keep"))
			if _, err := os.Lstat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); err != nil {
				t.Fatalf("recovery journal = %v; want retained for inspection", err)
			}
		})
	}
}

func TestSymlinkTransitionTransactionsRetainJournalAfterRollbackOrCommitFailure(t *testing.T) {
	for _, tc := range []struct {
		name      string
		committed bool
		run       func(*symlinkTransitionTransactions) error
	}{
		{
			name: "rollback failure",
			run: func(transactions *symlinkTransitionTransactions) error {
				_ = unix.Close(transactions.entries[0].parentFD)
				transactions.entries[0].parentFD = -1
				return transactions.rollback()
			},
		},
		{
			name:      "committed cleanup failure",
			committed: true,
			run: func(transactions *symlinkTransitionTransactions) error {
				if err := transactions.markArtifactsCommitted(); err != nil {
					return err
				}
				_ = unix.Close(transactions.entries[0].parentFD)
				transactions.entries[0].parentFD = -1
				return transactions.commit()
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			setupTestTemplate(t, root, testHash1)
			managedDir := filepath.Join(root, ".claude")
			if err := os.MkdirAll(managedDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(managedDir, "settings.json"), []byte("old"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := config.SaveIgnManifest(filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile), &model.IgnManifest{Files: []string{filepath.Join(managedDir, "settings.json")}}); err != nil {
				t.Fatal(err)
			}
			transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
				managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.run(transactions); err == nil {
				t.Fatal("transaction failure was not reported")
			}
			journalPath := filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)
			if _, err := os.Lstat(journalPath); err != nil {
				t.Fatalf("recovery journal = %v; want retained", err)
			}
			journal, err := loadSymlinkTransitionJournal(root)
			if err != nil {
				t.Fatalf("load retained recovery journal: %v", err)
			}
			if journal == nil {
				t.Fatal("retained recovery journal = nil")
			}
			if err := validateSymlinkTransitionJournal(journal); err != nil {
				journal.close()
				t.Fatalf("validate retained recovery journal: %v", err)
			}
			if journal.committed != tc.committed {
				journal.close()
				t.Fatalf("retained recovery journal committed = %t; want %t", journal.committed, tc.committed)
			}
			if len(journal.entries) != 1 || journal.entries[0].Path != ".claude" || journal.entries[0].Target != ".agents" {
				journal.close()
				t.Fatalf("retained recovery journal entries = %#v; want actionable .claude -> .agents record", journal.entries)
			}
			backup := filepath.Join(root, journal.entries[0].Backup)
			journal.close()

			if err := recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); err != nil {
				t.Fatalf("recover retained transition journal: %v", err)
			}
			info, err := os.Lstat(managedDir)
			if err != nil {
				t.Fatal(err)
			}
			if tc.committed {
				if info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("committed recovered transition = %v; want symlink", info.Mode())
				}
				target, err := os.Readlink(managedDir)
				if err != nil || target != ".agents" {
					t.Fatalf("committed recovered symlink target = %q, %v; want .agents", target, err)
				}
				if _, err := os.Lstat(backup); !os.IsNotExist(err) {
					t.Fatalf("committed recovered backup = %v; want removed", err)
				}
			} else {
				if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					t.Fatalf("uncommitted recovered transition = %v; want directory", info.Mode())
				}
				if content, err := os.ReadFile(filepath.Join(managedDir, "settings.json")); err != nil || string(content) != "old" {
					t.Fatalf("uncommitted recovered file = %q, %v; want old", content, err)
				}
			}
			if _, err := os.Lstat(journalPath); !os.IsNotExist(err) {
				t.Fatalf("recovered journal = %v; want removed", err)
			}
		})
	}
}

func TestCheckoutRollbackRestoresNestedSymlinkWithoutTargetTraversal(t *testing.T) {
	tempDir := t.TempDir()
	managedDir := filepath.Join(tempDir, ".claude")
	targetDir := filepath.Join(tempDir, "outside")
	if err := os.MkdirAll(filepath.Join(managedDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(targetDir, "keep.txt")
	if err := os.WriteFile(targetFile, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(managedDir, "nested", "outside-link")
	if err := os.Symlink(targetDir, link); err != nil {
		t.Fatal(err)
	}

	// Anchor the private archive under the test-owned tree so retained
	// backups cannot land in the process working directory.
	rollback := &checkoutGenerationRollback{outputDir: managedDir}
	entry, err := rollback.snapshotPath(managedDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(managedDir); err != nil {
		t.Fatal(err)
	}
	if err := restoreCheckoutRollbackEntry(entry); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("restored nested link = %v, %v", info, err)
	}
	if content, err := os.ReadFile(targetFile); err != nil || string(content) != "keep" {
		t.Fatalf("target content = %q, %v; rollback traversed link target", content, err)
	}
}

func setupSymlinkTransitionUpdate(t *testing.T) (string, *PrepareUpdateResult, string, string, string, string) {
	t.Helper()
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)
	managedDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	retiredPath := filepath.Join(managedDir, "settings.json")
	if err := os.WriteFile(retiredPath, []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	agentsPath := filepath.Join(tempDir, ".agents", "settings.json")
	if err := os.MkdirAll(filepath.Dir(agentsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentsPath, []byte("current settings"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(tempDir, ".ign", model.IgnManifestFile)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{retiredPath, agentsPath}}); err != nil {
		t.Fatal(err)
	}
	return tempDir, symlinkTransitionPrepareResult(tempDir, symlinkTransitionTemplate(nil)), managedDir, retiredPath, agentsPath, manifestPath
}

func symlinkTransitionTemplate(ignore []byte) *model.Template {
	files := []model.TemplateFile{
		{Path: ".agents/settings.json", Content: []byte("current settings"), Mode: 0644},
		{Path: ".claude", Mode: 0777 | os.ModeSymlink, SymlinkTarget: ".agents"},
	}
	if len(ignore) > 0 {
		files = append(files, model.TemplateFile{Path: model.IgnOverwriteIgnoreFile, Content: ignore, Mode: 0644})
	}
	return &model.Template{Config: model.IgnJson{Name: "test-template", Version: "1.0.0", Hash: testHash2, Variables: map[string]model.VarDef{}}, Files: files}
}

func symlinkTransitionPrepareResult(tempDir string, template *model.Template) *PrepareUpdateResult {
	return &PrepareUpdateResult{
		Template: template, IgnJson: &template.Config, ExistingVars: map[string]interface{}{},
		CurrentHash: testHash1, NewHash: testHash2, HashChanged: true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", model.IgnProjectConfigFile),
		IgnVarPath:    filepath.Join(tempDir, ".ign", model.IgnVarFile),
		IgnConfig:     &model.IgnConfig{Template: model.TemplateSource{URL: "https://github.com/test/template"}, Hash: testHash1},
	}
}

// assertUnresolvedTransitionReported checks that an unproven directory-to-symlink
// transition was surfaced rather than silently skipped. The update itself still
// succeeds so unrelated template changes are not held hostage by one directory.
func assertUnresolvedTransitionReported(t *testing.T, result *UpdateResult, path string) {
	t.Helper()
	if result == nil {
		t.Fatal("update returned no result")
	}
	if !slices.Contains(result.UnresolvedTransitionPaths, path) {
		t.Fatalf("UnresolvedTransitionPaths = %v, want %s", result.UnresolvedTransitionPaths, path)
	}
	for _, err := range result.Errors {
		if strings.Contains(err.Error(), "issue-45 partial-state") {
			return
		}
	}
	t.Fatalf("errors = %v, want issue-45 partial-state diagnostic", result.Errors)
}
