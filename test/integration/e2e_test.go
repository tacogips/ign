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

// TestE2E_CompleteWorkflow tests the complete init -> checkout workflow
func TestE2E_CompleteWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")
	outputDir := filepath.Join(tempDir, "my-project")

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

	// Step 1: Init
	t.Log("Step 1: Running init")
	if err := app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify config directory created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("config directory not created")
	}

	// Step 2: Edit variables (simulating user editing ign-var.json)
	t.Log("Step 2: Editing variables")
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Set realistic values
	ignVar.Variables["project_name"] = "awesome-api"
	ignVar.Variables["port"] = 3000
	ignVar.Variables["enable_feature"] = true

	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Step 3: Checkout (generate project)
	t.Log("Step 3: Running checkout")

	if _, err := app.Checkout(context.Background(), app.CheckoutOptions{
		OutputDir: outputDir,
	}); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	// Step 4: Verify generated project
	t.Log("Step 4: Verifying generated project")

	// Check output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("output directory not created")
	}

	// Check README.md
	readmePath := filepath.Join(outputDir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	readmeContent := string(readmeData)
	expectedValues := map[string]string{
		"awesome-api": "project_name",
		"3000":        "port",
		"true":        "enable_feature",
	}

	for expected, varName := range expectedValues {
		if !contains(readmeContent, expected) {
			t.Errorf("README.md does not contain %s (variable: %s)", expected, varName)
		}
	}

	// Check config.yaml
	configPath := filepath.Join(outputDir, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}

	configContent := string(configData)
	if !contains(configContent, "awesome-api") {
		t.Errorf("config.yaml does not contain project_name")
	}
	if !contains(configContent, "3000") {
		t.Errorf("config.yaml does not contain port")
	}

	// Verify ign.json NOT copied
	ignJsonPath := filepath.Join(outputDir, "ign.json")
	if _, err := os.Stat(ignJsonPath); !os.IsNotExist(err) {
		t.Errorf("ign.json should not be in output directory")
	}

	// Verify .ign NOT copied
	ignConfigPath := filepath.Join(outputDir, ".ign")
	if _, err := os.Stat(ignConfigPath); !os.IsNotExist(err) {
		t.Errorf(".ign should not be in output directory")
	}

	t.Log("E2E test completed successfully")
}

// TestE2E_MultipleTemplates tests using different templates sequentially
func TestE2E_MultipleTemplates(t *testing.T) {
	templates := []struct {
		name          string
		fixtureName   string
		variables     map[string]interface{}
		verifyFile    string
		verifyContent string
	}{
		{
			name:        "simple-template",
			fixtureName: "simple-template",
			variables: map[string]interface{}{
				"project_name":   "project-one",
				"port":           8080,
				"enable_feature": false,
			},
			verifyFile:    "README.md",
			verifyContent: "project-one",
		},
		{
			name:        "go-project",
			fixtureName: "go-project",
			variables: map[string]interface{}{
				"module_name": "github.com/user/myapp",
				"author":      "John Doe",
			},
			verifyFile:    "main.go",
			verifyContent: "myapp",
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for this subtest
			tempDir := t.TempDir()
			configDir := filepath.Join(tempDir, ".ign")
			outputDir := filepath.Join(tempDir, "output")

			// Copy fixture to temp directory
			templatePath := copyFixtureToTemp(t, tt.fixtureName, tempDir)

			// Change to temp directory for relative path resolution
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			// Init
			if err := app.Init(context.Background(), app.InitOptions{
				URL: templatePath,
			}); err != nil {
				t.Fatalf("Init failed: %v", err)
			}

			// Set variables
			ignVarPath := filepath.Join(configDir, "ign-var.json")
			data, err := os.ReadFile(ignVarPath)
			if err != nil {
				t.Fatalf("failed to read ign-var.json: %v", err)
			}

			var ignVar model.IgnVarJson
			if err := json.Unmarshal(data, &ignVar); err != nil {
				t.Fatalf("failed to parse ign-var.json: %v", err)
			}

			for k, v := range tt.variables {
				ignVar.Variables[k] = v
			}

			updatedData, err := json.MarshalIndent(ignVar, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal ign-var.json: %v", err)
			}

			if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
				t.Fatalf("failed to write ign-var.json: %v", err)
			}

			// Checkout
			if _, err := app.Checkout(context.Background(), app.CheckoutOptions{
				OutputDir: outputDir,
			}); err != nil {
				t.Fatalf("Checkout failed: %v", err)
			}

			// Verify
			verifyPath := filepath.Join(outputDir, tt.verifyFile)
			verifyData, err := os.ReadFile(verifyPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.verifyFile, err)
			}

			if !contains(string(verifyData), tt.verifyContent) {
				t.Errorf("%s does not contain expected content: %s", tt.verifyFile, tt.verifyContent)
			}
		})
	}
}

