package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestCollectVars(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		opts           func(path string) CollectVarsOptions
		wantErr        bool
		validateResult func(t *testing.T, result *CollectVarsResult, path string)
	}{
		{
			name: "collect variables from template files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create a template file with variables
				content := `Hello @ign-var:name@!
Port: @ign-var:port:int=8080@
@ign-if:debug@
Debug mode enabled
@ign-endif@`
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path:      path,
					Recursive: false,
					DryRun:    true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CollectVarsResult, path string) {
				if result.FilesScanned != 1 {
					t.Errorf("Expected 1 file scanned, got %d", result.FilesScanned)
				}
				if len(result.Variables) != 3 {
					t.Errorf("Expected 3 variables, got %d", len(result.Variables))
				}

				// Check 'name' variable
				if v, ok := result.Variables["name"]; !ok {
					t.Error("Expected 'name' variable")
				} else if v.Required != true {
					t.Error("Expected 'name' to be required")
				}

				// Check 'port' variable
				if v, ok := result.Variables["port"]; !ok {
					t.Error("Expected 'port' variable")
				} else {
					if v.Type != model.VarTypeInt {
						t.Errorf("Expected 'port' type to be int, got %s", v.Type)
					}
					if !v.HasDefault || v.Default != 8080 {
						t.Errorf("Expected 'port' default to be 8080, got %v", v.Default)
					}
				}

				// Check 'debug' variable (from @ign-if:)
				if v, ok := result.Variables["debug"]; !ok {
					t.Error("Expected 'debug' variable")
				} else if v.Type != model.VarTypeBool {
					t.Errorf("Expected 'debug' type to be bool, got %s", v.Type)
				}
			},
		},
		{
			name: "recursive scan",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}

				// File in root
				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
					t.Fatalf("Failed to create root file: %v", err)
				}

				// File in subdir
				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("@ign-var:sub_var@"), 0644); err != nil {
					t.Fatalf("Failed to create sub file: %v", err)
				}

				return dir
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path:      path,
					Recursive: true,
					DryRun:    true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CollectVarsResult, path string) {
				if result.FilesScanned != 2 {
					t.Errorf("Expected 2 files scanned, got %d", result.FilesScanned)
				}
				if len(result.Variables) != 2 {
					t.Errorf("Expected 2 variables, got %d", len(result.Variables))
				}
				if _, ok := result.Variables["root_var"]; !ok {
					t.Error("Expected 'root_var' variable")
				}
				if _, ok := result.Variables["sub_var"]; !ok {
					t.Error("Expected 'sub_var' variable")
				}
			},
		},
		{
			name: "non-recursive does not scan subdirs",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}

				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
					t.Fatalf("Failed to create root file: %v", err)
				}

				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("@ign-var:sub_var@"), 0644); err != nil {
					t.Fatalf("Failed to create sub file: %v", err)
				}

				return dir
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path:      path,
					Recursive: false,
					DryRun:    true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CollectVarsResult, path string) {
				if result.FilesScanned != 1 {
					t.Errorf("Expected 1 file scanned, got %d", result.FilesScanned)
				}
				if len(result.Variables) != 1 {
					t.Errorf("Expected 1 variable, got %d", len(result.Variables))
				}
				if _, ok := result.Variables["root_var"]; !ok {
					t.Error("Expected 'root_var' variable")
				}
			},
		},
		{
			name: "dry-run does not write file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:test@"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path:   path,
					DryRun: true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CollectVarsResult, path string) {
				if result.Updated {
					t.Error("Expected Updated to be false in dry-run mode")
				}
				// ign.json should not exist
				ignJsonPath := filepath.Join(path, "ign.json")
				if _, err := os.Stat(ignJsonPath); !os.IsNotExist(err) {
					t.Error("Expected ign.json to not exist in dry-run mode")
				}
			},
		},
		{
			name: "creates ign.json when not exists",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:test@"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path:   path,
					DryRun: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CollectVarsResult, path string) {
				if !result.Updated {
					t.Error("Expected Updated to be true")
				}
				ignJsonPath := filepath.Join(path, "ign.json")
				if _, err := os.Stat(ignJsonPath); os.IsNotExist(err) {
					t.Error("Expected ign.json to be created")
				}
			},
		},
		{
			name: "fail on non-existent path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path: path,
				}
			},
			wantErr: true,
		},
		{
			name: "fail when path is a file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return filePath
			},
			opts: func(path string) CollectVarsOptions {
				return CollectVarsOptions{
					Path: path,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			opts := tt.opts(path)

			result, err := CollectVars(context.Background(), opts)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, path)
			}
		})
	}
}

func TestParseVarArgs(t *testing.T) {
	tests := []struct {
		args       string
		wantName   string
		wantType   model.VarType
		wantDefVal interface{}
		wantHasDef bool
	}{
		{"name", "name", "", nil, false},
		{"name:string", "name", model.VarTypeString, nil, false},
		{"port:int", "port", model.VarTypeInt, nil, false},
		{"debug:bool", "debug", model.VarTypeBool, nil, false},
		{"name=default", "name", model.VarTypeString, "default", true},
		{"port:int=8080", "port", model.VarTypeInt, 8080, true},
		{"debug:bool=true", "debug", model.VarTypeBool, true, true},
		{"flag:bool=false", "flag", model.VarTypeBool, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			name, varType, defVal, hasDef := parseVarArgs(tt.args)

			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if varType != tt.wantType {
				t.Errorf("varType = %q, want %q", varType, tt.wantType)
			}
			if hasDef != tt.wantHasDef {
				t.Errorf("hasDefault = %v, want %v", hasDef, tt.wantHasDef)
			}
			if hasDef && defVal != tt.wantDefVal {
				t.Errorf("default = %v, want %v", defVal, tt.wantDefVal)
			}
		})
	}
}

func TestMergeMode(t *testing.T) {
	dir := t.TempDir()

	// Create existing ign.json with one variable
	existingIgnJson := `{
  "name": "test",
  "version": "1.0.0",
  "variables": {
    "existing_var": {
      "type": "string",
      "description": "Existing variable",
      "required": true
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "ign.json"), []byte(existingIgnJson), 0644); err != nil {
		t.Fatalf("Failed to create ign.json: %v", err)
	}

	// Create template file with new variable
	if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:new_var@"), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Test merge mode
	result, err := CollectVars(context.Background(), CollectVarsOptions{
		Path:  dir,
		Merge: true,
	})
	if err != nil {
		t.Fatalf("CollectVars failed: %v", err)
	}

	// Should have new_var in NewVars
	found := false
	for _, name := range result.NewVars {
		if name == "new_var" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'new_var' in NewVars")
	}

	// UpdatedVars should be empty in merge mode
	if len(result.UpdatedVars) != 0 {
		t.Errorf("Expected empty UpdatedVars in merge mode, got %v", result.UpdatedVars)
	}
}
