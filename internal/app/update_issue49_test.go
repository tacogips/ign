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

func TestCompleteUpdate_OverwritePrunesMissingIgnoredFilesStillInTemplate(t *testing.T) {
	tempDir := t.TempDir()
	ignDir := filepath.Join(tempDir, model.IgnConfigDir)
	if err := os.MkdirAll(ignDir, 0o755); err != nil {
		t.Fatalf("create .ign directory: %v", err)
	}

	ignoredDir := filepath.Join(tempDir, "src", "kestra-playground")
	if err := os.MkdirAll(ignoredDir, 0o755); err != nil {
		t.Fatalf("create ignored source directory: %v", err)
	}

	missingNames := []string{"__init__.py", "__main__.py", "cli.py", "py.typed"}
	manifestFiles := make([]string, 0, len(missingNames)+1)
	templateFiles := []model.TemplateFile{
		{Path: model.IgnOverwriteIgnoreFile, Content: []byte("src/kestra-playground/\n"), Mode: 0o644},
		{Path: "README.md", Content: []byte("current readme"), Mode: 0o644},
	}
	for _, name := range missingNames {
		path := filepath.Join(ignoredDir, name)
		manifestFiles = append(manifestFiles, path)
		templateFiles = append(templateFiles, model.TemplateFile{
			Path:    filepath.ToSlash(filepath.Join("src", "kestra-playground", name)),
			Content: []byte("template content"),
			Mode:    0o644,
		})
	}

	existingPath := filepath.Join(ignoredDir, "existing.py")
	const existingContent = "project-owned content"
	if err := os.WriteFile(existingPath, []byte(existingContent), 0o644); err != nil {
		t.Fatalf("write existing ignored file: %v", err)
	}
	manifestFiles = append(manifestFiles, existingPath)
	templateFiles = append(templateFiles, model.TemplateFile{
		Path:    "src/kestra-playground/existing.py",
		Content: []byte("template replacement"),
		Mode:    0o644,
	})

	manifestPath := filepath.Join(ignDir, model.IgnManifestFile)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: manifestFiles}); err != nil {
		t.Fatalf("save pre-fix manifest: %v", err)
	}

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		},
		Files:    templateFiles,
		RootPath: tempDir,
	}
	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
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
	if dryRunResult.FilesDeleted != len(missingNames) {
		t.Fatalf("dry-run FilesDeleted = %d, want %d", dryRunResult.FilesDeleted, len(missingNames))
	}
	manifestAfterDryRun, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest after dry-run: %v", err)
	}
	if len(manifestAfterDryRun.Files) != len(manifestFiles) {
		t.Fatalf("dry-run changed manifest entries: got %v, want %v", manifestAfterDryRun.Files, manifestFiles)
	}
	for _, path := range manifestFiles {
		if !slices.Contains(manifestAfterDryRun.Files, path) {
			t.Fatalf("dry-run removed manifest entry %s: %v", path, manifestAfterDryRun.Files)
		}
	}

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
	if result.FilesDeleted != len(missingNames) {
		t.Fatalf("FilesDeleted = %d, want %d", result.FilesDeleted, len(missingNames))
	}

	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatalf("load reconciled manifest: %v", err)
	}
	for _, name := range missingNames {
		path := filepath.Join(ignoredDir, name)
		if slices.Contains(manifest.Files, path) {
			t.Errorf("manifest retained missing ignored path %s: %v", path, manifest.Files)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("missing ignored path %s was recreated, lstat error = %v", path, err)
		}
	}
	if !slices.Contains(manifest.Files, existingPath) {
		t.Errorf("manifest dropped existing ignored path %s: %v", existingPath, manifest.Files)
	}
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("read existing ignored file: %v", err)
	}
	if string(content) != existingContent {
		t.Fatalf("existing ignored file content = %q, want %q", content, existingContent)
	}
}
