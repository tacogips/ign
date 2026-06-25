package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

func TestNormalizeTemplateURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full HTTPS URL",
			input:    "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "github.com prefix",
			input:    "github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "owner/repo format",
			input:    "owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "git@ format",
			input:    "git@github.com:owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "with subdirectory",
			input:    "github.com/owner/repo/templates/go-basic",
			expected: "https://github.com/owner/repo/templates/go-basic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTemplateURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTemplateURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateOutputDir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "/tmp/test",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			path:    "./test",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "path traversal",
			path:    "../../../etc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputDir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputDir(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestCreateEmptyVariablesMap(t *testing.T) {
	ignJson := &model.IgnJson{
		Name:    "test-template",
		Version: "1.0.0",
		Variables: map[string]model.VarDef{
			"project_name": {
				Type:        model.VarTypeString,
				Description: "Project name",
				Default:     "{current_dir}",
			},
			"port": {
				Type:        model.VarTypeInt,
				Description: "Port number",
				Default:     8080,
			},
			"enable_tls": {
				Type:        model.VarTypeBool,
				Description: "Enable TLS",
				Required:    false,
			},
		},
	}

	vars := CreateEmptyVariablesMap(ignJson)

	// Check string variable with raw dynamic default
	if val, ok := vars["project_name"]; !ok || val != "{current_dir}" {
		t.Errorf("Expected project_name to keep raw default, got %v", val)
	}

	// Check int variable with default
	if val, ok := vars["port"]; !ok || val != 8080 {
		t.Errorf("Expected port to be 8080, got %v", val)
	}

	// Check bool variable (no default)
	if val, ok := vars["enable_tls"]; !ok || val != false {
		t.Errorf("Expected enable_tls to be false, got %v", val)
	}
}

func TestCreateVariablesMap_ProvidedValues(t *testing.T) {
	ignJson := &model.IgnJson{
		Name:    "test-template",
		Version: "1.0.0",
		Variables: map[string]model.VarDef{
			"project_name": {
				Type:    model.VarTypeString,
				Default: "default-name",
			},
			"port": {
				Type:    model.VarTypeInt,
				Default: 8080,
			},
			"enable_tls": {
				Type: model.VarTypeBool,
			},
		},
	}

	vars := CreateVariablesMap(ignJson, map[string]interface{}{
		"project_name": "provided-name",
		"enable_tls":   true,
	})

	if got := vars["project_name"]; got != "provided-name" {
		t.Fatalf("project_name = %v, want provided-name", got)
	}
	if got := vars["port"]; got != 8080 {
		t.Fatalf("port = %v, want default 8080", got)
	}
	if got := vars["enable_tls"]; got != true {
		t.Fatalf("enable_tls = %v, want true", got)
	}
}

func TestCountVariablesByType_NilIgnJSON(t *testing.T) {
	stringCount, intCount, boolCount := CountVariablesByType(nil)
	if stringCount != 0 || intCount != 0 || boolCount != 0 {
		t.Fatalf("CountVariablesByType(nil) = (%d, %d, %d), want (0, 0, 0)", stringCount, intCount, boolCount)
	}
}

func TestValidateTemplateHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{
			name:    "missing hash",
			hash:    "",
			wantErr: true,
		},
		{
			name:    "invalid hash format",
			hash:    "test-hash",
			wantErr: true,
		},
		{
			name:    "valid sha256 hash",
			hash:    strings.Repeat("a", 64),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateHash(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateTemplateHash(%q) error = %v, wantErr %v", tt.hash, err, tt.wantErr)
			}
		})
	}
}

