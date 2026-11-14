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
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
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
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
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
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
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
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
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
				hash3, err := CalculateTemplateHashFromDir(dir, nil)
				if err != nil {
					t.Fatalf("Third hash calculation failed: %v", err)
				}
				if hash2 == hash3 {
					t.Error("Hash should change when .gitignore is removed (dotfiles are included in hash)")
				}
			},
		},
		{
			name: "symlink to directory is skipped gracefully",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create a regular file
				if err := os.WriteFile(filepath.Join(dir, "regular.txt"), []byte("regular content"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				// Create a subdirectory with a file
				subDir := filepath.Join(dir, "realdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(subDir, "inner.txt"), []byte("inner content"), 0644); err != nil {
					t.Fatalf("Failed to create inner file: %v", err)
				}
				// Create a symlink to the subdirectory
				if err := os.Symlink(subDir, filepath.Join(dir, "linkdir")); err != nil {
					t.Fatalf("Failed to create symlink: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				if hash == "" {
					t.Error("Expected non-empty hash")
				}
				// Hash should be deterministic and not fail due to symlink
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
				if err != nil {
					t.Fatalf("Second hash calculation failed: %v", err)
				}
				if hash != hash2 {
					t.Errorf("Hash not deterministic with symlink: got %s then %s", hash, hash2)
				}
			},
		},
		{
			name: "dangling symlink is skipped gracefully",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "regular.txt"), []byte("regular content"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				// Create a dangling symlink
				if err := os.Symlink(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "broken-link.txt")); err != nil {
					t.Fatalf("Failed to create dangling symlink: %v", err)
				}
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, hash string, dir string) {
				if hash == "" {
					t.Error("Expected non-empty hash (from regular.txt)")
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
				hash2, err := CalculateTemplateHashFromDir(dir, nil)
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
			hash, err := CalculateTemplateHashFromDir(dir, nil)

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

func TestCalculateTemplateHashFromDir_IgnorePatterns(t *testing.T) {
	t.Run("ignore patterns skip directories in hash", func(t *testing.T) {
		dir := t.TempDir()

		// Create a file in root
		if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("main content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create a subdirectory that should be ignored
		claudeDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "config.json"), []byte("claude config"), 0644); err != nil {
			t.Fatalf("Failed to create .claude/config.json: %v", err)
		}

		// Hash WITHOUT ignore patterns (includes .claude/)
		hashWithClaude, err := CalculateTemplateHashFromDir(dir, nil)
		if err != nil {
			t.Fatalf("Hash calculation failed: %v", err)
		}

		// Hash WITH ignore patterns (excludes .claude/)
		hashWithoutClaude, err := CalculateTemplateHashFromDir(dir, []string{".claude"})
		if err != nil {
			t.Fatalf("Hash calculation with ignore failed: %v", err)
		}

		if hashWithClaude == hashWithoutClaude {
			t.Error("Hash should differ when .claude/ directory is ignored vs included")
		}

		// Hash with ignore should match hash of directory without the .claude/ dir
		dirNoIgnored := t.TempDir()
		if err := os.WriteFile(filepath.Join(dirNoIgnored, "main.txt"), []byte("main content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		hashClean, err := CalculateTemplateHashFromDir(dirNoIgnored, nil)
		if err != nil {
			t.Fatalf("Clean hash calculation failed: %v", err)
		}

		if hashWithoutClaude != hashClean {
			t.Error("Hash with ignored dir should equal hash of directory without that dir")
		}
	})

	t.Run("ignore patterns skip files in hash", func(t *testing.T) {
		dir := t.TempDir()

		if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("main content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Hash WITHOUT ignore patterns
		hashAll, err := CalculateTemplateHashFromDir(dir, nil)
		if err != nil {
			t.Fatalf("Hash calculation failed: %v", err)
		}

		// Hash WITH ignore patterns for *.log
		hashNoLogs, err := CalculateTemplateHashFromDir(dir, []string{"*.log"})
		if err != nil {
			t.Fatalf("Hash calculation with ignore failed: %v", err)
		}

		if hashAll == hashNoLogs {
			t.Error("Hash should differ when *.log files are ignored")
		}
	})

	t.Run("ignore patterns with nested directories", func(t *testing.T) {
		dir := t.TempDir()

		if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create nested ignored directory
		nodeModules := filepath.Join(dir, "node_modules")
		subPkg := filepath.Join(nodeModules, "pkg")
		if err := os.MkdirAll(subPkg, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subPkg, "index.js"), []byte("module"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Hash with node_modules ignored should not include any nested files
		hashIgnored, err := CalculateTemplateHashFromDir(dir, []string{"node_modules"})
		if err != nil {
			t.Fatalf("Hash calculation failed: %v", err)
		}

		// Should equal hash of directory with only root.txt
		dirClean := t.TempDir()
		if err := os.WriteFile(filepath.Join(dirClean, "root.txt"), []byte("root"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		hashClean, err := CalculateTemplateHashFromDir(dirClean, nil)
		if err != nil {
			t.Fatalf("Clean hash calculation failed: %v", err)
		}

		if hashIgnored != hashClean {
			t.Error("Hash with ignored nested dir should equal hash without that dir")
		}
	})

	t.Run("empty ignore patterns same as nil", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		hashNil, err := CalculateTemplateHashFromDir(dir, nil)
		if err != nil {
			t.Fatalf("Hash with nil failed: %v", err)
		}

		hashEmpty, err := CalculateTemplateHashFromDir(dir, []string{})
		if err != nil {
			t.Fatalf("Hash with empty failed: %v", err)
		}

		if hashNil != hashEmpty {
			t.Error("Hash with nil and empty ignore patterns should be identical")
		}
	})
}

func TestScanTemplateFiles_IgnorePatterns(t *testing.T) {
	t.Run("ignore patterns skip directories during scan", func(t *testing.T) {
		dir := t.TempDir()

		// Create a template file in root
		if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create a subdirectory with template files that should be ignored
		claudeDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "config.txt"), []byte("@ign-var:ignored_var@"), 0644); err != nil {
			t.Fatalf("Failed to create .claude/config.txt: %v", err)
		}

		// Scan WITHOUT ignore patterns
		resultAll := &UpdateTemplateResult{Variables: make(map[string]*CollectedVar)}
		err := scanTemplateFiles(context.Background(), dir, nil, resultAll)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if resultAll.FilesScanned != 2 {
			t.Errorf("Expected 2 files scanned without ignore, got %d", resultAll.FilesScanned)
		}
		if _, ok := resultAll.Variables["ignored_var"]; !ok {
			t.Error("Expected 'ignored_var' without ignore patterns")
		}

		// Scan WITH ignore patterns
		resultIgnored := &UpdateTemplateResult{Variables: make(map[string]*CollectedVar)}
		err = scanTemplateFiles(context.Background(), dir, []string{".claude"}, resultIgnored)
		if err != nil {
			t.Fatalf("Scan with ignore failed: %v", err)
		}

		if resultIgnored.FilesScanned != 1 {
			t.Errorf("Expected 1 file scanned with ignore, got %d", resultIgnored.FilesScanned)
		}
		if _, ok := resultIgnored.Variables["ignored_var"]; ok {
			t.Error("Expected 'ignored_var' to be excluded with ignore patterns")
		}
		if _, ok := resultIgnored.Variables["root_var"]; !ok {
			t.Error("Expected 'root_var' to still be included")
		}
	})

	t.Run("ignore patterns skip files during scan", func(t *testing.T) {
		dir := t.TempDir()

		if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("@ign-var:main_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.log"), []byte("@ign-var:log_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		result := &UpdateTemplateResult{Variables: make(map[string]*CollectedVar)}
		err := scanTemplateFiles(context.Background(), dir, []string{"*.log"}, result)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if result.FilesScanned != 1 {
			t.Errorf("Expected 1 file scanned, got %d", result.FilesScanned)
		}
		if _, ok := result.Variables["log_var"]; ok {
			t.Error("Expected 'log_var' to be excluded")
		}
		if _, ok := result.Variables["main_var"]; !ok {
			t.Error("Expected 'main_var' to be included")
		}
	})
}

func TestScanTemplateFiles_SymlinkHandling(t *testing.T) {
	t.Run("symlink to directory is traversed during scan", func(t *testing.T) {
		dir := t.TempDir()

		// Create a template file in root
		if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create a subdirectory with a template file
		subDir := filepath.Join(dir, "realdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "inner.txt"), []byte("@ign-var:inner_var@"), 0644); err != nil {
			t.Fatalf("Failed to create inner file: %v", err)
		}

		// Create a symlink to the subdirectory
		if err := os.Symlink(subDir, filepath.Join(dir, "linkdir")); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}

		result := &UpdateTemplateResult{Variables: make(map[string]*CollectedVar)}
		err := scanTemplateFiles(context.Background(), dir, nil, result)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		// Should have scanned root.txt, realdir/inner.txt, and linkdir/inner.txt
		if result.FilesScanned < 2 {
			t.Errorf("Expected at least 2 files scanned, got %d", result.FilesScanned)
		}

		if _, ok := result.Variables["root_var"]; !ok {
			t.Error("Expected 'root_var' to be found")
		}
		if _, ok := result.Variables["inner_var"]; !ok {
			t.Error("Expected 'inner_var' to be found")
		}
	})

	t.Run("dangling symlink is skipped during scan", func(t *testing.T) {
		dir := t.TempDir()

		// Create a template file in root
		if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create a dangling symlink
		if err := os.Symlink(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "broken-link.txt")); err != nil {
			t.Fatalf("Failed to create dangling symlink: %v", err)
		}

		result := &UpdateTemplateResult{Variables: make(map[string]*CollectedVar)}
		err := scanTemplateFiles(context.Background(), dir, nil, result)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if result.FilesScanned != 1 {
			t.Errorf("Expected 1 file scanned, got %d", result.FilesScanned)
		}
		if _, ok := result.Variables["root_var"]; !ok {
			t.Error("Expected 'root_var' to be found")
		}
	})
}

func TestUpdateTemplate_IgnorePatterns(t *testing.T) {
	t.Run("ignore patterns from existing config applied during scan and hash", func(t *testing.T) {
		dir := t.TempDir()

		// Create existing ign-template.json with ignore patterns
		existingConfig := `{
  "name": "test",
  "version": "1.0.0",
  "variables": {},
  "settings": {
    "ignore_patterns": [".claude"]
  }
}`
		if err := os.WriteFile(filepath.Join(dir, model.IgnTemplateConfigFile), []byte(existingConfig), 0644); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Create a template file in root
		if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("@ign-var:root_var@"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create .claude directory with files that should be ignored
		claudeDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "agent.md"), []byte("@ign-var:ignored_var@"), 0644); err != nil {
			t.Fatalf("Failed to create .claude/agent.md: %v", err)
		}

		result, err := UpdateTemplate(context.Background(), UpdateTemplateOptions{
			Path:   dir,
			DryRun: true,
		})
		if err != nil {
			t.Fatalf("UpdateTemplate failed: %v", err)
		}

		// Should only scan root file, not .claude/agent.md
		if result.FilesScanned != 1 {
			t.Errorf("Expected 1 file scanned, got %d", result.FilesScanned)
		}

		if _, ok := result.Variables["root_var"]; !ok {
			t.Error("Expected 'root_var' to be found")
		}
		if _, ok := result.Variables["ignored_var"]; ok {
			t.Error("Expected 'ignored_var' to be excluded by ignore patterns")
		}
	})
}
