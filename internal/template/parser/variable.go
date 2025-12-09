package parser

import (
	"fmt"
	"strconv"
	"strings"
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
func processVarDirective(args string, vars Variables) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", newParseError(InvalidDirectiveSyntax, "variable name is empty")
	}

	val, ok := vars.Get(args)
	if !ok {
		return "", newParseErrorWithDirective(MissingVariable,
			fmt.Sprintf("variable not found: %s", args),
			"@ign-var:"+args+"@")
	}

	// Convert value to string
	return valueToString(val), nil
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
