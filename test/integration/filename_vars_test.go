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

// TestE2E_FilenameVariableSubstitution tests filename variable substitution feature
func TestE2E_FilenameVariableSubstitution(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")
	outputDir := filepath.Join(tempDir, "output")

	// Get fixture path from testdata
	fixtureDir := filepath.Join("..", "testdata", "filename-vars-template")
	templatePath := copyTestdataToTemp(t, fixtureDir, tempDir)

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
	t.Log("Step 1: Running init with filename-vars-template")
	if err := app.Init(context.Background(), app.InitOptions{
		URL: templatePath,
	}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify config directory created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("config directory not created")
	}

	// Step 2: Edit variables
	t.Log("Step 2: Setting variables")
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Set variable values
	ignVar.Variables["app_name"] = "myapp"
	ignVar.Variables["module_name"] = "handler"
	ignVar.Variables["env"] = "production"

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

	// Step 4: Verify generated files with substituted filenames
	t.Log("Step 4: Verifying filename substitution")

	// Verify module file: handler.txt
	moduleFilePath := filepath.Join(outputDir, "handler.txt")
	if _, err := os.Stat(moduleFilePath); os.IsNotExist(err) {
		t.Fatalf("handler.txt not created (filename variable not substituted)")
	}

	moduleContent, err := os.ReadFile(moduleFilePath)
	if err != nil {
		t.Fatalf("failed to read handler.txt: %v", err)
	}

	if !contains(string(moduleContent), "package handler") {
		t.Errorf("handler.txt does not contain 'package handler'")
	}

	// Verify app directory: cmd/myapp/main.txt
	appDirPath := filepath.Join(outputDir, "cmd", "myapp")
	if _, err := os.Stat(appDirPath); os.IsNotExist(err) {
		t.Fatalf("cmd/myapp directory not created (directory name variable not substituted)")
	}

	mainFilePath := filepath.Join(appDirPath, "main.txt")
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		t.Fatalf("cmd/myapp/main.txt not created")
	}

	mainContent, err := os.ReadFile(mainFilePath)
	if err != nil {
		t.Fatalf("failed to read main.txt: %v", err)
	}

	if !contains(string(mainContent), "myapp application") {
		t.Errorf("main.txt does not contain 'myapp application'")
	}

	// Verify config file with env variable: config-production.yaml
	configFilePath := filepath.Join(outputDir, "config-production.yaml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		t.Fatalf("config-production.yaml not created (filename variable not substituted)")
	}

	configContent, err := os.ReadFile(configFilePath)
	if err != nil {
		t.Fatalf("failed to read config-production.yaml: %v", err)
	}

	configStr := string(configContent)
	if !contains(configStr, "environment: production") {
		t.Errorf("config-production.yaml does not contain 'environment: production'")
	}
	if !contains(configStr, "app_name: myapp") {
		t.Errorf("config-production.yaml does not contain 'app_name: myapp'")
	}
	if !contains(configStr, "module: handler") {
		t.Errorf("config-production.yaml does not contain 'module: handler'")
	}

	// Verify README.md
	readmePath := filepath.Join(outputDir, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	readmeStr := string(readmeContent)
	if !contains(readmeStr, "# myapp Project") {
		t.Errorf("README.md does not contain '# myapp Project'")
	}
	if !contains(readmeStr, "Module name: handler") {
		t.Errorf("README.md does not contain 'Module name: handler'")
	}
	if !contains(readmeStr, "Current environment: production") {
		t.Errorf("README.md does not contain 'Current environment: production'")
	}

	t.Log("Filename variable substitution test completed successfully")
}

// TestE2E_FilenameVariables_DefaultValues tests filename variables with default values
func TestE2E_FilenameVariables_DefaultValues(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".ign")
	outputDir := filepath.Join(tempDir, "output")

	// Get fixture path from testdata
	fixtureDir := filepath.Join("..", "testdata", "filename-vars-template")
	templatePath := copyTestdataToTemp(t, fixtureDir, tempDir)

	// Change to temp directory
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

	// Step 2: Edit variables - only set required ones, leave env with default
	t.Log("Step 2: Setting required variables only")
	ignVarPath := filepath.Join(configDir, "ign-var.json")
	data, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	// Only set required variables, leave env to use default value
	ignVar.Variables["app_name"] = "testapp"
	ignVar.Variables["module_name"] = "utils"
	// env is not set - should use default "dev"

	updatedData, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal ign-var.json: %v", err)
	}

	if err := os.WriteFile(ignVarPath, updatedData, 0644); err != nil {
		t.Fatalf("failed to write ign-var.json: %v", err)
	}

	// Step 3: Checkout
	t.Log("Step 3: Running checkout")
	if _, err := app.Checkout(context.Background(), app.CheckoutOptions{
		OutputDir: outputDir,
	}); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	// Step 4: Verify config file uses default value
	t.Log("Step 4: Verifying default value in filename")

	// Should create config-dev.yaml (using default value)
	configFilePath := filepath.Join(outputDir, "config-dev.yaml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		t.Fatalf("config-dev.yaml not created (default value not used in filename)")
	}

	configContent, err := os.ReadFile(configFilePath)
	if err != nil {
		t.Fatalf("failed to read config-dev.yaml: %v", err)
	}

	if !contains(string(configContent), "environment: dev") {
		t.Errorf("config-dev.yaml does not contain 'environment: dev'")
	}

	t.Log("Filename variable default value test completed successfully")
}
