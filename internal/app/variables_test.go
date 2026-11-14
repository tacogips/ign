package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadVariablesFromMap(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("nil map returns error", func(t *testing.T) {
		_, err := LoadVariablesFromMap(nil, tmpDir)
		if err == nil {
			t.Error("Expected error for nil map")
		}
	})

	t.Run("empty map returns valid Variables", func(t *testing.T) {
		vars, err := LoadVariablesFromMap(map[string]interface{}{}, tmpDir)
		if err != nil {
			t.Errorf("Empty map should not error: %v", err)
		}
		if vars == nil {
			t.Error("Expected non-nil Variables")
		}
	})

	t.Run("direct values without @file: prefix", func(t *testing.T) {
		input := map[string]interface{}{
			"name":    "test-project",
			"version": "1.0.0",
			"port":    8080,
			"enabled": true,
		}

		vars, err := LoadVariablesFromMap(input, tmpDir)
		if err != nil {
			t.Errorf("Direct values should not error: %v", err)
		}

		// Verify all values are preserved
		if val, ok := vars.Get("name"); !ok || val != "test-project" {
			t.Errorf("Expected name=test-project, got %v", val)
		}
		if val, ok := vars.Get("version"); !ok || val != "1.0.0" {
			t.Errorf("Expected version=1.0.0, got %v", val)
		}
		if val, ok := vars.Get("port"); !ok || val != 8080 {
			t.Errorf("Expected port=8080, got %v", val)
		}
		if val, ok := vars.Get("enabled"); !ok || val != true {
			t.Errorf("Expected enabled=true, got %v", val)
		}
	})

	t.Run("@file: prefix loads file content", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "file content from disk"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		input := map[string]interface{}{
			"secret": "@file:test.txt",
		}

		vars, err := LoadVariablesFromMap(input, tmpDir)
		if err != nil {
			t.Errorf("@file: reference should work: %v", err)
		}

		if val, ok := vars.Get("secret"); !ok || val != content {
			t.Errorf("Expected secret=%s, got %v", content, val)
		}
	})

	t.Run("@file: with whitespace around filename", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "whitespace.txt")
		content := "whitespace test"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		input := map[string]interface{}{
			"data": "@file:  whitespace.txt  ",
		}

		vars, err := LoadVariablesFromMap(input, tmpDir)
		if err != nil {
			t.Errorf("@file: with whitespace should work: %v", err)
		}

		if val, ok := vars.Get("data"); !ok || val != content {
			t.Errorf("Expected data=%s, got %v", content, val)
		}
	})

	t.Run("@file: without filename returns error", func(t *testing.T) {
		input := map[string]interface{}{
			"empty": "@file:",
		}

		_, err := LoadVariablesFromMap(input, tmpDir)
		if err == nil {
			t.Error("Expected error for @file: without filename")
		}
	})

	t.Run("@file: with path traversal (..) returns error", func(t *testing.T) {
		input := map[string]interface{}{
			"malicious": "@file:../../../etc/passwd",
		}

		_, err := LoadVariablesFromMap(input, tmpDir)
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}
		if !strings.Contains(err.Error(), "..") {
			t.Errorf("Error should mention path traversal: %v", err)
		}
	})

	t.Run("@file: with absolute path returns error", func(t *testing.T) {
		input := map[string]interface{}{
			"absolute": "@file:/etc/passwd",
		}

		_, err := LoadVariablesFromMap(input, tmpDir)
		if err == nil {
			t.Error("Expected error for absolute path")
		}
	})

	t.Run("@file: non-existent file returns error", func(t *testing.T) {
		input := map[string]interface{}{
			"missing": "@file:nonexistent.txt",
		}

		_, err := LoadVariablesFromMap(input, tmpDir)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("mixed @file: and direct values", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "secret.key")
		secretContent := "secret-api-key-123"
		if err := os.WriteFile(testFile, []byte(secretContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		input := map[string]interface{}{
			"name":    "my-project",
			"api_key": "@file:secret.key",
			"port":    3000,
		}

		vars, err := LoadVariablesFromMap(input, tmpDir)
		if err != nil {
			t.Errorf("Mixed values should work: %v", err)
		}

		// Verify direct value
		if val, ok := vars.Get("name"); !ok || val != "my-project" {
			t.Errorf("Expected name=my-project, got %v", val)
		}

		// Verify file-loaded value
		if val, ok := vars.Get("api_key"); !ok || val != secretContent {
			t.Errorf("Expected api_key=%s, got %v", secretContent, val)
		}

		// Verify integer value
		if val, ok := vars.Get("port"); !ok || val != 3000 {
			t.Errorf("Expected port=3000, got %v", val)
		}
	})
}
