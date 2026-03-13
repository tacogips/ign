package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