func TestLoadVariables(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test.txt")
	testFileContent := "Hello, World!"
	if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		ignVar  *model.IgnVarJson
		wantErr bool
		check   func(t *testing.T, vars parser.Variables)
	}{
		{
			name: "simple variables",
			ignVar: &model.IgnVarJson{
				Variables: map[string]interface{}{
					"name":    "test",
					"version": "1.0.0",
					"port":    8080,
				},
			},
			wantErr: false,
			check: func(t *testing.T, vars parser.Variables) {
				if val, ok := vars.Get("name"); !ok || val != "test" {
					t.Errorf("Expected name=test, got %v", val)
				}
				if val, ok := vars.Get("port"); !ok || val != 8080 {
					t.Errorf("Expected port=8080, got %v", val)
				}
			},
		},
		{
			name: "@file: reference",
			ignVar: &model.IgnVarJson{
				Variables: map[string]interface{}{
					"content": "@file:test.txt",
				},
			},
			wantErr: false,
			check: func(t *testing.T, vars parser.Variables) {
				val, ok := vars.Get("content")
				if !ok {
					t.Error("Expected content variable to exist")
					return
				}
				if val != testFileContent {
					t.Errorf("Expected content=%q, got %q", testFileContent, val)
				}
			},
		},
		{
			name: "@file: missing file",
			ignVar: &model.IgnVarJson{
				Variables: map[string]interface{}{
					"content": "@file:nonexistent.txt",
				},
			},
			wantErr: true,
			check:   nil,
		},
		{
			name: "@file: empty filename",
			ignVar: &model.IgnVarJson{
				Variables: map[string]interface{}{
					"content": "@file:",
				},
			},
			wantErr: true,
			check:   nil,
		},
		{
			name:    "nil ignVar",
			ignVar:  nil,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars, err := LoadVariables(tt.ignVar, tempDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil && vars != nil {
				tt.check(t, vars)
			}
		})
	}
}

func TestCompleteCheckout_PreparesRuntimeVariables(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Mkdir(".ign", 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}

	licenseContent := "Licensed under test terms"
	if err := os.WriteFile(filepath.Join(".ign", "license.txt"), []byte(licenseContent), 0644); err != nil {
		t.Fatalf("failed to write license file: %v", err)
	}

	template := &model.Template{
		Config: model.IgnJson{
			Name:    "runtime-defaults",
			Version: "1.0.0",
			Hash:    strings.Repeat("a", 64),
			Variables: map[string]model.VarDef{
				"project_name": {
					Type:    model.VarTypeString,
					Default: "{current_dir}",
				},
				"license_text": {
					Type:     model.VarTypeString,
					Required: true,
				},
			},
		},
		Files: []model.TemplateFile{
			{
				Path:    "cmd/@ign-var:project_name@/main.txt",
				Content: []byte("project=@ign-var:project_name@\nlicense=@ign-var:license_text@\n"),
				Mode:    0644,
			},
		},
	}

	outputDir := filepath.Join(tempDir, "sample-app")
	prep := &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   model.TemplateRef{Provider: "local", Repo: "runtime-defaults"},
		NormalizedURL: "./template",
	}

	if _, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: prep,
		Variables: map[string]interface{}{
			"license_text": "@file:license.txt",
		},
		OutputDir: outputDir,
	}); err != nil {
		t.Fatalf("CompleteCheckout failed: %v", err)
	}

	generatedPath := filepath.Join(outputDir, "cmd", "sample-app", "main.txt")
	data, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "project=sample-app") {
		t.Fatalf("generated file did not use current_dir default: %s", content)
	}
	if !strings.Contains(content, "license="+licenseContent) {
		t.Fatalf("generated file did not resolve @file reference: %s", content)
	}

	var ignVar model.IgnVarJson
	ignVarData, err := os.ReadFile(filepath.Join(".ign", "ign-var.json"))
	if err != nil {
		t.Fatalf("failed to read ign-var.json: %v", err)
	}
	if err := json.Unmarshal(ignVarData, &ignVar); err != nil {
		t.Fatalf("failed to parse ign-var.json: %v", err)
	}

	if ignVar.Variables["project_name"] != "{current_dir}" {
		t.Fatalf("project_name default should remain dynamic in ign-var.json, got %v", ignVar.Variables["project_name"])
	}
	if ignVar.Variables["license_text"] != "@file:license.txt" {
		t.Fatalf("license_text should preserve raw @file reference, got %v", ignVar.Variables["license_text"])
	}
}

