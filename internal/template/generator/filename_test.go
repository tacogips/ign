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
		{
			name:     "plain @ character in filename (not a directive, no escaping needed)",
			filePath: "email@example.com.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "email@example.com.txt",
			wantErr: false,
		},
		{
			name:     "multiple @ characters in filename (not directives)",
			filePath: "user@host@domain.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "user@host@domain.txt",
			wantErr: false,
		},
		{
			name:     "@ at start and end of filename (not directives)",
			filePath: "@file@.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "@file@.txt",
			wantErr: false,
		},
		{
			name:     "raw directive to escape directive pattern in filename",
			filePath: "email@ign-raw:@@example.com.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "email@example.com.txt",
			wantErr: false,
		},
		{
			name:     "raw directive to escape @ign-var directive in filename",
			filePath: "doc-@ign-raw:@ign-var:name@@.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "doc-@ign-var:name@.txt",
			wantErr: false,
		},
		{
			name:     "raw directive preserves literal directive text",
			filePath: "template-@ign-raw:@ign-if:flag@@.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "template-@ign-if:flag@.txt",
			wantErr: false,
		},
		{
			name:     "ign-if directive is NOT processed in filename (kept as-is)",
			filePath: "config@ign-if:debug@-debug@ign-endif@.txt",
			variables: map[string]interface{}{
				"debug": true,
			},
			want:    "config@ign-if:debug@-debug@ign-endif@.txt",
			wantErr: false,
		},
		{
			name:     "ign-comment directive is NOT processed in filename (kept as-is)",
			filePath: "file@ign-comment:note@.txt",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "file@ign-comment:note@.txt",
			wantErr: false,
		},
		{
			name:     "ign-include directive is NOT processed in filename (kept as-is)",
			filePath: "output@ign-include:header.txt@.log",
			variables: map[string]interface{}{
				"unused": "value",
			},
			want:    "output@ign-include:header.txt@.log",
			wantErr: false,
		},
		{
			name:     "only ign-var and ign-raw are processed in filenames",
			filePath: "@ign-var:name@-@ign-raw:@@@ign-if:flag@-test.txt",
			variables: map[string]interface{}{
				"name": "output",
				"flag": true,
			},
			want:    "output-@@ign-if:flag@-test.txt",
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
			errMsg:  "forward slash", // Now caught at parser layer
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
			errMsg:  "forward slash", // Now caught at parser layer
		},
		{
			name:     "backslash in variable value (Windows path separator)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "dir\\file",
			},
			wantErr: true,
			errMsg:  "backslash", // Now caught at parser layer
		},
		{
			name:     "dot-dot in variable value",
			filePath: "cmd/@ign-var:dir@/main.go",
			variables: map[string]interface{}{
				"dir": "..",
			},
			wantErr: true,
			errMsg:  "parent directory", // Now caught at parser layer
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
		{
			name:     "null byte in variable value",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "file\x00name",
			},
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:     "colon in variable value (Windows drive separator)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "C:",
			},
			wantErr: true,
			errMsg:  "colon",
		},
		{
			name:     "colon in variable value (NTFS alternate data stream)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "file:stream",
			},
			wantErr: true,
			errMsg:  "colon",
		},
		{
			name:     "single dot in variable value (current directory)",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": ".",
			},
			wantErr: true,
			errMsg:  "current directory",
		},
		{
			name:     "path traversal with embedded dots at start",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "..hidden",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:     "path traversal with embedded dots in middle",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "data..backup",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:     "path traversal with embedded dots at end",
			filePath: "@ign-var:name@.go",
			variables: map[string]interface{}{
				"name": "test..",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		// Edge case: whitespace in middle of filename component
		{
			name:     "whitespace in middle of filename",
			filePath: "@ign-var:name@.txt",
			variables: map[string]interface{}{
				"name": "my file",
			},
			wantErr: false, // Spaces are allowed in filenames
			want:    "my file.txt",
		},
		{
			name:     "tab character in filename",
			filePath: "@ign-var:name@.txt",
			variables: map[string]interface{}{
				"name": "handler\tname",
			},
			wantErr: false, // Tab is allowed (though unusual)
			want:    "handler\tname.txt",
		},
		// Edge case: multiple path separators in sequence
		{
			name:     "multiple forward slashes",
			filePath: "@ign-var:name@.txt",
			variables: map[string]interface{}{
				"name": "dir//file",
			},
			wantErr: true,
			errMsg:  "forward slash",
		},
		{
			name:     "multiple backslashes",
			filePath: "@ign-var:name@.txt",
			variables: map[string]interface{}{
				"name": "path\\\\file",
			},
			wantErr: true,
			errMsg:  "backslash",
		},
		// Edge case: mixed valid and invalid characters
		{
			name:     "null byte in middle of valid filename",
			filePath: "@ign-var:name@.backup",
			variables: map[string]interface{}{
				"name": "handler\x00",
			},
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:     "colon in middle of filename",
			filePath: "@ign-var:name@.yaml",
			variables: map[string]interface{}{
				"name": "config:debug",
			},
			wantErr: true,
			errMsg:  "colon",
		},
		// Integration tests: Multiple path components
		{
			name:     "multiple components - each valid individually",
			filePath: "@ign-var:dir1@/@ign-var:dir2@/file.go",
			variables: map[string]interface{}{
				"dir1": "pkg",
				"dir2": "handler",
			},
			wantErr: false,
			want:    "pkg/handler/file.go",
		},
		{
			name:     "multiple components - second component with path separator",
			filePath: "@ign-var:dir1@/@ign-var:dir2@/file.go",
			variables: map[string]interface{}{
				"dir1": "pkg",
				"dir2": "../etc",
			},
			wantErr: true,
			errMsg:  "forward slash",
		},
		{
			name:     "multiple components - second component with path in value",
			filePath: "@ign-var:dir1@/@ign-var:dir2@",
			variables: map[string]interface{}{
				"dir1": "valid",
				"dir2": "etc/passwd",
			},
			wantErr: true,
			errMsg:  "forward slash",
		},
		{
			name:     "three-level deep path with valid values",
			filePath: "a/@ign-var:b@/c.go",
			variables: map[string]interface{}{
				"b": "middleware",
			},
			wantErr: false,
			want:    "a/middleware/c.go",
		},
		{
			name:     "three-level deep path with path separator in middle",
			filePath: "a/@ign-var:b@/c.go",
			variables: map[string]interface{}{
				"b": "../secret",
			},
			wantErr: true,
			errMsg:  "forward slash",
		},
		{
			name:     "path with absolute path in variable value",
			filePath: "@ign-var:dir@/@ign-var:file@",
			variables: map[string]interface{}{
				"dir":  "valid",
				"file": "/etc/passwd",
			},
			wantErr: true,
			errMsg:  "forward slash",
		},
		// Note: Directive syntax validation (e.g., @ign-var:name@ with colons) happens during parsing,
		// not during filename generation. The parser validates directive syntax separately from
		// variable value validation. See internal/template/parser/parser_test.go for directive syntax tests.
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
		// Note: null byte and colon validation is done in the parser layer
		// during variable substitution, not in validateFilenameComponent
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
		// Note: Windows absolute paths are only detected as absolute on Windows systems
		// due to filepath.IsAbs() platform-specific behavior. On Unix systems, these
		// look like relative paths, so we document expected behavior rather than enforce it.
		// Real protection happens at parser level where path separators are rejected.
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
