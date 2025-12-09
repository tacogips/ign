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

// TestInit_SimpleTemplate tests project initialization with simple template
func TestInit_SimpleTemplate(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")
	outputDir := filepath.Join(tempDir, "output")

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

	if err := app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       templatePath,
		OutputDir: buildDir,
	}); err != nil {
		t.Fatalf("BuildInit failed: %v", err)
	}

	// Edit ign-var.json with test values
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Set variable values
	ignVar.Variables["project_name"] = "test-project"
	ignVar.Variables["port"] = 9090
	ignVar.Variables["enable_feature"] = true

	// Save updated ign-var.json
	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Change to temp directory for init
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Execute init
	_, err = app.Init(context.Background(), app.InitOptions{
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(buildDir, "ign-var.json"),
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify output directory created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("output directory not created: %s", outputDir)
	}

	// Verify README.md created with substituted variables
	readmePath := filepath.Join(outputDir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	readmeContent := string(readmeData)

	// Check variable substitution
	if !contains(readmeContent, "test-project") {
		t.Errorf("README.md does not contain project_name substitution")
	}

	if !contains(readmeContent, "9090") {
		t.Errorf("README.md does not contain port substitution")
	}

	if !contains(readmeContent, "true") {
		t.Errorf("README.md does not contain enable_feature substitution")
	}

	// Verify config.yaml created
	configPath := filepath.Join(outputDir, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}

	configContent := string(configData)

	if !contains(configContent, "test-project") {
		t.Errorf("config.yaml does not contain project_name substitution")
	}

	// Verify ign.json is NOT copied
	ignJsonPath := filepath.Join(outputDir, "ign.json")
	if _, err := os.Stat(ignJsonPath); !os.IsNotExist(err) {
		t.Errorf("ign.json should not be copied to output")
	}
}

// TestInit_ConditionalTemplate tests project initialization with conditional directives
func TestInit_ConditionalTemplate(t *testing.T) {
	tests := []struct {
		name       string
		useDocker  bool
		hasLicense bool
		expectFile bool
	}{
		{
			name:       "with docker",
			useDocker:  true,
			hasLicense: true,
			expectFile: true,
		},
		{
			name:       "without docker",
			useDocker:  false,
			hasLicense: false,
			expectFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()
			buildDir := filepath.Join(tempDir, ".ign-build")
			outputDir := filepath.Join(tempDir, "output")

			// Copy fixture and build init
			templatePath := copyFixtureToTemp(t, "conditional-template", tempDir)

			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}
			defer os.Chdir(origDir)

			if err := app.BuildInit(context.Background(), app.BuildInitOptions{
				URL:       templatePath,
				OutputDir: buildDir,
			}); err != nil {
				t.Fatalf("BuildInit failed: %v", err)
			}

			// Set variables
			ignVarPath := filepath.Join(buildDir, "ign-var.json")
			data, err := os.ReadFile(ignVarPath)
			if err != nil {
				t.Fatalf("failed to read ign-var.json: %v", err)
			}

			var ignVar model.IgnVarJson
			if err := json.Unmarshal(data, &ignVar); err != nil {
				t.Fatalf("failed to parse ign-var.json: %v", err)
			}

			ignVar.Variables["project_name"] = "test-conditional"
			ignVar.Variables["use_docker"] = tt.useDocker
			ignVar.Variables["has_license"] = tt.hasLicense

			updatedData, err := json.MarshalIndent(ignVar, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal ign-var.json: %v", err)
			}

			if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
				t.Fatalf("failed to write ign-var.json: %v", err)
			}

			// Execute init
			if _, err := app.Init(context.Background(), app.InitOptions{
				OutputDir:  outputDir,
				ConfigPath: filepath.Join(buildDir, "ign-var.json"),
			}); err != nil {
				t.Fatalf("Init failed: %v", err)
			}

			// Verify README.md content based on conditions
			readmePath := filepath.Join(outputDir, "README.md")
			readmeData, err := os.ReadFile(readmePath)
			if err != nil {
				t.Fatalf("failed to read README.md: %v", err)
			}

			readmeContent := string(readmeData)

			if tt.useDocker {
				if !contains(readmeContent, "Docker Support") {
					t.Errorf("README.md should contain Docker section when use_docker=true")
				}
			} else {
				if contains(readmeContent, "Docker Support") {
					t.Errorf("README.md should not contain Docker section when use_docker=false")
				}
			}

			if tt.hasLicense {
				if !contains(readmeContent, "License") {
					t.Errorf("README.md should contain License section when has_license=true")
				}
			} else {
				if contains(readmeContent, "Licensed under") {
					t.Errorf("README.md should not contain License details when has_license=false")
				}
			}

			// Verify docker-compose.yml
			dockerComposePath := filepath.Join(outputDir, "docker-compose.yml")
			dockerComposeData, err := os.ReadFile(dockerComposePath)
			if err != nil {
				t.Fatalf("failed to read docker-compose.yml: %v", err)
			}

			dockerComposeContent := string(dockerComposeData)

			if tt.useDocker {
				if !contains(dockerComposeContent, "version:") {
					t.Errorf("docker-compose.yml should contain content when use_docker=true")
				}
			} else {
				// When use_docker=false, file should be empty or only contain whitespace
				trimmed := string(dockerComposeData)
				if len(trimmed) > 10 { // Allow some whitespace
					t.Errorf("docker-compose.yml should be mostly empty when use_docker=false, got: %s", trimmed)
				}
			}
		})
	}
}