func TestCompleteCheckout_InvalidRuntimeVariableDoesNotWriteConfigOrGenerate(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Mkdir(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	template := &model.Template{
		Config: model.IgnJson{
			Name:    "invalid-runtime-variable",
			Version: "1.0.0",
			Hash:    strings.Repeat("a", 64),
			Variables: map[string]model.VarDef{
				"license_text": {
					Type:     model.VarTypeString,
					Required: true,
				},
			},
		},
		Files: []model.TemplateFile{
			{
				Path:    "README.md",
				Content: []byte("@ign-var:license_text@"),
				Mode:    0644,
			},
		},
	}

	outputDir := filepath.Join(tempDir, "out")
	prep := &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   model.TemplateRef{Provider: "local", Repo: "invalid-runtime-variable"},
		NormalizedURL: "./template",
	}

	_, err = CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: prep,
		Variables: map[string]interface{}{
			"license_text": "@file:missing.txt",
		},
		OutputDir: outputDir,
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected missing @file variable error")
	}

	for _, path := range []string{
		filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile),
		filepath.Join(model.IgnConfigDir, model.IgnVarFile),
		filepath.Join(model.IgnConfigDir, model.IgnManifestFile),
		filepath.Join(outputDir, "README.md"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s was written before runtime variable validation completed: %v", path, err)
		}
	}
}

func TestValidateCompleteCheckoutOptionsRequiresTemplate(t *testing.T) {
	err := ValidateCompleteCheckoutOptions(CompleteCheckoutOptions{
		PrepareResult: &PrepareCheckoutResult{
			IgnJson: &model.IgnJson{
				Name:      "missing-template",
				Version:   "1.0.0",
				Hash:      strings.Repeat("a", 64),
				Variables: map[string]model.VarDef{},
			},
		},
		Variables: map[string]interface{}{},
		OutputDir: ".",
	})
	if err == nil {
		t.Fatalf("ValidateCompleteCheckoutOptions expected nil template error")
	}
}

func TestCompleteCheckout_GenerationFailureDoesNotWriteConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "empty-template",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
			Hash:      strings.Repeat("b", 64),
		},
	}
	prep := &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   model.TemplateRef{Provider: "local", Repo: "empty-template"},
		NormalizedURL: "./template",
	}

	_, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: prep,
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected generation error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("CompleteCheckout error type = %T, want *AppError", err)
	}
	if appErr.Type != CheckoutFailed {
		t.Fatalf("CompleteCheckout error app type = %v, want %v", appErr.Type, CheckoutFailed)
	}
	if !strings.Contains(err.Error(), "template has no files") {
		t.Fatalf("CompleteCheckout error = %q, want template has no files generation error", err.Error())
	}

	for _, path := range []string{
		filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile),
		filepath.Join(model.IgnConfigDir, model.IgnVarFile),
		filepath.Join(model.IgnConfigDir, model.IgnManifestFile),
		"output",
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s was written before generation completed: %v", path, err)
		}
	}
}

func TestCompleteCheckout_ManifestFailureRollsBackConfigFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("failed to create manifest directory fixture: %v", err)
	}

	_, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: singleFilePreparedCheckout(),
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected manifest save error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("CompleteCheckout error type = %T, want *AppError", err)
	}
	if appErr.Type != CheckoutFailed {
		t.Fatalf("CompleteCheckout error app type = %v, want %v", appErr.Type, CheckoutFailed)
	}
	if !strings.Contains(err.Error(), "failed to save ign-files.json") {
		t.Fatalf("CompleteCheckout error = %q, want manifest save error", err.Error())
	}

	for _, path := range []string{
		filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile),
		filepath.Join(model.IgnConfigDir, model.IgnVarFile),
		filepath.Join("output", "docs", "README.md"),
		"output",
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s was not rolled back after manifest save failure: %v", path, err)
		}
	}
}

func TestCompleteCheckout_ManifestFailureRestoresOverwrittenFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	if err := os.MkdirAll(filepath.Join("output", "docs"), 0755); err != nil {
		t.Fatalf("failed to create output fixture: %v", err)
	}
	overwrittenPath := filepath.Join("output", "docs", "README.md")
	if err := os.WriteFile(overwrittenPath, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create existing output file: %v", err)
	}
	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("failed to create manifest directory fixture: %v", err)
	}

	_, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: twoFilePreparedCheckout(),
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
		Overwrite:     true,
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected manifest save error")
	}
	if !strings.Contains(err.Error(), "failed to save ign-files.json") {
		t.Fatalf("CompleteCheckout error = %q, want manifest save error", err.Error())
	}

	content, err := os.ReadFile(overwrittenPath)
	if err != nil {
		t.Fatalf("failed to read restored output file: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("overwritten file content = %q, want original", content)
	}
	if _, err := os.Stat(filepath.Join("output", "docs", "NEW.md")); !os.IsNotExist(err) {
		t.Fatalf("newly created file was not rolled back after manifest save failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)); !os.IsNotExist(err) {
		t.Fatalf("ign.json was not rolled back after manifest save failure: %v", err)
	}
}

func TestCompleteCheckout_ConfigSaveFailureRollsBackGeneratedFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := os.MkdirAll(ignConfigPath, 0755); err != nil {
		t.Fatalf("failed to create ign.json directory fixture: %v", err)
	}

	_, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: singleFilePreparedCheckout(),
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected ign.json save error")
	}
	if !strings.Contains(err.Error(), "failed to save ign.json") {
		t.Fatalf("CompleteCheckout error = %q, want ign.json save error", err.Error())
	}
	if _, err := os.Stat(filepath.Join("output", "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("generated file was not rolled back after ign.json save failure: %v", err)
	}
	if _, err := os.Stat("output"); !os.IsNotExist(err) {
		t.Fatalf("generated output directory was not rolled back after ign.json save failure: %v", err)
	}
}

