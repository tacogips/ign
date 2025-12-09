package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create test variables
func testVars(vars map[string]interface{}) Variables {
	return NewMapVariables(vars)
}

// TestVarDirective tests @ign-var:NAME@ substitution
func TestVarDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "simple string substitution",
			input:    "name: @ign-var:project_name@",
			vars:     map[string]interface{}{"project_name": "my-app"},
			expected: "name: my-app",
			wantErr:  false,
		},
		{
			name:     "integer substitution",
			input:    "port: @ign-var:port@",
			vars:     map[string]interface{}{"port": 8080},
			expected: "port: 8080",
			wantErr:  false,
		},
		{
			name:     "boolean substitution",
			input:    "debug: @ign-var:debug@",
			vars:     map[string]interface{}{"debug": true},
			expected: "debug: true",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			input:    "host: @ign-var:host@, port: @ign-var:port@",
			vars:     map[string]interface{}{"host": "localhost", "port": 8080},
			expected: "host: localhost, port: 8080",
			wantErr:  false,
		},
		{
			name:     "missing variable",
			input:    "@ign-var:missing@",
			vars:     map[string]interface{}{},
			expected: "",
			wantErr:  true,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), []byte(tt.input), testVars(tt.vars))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

// TestCommentDirective tests @ign-comment:NAME@ with comment removal
func TestCommentDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "single-line comment //",
			input:    "// @ign-comment:code@",
			vars:     map[string]interface{}{"code": "fmt.Println(\"hello\")"},
			expected: "fmt.Println(\"hello\")",
			wantErr:  false,
		},
		{
			name:     "single-line comment #",
			input:    "# @ign-comment:code@",
			vars:     map[string]interface{}{"code": "print('hello')"},
			expected: "print('hello')",
			wantErr:  false,
		},
		{
			name:     "single-line comment --",
			input:    "-- @ign-comment:code@",
			vars:     map[string]interface{}{"code": "SELECT * FROM users"},
			expected: "SELECT * FROM users",
			wantErr:  false,
		},
		{
			name:     "block comment /* */",
			input:    "/* @ign-comment:code@ */",
			vars:     map[string]interface{}{"code": "const x = 1;"},
			expected: "const x = 1;",
			wantErr:  false,
		},
		{
			name:     "HTML comment",
			input:    "<!-- @ign-comment:tag@ -->",
			vars:     map[string]interface{}{"tag": "<div>content</div>"},
			expected: "<div>content</div>",
			wantErr:  false,
		},
		{
			name:     "with indentation",
			input:    "    // @ign-comment:code@",
			vars:     map[string]interface{}{"code": "return true"},
			expected: "    return true",
			wantErr:  false,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), []byte(tt.input), testVars(tt.vars))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

// TestRawDirective tests @ign-raw:CONTENT@
func TestRawDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
	}{
		{
			name:     "literal directive syntax",
			input:    "To use: @ign-raw:@ign-var:name@@",
			vars:     map[string]interface{}{},
			expected: "To use: @ign-var:name@",
		},
		{
			name:     "raw with other directives",
			input:    "Project: @ign-var:name@, Syntax: @ign-raw:@ign-var:x@@",
			vars:     map[string]interface{}{"name": "myapp"},
			expected: "Project: myapp, Syntax: @ign-var:x@",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), []byte(tt.input), testVars(tt.vars))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

// TestConditionalDirective tests @ign-if:/@ign-else@/@ign-endif@
func TestConditionalDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "if true",
			input:    "config:\n  @ign-if:use_tls@\n  tls: enabled\n  @ign-endif@",
			vars:     map[string]interface{}{"use_tls": true},
			expected: "config:\n  \n  tls: enabled\n  ",
			wantErr:  false,
		},
		{
			name:     "if false",
			input:    "config:\n  @ign-if:use_tls@\n  tls: enabled\n  @ign-endif@",
			vars:     map[string]interface{}{"use_tls": false},
			expected: "config:\n  ",
			wantErr:  false,
		},
		{
			name:     "if-else true",
			input:    "@ign-if:use_cache@\ncache: redis\n@ign-else@\ncache: memory\n@ign-endif@",
			vars:     map[string]interface{}{"use_cache": true},
			expected: "\ncache: redis\n",
			wantErr:  false,
		},
		{
			name:     "if-else false",
			input:    "@ign-if:use_cache@\ncache: redis\n@ign-else@\ncache: memory\n@ign-endif@",
			vars:     map[string]interface{}{"use_cache": false},
			expected: "\ncache: memory\n",
			wantErr:  false,
		},
		{
			name:     "nested conditionals",
			input:    "@ign-if:enable_api@\napi: true\n@ign-if:api_auth@\nauth: jwt\n@ign-endif@\n@ign-endif@",
			vars:     map[string]interface{}{"enable_api": true, "api_auth": true},
			expected: "\napi: true\n\nauth: jwt\n\n",
			wantErr:  false,
		},
		{
			name: "unclosed block",
			input: `@ign-if:test@
content`,
			vars:    map[string]interface{}{"test": true},
			wantErr: true,
		},
		{
			name: "non-boolean condition",
			input: `@ign-if:name@
content
@ign-endif@`,
			vars:    map[string]interface{}{"name": "string"},
			wantErr: true,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), []byte(tt.input), testVars(tt.vars))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

