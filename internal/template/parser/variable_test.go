package parser

import (
	"context"
	"testing"
)

// TestExtendedVarSyntax tests the extended variable syntax with types and defaults
func TestExtendedVarSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]interface{}
		expected string
		wantErr  bool
	}{
		// Basic syntax (existing behavior, required variables)
		{
			name:     "basic variable - string from vars",
			input:    "@ign-var:name@",
			vars:     map[string]interface{}{"name": "test"},
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "basic variable - missing required",
			input:    "@ign-var:missing@",
			vars:     map[string]interface{}{},
			expected: "",
			wantErr:  true,
		},

		// Type-only syntax (required variables with type validation)
		{
			name:     "type annotation - string",
			input:    "@ign-var:name:string@",
			vars:     map[string]interface{}{"name": "test"},
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "type annotation - int",
			input:    "@ign-var:port:int@",
			vars:     map[string]interface{}{"port": 8080},
			expected: "8080",
			wantErr:  false,
		},
		{
			name:     "type annotation - bool",
			input:    "@ign-var:debug:bool@",
			vars:     map[string]interface{}{"debug": true},
			expected: "true",
			wantErr:  false,
		},
		{
			name:     "type annotation - type mismatch",
			input:    "@ign-var:port:string@",
			vars:     map[string]interface{}{"port": 8080},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "type annotation - missing required",
			input:    "@ign-var:name:string@",
			vars:     map[string]interface{}{},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "type annotation - invalid type",
			input:    "@ign-var:name:float@",
			vars:     map[string]interface{}{"name": "test"},
			expected: "",
			wantErr:  true,
		},

		// Default-only syntax (optional variables)
		{
			name:     "default value - string from vars",
			input:    "@ign-var:name=default@",
			vars:     map[string]interface{}{"name": "actual"},
			expected: "actual",
			wantErr:  false,
		},
		{
			name:     "default value - use default when missing",
			input:    "@ign-var:name=default@",
			vars:     map[string]interface{}{},
			expected: "default",
			wantErr:  false,
		},
		{
			name:     "default value - int inferred",
			input:    "@ign-var:port=8080@",
			vars:     map[string]interface{}{},
			expected: "8080",
			wantErr:  false,
		},
		{
			name:     "default value - bool inferred",
			input:    "@ign-var:debug=true@",
			vars:     map[string]interface{}{},
			expected: "true",
			wantErr:  false,
		},
		{
			name:     "default value - bool false",
			input:    "@ign-var:debug=false@",
			vars:     map[string]interface{}{},
			expected: "false",
			wantErr:  false,
		},

		// Type and default syntax (optional variables with type validation)
		{
			name:     "type and default - string from vars",
			input:    "@ign-var:name:string=default@",
			vars:     map[string]interface{}{"name": "actual"},
			expected: "actual",
			wantErr:  false,
		},
		{
			name:     "type and default - use default when missing",
			input:    "@ign-var:name:string=default@",
			vars:     map[string]interface{}{},
			expected: "default",
			wantErr:  false,
		},
		{
			name:     "type and default - int",
			input:    "@ign-var:port:int=8080@",
			vars:     map[string]interface{}{},
			expected: "8080",
			wantErr:  false,
		},
		{
			name:     "type and default - bool",
			input:    "@ign-var:debug:bool=true@",
			vars:     map[string]interface{}{},
			expected: "true",
			wantErr:  false,
		},
		{
			name:     "type and default - type mismatch from vars",
			input:    "@ign-var:port:string=8080@",
			vars:     map[string]interface{}{"port": 9000},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "type and default - default type mismatch",
			input:    "@ign-var:name:int=notanumber@",
			vars:     map[string]interface{}{},
			expected: "",
			wantErr:  true,
		},

		// Edge cases
		{
			name:     "empty default value - string",
			input:    "@ign-var:name:string=@",
			vars:     map[string]interface{}{},
			expected: "",
			wantErr:  false,
		},
		{
			name:     "default with spaces",
			input:    "@ign-var:msg=hello world@",
			vars:     map[string]interface{}{},
			expected: "hello world",
			wantErr:  false,
		},
		{
			name:     "whitespace around syntax",
			input:    "@ign-var: name : string = default @",
			vars:     map[string]interface{}{},
			expected: "default",
			wantErr:  false,
		},

		// Multiple variables in one template
		{
			name:     "mixed syntax",
			input:    "host=@ign-var:host:string=localhost@, port=@ign-var:port:int=8080@, debug=@ign-var:debug:bool@",
			vars:     map[string]interface{}{"debug": false},
			expected: "host=localhost, port=8080, debug=false",
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

// TestParseVarSyntax tests the parseVarSyntax helper function directly
func TestParseVarSyntax(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedName       string
		expectedType       string
		expectedDefault    interface{}
		expectedHasDefault bool
		wantErr            bool
	}{
		{
			name:               "basic name only",
			input:              "myvar",
			expectedName:       "myvar",
			expectedType:       "",
			expectedDefault:    nil,
			expectedHasDefault: false,
			wantErr:            false,
		},
		{
			name:               "name with type",
			input:              "myvar:string",
			expectedName:       "myvar",
			expectedType:       "string",
			expectedDefault:    nil,
			expectedHasDefault: false,
			wantErr:            false,
		},
		{
			name:               "name with default string",
			input:              "myvar=hello",
			expectedName:       "myvar",
			expectedType:       "",
			expectedDefault:    "hello",
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:               "name with default int",
			input:              "port=8080",
			expectedName:       "port",
			expectedType:       "",
			expectedDefault:    8080,
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:               "name with default bool true",
			input:              "debug=true",
			expectedName:       "debug",
			expectedType:       "",
			expectedDefault:    true,
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:               "name with default bool false",
			input:              "debug=false",
			expectedName:       "debug",
			expectedType:       "",
			expectedDefault:    false,
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:               "name, type, and default",
			input:              "port:int=8080",
			expectedName:       "port",
			expectedType:       "int",
			expectedDefault:    8080,
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:               "empty default",
			input:              "name=",
			expectedName:       "name",
			expectedType:       "",
			expectedDefault:    "",
			expectedHasDefault: true,
			wantErr:            false,
		},
		{
			name:    "invalid type",
			input:   "name:float",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   ":string",
			wantErr: true,
		},
		{
			name:               "whitespace trimming",
			input:              " name : string = value ",
			expectedName:       "name",
			expectedType:       "string",
			expectedDefault:    "value",
			expectedHasDefault: true,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, varType, defaultValue, hasDefault, err := parseVarSyntax(tt.input)

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

			if name != tt.expectedName {
				t.Errorf("name: expected %q, got %q", tt.expectedName, name)
			}

			if varType != tt.expectedType {
				t.Errorf("type: expected %q, got %q", tt.expectedType, varType)
			}

			if hasDefault != tt.expectedHasDefault {
				t.Errorf("hasDefault: expected %v, got %v", tt.expectedHasDefault, hasDefault)
			}

			if hasDefault && defaultValue != tt.expectedDefault {
				t.Errorf("default: expected %v (%T), got %v (%T)", tt.expectedDefault, tt.expectedDefault, defaultValue, defaultValue)
			}
		})
	}
}