func TestCompleteCheckout_VarSaveFailureRollsBackGeneratedFilesAndConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := os.MkdirAll(ignVarPath, 0755); err != nil {
		t.Fatalf("failed to create ign-var.json directory fixture: %v", err)
	}

	_, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: singleFilePreparedCheckout(),
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
	})
	if err == nil {
		t.Fatalf("CompleteCheckout expected ign-var.json save error")
	}
	if !strings.Contains(err.Error(), "failed to save ign-var.json") {
		t.Fatalf("CompleteCheckout error = %q, want ign-var.json save error", err.Error())
	}
	for _, path := range []string{
		filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile),
		filepath.Join("output", "docs", "README.md"),
		"output",
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s was not rolled back after ign-var.json save failure: %v", path, err)
		}
	}
}

func TestCheckout_MissingRequiredVariableDoesNotUpdateConfigHash(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	templateDir := "template"
	templateConfig := &model.IgnJson{
		Name:    "required-variable-template",
		Version: "1.0.0",
		Variables: map[string]model.VarDef{
			"project_name": {
				Type:     model.VarTypeString,
				Required: true,
			},
		},
		Hash: newHash,
	}
	writeLocalTemplate(t, templateDir, templateConfig, map[string]string{
		"README.md": "@ign-var:project_name@",
	})

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := config.SaveIgnConfig(ignConfigPath, &model.IgnConfig{
		Template: model.TemplateSource{URL: "./" + templateDir},
		Hash:     oldHash,
	}); err != nil {
		t.Fatalf("failed to write ign config: %v", err)
	}
	oldConfigData, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read initial ign config: %v", err)
	}
	oldVariables := map[string]interface{}{
		"project_name": "",
		"preserved":    "keep-me",
	}
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := config.SaveIgnVarJson(ignVarPath, &model.IgnVarJson{
		Variables: oldVariables,
	}); err != nil {
		t.Fatalf("failed to write ign vars: %v", err)
	}
	oldVarData, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read initial ign vars: %v", err)
	}

	_, err = Checkout(context.Background(), CheckoutOptions{
		OutputDir: "output",
	})
	if err == nil {
		t.Fatalf("Checkout expected missing required variable error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("Checkout error type = %T, want *AppError", err)
	}
	if appErr.Type != ValidationFailed {
		t.Fatalf("Checkout error app type = %v, want %v", appErr.Type, ValidationFailed)
	}
	if !strings.Contains(err.Error(), "missing required variables: project_name") {
		t.Fatalf("Checkout error = %q, want missing project_name validation error", err.Error())
	}

	gotConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to load ign config after failed checkout: %v", err)
	}
	if gotConfig.Hash != oldHash {
		t.Fatalf("checkout updated ign.json hash before variable validation, got %q want %q", gotConfig.Hash, oldHash)
	}
	gotConfigData, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read ign config after failed checkout: %v", err)
	}
	if string(gotConfigData) != string(oldConfigData) {
		t.Fatalf("checkout changed ign.json before variable validation")
	}
	gotVars, err := config.LoadIgnVarJson(ignVarPath)
	if err != nil {
		t.Fatalf("failed to load ign vars after failed checkout: %v", err)
	}
	if gotVars.Variables["preserved"] != oldVariables["preserved"] {
		t.Fatalf("checkout changed ign-var.json before variable validation, got %v want %v", gotVars.Variables["preserved"], oldVariables["preserved"])
	}
	gotVarData, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read ign vars after failed checkout: %v", err)
	}
	if string(gotVarData) != string(oldVarData) {
		t.Fatalf("checkout changed ign-var.json before variable validation")
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnManifestFile)); !os.IsNotExist(err) {
		t.Fatalf("checkout wrote manifest after variable validation failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join("output", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("checkout generated output after variable validation failure: %v", err)
	}
}

