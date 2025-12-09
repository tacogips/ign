package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tacogips/ign/internal/debug"
)

// Variables holds template variable values and provides type-safe access.
type Variables interface {
	// Get retrieves a variable value by name.
	// Returns (value, true) if found, (nil, false) if not found.
	Get(name string) (interface{}, bool)

	// GetString retrieves a string variable.
	// Returns error if variable not found or type mismatch.
	GetString(name string) (string, error)

	// GetInt retrieves an integer variable.
	// Returns error if variable not found or type mismatch.
	GetInt(name string) (int, error)

	// GetBool retrieves a boolean variable.
	// Returns error if variable not found or type mismatch.
	GetBool(name string) (bool, error)

	// Set sets a variable value.
	Set(name string, value interface{}) error

	// All returns all variables as a map.
	All() map[string]interface{}
}

// MapVariables implements Variables using a map[string]interface{}.
type MapVariables struct {
	data map[string]interface{}
}

// NewMapVariables creates a new MapVariables from a map.
func NewMapVariables(data map[string]interface{}) *MapVariables {
	if data == nil {
		data = make(map[string]interface{})
	}
	return &MapVariables{data: data}
}

// Get retrieves a variable value by name.
func (m *MapVariables) Get(name string) (interface{}, bool) {
	val, ok := m.data[name]
	return val, ok
}

// GetString retrieves a string variable.
func (m *MapVariables) GetString(name string) (string, error) {
	val, ok := m.data[name]
	if !ok {
		return "", fmt.Errorf("variable not found: %s", name)
	}

	switch v := val.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("variable %s is not a string (got %T)", name, val)
	}
}

// GetInt retrieves an integer variable.
func (m *MapVariables) GetInt(name string) (int, error) {
	val, ok := m.data[name]
	if !ok {
		return 0, fmt.Errorf("variable not found: %s", name)
	}

	switch v := val.(type) {
	case int:
		return v, nil
	case float64:
		// JSON unmarshaling produces float64 for all numbers
		return int(v), nil
	case int64:
		return int(v), nil
	case int32:
		return int(v), nil
	default:
		return 0, fmt.Errorf("variable %s is not an integer (got %T)", name, val)
	}
}

// GetBool retrieves a boolean variable.
func (m *MapVariables) GetBool(name string) (bool, error) {
	val, ok := m.data[name]
	if !ok {
		return false, fmt.Errorf("variable not found: %s", name)
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("variable %s is not a boolean (got %T)", name, val)
	}
}

// Set sets a variable value.
func (m *MapVariables) Set(name string, value interface{}) error {
	m.data[name] = value
	return nil
}

// All returns all variables as a map.
func (m *MapVariables) All() map[string]interface{} {
	result := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

// processVarDirective substitutes a @ign-var:NAME@ directive with its value.
// Supports the following syntax variants:
//
//	@ign-var:NAME@                      - Basic (required, type inferred)
//	@ign-var:NAME:TYPE@                 - With explicit type (required)
//	@ign-var:NAME=DEFAULT@              - With default value (optional)
//	@ign-var:NAME:TYPE=DEFAULT@         - With type and default value (optional)
func processVarDirective(args string, vars Variables) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", newParseError(InvalidDirectiveSyntax, "variable name is empty")
	}

	// Parse the variable syntax to extract name, type, and default value
	varName, varType, defaultValue, hasDefault, err := parseVarSyntax(args)
	if err != nil {
		return "", newParseErrorWithDirective(InvalidDirectiveSyntax, err.Error(), "@ign-var:"+args+"@")
	}

	debug.Debug("[parser] processVarDirective: name=%s, type=%s, hasDefault=%v, default=%v", varName, varType, hasDefault, defaultValue)

	// Try to get the value from variables
	val, ok := vars.Get(varName)

	// If variable not found
	if !ok {
		// If default value is provided, use it (optional variable)
		if hasDefault {
			debug.Debug("[parser] processVarDirective: using default value for %s: %v", varName, defaultValue)
			val = defaultValue
		} else {
			// No default value means required variable
			return "", newParseErrorWithDirective(MissingVariable,
				fmt.Sprintf("required variable not found: %s", varName),
				"@ign-var:"+args+"@")
		}
	}

	// Validate type if specified
	if varType != "" {
		if err := validateVarType(varName, val, varType); err != nil {
			return "", newParseErrorWithDirective(InvalidDirectiveSyntax, err.Error(), "@ign-var:"+args+"@")
		}
	}

	// Convert value to string
	result := valueToString(val)
	debug.Debug("[parser] processVarDirective: variable=%s, value=%v, resolved=%s", varName, val, result)
	return result, nil
}

