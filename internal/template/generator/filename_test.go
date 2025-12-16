package generator

import (
	"context"
	"testing"

	"github.com/tacogips/ign/internal/template/parser"
)

func TestProcessFilename(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		variables map[string]interface{}
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "simple filename with variable",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "handler",
			},
			want:    "handler.go",
			wantErr: false,
		},
		{
			name:     "directory with variable",
			filePath: "cmd/@ign-var:app_name@/main.go",
			variables: map[string]interface{}{
				"app_name": "myapp",
			},
			want:    "cmd/myapp/main.go",
			wantErr: false,
		},
		{
			name:     "multiple variables in path",
			filePath: "@ign-var:module@/@ign-var:type@/@ign-var:name@.go",
			variables: map[string]interface{}{
				"module": "api",
				"type":   "handlers",
				"name":   "user",
			},
			want:    "api/handlers/user.go",
			wantErr: false,
		},
		{
			name:     "variable with default value - provided",
			filePath: "config-@ign-var:env=dev@.yaml",
			variables: map[string]interface{}{
				"env": "production",
			},
			want:    "config-production.yaml",
			wantErr: false,
		},
		{
			name:      "variable with default value - not provided",
			filePath:  "config-@ign-var:env=dev@.yaml",
			variables: map[string]interface{}{},
			want:      "config-dev.yaml",
			wantErr:   false,
		},
		{
			name:     "no variables in path",
			filePath: "internal/app/main.go",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "internal/app/main.go",
			wantErr: false,
		},
		{
			name:     "variable in filename and directory",
			filePath: "@ign-var:package@/@ign-var:package@.go",
			variables: map[string]interface{}{
				"package": "utils",
			},
			want:    "utils/utils.go",
			wantErr: false,
		},
		{
			name:     "variable with type annotation",
			filePath: "version-@ign-var:ver:int@.txt",
			variables: map[string]interface{}{
				"ver": 2,
			},
			want:    "version-2.txt",
			wantErr: false,
		},
		// Error cases
		{
			name:     "missing required variable",
			filePath: "@ign-var:missing@.go",
			variables: map[string]interface{}{
				"other": "value",
			},
			wantErr: true,
			errMsg:  "required variable not found: missing",
		},
		{
			name:     "path traversal in variable value",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "../etc/passwd",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:     "empty variable value - filename only variable",
			filePath: "@ign-var:name@",
			variables: map[string]interface{}{
				"name": "",
			},
			wantErr: true,
			errMsg:  "resulted in empty value",
		},
		{
			name:     "whitespace variable value - filename only variable",
			filePath: "@ign-var:name@",
			variables: map[string]interface{}{
				"name": "   ",
			},
			wantErr: true,
			errMsg:  "resulted in empty value",
		},
		{
			name:     "empty variable with extension - produces .go (valid)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "",
			},
			want:    ".go",
			wantErr: false,
		},
		{
			name:     "path separator in variable value",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "dir/file",
			},
			wantErr: true,
			errMsg:  "contains path separator",
		},
		{
			name:     "backslash in variable value (Windows path separator)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "dir\\file",
			},
			wantErr: true,
			errMsg:  "contains path separator",
		},
		{
			name:     "dot-dot in variable value",
			filePath: "cmd/@ign-var:dir@/main.go",
			variables: map[string]interface{}{
				"dir": "..",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:     "type mismatch",
			filePath: "@ign-var:port:int@.txt",
			variables: map[string]interface{}{
				"port": "not-a-number",
			},
			wantErr: true,
			errMsg:  "type mismatch",
		},
	}

	p := parser.NewParser()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := parser.NewMapVariables(tt.variables)
			got, err := ProcessFilename(ctx, tt.filePath, vars, p)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ProcessFilename() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("ProcessFilename() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ProcessFilename() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ProcessFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateFilenameComponent(t *testing.T) {
	tests := []struct {
		name      string
		processed string
		original  string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid component",
			processed: "handler",
			original:  "@ign-var:name@",
			wantErr:   false,
		},
		{
			name:      "valid component with numbers",
			processed: "handler123",
			original:  "@ign-var:name@",
			wantErr:   false,
		},
		{
			name:      "valid component with dashes",
			processed: "my-handler",
			original:  "@ign-var:name@",
			wantErr:   false,
		},
		{
			name:      "empty after substitution",
			processed: "",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "resulted in empty value",
		},
		{
			name:      "whitespace only",
			processed: "   ",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "resulted in empty value",
		},
		{
			name:      "contains path traversal",
			processed: "..",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "path traversal",
		},
		{
			name:      "contains slash",
			processed: "dir/file",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "path separator",
		},
		{
			name:      "contains backslash",
			processed: "dir\\file",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "path separator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilenameComponent(tt.processed, tt.original)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateFilenameComponent() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("validateFilenameComponent() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateFilenameComponent() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateProcessedPath(t *testing.T) {
	tests := []struct {
		name      string
		processed string
		original  string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid relative path",
			processed: "cmd/myapp/main.go",
			original:  "cmd/@ign-var:app@/main.go",
			wantErr:   false,
		},
		{
			name:      "valid single file",
			processed: "handler.go",
			original:  "@ign-var:name@.go",
			wantErr:   false,
		},
		{
			name:      "absolute path",
			processed: "/etc/passwd",
			original:  "@ign-var:path@",
			wantErr:   true,
			errMsg:    "absolute path",
		},
		{
			name:      "path traversal at start",
			processed: "../etc/passwd",
			original:  "@ign-var:path@",
			wantErr:   true,
			errMsg:    "path traversal",
		},
		{
			name:      "current directory",
			processed: ".",
			original:  "@ign-var:name@",
			wantErr:   true,
			errMsg:    "current directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProcessedPath(tt.processed, tt.original)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateProcessedPath() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("validateProcessedPath() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateProcessedPath() unexpected error = %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