func TestCheckout_GenerationFailureDoesNotUpdateConfigHash(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	templateDir := "template"
	templateConfig := &model.IgnJson{
		Name:      "empty-template",
		Version:   "1.0.0",
		Variables: map[string]model.VarDef{},
		Hash:      newHash,
	}
	writeLocalTemplate(t, templateDir, templateConfig, nil)

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := config.SaveIgnConfig(ignConfigPath, &model.IgnConfig{
		Template: model.TemplateSource{URL: "./" + templateDir},
		Hash:     oldHash,
	}); err != nil {
		t.Fatalf("failed to write ign config: %v", err)
	}
	oldConfigData, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read initial ign config: %v", err)
	}
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := config.SaveIgnVarJson(ignVarPath, &model.IgnVarJson{
		Variables: map[string]interface{}{},
	}); err != nil {
		t.Fatalf("failed to write ign vars: %v", err)
	}

	_, err = Checkout(context.Background(), CheckoutOptions{
		OutputDir: "output",
	})
	if err == nil {
		t.Fatalf("Checkout expected generation error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("Checkout error type = %T, want *AppError", err)
	}
	if appErr.Type != CheckoutFailed {
		t.Fatalf("Checkout error app type = %v, want %v", appErr.Type, CheckoutFailed)
	}
	if !strings.Contains(err.Error(), "template has no files") {
		t.Fatalf("Checkout error = %q, want template has no files generation error", err.Error())
	}

	gotConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to load ign config after failed checkout: %v", err)
	}
	if gotConfig.Hash != oldHash {
		t.Fatalf("checkout updated ign.json hash before successful generation, got %q want %q", gotConfig.Hash, oldHash)
	}
	gotConfigData, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read ign config after failed checkout: %v", err)
	}
	if string(gotConfigData) != string(oldConfigData) {
		t.Fatalf("checkout changed ign.json before successful generation")
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnManifestFile)); !os.IsNotExist(err) {
		t.Fatalf("checkout wrote manifest after generation failure: %v", err)
	}
	if _, err := os.Stat("output"); !os.IsNotExist(err) {
		t.Fatalf("checkout created output directory after generation failure: %v", err)
	}
}

func TestCheckout_ManifestFailureRollsBackCreatedFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	templateDir := "template"
	templateConfig := &model.IgnJson{
		Name:      "single-file-template",
		Version:   "1.0.0",
		Variables: map[string]model.VarDef{},
		Hash:      newHash,
	}
	writeLocalTemplate(t, templateDir, templateConfig, map[string]string{
		filepath.Join("docs", "README.md"): "generated",
	})

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := config.SaveIgnConfig(ignConfigPath, &model.IgnConfig{
		Template: model.TemplateSource{URL: "./" + templateDir},
		Hash:     oldHash,
	}); err != nil {
		t.Fatalf("failed to write ign config: %v", err)
	}
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := config.SaveIgnVarJson(ignVarPath, &model.IgnVarJson{
		Variables: map[string]interface{}{},
	}); err != nil {
		t.Fatalf("failed to write ign vars: %v", err)
	}
	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("failed to create manifest directory fixture: %v", err)
	}

	_, err := Checkout(context.Background(), CheckoutOptions{
		OutputDir: "output",
	})
	if err == nil {
		t.Fatalf("Checkout expected manifest save error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("Checkout error type = %T, want *AppError", err)
	}
	if appErr.Type != CheckoutFailed {
		t.Fatalf("Checkout error app type = %v, want %v", appErr.Type, CheckoutFailed)
	}
	if !strings.Contains(err.Error(), "failed to save ign-files.json") {
		t.Fatalf("Checkout error = %q, want manifest save error", err.Error())
	}

	gotConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to load ign config after failed checkout: %v", err)
	}
	if gotConfig.Hash != oldHash {
		t.Fatalf("checkout updated ign.json hash after manifest save failure, got %q want %q", gotConfig.Hash, oldHash)
	}
	if _, err := os.Stat(filepath.Join("output", "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("checkout did not roll back generated file after manifest save failure: %v", err)
	}
	if _, err := os.Stat("output"); !os.IsNotExist(err) {
		t.Fatalf("checkout did not roll back generated output directory after manifest save failure: %v", err)
	}
}

