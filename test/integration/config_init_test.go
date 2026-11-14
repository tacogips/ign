package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/template/model"
)

// TestInit_LocalProvider tests init with local filesystem provider
func TestInit_LocalProvider(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")

	// Copy fixture to temp directory
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	// Change to temp directory for relative path resolution
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Execute init
	err = app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify .ign directory created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("config directory not created: %s", configDir)
	}

	// Verify ign.json created
	ignConfigPath := filepath.Join(configDir, "ign.json")
	if _, err := os.Stat(ignConfigPath); os.IsNotExist(err) {
		t.Errorf("ign.json not created: %s", ignConfigPath)
	}

	// Read and verify ign.json content
	ignConfigData, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read ign.json: %v", err)
	}

	var ignConfig model.IgnConfig
	if err := json.Unmarshal(ignConfigData, &ignConfig); err != nil {
		t.Fatalf("failed to parse ign.json: %v", err)
	}

	// Verify template source in ign.json
	if ignConfig.Template.URL == "" {
		t.Errorf("template URL is empty in ign.json")
	}

	// Verify hash is present
	if ignConfig.Hash == "" {
		t.Errorf("template hash is empty in ign.json")
	}

	// Verify ign-var.json created
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	if _, err := os.Stat(ignVarPath); os.IsNotExist(err) {
		t.Errorf("ign-var.json not created: %s", ignVarPath)
	}

	// Read and verify ign-var.json content
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Verify variables initialized
	if ignVar.Variables == nil {
		t.Errorf("variables map is nil")
	}

	// Verify expected variables exist
	expectedVars := []string{"project_name", "port", "enable_feature"}
	for _, varName := range expectedVars {
		if _, ok := ignVar.Variables[varName]; !ok {
			t.Errorf("variable %s not found in ign-var.json", varName)
		}
	}

	// Verify metadata
	if ignVar.Metadata == nil {
		t.Errorf("metadata is nil")
	} else {
		if ignVar.Metadata.TemplateName != "simple-template" {
			t.Errorf("template name = %s, want simple-template", ignVar.Metadata.TemplateName)
		}
		if ignVar.Metadata.TemplateVersion != "1.0.0" {
			t.Errorf("template version = %s, want 1.0.0", ignVar.Metadata.TemplateVersion)
		}
	}
}

// TestInit_WithAbsolutePath tests that absolute paths now work for local provider
func TestInit_WithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Copy fixture to temp directory first
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	// Get absolute path
	absPath, err := filepath.Abs(filepath.Join(tempDir, templatePath))
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Absolute paths should now work
	err = app.Init(context.Background(), app.InitOptions{
		URL: absPath,
	})
	if err != nil {
		t.Errorf("Init should succeed with absolute path, but failed: %v", err)
	}
}

// TestInit_WithPathTraversal tests that path traversal is rejected
func TestInit_WithPathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Try with path traversal (should fail)
	traversalPath := "../../../etc/passwd"

	err = app.Init(context.Background(), app.InitOptions{
		URL: traversalPath,
	})
	if err == nil {
		t.Errorf("Init should fail with path traversal, but succeeded")
	}
}

// TestInit_InvalidTemplate tests handling of invalid template
func TestInit_InvalidTemplate(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory without ign.json
	invalidTemplateDir := filepath.Join(tempDir, "invalid-template")
	if err := os.MkdirAll(invalidTemplateDir, 0755); err != nil {
		t.Fatalf("failed to create invalid template dir: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Get relative path
	relPath := "./invalid-template"

	// Should fail because ign.json is missing
	err = app.Init(context.Background(), app.InitOptions{
		URL: relPath,
	})
	if err == nil {
		t.Errorf("Init should fail with missing ign.json, but succeeded")
	}
}

// TestInit_ConditionalTemplate tests init with conditional template
func TestInit_ConditionalTemplate(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")

	// Copy fixture to temp directory
	templatePath := copyFixtureToTemp(t, "conditional-template", tempDir)

	// Change to temp directory for relative path resolution
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Execute init
	err = app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Read ign-var.json
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Verify conditional variables
	expectedVars := []string{"project_name", "use_docker", "has_license", "license"}
	for _, varName := range expectedVars {
		if _, ok := ignVar.Variables[varName]; !ok {
			t.Errorf("variable %s not found in ign-var.json", varName)
		}
	}
}

// TestInit_Force tests the --force flag for backup and reinitialize
func TestInit_Force(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")

	// Copy fixture to temp directory
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	// Change to temp directory for relative path resolution
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// First init
	err = app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	})
	if err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// Verify ign-var.json created
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	if _, err := os.Stat(ignVarPath); os.IsNotExist(err) {
		t.Fatalf("ign-var.json not created: %s", ignVarPath)
	}

	// Second init without --force should fail
	err = app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	})
	if err == nil {
		t.Errorf("Second Init without --force should fail, but succeeded")
	}

	// Third init with --force should succeed and create backup
	err = app.Init(context.Background(), app.InitOptions{
		URL:   templatePath,
		Force: true,
	})
	if err != nil {
		t.Fatalf("Third Init with --force failed: %v", err)
	}

	// Verify backup created
	backupPath := filepath.Join(configDir, "ign-var.json.bk1")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file not created: %s", backupPath)
	}

	// Fourth init with --force should create second backup
	err = app.Init(context.Background(), app.InitOptions{
		URL:   templatePath,
		Force: true,
	})
	if err != nil {
		t.Fatalf("Fourth Init with --force failed: %v", err)
	}

	// Verify second backup created
	backupPath2 := filepath.Join(configDir, "ign-var.json.bk2")
	if _, err := os.Stat(backupPath2); os.IsNotExist(err) {
		t.Errorf("second backup file not created: %s", backupPath2)
	}
}