// TestIncludeDirective tests @ign-include:PATH@
func TestIncludeDirective(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test include file
	includeFile := filepath.Join(tmpDir, "header.txt")
	includeContent := "// Header: @ign-var:project@\n"
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatalf("failed to write include file: %v", err)
	}

	// Create main template
	mainFile := filepath.Join(tmpDir, "main.txt")
	mainContent := "@ign-include:header.txt@\ncode here"

	vars := testVars(map[string]interface{}{"project": "myapp"})
	pctx := &ParseContext{
		Variables:    vars,
		IncludeDepth: 0,
		IncludeStack: []string{},
		TemplateRoot: tmpDir,
		CurrentFile:  mainFile,
	}

	parser := NewParser()
	result, err := parser.ParseWithContext(context.Background(), []byte(mainContent), pctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := "// Header: myapp\n\ncode here"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestCircularInclude tests circular include detection
func TestCircularInclude(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create circular includes: a.txt -> b.txt -> a.txt
	fileA := filepath.Join(tmpDir, "a.txt")
	fileB := filepath.Join(tmpDir, "b.txt")

	if err := os.WriteFile(fileA, []byte("@ign-include:b.txt@"), 0644); err != nil {
		t.Fatalf("failed to write file a: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("@ign-include:a.txt@"), 0644); err != nil {
		t.Fatalf("failed to write file b: %v", err)
	}

	vars := testVars(map[string]interface{}{})
	pctx := &ParseContext{
		Variables:    vars,
		IncludeDepth: 0,
		IncludeStack: []string{},
		TemplateRoot: tmpDir,
		CurrentFile:  fileA,
	}

	parser := NewParser()
	_, err = parser.ParseWithContext(context.Background(), []byte("@ign-include:b.txt@"), pctx)
	if err == nil {
		t.Errorf("expected circular include error, got none")
	}

	if !strings.Contains(err.Error(), "circular include") {
		t.Errorf("expected circular include error, got: %v", err)
	}
}

// TestValidate tests template validation
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid template",
			input:   "@ign-var:name@ and @ign-if:debug@debug@ign-endif@",
			wantErr: false,
		},
		{
			name:    "unknown directive",
			input:   "@ign-loop:items@",
			wantErr: true,
		},
		{
			name:    "empty variable name",
			input:   "@ign-var:@",
			wantErr: true,
		},
		{
			name:    "unclosed if block",
			input:   "@ign-if:test@content",
			wantErr: true,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(context.Background(), []byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestExtractVariables tests variable extraction
func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single variable",
			input:    "@ign-var:name@",
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			input:    "@ign-var:host@ @ign-var:port@ @ign-var:host@",
			expected: []string{"host", "port"},
		},
		{
			name:     "variables in conditionals",
			input:    "@ign-if:debug@Debug: @ign-var:level@@ign-endif@",
			expected: []string{"debug", "level"},
		},
		{
			name:     "comment variables",
			input:    "// @ign-comment:code@",
			expected: []string{"code"},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractVariables([]byte(tt.input))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Convert to map for order-independent comparison
			resultMap := make(map[string]bool)
			for _, v := range result {
				resultMap[v] = true
			}

			expectedMap := make(map[string]bool)
			for _, v := range tt.expected {
				expectedMap[v] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("expected %d variables, got %d", len(expectedMap), len(resultMap))
				return
			}

			for v := range expectedMap {
				if !resultMap[v] {
					t.Errorf("expected variable %q not found", v)
				}
			}
		})
	}
}

// TestMapVariables tests the MapVariables implementation
func TestMapVariables(t *testing.T) {
	vars := NewMapVariables(map[string]interface{}{
		"name":  "test",
		"count": 42,
		"flag":  true,
	})

	// Test Get
	if val, ok := vars.Get("name"); !ok || val != "test" {
		t.Errorf("Get failed")
	}

	// Test GetString
	if val, err := vars.GetString("name"); err != nil || val != "test" {
		t.Errorf("GetString failed: %v", err)
	}

	// Test GetInt
	if val, err := vars.GetInt("count"); err != nil || val != 42 {
		t.Errorf("GetInt failed: %v", err)
	}

	// Test GetBool
	if val, err := vars.GetBool("flag"); err != nil || val != true {
		t.Errorf("GetBool failed: %v", err)
	}

	// Test Set
	if err := vars.Set("new", "value"); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// Test All
	all := vars.All()
	if len(all) != 4 {
		t.Errorf("All returned wrong count: %d", len(all))
	}
}