func TestCheckout_ManifestFailureRestoresOverwrittenFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	templateDir := "template"
	templateConfig := &model.IgnJson{
		Name:      "two-file-template",
		Version:   "1.0.0",
		Variables: map[string]model.VarDef{},
		Hash:      newHash,
	}
	writeLocalTemplate(t, templateDir, templateConfig, map[string]string{
		filepath.Join("docs", "README.md"): "generated",
		filepath.Join("docs", "NEW.md"):    "new",
	})

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := config.SaveIgnConfig(ignConfigPath, &model.IgnConfig{
		Template: model.TemplateSource{URL: "./" + templateDir},
		Hash:     oldHash,
	}); err != nil {
		t.Fatalf("failed to write ign config: %v", err)
	}
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := config.SaveIgnVarJson(ignVarPath, &model.IgnVarJson{
		Variables: map[string]interface{}{},
	}); err != nil {
		t.Fatalf("failed to write ign vars: %v", err)
	}
	if err := os.MkdirAll(filepath.Join("output", "docs"), 0755); err != nil {
		t.Fatalf("failed to create output fixture: %v", err)
	}
	overwrittenPath := filepath.Join("output", "docs", "README.md")
	if err := os.WriteFile(overwrittenPath, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create existing output file: %v", err)
	}
	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("failed to create manifest directory fixture: %v", err)
	}

	_, err := Checkout(context.Background(), CheckoutOptions{
		OutputDir: "output",
		Overwrite: true,
	})
	if err == nil {
		t.Fatalf("Checkout expected manifest save error")
	}
	if !strings.Contains(err.Error(), "failed to save ign-files.json") {
		t.Fatalf("Checkout error = %q, want manifest save error", err.Error())
	}

	content, err := os.ReadFile(overwrittenPath)
	if err != nil {
		t.Fatalf("failed to read restored output file: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("overwritten file content = %q, want original", content)
	}
	if _, err := os.Stat(filepath.Join("output", "docs", "NEW.md")); !os.IsNotExist(err) {
		t.Fatalf("newly created file was not rolled back after manifest save failure: %v", err)
	}
	gotConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to load ign config after failed checkout: %v", err)
	}
	if gotConfig.Hash != oldHash {
		t.Fatalf("checkout updated ign.json hash after manifest save failure, got %q want %q", gotConfig.Hash, oldHash)
	}
}

func TestCheckout_ManifestFailurePreservesPreexistingOutputDir(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	templateDir := "template"
	templateConfig := &model.IgnJson{
		Name:      "nested-file-template",
		Version:   "1.0.0",
		Variables: map[string]model.VarDef{},
		Hash:      newHash,
	}
	writeLocalTemplate(t, templateDir, templateConfig, map[string]string{
		filepath.Join("docs", "README.md"): "generated",
	})

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	if err := config.SaveIgnConfig(ignConfigPath, &model.IgnConfig{
		Template: model.TemplateSource{URL: "./" + templateDir},
		Hash:     oldHash,
	}); err != nil {
		t.Fatalf("failed to write ign config: %v", err)
	}
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := config.SaveIgnVarJson(ignVarPath, &model.IgnVarJson{
		Variables: map[string]interface{}{},
	}); err != nil {
		t.Fatalf("failed to write ign vars: %v", err)
	}
	if err := os.Mkdir("output", 0755); err != nil {
		t.Fatalf("failed to create preexisting output directory: %v", err)
	}
	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("failed to create manifest directory fixture: %v", err)
	}

	_, err := Checkout(context.Background(), CheckoutOptions{
		OutputDir: "output",
	})
	if err == nil {
		t.Fatalf("Checkout expected manifest save error")
	}

	if _, err := os.Stat(filepath.Join("output", "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("checkout did not roll back generated file after manifest save failure: %v", err)
	}
	if info, err := os.Stat("output"); err != nil || !info.IsDir() {
		t.Fatalf("checkout removed preexisting output directory, info=%v err=%v", info, err)
	}
	if _, err := os.Stat(filepath.Join("output", "docs")); !os.IsNotExist(err) {
		t.Fatalf("checkout left empty generated subdirectory after manifest save failure: %v", err)
	}
}

