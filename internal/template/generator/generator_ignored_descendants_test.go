package generator

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

func TestGenerator_GenerateWithSelectiveOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	if err := os.MkdirAll(filepath.Join(tmpDir, "config"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("old readme"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config", "local.yaml"), []byte("old local"), 0644); err != nil {
		t.Fatalf("failed to create local config: %v", err)
	}

	template := &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
		},
		Files: []model.TemplateFile{
			{
				Path:     model.IgnOverwriteIgnoreFile,
				Content:  []byte("config/\n"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "nested/" + model.IgnOverwriteIgnoreFile,
				Content:  []byte("nested metadata-like file"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "README.md",
				Content:  []byte("new readme"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "config/local.yaml",
				Content:  []byte("new local"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "config/default.yaml",
				Content:  []byte("new default"),
				Mode:     0644,
				IsBinary: false,
			},
		},
		RootPath: tmpDir,
	}

	result, err := gen.Generate(ctx, GenerateOptions{
		Template:      template,
		Variables:     parser.NewMapVariables(map[string]interface{}{}),
		OutputDir:     tmpDir,
		OverwriteMode: OverwriteSelective,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.FilesOverwritten != 1 {
		t.Fatalf("FilesOverwritten = %d, want 1", result.FilesOverwritten)
	}
	if result.FilesSkipped != 2 {
		t.Fatalf("FilesSkipped = %d, want 2", result.FilesSkipped)
	}
	if result.FilesCreated != 1 {
		t.Fatalf("FilesCreated = %d, want 1", result.FilesCreated)
	}

	readme, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}
	if string(readme) != "new readme" {
		t.Fatalf("README.md = %q, want new readme", string(readme))
	}

	localConfig, err := os.ReadFile(filepath.Join(tmpDir, "config", "local.yaml"))
	if err != nil {
		t.Fatalf("failed to read local config: %v", err)
	}
	if string(localConfig) != "old local" {
		t.Fatalf("config/local.yaml = %q, want old local", string(localConfig))
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "config", "default.yaml")); !os.IsNotExist(err) {
		t.Fatalf("config/default.yaml should not be generated, stat error = %v", err)
	}

	nestedOverwriteIgnore, err := os.ReadFile(filepath.Join(tmpDir, "nested", model.IgnOverwriteIgnoreFile))
	if err != nil {
		t.Fatalf("failed to read nested %s: %v", model.IgnOverwriteIgnoreFile, err)
	}
	if string(nestedOverwriteIgnore) != "nested metadata-like file" {
		t.Fatalf("nested %s = %q, want nested metadata-like file", model.IgnOverwriteIgnoreFile, string(nestedOverwriteIgnore))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, model.IgnOverwriteIgnoreFile)); !os.IsNotExist(err) {
		t.Fatalf("%s should not be generated", model.IgnOverwriteIgnoreFile)
	}
}

func TestGenerateSelectiveOverwriteSkipsMissingIgnoredDescendants(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	existingPath := filepath.Join(tmpDir, "src", "existing.go")
	if err := os.WriteFile(existingPath, []byte("old existing"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	template := ignoredDescendantsTemplate(tmpDir)
	result, err := NewGenerator().Generate(context.Background(), GenerateOptions{
		Template:      template,
		Variables:     parser.NewMapVariables(map[string]interface{}{}),
		OutputDir:     tmpDir,
		OverwriteMode: OverwriteSelective,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.FilesCreated != 1 {
		t.Fatalf("FilesCreated = %d, want 1", result.FilesCreated)
	}
	if result.FilesSkipped != 5 {
		t.Fatalf("FilesSkipped = %d, want 5", result.FilesSkipped)
	}
	if got := string(mustReadGeneratorFile(t, existingPath)); got != "old existing" {
		t.Fatalf("src/existing.go = %q, want old existing", got)
	}

	readmePath := filepath.Join(tmpDir, "README.md")
	if !slices.Contains(result.CreatedFiles, readmePath) || !slices.Contains(result.WrittenFiles, readmePath) {
		t.Fatalf("README.md should be the only tracked write, created=%v written=%v", result.CreatedFiles, result.WrittenFiles)
	}
	for _, rel := range []string{"src/new.go", "Sources/App.swift", "Tests/AppTests.swift", "Tests/Current"} {
		abs := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if slices.Contains(result.CreatedFiles, abs) || slices.Contains(result.WrittenFiles, abs) {
			t.Fatalf("%s should not be tracked as created or written, created=%v written=%v", rel, result.CreatedFiles, result.WrittenFiles)
		}
		if _, err := os.Lstat(abs); !os.IsNotExist(err) {
			t.Fatalf("%s should remain absent, lstat error = %v", rel, err)
		}
	}
}

func TestDryRunSelectiveOverwriteSkipsMissingIgnoredDescendants(t *testing.T) {
	tmpDir := t.TempDir()
	template := ignoredDescendantsTemplate(tmpDir)

	result, err := NewGenerator().DryRun(context.Background(), GenerateOptions{
		Template:      template,
		Variables:     parser.NewMapVariables(map[string]interface{}{}),
		OutputDir:     tmpDir,
		OverwriteMode: OverwriteSelective,
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	if result.FilesCreated != 1 {
		t.Fatalf("FilesCreated = %d, want 1", result.FilesCreated)
	}
	if result.FilesSkipped != 5 {
		t.Fatalf("FilesSkipped = %d, want 5", result.FilesSkipped)
	}

	skipped := map[string]bool{}
	for _, file := range result.DryRunFiles {
		rel, err := filepath.Rel(tmpDir, file.Path)
		if err != nil {
			t.Fatalf("failed to relativize %s: %v", file.Path, err)
		}
		if file.WouldSkip {
			skipped[filepath.ToSlash(rel)] = true
		}
	}
	for _, rel := range []string{"src/existing.go", "src/new.go", "Sources/App.swift", "Tests/AppTests.swift", "Tests/Current"} {
		if !skipped[rel] {
			t.Fatalf("%s should be reported as skipped in dry-run files: %#v", rel, result.DryRunFiles)
		}
	}
	for _, dir := range result.Directories {
		rel, err := filepath.Rel(tmpDir, dir)
		if err != nil {
			t.Fatalf("failed to relativize dry-run dir %s: %v", dir, err)
		}
		rel = filepath.ToSlash(rel)
		if rel == "src" || rel == "Sources" || rel == "Tests" {
			t.Fatalf("dry-run directories should not include protected directory %s: %v", rel, result.Directories)
		}
	}
}

func TestGenerateOverwriteAllCreatesIgnoredDescendants(t *testing.T) {
	tmpDir := t.TempDir()
	template := ignoredDescendantsTemplate(tmpDir)

	result, err := NewGenerator().Generate(context.Background(), GenerateOptions{
		Template:      template,
		Variables:     parser.NewMapVariables(map[string]interface{}{}),
		OutputDir:     tmpDir,
		OverwriteMode: OverwriteAll,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.FilesCreated != 6 {
		t.Fatalf("FilesCreated = %d, want 6", result.FilesCreated)
	}
	for _, rel := range []string{"src/new.go", "Sources/App.swift", "Tests/AppTests.swift", "Tests/Current"} {
		if _, err := os.Lstat(filepath.Join(tmpDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s should be generated with overwrite-all: %v", rel, err)
		}
	}
}

func ignoredDescendantsTemplate(root string) *model.Template {
	return &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
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

func mustReadGeneratorFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return content
}
