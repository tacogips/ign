package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestRewind_RemovesManagedFilesAndPreservesUserFiles(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	prep, err := app.PrepareCheckout(context.Background(), app.PrepareCheckoutOptions{
		URL:          templatePath,
		Ref:          "main",
		ConfigExists: false,
	})
	if err != nil {
		t.Fatalf("PrepareCheckout failed: %v", err)
	}

	_, err = app.CompleteCheckout(context.Background(), app.CompleteCheckoutOptions{
		PrepareResult: prep,
		Variables: map[string]interface{}{
			"project_name":   "rewind-test",
			"port":           "8080",
			"enable_feature": "true",
		},
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("CompleteCheckout failed: %v", err)
	}

	manifestPath := filepath.Join(tempDir, ".ign", model.IgnManifestFile)
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read ign-files.json: %v", err)
	}

	var manifest model.IgnManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("failed to parse ign-files.json: %v", err)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("ign-files.json should record created files")
	}

	userFile := filepath.Join(outputDir, "notes.txt")
	if err := os.WriteFile(userFile, []byte("keep me"), 0644); err != nil {
		t.Fatalf("failed to create user file: %v", err)
	}

	result, err := app.Rewind(context.Background(), app.RewindOptions{
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("Rewind failed: %v", err)
	}
	if result.FilesRemoved == 0 {
		t.Fatal("Rewind should remove at least one generated file")
	}

	for _, path := range manifest.Files {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("managed file still exists after rewind: %s", path)
		}
	}

	if _, err := os.Stat(userFile); err != nil {
		t.Fatalf("user file should be preserved: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".ign")); !os.IsNotExist(err) {
		t.Fatal(".ign should be removed by rewind")
	}
}

func TestRewind_RemovesFilesOverwrittenDuringCheckout(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output directory: %v", err)
	}

	overwrittenFile := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(overwrittenFile, []byte("preexisting content"), 0644); err != nil {
		t.Fatalf("failed to create preexisting file: %v", err)
	}

	prep, err := app.PrepareCheckout(context.Background(), app.PrepareCheckoutOptions{
		URL:          templatePath,
		Ref:          "main",
		ConfigExists: false,
	})
	if err != nil {
		t.Fatalf("PrepareCheckout failed: %v", err)
	}

	_, err = app.CompleteCheckout(context.Background(), app.CompleteCheckoutOptions{
		PrepareResult: prep,
		Variables: map[string]interface{}{
			"project_name":   "rewind-overwrite-test",
			"port":           "8080",
			"enable_feature": "true",
		},
		OutputDir: outputDir,
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("CompleteCheckout failed: %v", err)
	}

	manifest, err := config.LoadIgnManifest(filepath.Join(tempDir, ".ign", model.IgnManifestFile))
	if err != nil {
		t.Fatalf("failed to load ign-files.json: %v", err)
	}

	foundOverwrittenFile := false
	for _, path := range manifest.Files {
		if path == overwrittenFile {
			foundOverwrittenFile = true
			break
		}
	}
	if !foundOverwrittenFile {
		t.Fatalf("ign-files.json should record overwritten file %s", overwrittenFile)
	}

	if _, err := app.Rewind(context.Background(), app.RewindOptions{
		OutputDir: outputDir,
	}); err != nil {
		t.Fatalf("Rewind failed: %v", err)
	}

	if _, err := os.Stat(overwrittenFile); !os.IsNotExist(err) {
		t.Fatalf("overwritten file should be removed by rewind: %v", err)
	}
}