func writeLocalTemplate(t *testing.T, templateDir string, templateConfig *model.IgnJson, files map[string]string) {
	t.Helper()

	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}
	templateConfigData, err := json.MarshalIndent(templateConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), templateConfigData, 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(templateDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create template file directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write template file: %v", err)
		}
	}
}

func singleFilePreparedCheckout() *PrepareCheckoutResult {
	template := &model.Template{
		Config: model.IgnJson{
			Name:      "single-file-template",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
			Hash:      strings.Repeat("b", 64),
		},
		Files: []model.TemplateFile{
			{
				Path:    filepath.Join("docs", "README.md"),
				Content: []byte("generated"),
				Mode:    0644,
			},
		},
	}
	return &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   model.TemplateRef{Provider: "local", Repo: "single-file-template"},
		NormalizedURL: "./template",
	}
}

func twoFilePreparedCheckout() *PrepareCheckoutResult {
	template := &model.Template{
		Config: model.IgnJson{
			Name:      "two-file-template",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
			Hash:      strings.Repeat("b", 64),
		},
		Files: []model.TemplateFile{
			{
				Path:    filepath.Join("docs", "README.md"),
				Content: []byte("generated"),
				Mode:    0644,
			},
			{
				Path:    filepath.Join("docs", "NEW.md"),
				Content: []byte("new"),
				Mode:    0644,
			},
		},
	}
	return &PrepareCheckoutResult{
		Template:      template,
		IgnJson:       &template.Config,
		TemplateRef:   model.TemplateRef{Provider: "local", Repo: "two-file-template"},
		NormalizedURL: "./template",
	}
}

func TestValidateVariables(t *testing.T) {
	ignJson := &model.IgnJson{
		Variables: map[string]model.VarDef{
			"required_string": {
				Type:     model.VarTypeString,
				Required: true,
			},
			"optional_string": {
				Type:     model.VarTypeString,
				Required: false,
			},
			"required_int": {
				Type:     model.VarTypeInt,
				Required: true,
			},
		},
	}

	tests := []struct {
		name    string
		vars    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "all required variables set",
			vars: map[string]interface{}{
				"required_string": "value",
				"required_int":    42,
			},
			wantErr: false,
		},
		{
			name: "missing required string",
			vars: map[string]interface{}{
				"required_int": 42,
			},
			wantErr: true,
			errMsg:  "required_string",
		},
		{
			name: "empty required string",
			vars: map[string]interface{}{
				"required_string": "",
				"required_int":    42,
			},
			wantErr: true,
			errMsg:  "required_string",
		},
		{
			name: "missing required int",
			vars: map[string]interface{}{
				"required_string": "value",
			},
			wantErr: true,
			errMsg:  "required_int",
		},
		{
			name: "optional variable missing is OK",
			vars: map[string]interface{}{
				"required_string": "value",
				"required_int":    42,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := parser.NewMapVariables(tt.vars)
			err := ValidateVariables(ignJson, vars)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestAppErrors(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() error
		wantType  AppErrorType
	}{
		{
			name:      "InitError",
			createErr: func() error { return NewInitError("test", nil) },
			wantType:  InitFailed,
		},
		{
			name:      "CheckoutError",
			createErr: func() error { return NewCheckoutError("test", nil) },
			wantType:  CheckoutFailed,
		},
		{
			name:      "VariableLoadError",
			createErr: func() error { return NewVariableLoadError("test", nil) },
			wantType:  VariableLoadFailed,
		},
		{
			name:      "TemplateFetchError",
			createErr: func() error { return NewTemplateFetchError("test", nil) },
			wantType:  TemplateFetchFailed,
		},
		{
			name:      "ValidationError",
			createErr: func() error { return NewValidationError("test", nil) },
			wantType:  ValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()

			appErr, ok := err.(*AppError)
			if !ok {
				t.Fatalf("Expected *AppError, got %T", err)
			}

			if appErr.Type != tt.wantType {
				t.Errorf("Expected error type %v, got %v", tt.wantType, appErr.Type)
			}

			if appErr.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}
