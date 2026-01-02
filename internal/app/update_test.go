package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

// setupTestTemplate creates a minimal template structure for testing
// Uses valid SHA256 hash format (64 hex characters)
func setupTestTemplate(t *testing.T, dir string, hash string) {
	t.Helper()

	// Create .ign directory
	ignDir := filepath.Join(dir, ".ign")
	if err := os.MkdirAll(ignDir, 0755); err != nil {
		t.Fatalf("Failed to create .ign directory: %v", err)
	}

	// Create ign.json
	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL:  "https://github.com/test/template",
			Ref:  "main",
			Path: "",
		},
		Hash: hash,
	}
	ignConfigPath := filepath.Join(ignDir, "ign.json")
	if err := config.SaveIgnConfig(ignConfigPath, ignConfig); err != nil {
		t.Fatalf("Failed to save ign.json: %v", err)
	}

	// Create ign-var.json
	ignVar := &model.IgnVarJson{
		Variables: map[string]interface{}{
			"project_name": "test-project",
			"version":      "1.0.0",
		},
	}
	ignVarPath := filepath.Join(ignDir, "ign-var.json")
	if err := config.SaveIgnVarJson(ignVarPath, ignVar); err != nil {
		t.Fatalf("Failed to save ign-var.json: %v", err)
	}
}

// Valid SHA256 hashes for testing (64 hex characters)
const (
	testHash1 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testHash2 = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

func TestPrepareUpdate_NoIgnDirectory(t *testing.T) {
	// Create temporary directory without .ign
	tempDir := t.TempDir()

	opts := UpdateOptions{
		OutputDir: tempDir,
	}

	result, err := PrepareUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error when .ign directory does not exist")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	// Check error type
	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("Expected *AppError, got %T", err)
	}
	if appErr.Type != ValidationFailed {
		t.Errorf("Expected ValidationFailed error type, got %v", appErr.Type)
	}
}

func TestPrepareUpdate_MissingIgnConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create .ign directory but no ign.json
	ignDir := filepath.Join(tempDir, ".ign")
	if err := os.MkdirAll(ignDir, 0755); err != nil {
		t.Fatalf("Failed to create .ign directory: %v", err)
	}

	opts := UpdateOptions{
		OutputDir: tempDir,
	}

	result, err := PrepareUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error when ign.json is missing")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestFindVariableChanges(t *testing.T) {
	tests := []struct {
		name            string
		existing        map[string]interface{}
		templateVars    map[string]model.VarDef
		wantNewVars     []string
		wantRemovedVars []string
	}{
		{
			name: "no changes",
			existing: map[string]interface{}{
				"var1": "value1",
				"var2": "value2",
			},
			templateVars: map[string]model.VarDef{
				"var1": {Type: model.VarTypeString},
				"var2": {Type: model.VarTypeString},
			},
			wantNewVars:     []string{},
			wantRemovedVars: []string{},
		},
		{
			name: "new variables added",
			existing: map[string]interface{}{
				"var1": "value1",
			},
			templateVars: map[string]model.VarDef{
				"var1": {Type: model.VarTypeString},
				"var2": {Type: model.VarTypeString},
				"var3": {Type: model.VarTypeInt},
			},
			wantNewVars:     []string{"var2", "var3"},
			wantRemovedVars: []string{},
		},
		{
			name: "variables removed",
			existing: map[string]interface{}{
				"var1": "value1",
				"var2": "value2",
				"var3": "value3",
			},
			templateVars: map[string]model.VarDef{
				"var1": {Type: model.VarTypeString},
			},
			wantNewVars:     []string{},
			wantRemovedVars: []string{"var2", "var3"},
		},
		{
			name: "both new and removed variables",
			existing: map[string]interface{}{
				"old_var1": "value1",
				"old_var2": "value2",
			},
			templateVars: map[string]model.VarDef{
				"new_var1": {Type: model.VarTypeString},
				"new_var2": {Type: model.VarTypeInt},
			},
			wantNewVars:     []string{"new_var1", "new_var2"},
			wantRemovedVars: []string{"old_var1", "old_var2"},
		},
		{
			name:     "empty existing variables",
			existing: map[string]interface{}{},
			templateVars: map[string]model.VarDef{
				"var1": {Type: model.VarTypeString},
			},
			wantNewVars:     []string{"var1"},
			wantRemovedVars: []string{},
		},
		{
			name: "empty template variables",
			existing: map[string]interface{}{
				"var1": "value1",
			},
			templateVars:    map[string]model.VarDef{},
			wantNewVars:     []string{},
			wantRemovedVars: []string{"var1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newVars, removedVars := findVariableChanges(tt.existing, tt.templateVars)

			if len(newVars) != len(tt.wantNewVars) {
				t.Errorf("Expected %d new variables, got %d: %v", len(tt.wantNewVars), len(newVars), newVars)
			}
			for i, want := range tt.wantNewVars {
				if i >= len(newVars) || newVars[i] != want {
					t.Errorf("Expected new variable %d to be %q, got %q", i, want, newVars[i])
				}
			}

			if len(removedVars) != len(tt.wantRemovedVars) {
				t.Errorf("Expected %d removed variables, got %d: %v", len(tt.wantRemovedVars), len(removedVars), removedVars)
			}
			for i, want := range tt.wantRemovedVars {
				if i >= len(removedVars) || removedVars[i] != want {
					t.Errorf("Expected removed variable %d to be %q, got %q", i, want, removedVars[i])
				}
			}
		})
	}
}

