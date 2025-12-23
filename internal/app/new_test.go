package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestAvailableScaffoldTypes(t *testing.T) {
	types, err := AvailableScaffoldTypes()
	if err != nil {
		t.Fatalf("AvailableScaffoldTypes() error = %v", err)
	}

	if len(types) == 0 {
		t.Error("Expected at least one scaffold type, got none")
	}

	// Check that 'default' type exists
	found := false
	for _, typ := range types {
		if typ == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'default' scaffold type, got %v", types)
	}
}

func TestNewTemplate(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		opts        func(path string) NewTemplateOptions
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *NewTemplateResult, path string)
	}{
		{
			name: "create new template in empty directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-template")
			},
			opts: func(path string) NewTemplateOptions {
				return NewTemplateOptions{
					Path:  path,
					Type:  "default",
					Force: false,
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *NewTemplateResult, path string) {
				if result.FilesCreated < 1 {
					t.Errorf("Expected at least 1 file created, got %d", result.FilesCreated)
				}

				// Check ign-template.json exists
				ignJsonPath := filepath.Join(path, model.IgnTemplateConfigFile)
				if _, err := os.Stat(ignJsonPath); os.IsNotExist(err) {
					t.Errorf("Expected %s to be created", model.IgnTemplateConfigFile)
				}

				// Check README.md exists
				readmePath := filepath.Join(path, "README.md")
				if _, err := os.Stat(readmePath); os.IsNotExist(err) {
					t.Error("Expected README.md to be created")
				}
			},
		},
		{
			name: "fail on non-empty directory without force",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "non-empty")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				// Create a file to make directory non-empty
				if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			opts: func(path string) NewTemplateOptions {
				return NewTemplateOptions{
					Path:  path,
					Type:  "default",
					Force: false,
				}
			},
			wantErr:     true,
			errContains: "not empty",
		},
		{
			name: "succeed on non-empty directory with force",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "force-overwrite")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return dir
			},
			opts: func(path string) NewTemplateOptions {
				return NewTemplateOptions{
					Path:  path,
					Type:  "default",
					Force: true,
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *NewTemplateResult, path string) {
				if result.FilesCreated < 1 {
					t.Errorf("Expected at least 1 file created, got %d", result.FilesCreated)
				}
			},
		},
		{
			name: "fail on unknown scaffold type",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "unknown-type")
			},
			opts: func(path string) NewTemplateOptions {
				return NewTemplateOptions{
					Path:  path,
					Type:  "nonexistent-type",
					Force: false,
				}
			},
			wantErr:     true,
			errContains: "unknown scaffold type",
		},
		{
			name: "fail when path is existing file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "existing-file")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return filePath
			},
			opts: func(path string) NewTemplateOptions {
				return NewTemplateOptions{
					Path:  path,
					Type:  "default",
					Force: false,
				}
			},
			wantErr:     true,
			errContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			opts := tt.opts(path)

			result, err := NewTemplate(context.Background(), opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTemplate() expected error, got nil")
					return
				}
				if tt.errContains != "" {
					if !containsString(err.Error(), tt.errContains) {
						t.Errorf("NewTemplate() error = %v, want error containing %q", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("NewTemplate() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, result, path)
			}
		})
	}
}

func TestNewTemplateCreatesValidTemplate(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "valid-template")

	result, err := NewTemplate(context.Background(), NewTemplateOptions{
		Path:  targetPath,
		Type:  "default",
		Force: false,
	})
	if err != nil {
		t.Fatalf("NewTemplate() error = %v", err)
	}

	// Verify result path is absolute
	if !filepath.IsAbs(result.Path) {
		t.Errorf("Expected absolute path, got %s", result.Path)
	}

	// Verify files list matches FilesCreated count
	if len(result.Files) != result.FilesCreated {
		t.Errorf("Files list length (%d) != FilesCreated (%d)", len(result.Files), result.FilesCreated)
	}

	// Verify each file in the list exists
	for _, file := range result.Files {
		fullPath := filepath.Join(result.Path, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Listed file does not exist: %s", file)
		}
	}

	// Verify ign-template.json is valid JSON
	ignJsonPath := filepath.Join(result.Path, model.IgnTemplateConfigFile)
	content, err := os.ReadFile(ignJsonPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", model.IgnTemplateConfigFile, err)
	}
	if len(content) == 0 {
		t.Errorf("%s is empty", model.IgnTemplateConfigFile)
	}

	// Check it contains expected fields
	contentStr := string(content)
	expectedFields := []string{"name", "version", "variables"}
	for _, field := range expectedFields {
		if !containsString(contentStr, field) {
			t.Errorf("%s missing expected field: %s", model.IgnTemplateConfigFile, field)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
