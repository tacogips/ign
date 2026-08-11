package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestCompleteUpdate_SelectiveOverwriteDoesNotCreateMissingIgnoredDescendants(t *testing.T) {
	t.Run("selective overwrite skips protected descendants", func(t *testing.T) {
		tempDir := t.TempDir()
		setupTestTemplate(t, tempDir, testHash1)
		if err := os.MkdirAll(filepath.Join(tempDir, "src"), 0755); err != nil {
			t.Fatalf("Failed to create src directory: %v", err)
		}
		existingPath := filepath.Join(tempDir, "src", "existing.go")
		if err := os.WriteFile(existingPath, []byte("old existing"), 0644); err != nil {
			t.Fatalf("Failed to write existing source: %v", err)
		}

		template := updateIgnoredDescendantsTemplate(tempDir)
		prep := updatePrepareResultForTemplate(tempDir, template)
		result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     true,
			OverwriteMode: generator.OverwriteSelective,
		})
		if err != nil {
			t.Fatalf("CompleteUpdate failed: %v", err)
		}
		if result.FilesCreated != 1 {
			t.Fatalf("FilesCreated = %d, want 1", result.FilesCreated)
		}
		if result.FilesSkipped != 5 {
			t.Fatalf("FilesSkipped = %d, want 5", result.FilesSkipped)
		}
		if got := string(mustReadAppFile(t, existingPath)); got != "old existing" {
			t.Fatalf("src/existing.go = %q, want old existing", got)
		}

		manifest, err := config.LoadIgnManifest(filepath.Join(tempDir, ".ign", model.IgnManifestFile))
		if err != nil {
			t.Fatalf("Failed to load manifest: %v", err)
		}
		readmePath := filepath.Join(tempDir, "README.md")
		if !slices.Contains(manifest.Files, readmePath) {
			t.Fatalf("manifest should contain generated README %s: %v", readmePath, manifest.Files)
		}
		for _, rel := range []string{"src/existing.go", "src/new.go", "Sources/App.swift", "Tests/AppTests.swift", "Tests/Current"} {
			abs := filepath.Join(tempDir, filepath.FromSlash(rel))
			if slices.Contains(manifest.Files, abs) {
				t.Fatalf("manifest should not contain protected path %s: %v", abs, manifest.Files)
			}
			if rel != "src/existing.go" {
				if _, err := os.Lstat(abs); !os.IsNotExist(err) {
					t.Fatalf("%s should remain absent, lstat error = %v", rel, err)
				}
			}
		}
	})

	t.Run("dry-run and overwrite-all keep expected boundaries", func(t *testing.T) {
		tempDir := t.TempDir()
		setupTestTemplate(t, tempDir, testHash1)
		template := updateIgnoredDescendantsTemplate(tempDir)
		prep := updatePrepareResultForTemplate(tempDir, template)

		dryRunResult, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     true,
			OverwriteMode: generator.OverwriteSelective,
			DryRun:        true,
		})
		if err != nil {
			t.Fatalf("CompleteUpdate dry-run failed: %v", err)
		}
		if dryRunResult.FilesSkipped != 5 {
			t.Fatalf("dry-run FilesSkipped = %d, want 5", dryRunResult.FilesSkipped)
		}
		for _, file := range dryRunResult.DryRunFiles {
			rel, err := filepath.Rel(tempDir, file.Path)
			if err != nil {
				t.Fatalf("failed to relativize %s: %v", file.Path, err)
			}
			if rel != "README.md" && !file.WouldSkip {
				t.Fatalf("%s should be marked skipped in dry-run: %#v", rel, file)
			}
		}
		if _, err := os.Lstat(filepath.Join(tempDir, "src", "new.go")); !os.IsNotExist(err) {
			t.Fatalf("dry-run should not create src/new.go, lstat error = %v", err)
		}

		overwriteAllDir := t.TempDir()
		setupTestTemplate(t, overwriteAllDir, testHash1)
		overwriteAllTemplate := updateIgnoredDescendantsTemplate(overwriteAllDir)
		overwriteAllPrep := updatePrepareResultForTemplate(overwriteAllDir, overwriteAllTemplate)
		overwriteAllResult, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
			PrepareResult: overwriteAllPrep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     overwriteAllDir,
			Overwrite:     true,
			OverwriteMode: generator.OverwriteAll,
		})
		if err != nil {
			t.Fatalf("CompleteUpdate overwrite-all failed: %v", err)
		}
		if overwriteAllResult.FilesCreated != 6 {
			t.Fatalf("overwrite-all FilesCreated = %d, want 6", overwriteAllResult.FilesCreated)
		}
		for _, rel := range []string{"src/new.go", "Sources/App.swift", "Tests/AppTests.swift", "Tests/Current"} {
			if _, err := os.Lstat(filepath.Join(overwriteAllDir, filepath.FromSlash(rel))); err != nil {
				t.Fatalf("%s should be generated with overwrite-all: %v", rel, err)
			}
		}
	})
}

func updateIgnoredDescendantsTemplate(root string) *model.Template {
	return &model.Template{
		Config: model.IgnJson{
			Name:      "test",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{Path: model.IgnOverwriteIgnoreFile, Content: []byte("src/\nSources/\nTests/\n"), Mode: 0644},
			{Path: "README.md", Content: []byte("readme"), Mode: 0644},
			{Path: "src/existing.go", Content: []byte("new existing"), Mode: 0644},
			{Path: "src/new.go", Content: []byte("new file"), Mode: 0644},
			{Path: "Sources/App.swift", Content: []byte("app"), Mode: 0644},
			{Path: "Tests/AppTests.swift", Content: []byte("tests"), Mode: 0644},
			{Path: "Tests/Current", Mode: 0777 | os.ModeSymlink, SymlinkTarget: "../Sources"},
		},
		RootPath: root,
	}
}

func updatePrepareResultForTemplate(outputDir string, template *model.Template) *PrepareUpdateResult {
	ignDir := filepath.Join(outputDir, ".ign")
	return &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(ignDir, model.IgnProjectConfigFile),
		IgnVarPath:    filepath.Join(ignDir, model.IgnVarFile),
		IgnConfig: &model.IgnConfig{
			Template: model.TemplateSource{URL: "https://github.com/test/template"},
			Hash:     testHash1,
		},
	}
}

func mustReadAppFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", path, err)
	}
	return content
}