func TestCompleteUpdate_NilPrepareResult(t *testing.T) {
	opts := CompleteUpdateOptions{
		PrepareResult: nil,
		OutputDir:     "/tmp/test",
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error when PrepareResult is nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	// Verify error message contains expected text
	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("Expected *AppError, got %T", err)
	}
	if appErr.Type != ValidationFailed {
		t.Errorf("Expected ValidationFailed error type, got %v", appErr.Type)
	}
}

func TestCompleteUpdate_NilIgnJson(t *testing.T) {
	prep := &PrepareUpdateResult{
		IgnJson: nil,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     "/tmp/test",
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error when IgnJson is nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestCompleteUpdate_EmptyOutputDirectory(t *testing.T) {
	prep := &PrepareUpdateResult{
		IgnJson: &model.IgnJson{
			Name:      "test",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		},
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     "",
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error when output directory is empty")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	// Verify error message mentions "update output directory"
	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("Expected *AppError, got %T", err)
	}
	if appErr.Type != ValidationFailed {
		t.Errorf("Expected ValidationFailed error type, got %v", appErr.Type)
	}
}

func TestCompleteUpdate_VariableMerging(t *testing.T) {
	tempDir := t.TempDir()

	// Setup .ign directory with configuration
	setupTestTemplate(t, tempDir, testHash1)

	// Create template
	template := &model.Template{
		Config: model.IgnJson{
			Name:    "test-template",
			Version: "1.0.0",
			Hash:    testHash2,
			Variables: map[string]model.VarDef{
				"project_name": {
					Type:        model.VarTypeString,
					Description: "Project name",
					Required:    true,
				},
				"new_var": {
					Type:        model.VarTypeString,
					Description: "New variable",
					Default:     "default_value",
					Required:    false,
				},
			},
		},
		Files: []model.TemplateFile{
			{
				Path:    "README.md",
				Content: []byte("# Project"),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL:  "https://github.com/test/template",
			Ref:  "main",
			Path: "",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{"project_name": "test-project", "version": "1.0.0"},
		NewVars:       []string{"new_var"},
		RemovedVars:   []string{"version"},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{"new_var": "new_value"},
		OutputDir:     tempDir,
		DryRun:        true,
		Overwrite:     false,
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify results
	if !result.HashChanged {
		t.Error("Expected HashChanged to be true")
	}

	if len(result.NewVariables) != 1 || result.NewVariables[0] != "new_var" {
		t.Errorf("Expected NewVariables to be [new_var], got %v", result.NewVariables)
	}

	if len(result.RemovedVariables) != 1 || result.RemovedVariables[0] != "version" {
		t.Errorf("Expected RemovedVariables to be [version], got %v", result.RemovedVariables)
	}
}

func TestCompleteUpdate_DryRunMode(t *testing.T) {
	tempDir := t.TempDir()

	// Setup .ign directory
	setupTestTemplate(t, tempDir, testHash1)

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test-template",
			Version:   "1.0.0",
			Hash:      testHash2,
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{
				Path:    "test.txt",
				Content: []byte("test content"),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     tempDir,
		DryRun:        true,
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	// In dry-run mode, configuration files should NOT be updated
	// Verify ign.json still has old hash
	ignConfigPath := filepath.Join(tempDir, ".ign", "ign.json")
	loadedConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("Failed to load ign.json: %v", err)
	}

	if loadedConfig.Hash != testHash1 {
		t.Errorf("Expected hash to remain '%s' in dry-run mode, got %s", testHash1, loadedConfig.Hash)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestCompleteUpdate_ConfigurationFileUpdates(t *testing.T) {
	tempDir := t.TempDir()

	// Setup .ign directory
	setupTestTemplate(t, tempDir, testHash1)

	template := &model.Template{
		Config: model.IgnJson{
			Name:    "test-template",
			Version: "2.0.0",
			Hash:    testHash2,
			Variables: map[string]model.VarDef{
				"var1": {
					Type:     model.VarTypeString,
					Required: true,
				},
			},
		},
		Files: []model.TemplateFile{
			{
				Path:    "file.txt",
				Content: []byte("content"),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{"var1"},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{"var1": "value1"},
		OutputDir:     tempDir,
		DryRun:        false,
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify ign.json was updated with new hash
	ignConfigPath := filepath.Join(tempDir, ".ign", "ign.json")
	loadedConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		t.Fatalf("Failed to load ign.json: %v", err)
	}

	if loadedConfig.Hash != testHash2 {
		t.Errorf("Expected hash to be '%s', got %s", testHash2, loadedConfig.Hash)
	}

	if loadedConfig.Metadata == nil {
		t.Error("Expected metadata to be set")
	}

	// Verify ign-var.json was updated with new variables
	ignVarPath := filepath.Join(tempDir, ".ign", "ign-var.json")
	loadedVar, err := config.LoadIgnVarJson(ignVarPath)
	if err != nil {
		t.Fatalf("Failed to load ign-var.json: %v", err)
	}

	if val, ok := loadedVar.Variables["var1"]; !ok || val != "value1" {
		t.Errorf("Expected var1 to be 'value1', got %v", val)
	}

	// Note: ign-var.json no longer contains metadata (it's only in ign.json)
}

func TestGetNewVariableDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		prep     *PrepareUpdateResult
		wantVars []string
	}{
		{
			name: "returns definitions for new variables",
			prep: &PrepareUpdateResult{
				NewVars: []string{"var1", "var2"},
				IgnJson: &model.IgnJson{
					Variables: map[string]model.VarDef{
						"var1": {Type: model.VarTypeString},
						"var2": {Type: model.VarTypeInt},
						"var3": {Type: model.VarTypeBool},
					},
				},
			},
			wantVars: []string{"var1", "var2"},
		},
		{
			name:     "nil prepare result",
			prep:     nil,
			wantVars: []string{},
		},
		{
			name: "nil ign json",
			prep: &PrepareUpdateResult{
				NewVars: []string{"var1"},
				IgnJson: nil,
			},
			wantVars: []string{},
		},
		{
			name: "empty new vars",
			prep: &PrepareUpdateResult{
				NewVars: []string{},
				IgnJson: &model.IgnJson{
					Variables: map[string]model.VarDef{
						"var1": {Type: model.VarTypeString},
					},
				},
			},
			wantVars: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNewVariableDefinitions(tt.prep)

			if len(result) != len(tt.wantVars) {
				t.Errorf("Expected %d variables, got %d", len(tt.wantVars), len(result))
			}

			for _, varName := range tt.wantVars {
				if _, ok := result[varName]; !ok {
					t.Errorf("Expected variable %q in result", varName)
				}
			}
		})
	}
}

func TestFilterVariablesForPrompt(t *testing.T) {
	tests := []struct {
		name          string
		newVarDefs    map[string]model.VarDef
		wantPromptFor []string
	}{
		{
			name: "required variables need prompting",
			newVarDefs: map[string]model.VarDef{
				"required_var": {
					Type:     model.VarTypeString,
					Required: true,
				},
				"optional_with_default": {
					Type:     model.VarTypeString,
					Required: false,
					Default:  "default_value",
				},
			},
			wantPromptFor: []string{"required_var"},
		},
		{
			name: "variables without defaults need prompting",
			newVarDefs: map[string]model.VarDef{
				"no_default": {
					Type:     model.VarTypeString,
					Required: false,
					Default:  nil,
				},
				"with_default": {
					Type:     model.VarTypeString,
					Required: false,
					Default:  "default",
				},
			},
			wantPromptFor: []string{"no_default"},
		},
		{
			name: "optional with defaults do not need prompting",
			newVarDefs: map[string]model.VarDef{
				"optional_default": {
					Type:     model.VarTypeString,
					Required: false,
					Default:  "value",
				},
			},
			wantPromptFor: []string{},
		},
		{
			name:          "empty input",
			newVarDefs:    map[string]model.VarDef{},
			wantPromptFor: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterVariablesForPrompt(tt.newVarDefs)

			if len(result) != len(tt.wantPromptFor) {
				t.Errorf("Expected %d variables for prompt, got %d", len(tt.wantPromptFor), len(result))
			}

			for _, varName := range tt.wantPromptFor {
				if _, ok := result[varName]; !ok {
					t.Errorf("Expected variable %q to need prompting", varName)
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name         string
		newVarDefs   map[string]model.VarDef
		providedVars map[string]interface{}
		wantResult   map[string]interface{}
	}{
		{
			name: "applies defaults for missing variables",
			newVarDefs: map[string]model.VarDef{
				"var1": {
					Type:    model.VarTypeString,
					Default: "default1",
				},
				"var2": {
					Type:    model.VarTypeInt,
					Default: 42,
				},
			},
			providedVars: map[string]interface{}{},
			wantResult: map[string]interface{}{
				"var1": "default1",
				"var2": 42,
			},
		},
		{
			name: "preserves provided variables",
			newVarDefs: map[string]model.VarDef{
				"var1": {
					Type:    model.VarTypeString,
					Default: "default1",
				},
			},
			providedVars: map[string]interface{}{
				"var1": "provided1",
			},
			wantResult: map[string]interface{}{
				"var1": "provided1",
			},
		},
		{
			name: "handles nil providedVars",
			newVarDefs: map[string]model.VarDef{
				"var1": {
					Type:    model.VarTypeString,
					Default: "default1",
				},
			},
			providedVars: nil,
			wantResult: map[string]interface{}{
				"var1": "default1",
			},
		},
		{
			name: "skips variables without defaults",
			newVarDefs: map[string]model.VarDef{
				"var1": {
					Type:    model.VarTypeString,
					Default: nil,
				},
				"var2": {
					Type:    model.VarTypeString,
					Default: "default2",
				},
			},
			providedVars: map[string]interface{}{},
			wantResult: map[string]interface{}{
				"var2": "default2",
			},
		},
		{
			name: "mixed scenario",
			newVarDefs: map[string]model.VarDef{
				"provided": {
					Type:    model.VarTypeString,
					Default: "default_provided",
				},
				"defaulted": {
					Type:    model.VarTypeString,
					Default: "default_value",
				},
				"no_default": {
					Type:    model.VarTypeString,
					Default: nil,
				},
			},
			providedVars: map[string]interface{}{
				"provided": "user_value",
			},
			wantResult: map[string]interface{}{
				"provided":  "user_value",
				"defaulted": "default_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyDefaults(tt.newVarDefs, tt.providedVars)

			if len(result) != len(tt.wantResult) {
				t.Errorf("Expected %d variables in result, got %d", len(tt.wantResult), len(result))
			}

			for key, wantVal := range tt.wantResult {
				gotVal, ok := result[key]
				if !ok {
					t.Errorf("Expected variable %q in result", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("Variable %q: expected %v, got %v", key, wantVal, gotVal)
				}
			}

			// Ensure no extra variables
			for key := range result {
				if _, ok := tt.wantResult[key]; !ok {
					t.Errorf("Unexpected variable %q in result", key)
				}
			}
		})
	}
}

func TestFormatVariableChanges(t *testing.T) {
	tests := []struct {
		name       string
		prep       *PrepareUpdateResult
		wantOutput string
	}{
		{
			name: "both new and removed variables",
			prep: &PrepareUpdateResult{
				NewVars:     []string{"new1", "new2"},
				RemovedVars: []string{"old1", "old2"},
			},
			wantOutput: "New variables: [new1 new2]\nRemoved variables: [old1 old2]\n",
		},
		{
			name: "only new variables",
			prep: &PrepareUpdateResult{
				NewVars:     []string{"new1"},
				RemovedVars: []string{},
			},
			wantOutput: "New variables: [new1]\n",
		},
		{
			name: "only removed variables",
			prep: &PrepareUpdateResult{
				NewVars:     []string{},
				RemovedVars: []string{"old1"},
			},
			wantOutput: "Removed variables: [old1]\n",
		},
		{
			name: "no changes",
			prep: &PrepareUpdateResult{
				NewVars:     []string{},
				RemovedVars: []string{},
			},
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVariableChanges(tt.prep)
			if result != tt.wantOutput {
				t.Errorf("Expected output:\n%s\nGot:\n%s", tt.wantOutput, result)
			}
		})
	}
}

func TestCompleteUpdate_InvalidOutputDirectory(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{
				Path:    "test.txt",
				Content: []byte("test"),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     "../../../etc/passwd", // Invalid path with traversal
		DryRun:        false,
	}

	// This should fail due to invalid output directory
	result, err := CompleteUpdate(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error with invalid output directory")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	// Verify it's a validation error
	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("Expected *AppError, got %T", err)
	}
	if appErr.Type != ValidationFailed {
		t.Errorf("Expected ValidationFailed error type, got %v", appErr.Type)
	}
}

func TestCompleteUpdate_EmptyVariables(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test-template",
			Version:   "1.0.0",
			Hash:      testHash2,
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{
				Path:    "empty.txt",
				Content: []byte(""),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	opts := CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     tempDir,
		DryRun:        true,
	}

	result, err := CompleteUpdate(context.Background(), opts)
	if err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.NewVariables) != 0 {
		t.Errorf("Expected no new variables, got %v", result.NewVariables)
	}

	if len(result.RemovedVariables) != 0 {
		t.Errorf("Expected no removed variables, got %v", result.RemovedVariables)
	}
}

func TestCompleteUpdate_OverwriteFlag(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)

	// Create an existing file that will be overwritten
	existingFilePath := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("original content")
	if err := os.WriteFile(existingFilePath, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	newContent := []byte("new content from template")
	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test-template",
			Version:   "1.0.0",
			Hash:      testHash2,
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{
				Path:    "test.txt",
				Content: newContent,
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	t.Run("without overwrite - file should be skipped", func(t *testing.T) {
		// Reset file to original content
		if err := os.WriteFile(existingFilePath, originalContent, 0644); err != nil {
			t.Fatalf("Failed to reset file: %v", err)
		}

		opts := CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     false, // Do not overwrite
			DryRun:        false,
		}

		result, err := CompleteUpdate(context.Background(), opts)
		if err != nil {
			t.Fatalf("CompleteUpdate failed: %v", err)
		}

		if result.FilesSkipped != 1 {
			t.Errorf("Expected 1 file skipped, got %d", result.FilesSkipped)
		}
		if result.FilesOverwritten != 0 {
			t.Errorf("Expected 0 files overwritten, got %d", result.FilesOverwritten)
		}

		// Verify file content unchanged
		content, err := os.ReadFile(existingFilePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != string(originalContent) {
			t.Errorf("File content should not have changed. Got: %s, Expected: %s", string(content), string(originalContent))
		}
	})

	t.Run("with overwrite - file should be overwritten", func(t *testing.T) {
		// Reset file to original content
		if err := os.WriteFile(existingFilePath, originalContent, 0644); err != nil {
			t.Fatalf("Failed to reset file: %v", err)
		}

		opts := CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     true, // Overwrite enabled
			DryRun:        false,
		}

		result, err := CompleteUpdate(context.Background(), opts)
		if err != nil {
			t.Fatalf("CompleteUpdate failed: %v", err)
		}

		if result.FilesSkipped != 0 {
			t.Errorf("Expected 0 files skipped, got %d", result.FilesSkipped)
		}
		if result.FilesOverwritten != 1 {
			t.Errorf("Expected 1 file overwritten, got %d", result.FilesOverwritten)
		}

		// Verify file content changed
		content, err := os.ReadFile(existingFilePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != string(newContent) {
			t.Errorf("File content should have been overwritten. Got: %s, Expected: %s", string(content), string(newContent))
		}
	})
}

func TestCompleteUpdate_DryRunWithOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	setupTestTemplate(t, tempDir, testHash1)

	// Create an existing file
	existingFilePath := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("original content")
	if err := os.WriteFile(existingFilePath, originalContent, 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	template := &model.Template{
		Config: model.IgnJson{
			Name:      "test-template",
			Version:   "1.0.0",
			Hash:      testHash2,
			Variables: map[string]model.VarDef{},
		},
		Files: []model.TemplateFile{
			{
				Path:    "test.txt",
				Content: []byte("new content"),
			},
		},
	}

	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{
			URL: "https://github.com/test/template",
		},
		Hash: testHash1,
	}

	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		NewVars:       []string{},
		RemovedVars:   []string{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(tempDir, ".ign", "ign.json"),
		IgnVarPath:    filepath.Join(tempDir, ".ign", "ign-var.json"),
		IgnConfig:     ignConfig,
	}

	t.Run("dry-run without overwrite - shows skip", func(t *testing.T) {
		opts := CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     false,
			DryRun:        true,
		}

		result, err := CompleteUpdate(context.Background(), opts)
		if err != nil {
			t.Fatalf("CompleteUpdate failed: %v", err)
		}

		if result.FilesSkipped != 1 {
			t.Errorf("Expected 1 file skipped in dry-run, got %d", result.FilesSkipped)
		}

		// Verify DryRunFiles shows WouldSkip
		if len(result.DryRunFiles) != 1 {
			t.Fatalf("Expected 1 dry-run file, got %d", len(result.DryRunFiles))
		}
		if !result.DryRunFiles[0].WouldSkip {
			t.Error("Expected WouldSkip to be true")
		}

		// Verify file was not modified (dry-run)
		content, err := os.ReadFile(existingFilePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != string(originalContent) {
			t.Error("File should not have been modified in dry-run mode")
		}
	})

	t.Run("dry-run with overwrite - shows overwrite", func(t *testing.T) {
		opts := CompleteUpdateOptions{
			PrepareResult: prep,
			NewVariables:  map[string]interface{}{},
			OutputDir:     tempDir,
			Overwrite:     true,
			DryRun:        true,
		}

		result, err := CompleteUpdate(context.Background(), opts)
		if err != nil {
			t.Fatalf("CompleteUpdate failed: %v", err)
		}

		if result.FilesOverwritten != 1 {
			t.Errorf("Expected 1 file overwritten in dry-run, got %d", result.FilesOverwritten)
		}

		// Verify DryRunFiles shows WouldOverwrite
		if len(result.DryRunFiles) != 1 {
			t.Fatalf("Expected 1 dry-run file, got %d", len(result.DryRunFiles))
		}
		if !result.DryRunFiles[0].WouldOverwrite {
			t.Error("Expected WouldOverwrite to be true")
		}

		// Verify file was not modified (dry-run)
		content, err := os.ReadFile(existingFilePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != string(originalContent) {
			t.Error("File should not have been modified in dry-run mode")
		}
	})
}