// TestE2E_DryRun tests the dry-run mode
func TestE2E_DryRun(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")
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
	defer func() { _ = os.Chdir(origDir) }()

	// Init
	if err := app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Set variables
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	ignVar.Variables["project_name"] = "dry-run-test"
	ignVar.Variables["port"] = 5000
	ignVar.Variables["enable_feature"] = true

	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Execute checkout with dry-run
	if _, err := app.Checkout(context.Background(), app.CheckoutOptions{
		OutputDir: outputDir,
		DryRun:    true,
	}); err != nil {
		t.Fatalf("Checkout (dry-run) failed: %v", err)
	}

	// Verify output directory NOT created in dry-run mode
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("output directory should not be created in dry-run mode")
	}

	// Now run without dry-run
	if _, err := app.Checkout(context.Background(), app.CheckoutOptions{
		OutputDir: outputDir,
	}); err != nil {
		t.Fatalf("Checkout (real) failed: %v", err)
	}

	// Verify output directory created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("output directory should be created in real mode")
	}

	// Verify files created
	readmePath := filepath.Join(outputDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Errorf("README.md should be created in real mode")
	}
}

// TestE2E_ErrorHandling tests error scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	t.Run("missing required variable", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".ign")
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
		defer func() { _ = os.Chdir(origDir) }()

		// Init with simple template
		if err := app.Init(context.Background(), app.InitOptions{
			URL: templatePath,
		}); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Read ign-var.json but don't set required variable
		ignVarPath := filepath.Join(configDir, "ign-var.json")
		data, err := os.ReadFile(ignVarPath)
		if err != nil {
			t.Fatalf("failed to read ign-var.json: %v", err)
		}

		var ignVar model.IgnVarJson
		if err := json.Unmarshal(data, &ignVar); err != nil {
			t.Fatalf("failed to parse ign-var.json: %v", err)
		}

		// Ensure Variables map is initialized
		if ignVar.Variables == nil {
			ignVar.Variables = make(map[string]interface{})
		}

		// Leave project_name empty (required variable)
		ignVar.Variables["project_name"] = ""
		ignVar.Variables["port"] = 8080
		ignVar.Variables["enable_feature"] = false

		updatedData, _ := json.MarshalIndent(ignVar, "", "  ")
		_ = os.WriteFile(ignVarPath, updatedData, 0644)

		// Execute checkout
		_, err = app.Checkout(context.Background(), app.CheckoutOptions{
			OutputDir: outputDir,
		})

		if err == nil {
			t.Errorf("expected error but got nil")
		} else if !contains(err.Error(), "required") {
			t.Errorf("error message does not contain expected substring: required\nGot: %v", err)
		}
	})

	t.Run("missing ign-var.json", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".ign")
		outputDir := filepath.Join(tempDir, "output")

		// Create config dir but no ign-var.json
		_ = os.MkdirAll(configDir, 0755)

		// Change directory
		oldWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldWd) }()
		_ = os.Chdir(tempDir)

		// Execute checkout
		_, err := app.Checkout(context.Background(), app.CheckoutOptions{
			OutputDir: outputDir,
		})

		if err == nil {
			t.Errorf("expected error but got nil")
		} else if !contains(err.Error(), "ign.json") {
			t.Errorf("error message does not contain expected substring: ign.json\nGot: %v", err)
		}
	})
}
