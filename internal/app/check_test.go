package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckTemplate(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		opts           func(path string) CheckTemplateOptions
		wantErr        bool
		validateResult func(t *testing.T, result *CheckResult)
	}{
		{
			name: "valid template file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				content := `Hello @ign-var:name@!
Port: @ign-var:port:int=8080@
@ign-if:debug@
Debug mode enabled
@ign-endif@`
				filePath := filepath.Join(dir, "template.txt")
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return filePath
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: false,
					Verbose:   false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked, got %d", result.FilesChecked)
				}
				if result.FilesWithErrors != 0 {
					t.Errorf("Expected 0 files with errors, got %d", result.FilesWithErrors)
				}
				if len(result.Errors) != 0 {
					t.Errorf("Expected no errors, got %v", result.Errors)
				}
			},
		},
		{
			name: "invalid directive syntax",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Missing closing @ in directive
				content := `Hello @ign-var:name
This is invalid`
				filePath := filepath.Join(dir, "invalid.txt")
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return filePath
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path: path,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked, got %d", result.FilesChecked)
				}
				// Note: This may or may not be detected as an error depending on parser implementation
				// If no @ is found, it might just be treated as literal text
			},
		},
		{
			name: "directory with multiple files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()

				// Valid file
				if err := os.WriteFile(filepath.Join(dir, "valid.txt"), []byte("@ign-var:name@"), 0644); err != nil {
					t.Fatalf("Failed to create valid file: %v", err)
				}

				// Another valid file
				if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("port: @ign-var:port:int@"), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}

				// File without directives (should be skipped)
				if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("No directives here"), 0644); err != nil {
					t.Fatalf("Failed to create readme: %v", err)
				}

				return dir
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				// Only files with @ign- directives should be checked
				if result.FilesChecked != 2 {
					t.Errorf("Expected 2 files checked (only files with directives), got %d", result.FilesChecked)
				}
				if result.FilesWithErrors != 0 {
					t.Errorf("Expected 0 files with errors, got %d", result.FilesWithErrors)
				}
			},
		},
		{
			name: "recursive directory scan",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}

				// File in root
				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root@"), 0644); err != nil {
					t.Fatalf("Failed to create root file: %v", err)
				}

				// File in subdir
				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("@ign-var:sub@"), 0644); err != nil {
					t.Fatalf("Failed to create sub file: %v", err)
				}

				return dir
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: true,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 2 {
					t.Errorf("Expected 2 files checked, got %d", result.FilesChecked)
				}
				if result.FilesWithErrors != 0 {
					t.Errorf("Expected 0 files with errors, got %d", result.FilesWithErrors)
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

				if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("@ign-var:root@"), 0644); err != nil {
					t.Fatalf("Failed to create root file: %v", err)
				}

				if err := os.WriteFile(filepath.Join(subdir, "sub.txt"), []byte("@ign-var:sub@"), 0644); err != nil {
					t.Fatalf("Failed to create sub file: %v", err)
				}

				return dir
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked (non-recursive), got %d", result.FilesChecked)
				}
			},
		},
		{
			name: "skip binary files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()

				// Text file with directives
				if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte("@ign-var:name@"), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}

				// Binary file (PNG) - should be skipped
				if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47}, 0644); err != nil {
					t.Fatalf("Failed to create binary file: %v", err)
				}

				return dir
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked (binary file skipped), got %d", result.FilesChecked)
				}
			},
		},
		{
			name: "non-existent path",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path: path,
				}
			},
			wantErr: true,
		},
		{
			name: "unclosed conditional block",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				content := `@ign-if:debug@
This block is never closed`
				filePath := filepath.Join(dir, "unclosed.txt")
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return filePath
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path: path,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked, got %d", result.FilesChecked)
				}
				if result.FilesWithErrors != 1 {
					t.Errorf("Expected 1 file with errors (unclosed block), got %d", result.FilesWithErrors)
				}
				if len(result.Errors) == 0 {
					t.Error("Expected errors for unclosed block, got none")
				}
			},
		},
		{
			name: "valid filename variables",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// File with variables in filename
				content := `package @ign-var:module_name@`
				filePath := filepath.Join(dir, "@ign-var:module_name@.go")
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}
				return dir
			},
			opts: func(path string) CheckTemplateOptions {
				return CheckTemplateOptions{
					Path:      path,
					Recursive: false,
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *CheckResult) {
				if result.FilesChecked != 1 {
					t.Errorf("Expected 1 file checked, got %d", result.FilesChecked)
				}
				if result.FilesWithErrors != 0 {
					t.Errorf("Expected 0 files with errors, got %d", result.FilesWithErrors)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			opts := tt.opts(path)

			result, err := CheckTemplate(context.Background(), opts)

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
				tt.validateResult(t, result)
			}
		})
	}
}
