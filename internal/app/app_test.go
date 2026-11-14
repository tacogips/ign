package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
				Required:    true,
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

	// Check string variable (no default)
	if val, ok := vars["project_name"]; !ok || val != "" {
		t.Errorf("Expected project_name to be empty string, got %v", val)
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