// parseVarSyntax parses the variable directive arguments.
// Returns: (name, type, defaultValue, hasDefault, error)
// Syntax:
//
//	NAME                - Basic variable (required)
//	NAME:TYPE           - Variable with type (required)
//	NAME=DEFAULT        - Variable with default (optional)
//	NAME:TYPE=DEFAULT   - Variable with type and default (optional)
func parseVarSyntax(args string) (name string, varType string, defaultValue interface{}, hasDefault bool, err error) {
	// Check for default value (split on '=')
	equalIdx := strings.Index(args, "=")
	if equalIdx != -1 {
		hasDefault = true
		defaultStr := args[equalIdx+1:]
		args = args[:equalIdx] // Remove default part for further parsing

		// Parse the default value based on content
		defaultValue = parseDefaultValue(defaultStr)
	}

	// Check for type annotation (split on ':')
	colonIdx := strings.Index(args, ":")
	if colonIdx != -1 {
		name = strings.TrimSpace(args[:colonIdx])
		varType = strings.TrimSpace(args[colonIdx+1:])

		// Validate type
		if varType != "" && varType != "string" && varType != "int" && varType != "bool" {
			err = fmt.Errorf("invalid type %q (must be string, int, or bool)", varType)
			return
		}
	} else {
		name = strings.TrimSpace(args)
	}

	// Validate variable name
	if name == "" {
		err = fmt.Errorf("variable name is empty")
		return
	}

	// If type is specified and has default, validate default matches type
	if varType != "" && hasDefault {
		expectedType := inferType(defaultValue)
		if expectedType != varType {
			// Try to coerce the default value to match the specified type
			coerced, coerceErr := coerceValue(defaultValue, varType)
			if coerceErr != nil {
				err = fmt.Errorf("default value type mismatch: expected %s, got %s", varType, expectedType)
				return
			}
			defaultValue = coerced
		}
	}

	return
}

// parseDefaultValue parses a default value string and returns the appropriate type.
// - "true"/"false" -> bool
// - numeric string -> int
// - otherwise -> string
func parseDefaultValue(value string) interface{} {
	value = strings.TrimSpace(value)

	// Try boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Default to string
	return value
}

// inferType infers the type name from a value.
func inferType(val interface{}) string {
	switch val.(type) {
	case bool:
		return "bool"
	case int, int64, int32, float64:
		return "int"
	default:
		return "string"
	}
}

// coerceValue attempts to convert a value to the specified type.
func coerceValue(val interface{}, targetType string) (interface{}, error) {
	switch targetType {
	case "string":
		return valueToString(val), nil
	case "int":
		switch v := val.(type) {
		case int:
			return v, nil
		case int64:
			return int(v), nil
		case float64:
			return int(v), nil
		case string:
			return strconv.Atoi(v)
		default:
			return nil, fmt.Errorf("cannot coerce %T to int", val)
		}
	case "bool":
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return nil, fmt.Errorf("cannot coerce %T to bool", val)
		}
	default:
		return nil, fmt.Errorf("unknown type: %s", targetType)
	}
}

// validateVarType validates that a value matches the expected type.
func validateVarType(name string, val interface{}, expectedType string) error {
	actualType := inferType(val)

	// Special handling for numeric types (JSON unmarshaling produces float64)
	if expectedType == "int" {
		switch val.(type) {
		case int, int64, int32, float64:
			return nil
		default:
			return fmt.Errorf("variable %s: type mismatch, expected %s but got %s", name, expectedType, actualType)
		}
	}

	if actualType != expectedType {
		return fmt.Errorf("variable %s: type mismatch, expected %s but got %s", name, expectedType, actualType)
	}

	return nil
}

// valueToString converts a variable value to its string representation.
func valueToString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// Convert float to int if it's a whole number (for JSON numbers)
		if v == float64(int(v)) {
			return strconv.Itoa(int(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
