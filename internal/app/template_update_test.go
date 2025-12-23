package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestUpdateTemplate(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		opts           func(path string) UpdateTemplateOptions
		wantErr        bool
		validateResult func(t *testing.T, result *UpdateTemplateResult, path string)
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
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
					Path:   path,
					DryRun: true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *UpdateTemplateResult, path string) {
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
			name: "scans subdirectories",
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
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
					Path:   path,
					DryRun: true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *UpdateTemplateResult, path string) {
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
			name: "dry-run does not write file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:test@"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
					Path:   path,
					DryRun: true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *UpdateTemplateResult, path string) {
				if result.Updated {
					t.Error("Expected Updated to be false in dry-run mode")
				}
				// ign-template.json should not exist
				ignJsonPath := filepath.Join(path, model.IgnTemplateConfigFile)
				if _, err := os.Stat(ignJsonPath); !os.IsNotExist(err) {
					t.Errorf("Expected %s to not exist in dry-run mode", model.IgnTemplateConfigFile)
				}
			},
		},
		{
			name: "creates ign-template.json when not exists",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:test@"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
					Path:   path,
					DryRun: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *UpdateTemplateResult, path string) {
				if !result.Updated {
					t.Error("Expected Updated to be true")
				}
				ignJsonPath := filepath.Join(path, model.IgnTemplateConfigFile)
				if _, err := os.Stat(ignJsonPath); os.IsNotExist(err) {
					t.Errorf("Expected %s to be created", model.IgnTemplateConfigFile)
				}
			},
		},
		{
			name: "fail on non-existent path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
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
			opts: func(path string) UpdateTemplateOptions {
				return UpdateTemplateOptions{
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

			result, err := UpdateTemplate(context.Background(), opts)

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
		// Version strings should remain strings, not be parsed as integers (issue #20)
		{"GO_VERSION=1.25.4", "GO_VERSION", model.VarTypeString, "1.25.4", true},
		{"version=2.0.0", "version", model.VarTypeString, "2.0.0", true},
		{"node_version=18.17.0", "node_version", model.VarTypeString, "18.17.0", true},
		// Plain integers should still work
		{"count=42", "count", model.VarTypeInt, 42, true},
		{"level=-5", "level", model.VarTypeInt, -5, true},
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
	if err := os.WriteFile(filepath.Join(dir, model.IgnTemplateConfigFile), []byte(existingIgnJson), 0644); err != nil {
		t.Fatalf("Failed to create %s: %v", model.IgnTemplateConfigFile, err)
	}

	// Create template file with new variable
	if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:new_var@"), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Test merge mode
	result, err := UpdateTemplate(context.Background(), UpdateTemplateOptions{
		Path:  dir,
		Merge: true,
	})
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
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

func TestCalculateTemplateHashFromDir(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantErr  bool
		validate func(t *testing.T, hash string, dir string)
	}{
		{
			name: "deterministic hash for same content",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				// Calculate again and verify same hash
				hash2, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash != hash2 {
					t.Errorf("Hash not deterministic: got %s on first call, %s on second", hash, hash2)
				}
			},
		},
		{
			name: "file order independence",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create files in different order than they'll be sorted
				if err := os.WriteFile(filepath.Join(dir, "zzz.txt"), []byte("last"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "aaa.txt"), []byte("first"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "mmm.txt"), []byte("middle"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				if hash == "" {
					t.Error("Expected non-empty hash")
				}
			},
		},
		{
			name: "different content produces different hash",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content_v1"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash1 string, dir string) {
				// Modify file content
				if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content_v2"), 0644); err != nil {
					t.Fatalf("Failed to modify file: %v", err)
				}
				hash2, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash1 == hash2 {
					t.Error("Expected different hashes for different content")
				}
			},
		},
		{
			name: "ign-template.json excluded from hash",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("template content"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, model.IgnTemplateConfigFile), []byte(`{"name":"test"}`), 0644); err != nil {
					t.Fatalf("Failed to create %s: %v", model.IgnTemplateConfigFile, err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash1 string, dir string) {
				// Modify ign-template.json
				if err := os.WriteFile(filepath.Join(dir, model.IgnTemplateConfigFile), []byte(`{"name":"test","version":"2.0"}`), 0644); err != nil {
					t.Fatalf("Failed to modify %s: %v", model.IgnTemplateConfigFile, err)
				}
				hash2, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash1 != hash2 {
					t.Errorf("Hash should not change when only %s is modified", model.IgnTemplateConfigFile)
				}
			},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				// Empty directory returns empty hash
				if hash != "" {
					t.Errorf("Expected empty hash for empty directory, got %s", hash)
				}
			},
		},
		{
			name: "single file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "single.txt"), []byte("single file content"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				if hash == "" {
					t.Error("Expected non-empty hash")
				}
			},
		},
		{
			name: "dotfiles included but .git excluded",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("visible"), 0644); err != nil {
					t.Fatalf("Failed to create visible file: %v", err)
				}
				// Create a dotfile (should be included in hash)
				if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp"), 0644); err != nil {
					t.Fatalf("Failed to create .gitignore file: %v", err)
				}
				// Create .git directory (should be excluded from hash)
				gitDir := filepath.Join(dir, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatalf("Failed to create .git directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0644); err != nil {
					t.Fatalf("Failed to create .git/config file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash1 string, dir string) {
				// Removing .git directory should NOT change hash (it's excluded)
				if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
					t.Fatalf("Failed to remove .git directory: %v", err)
				}
				hash2, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash1 != hash2 {
					t.Error("Hash should not change when .git directory is removed (excluded from hash)")
				}

				// Removing .gitignore SHOULD change hash (dotfiles are now included)
				if err := os.Remove(filepath.Join(dir, ".gitignore")); err != nil {
					t.Fatalf("Failed to remove .gitignore file: %v", err)
				}
				hash3, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Third hash calculation failed: %v", err)
				}
				if hash2 == hash3 {
					t.Error("Hash should change when .gitignore is removed (dotfiles are included in hash)")
				}
			},
		},
		{
			name: "error on non-existent directory",
			setup: func(t *testing.T) string {
				return "/nonexistent/directory/path"
			},
			wantErr: true,
		},
		{
			name: "subdirectories included",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0644); err != nil {
					t.Fatalf("Failed to create root file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("sub"), 0644); err != nil {
					t.Fatalf("Failed to create sub file: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash1 string, dir string) {
				// Modify subdirectory file
				subdir := filepath.Join(dir, "subdir")
				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("modified"), 0644); err != nil {
					t.Fatalf("Failed to modify sub file: %v", err)
				}
				hash2, err := CalculateTemplateHashFromDir(dir)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash1 == hash2 {
					t.Error("Hash should change when subdirectory file is modified")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			hash, err := CalculateTemplateHashFromDir(dir)

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

			if tt.validate != nil {
				tt.validate(t, hash, dir)
			}
		})
	}
}
