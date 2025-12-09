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

// TestBuildInit_LocalProvider tests build init with local filesystem provider
func TestBuildInit_LocalProvider(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")

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
	defer os.Chdir(origDir)

	// Execute build init
	err = app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       templatePath,
		OutputDir: buildDir,
	})
	if err != nil {
		t.Fatalf("BuildInit failed: %v", err)
	}

	// Verify .ign-build directory created
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		t.Errorf("build directory not created: %s", buildDir)
	}

	// Verify ign-var.json created
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
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

	// Verify template source
	if ignVar.Template.URL == "" {
		t.Errorf("template URL is empty")
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

// TestBuildInit_WithAbsolutePath tests that absolute paths are rejected for local provider
func TestBuildInit_WithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")

	// Try with absolute path (should fail for local provider)
	absPath := "/tmp/some-template"

	err := app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       absPath,
		OutputDir: buildDir,
	})
	if err == nil {
		t.Errorf("BuildInit should fail with absolute path, but succeeded")
	}
}

// TestBuildInit_WithPathTraversal tests that path traversal is rejected
func TestBuildInit_WithPathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")

	// Try with path traversal (should fail)
	traversalPath := "../../../etc/passwd"

	err := app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       traversalPath,
		OutputDir: buildDir,
	})
	if err == nil {
		t.Errorf("BuildInit should fail with path traversal, but succeeded")
	}
}

// TestBuildInit_InvalidTemplate tests handling of invalid template
func TestBuildInit_InvalidTemplate(t *testing.T) {
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")

	// Create a directory without ign.json
	invalidTemplateDir := filepath.Join(tempDir, "invalid-template")
	if err := os.MkdirAll(invalidTemplateDir, 0755); err != nil {
		t.Fatalf("failed to create invalid template dir: %v", err)
	}

	// Get relative path
	relPath, err := filepath.Rel(tempDir, invalidTemplateDir)
	if err != nil {
		t.Fatalf("failed to get relative path: %v", err)
	}

	// Should fail because ign.json is missing
	err = app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       relPath,
		OutputDir: buildDir,
	})
	if err == nil {
		t.Errorf("BuildInit should fail with missing ign.json, but succeeded")
	}
}

// TestBuildInit_ConditionalTemplate tests build init with conditional template
func TestBuildInit_ConditionalTemplate(t *testing.T) {
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")

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
	defer os.Chdir(origDir)

	// Execute build init
	err = app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       templatePath,
		OutputDir: buildDir,
	})
	if err != nil {
		t.Fatalf("BuildInit failed: %v", err)
	}

	// Read ign-var.json
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
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
