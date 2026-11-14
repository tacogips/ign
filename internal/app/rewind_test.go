package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestRewind_PreservesIgnDirectoryOnRemovalErrors(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	managedDir := filepath.Join(tempDir, "generated-dir")
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatalf("failed to create managed directory: %v", err)
	}

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}

	manifestPath := filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile)
	manifestData, err := json.Marshal(&model.IgnManifest{
		Files: []string{managedDir},
	})
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	result, err := Rewind(context.Background(), RewindOptions{
		OutputDir: tempDir,
	})
	if err == nil {
		t.Fatal("Rewind should fail when manifest contains a directory entry")
	}
	if result == nil {
		t.Fatal("Rewind should return a partial result on failure")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 rewind error, got %d", len(result.Errors))
	}

	if _, statErr := os.Stat(filepath.Join(tempDir, model.IgnConfigDir)); statErr != nil {
		t.Fatalf(".ign should be preserved after rewind failure: %v", statErr)
	}
	if _, statErr := os.Stat(managedDir); statErr != nil {
		t.Fatalf("managed directory should remain after rewind failure: %v", statErr)
	}
}

func TestRewind_RejectsManagedPathsOutsideOutputDir(t *testing.T) {
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace directory: %v", err)
	}
	t.Chdir(workspaceDir)

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}

	outsideFile := filepath.Join(tempDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("keep me"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	if err := config.SaveIgnManifest(
		filepath.Join(workspaceDir, model.IgnConfigDir, model.IgnManifestFile),
		&model.IgnManifest{Files: []string{outsideFile}},
	); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	result, err := Rewind(context.Background(), RewindOptions{
		OutputDir: ".",
	})
	if err == nil {
		t.Fatal("Rewind should fail when manifest points outside the output directory")
	}
	if result == nil {
		t.Fatal("Rewind should return a partial result on failure")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 rewind error, got %d", len(result.Errors))
	}

	if _, statErr := os.Stat(outsideFile); statErr != nil {
		t.Fatalf("outside file should be preserved: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(workspaceDir, model.IgnConfigDir)); statErr != nil {
		t.Fatalf(".ign should be preserved after rewind failure: %v", statErr)
	}
}

func TestRewind_MissingManifestSkipsDifferentPreexistingFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := filepath.Join(tempDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}
	templateConfig := `{"name":"rewind-template","version":"1.0.0","hash":"` + testHash1 + `","variables":{}}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(templateConfig), 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("template content\n"), 0644); err != nil {
		t.Fatalf("failed to write template README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("user content\n"), 0644); err != nil {
		t.Fatalf("failed to write user README: %v", err)
	}

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}
	if err := config.SaveIgnConfig(
		filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile),
		&model.IgnConfig{Template: model.TemplateSource{URL: templateDir, Ref: "main"}, Hash: testHash1},
	); err != nil {
		t.Fatalf("failed to save ign config: %v", err)
	}
	if err := config.SaveIgnVarJson(
		filepath.Join(model.IgnConfigDir, model.IgnVarFile),
		&model.IgnVarJson{Variables: map[string]interface{}{}},
	); err != nil {
		t.Fatalf("failed to save ign vars: %v", err)
	}

	result, err := Rewind(context.Background(), RewindOptions{OutputDir: tempDir})
	if err != nil {
		t.Fatalf("Rewind failed: %v", err)
	}
	if result.FilesRemoved != 0 {
		t.Fatalf("FilesRemoved = %d, want 0", result.FilesRemoved)
	}
	if result.FilesSkipped != 1 {
		t.Fatalf("FilesSkipped = %d, want 1", result.FilesSkipped)
	}
	content, err := os.ReadFile(filepath.Join(tempDir, "README.md"))
	if err != nil {
		t.Fatalf("failed to read README after rewind: %v", err)
	}
	if string(content) != "user content\n" {
		t.Fatalf("README content = %q, want user content", content)
	}
}