// TestInit_GoProject tests Go project template with comment directive
func TestInit_GoProject(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")
	outputDir := filepath.Join(tempDir, "output")

	// Copy fixture and build init
	templatePath := copyFixtureToTemp(t, "go-project", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       templatePath,
		OutputDir: buildDir,
	}); err != nil {
		t.Fatalf("BuildInit failed: %v", err)
	}

	// Set variables
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	ignVar.Variables["module_name"] = "github.com/test/myproject"
	ignVar.Variables["author"] = "Test Author"

	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Change directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Execute init
	if _, err := app.Init(context.Background(), app.InitOptions{
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(buildDir, "ign-var.json"),
	}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify go.mod
	goModPath := filepath.Join(outputDir, "go.mod")
	goModData, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	goModContent := string(goModData)
	if !contains(goModContent, "github.com/test/myproject") {
		t.Errorf("go.mod does not contain module_name substitution")
	}

	// Verify main.go has variable substitution
	mainGoPath := filepath.Join(outputDir, "main.go")
	mainGoData, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	mainGoContent := string(mainGoData)

	// Should contain module name in code
	if !contains(mainGoContent, "github.com/test/myproject") {
		t.Errorf("main.go does not contain module_name substitution")
	}

	// Verify README.md with comment directive (lines should be removed)
	readmePath := filepath.Join(outputDir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	readmeContent := string(readmeData)

	// Should NOT contain @ign-comment directive (lines removed)
	if contains(readmeContent, "@ign-comment:") {
		t.Errorf("README.md should not contain @ign-comment directive (lines should be removed)")
	}

	// Should contain substituted variable values
	if !contains(readmeContent, "github.com/test/myproject") {
		t.Errorf("README.md does not contain module_name substitution")
	}

	if !contains(readmeContent, "Test Author") {
		t.Errorf("README.md does not contain author substitution")
	}
}

// TestInit_Overwrite tests the overwrite flag
func TestInit_Overwrite(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	buildDir := filepath.Join(tempDir, ".ign-build")
	outputDir := filepath.Join(tempDir, "output")

	// Copy fixture and build init
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := app.BuildInit(context.Background(), app.BuildInitOptions{
		URL:       templatePath,
		OutputDir: buildDir,
	}); err != nil {
		t.Fatalf("BuildInit failed: %v", err)
	}

	// Set variables
	ignVarPath := filepath.Join(buildDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	ignVar.Variables["project_name"] = "first-run"
	ignVar.Variables["port"] = 8080
	ignVar.Variables["enable_feature"] = false

	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Change directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// First init
	if _, err := app.Init(context.Background(), app.InitOptions{
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(buildDir, "ign-var.json"),
	}); err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// Verify first run
	readmePath := filepath.Join(outputDir, "README.md")
	firstRunData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md after first run: %v", err)
	}

	if !contains(string(firstRunData), "first-run") {
		t.Errorf("README.md does not contain first-run value")
	}

	// Update variables
	ignVar.Variables["project_name"] = "second-run"
	updatedData2, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData2, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Second init without overwrite (should skip)
	if _, err := app.Init(context.Background(), app.InitOptions{
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(buildDir, "ign-var.json"),
	}); err != nil {
		t.Fatalf("Second Init (no overwrite) failed: %v", err)
	}

	// Verify files not overwritten
	secondRunData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md after second run: %v", err)
	}

	if !contains(string(secondRunData), "first-run") {
		t.Errorf("README.md should still contain first-run (not overwritten)")
	}

	// Third init with overwrite
	if _, err := app.Init(context.Background(), app.InitOptions{
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(buildDir, "ign-var.json"),
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("Third Init (with overwrite) failed: %v", err)
	}

	// Verify files overwritten
	thirdRunData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md after third run: %v", err)
	}

	if !contains(string(thirdRunData), "second-run") {
		t.Errorf("README.md should contain second-run (overwritten)")
	}

	if contains(string(thirdRunData), "first-run") {
		t.Errorf("README.md should not contain first-run (overwritten)")
	}
}
