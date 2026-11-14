package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// copyFixtureToTemp copies a fixture template directory to a temp directory
// and returns the path to the copied template (relative to tempDir as "./template-name").
func copyFixtureToTemp(t *testing.T, fixtureName, tempDir string) string {
	t.Helper()

	// Get the absolute path to the fixture
	fixtureDir, err := filepath.Abs(filepath.Join("../fixtures/templates", fixtureName))
	if err != nil {
		t.Fatalf("failed to get fixture path: %v", err)
	}

	// Create destination directory
	destDir := filepath.Join(tempDir, fixtureName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create destination directory: %v", err)
	}

	// Copy all files from fixture to destination
	err = filepath.Walk(fixtureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from fixture root
		relPath, err := filepath.Rel(fixtureDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})

	if err != nil {
		t.Fatalf("failed to copy fixture: %v", err)
	}

	// Return relative path from tempDir (no ".." required)
	return "./" + fixtureName
}

// contains checks if string s contains substring substr
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
