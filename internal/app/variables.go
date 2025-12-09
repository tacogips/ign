package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

// LoadVariables loads and processes variables from IgnVarJson.
// Resolves @file: prefixed values by reading file content from buildDir.
// Returns a Variables interface suitable for template processing.
func LoadVariables(ignVar *model.IgnVarJson, buildDir string) (parser.Variables, error) {
	if ignVar == nil {
		return nil, NewVariableLoadError("ignVar is nil", nil)
	}

	if ignVar.Variables == nil {
		return nil, NewVariableLoadError("variables map is nil", nil)
	}

	// Create a copy of the variables map to avoid modifying the original
	processedVars := make(map[string]interface{})

	for name, value := range ignVar.Variables {
		// Check if value is a string with @file: prefix
		if strVal, ok := value.(string); ok && strings.HasPrefix(strVal, "@file:") {
			// Extract filename
			filename := strings.TrimPrefix(strVal, "@file:")
			filename = strings.TrimSpace(filename)

			if filename == "" {
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: @file: prefix without filename", name),
					nil,
				)
			}

			// Resolve file path relative to buildDir
			filePath := filepath.Join(buildDir, filename)

			// Read file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: failed to read @file:%s", name, filename),
					err,
				)
			}

			// Use file content as variable value
			processedVars[name] = string(content)
		} else {
			// Use value as-is
			processedVars[name] = value
		}
	}

	return parser.NewMapVariables(processedVars), nil
}

// ValidateVariables validates that all required variables from IgnJson are set.
// Returns an error if any required variable is missing or empty.
func ValidateVariables(ignJson *model.IgnJson, vars parser.Variables) error {
	if ignJson == nil {
		return NewValidationError("ignJson is nil", nil)
	}

	if vars == nil {
		return NewValidationError("variables are nil", nil)
	}

	var missingVars []string

	for name, varDef := range ignJson.Variables {
		// Skip if not required
		if !varDef.Required {
			continue
		}

		// Check if variable exists
		value, exists := vars.Get(name)
		if !exists {
			missingVars = append(missingVars, name)
			continue
		}

		// Check if string variable is empty
		if varDef.Type == model.VarTypeString {
			if strVal, ok := value.(string); ok && strings.TrimSpace(strVal) == "" {
				missingVars = append(missingVars, name)
			}
		}
	}

	if len(missingVars) > 0 {
		return NewValidationError(
			fmt.Sprintf("missing required variables: %s", strings.Join(missingVars, ", ")),
			nil,
		)
	}

	return nil
}