// TestParseDefaultValue tests the parseDefaultValue helper function
func TestParseDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "bool true",
			input:    "true",
			expected: true,
		},
		{
			name:     "bool false",
			input:    "false",
			expected: false,
		},
		{
			name:     "integer",
			input:    "42",
			expected: 42,
		},
		{
			name:     "negative integer",
			input:    "-10",
			expected: -10,
		},
		{
			name:     "zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "numeric-like string",
			input:    "123abc",
			expected: "123abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDefaultValue(tt.input)

			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

// TestValidateVarType tests type validation
func TestValidateVarType(t *testing.T) {
	tests := []struct {
		name         string
		varName      string
		value        interface{}
		expectedType string
		wantErr      bool
	}{
		{
			name:         "string valid",
			varName:      "name",
			value:        "test",
			expectedType: "string",
			wantErr:      false,
		},
		{
			name:         "int valid",
			varName:      "port",
			value:        8080,
			expectedType: "int",
			wantErr:      false,
		},
		{
			name:         "int64 as int",
			varName:      "count",
			value:        int64(100),
			expectedType: "int",
			wantErr:      false,
		},
		{
			name:         "float64 as int (JSON numbers)",
			varName:      "num",
			value:        float64(42),
			expectedType: "int",
			wantErr:      false,
		},
		{
			name:         "bool valid",
			varName:      "debug",
			value:        true,
			expectedType: "bool",
			wantErr:      false,
		},
		{
			name:         "type mismatch - string expected int",
			varName:      "port",
			value:        "8080",
			expectedType: "int",
			wantErr:      true,
		},
		{
			name:         "type mismatch - int expected string",
			varName:      "name",
			value:        42,
			expectedType: "string",
			wantErr:      true,
		},
		{
			name:         "type mismatch - string expected bool",
			varName:      "debug",
			value:        "true",
			expectedType: "bool",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVarType(tt.varName, tt.value, tt.expectedType)

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

// TestInferType tests type inference
func TestInferType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string",
			value:    "hello",
			expected: "string",
		},
		{
			name:     "int",
			value:    42,
			expected: "int",
		},
		{
			name:     "int64",
			value:    int64(100),
			expected: "int",
		},
		{
			name:     "float64",
			value:    float64(3.14),
			expected: "int",
		},
		{
			name:     "bool",
			value:    true,
			expected: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferType(tt.value)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
