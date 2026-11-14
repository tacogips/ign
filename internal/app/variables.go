package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

// LoadVariablesFromMap loads and processes variables from a map.
// Resolves @file: prefixed values by reading file content from buildDir.
// Returns a Variables interface suitable for template processing.
func LoadVariablesFromMap(variables map[string]interface{}, buildDir string) (parser.Variables, error) {
	debug.Debug("[app] LoadVariablesFromMap: starting variable loading")
	debug.DebugValue("[app] Build directory", buildDir)

	if variables == nil {
		debug.Debug("[app] LoadVariablesFromMap: variables map is nil")
		return nil, NewVariableLoadError("variables map is nil", nil)
	}

	debug.DebugValue("[app] Number of variables to process", len(variables))

	// Create a copy of the variables map to avoid modifying the original
	processedVars := make(map[string]interface{})

	for name, value := range variables {
		// Check if value is a string with @file: prefix
		if strVal, ok := value.(string); ok && strings.HasPrefix(strVal, "@file:") {
			// Extract filename
			filename := strings.TrimPrefix(strVal, "@file:")
			filename = strings.TrimSpace(filename)

			debug.Debug("[app] Variable '%s': resolving @file: reference", name)
			debug.DebugValue("[app] File reference", filename)

			if filename == "" {
				debug.Debug("[app] Variable '%s': @file: prefix without filename", name)
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: @file: prefix without filename", name),
					nil,
				)
			}

			// Security: Validate filename does not contain path traversal sequences
			if strings.Contains(filename, "..") {
				debug.Debug("[app] Variable '%s': path traversal attempt detected", name)
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: @file: path contains '..' which is not allowed for security reasons", name),
					nil,
				)
			}

			// Resolve file path relative to buildDir
			filePath := filepath.Join(buildDir, filename)
			debug.DebugValue("[app] Resolved file path", filePath)

			// Security: Verify resolved path is within buildDir
			absBuildDir, err := filepath.Abs(buildDir)
			if err != nil {
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: failed to resolve build directory", name),
					err,
				)
			}
			absFilePath, err := filepath.Abs(filePath)
			if err != nil {
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: failed to resolve file path", name),
					err,
				)
			}
			relPath, err := filepath.Rel(absBuildDir, absFilePath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				debug.Debug("[app] Variable '%s': file path escapes build directory", name)
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: @file: path must be within the configuration directory", name),
					nil,
				)
			}

			// Read file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				debug.Debug("[app] Variable '%s': failed to read file: %v", name, err)
				return nil, NewVariableLoadError(
					fmt.Sprintf("variable %s: failed to read @file:%s", name, filename),
					err,
				)
			}

			// Use file content as variable value
			processedVars[name] = string(content)
			debug.Debug("[app] Variable '%s': file content loaded (%d bytes)", name, len(content))
		} else {
			// Use value as-is
			processedVars[name] = value
			debug.Debug("[app] Variable '%s': using direct value", name)
		}
	}

	debug.Debug("[app] LoadVariablesFromMap: all variables processed successfully")
	return parser.NewMapVariables(processedVars), nil
}

// LoadVariables loads and processes variables from IgnVarJson.
// Resolves @file: prefixed values by reading file content from buildDir.
// Returns a Variables interface suitable for template processing.
func LoadVariables(ignVar *model.IgnVarJson, buildDir string) (parser.Variables, error) {
	debug.Debug("[app] LoadVariables: starting variable loading")
	debug.DebugValue("[app] Build directory", buildDir)

	if ignVar == nil {
		debug.Debug("[app] LoadVariables: ignVar is nil")
		return nil, NewVariableLoadError("ignVar is nil", nil)
	}

	if ignVar.Variables == nil {
		debug.Debug("[app] LoadVariables: variables map is nil")
		return nil, NewVariableLoadError("variables map is nil", nil)
	}

	// Delegate to LoadVariablesFromMap
	return LoadVariablesFromMap(ignVar.Variables, buildDir)
}

// ValidateVariables validates that all required variables from IgnJson are set.
// Returns an error if any required variable is missing or empty.
func ValidateVariables(ignJson *model.IgnJson, vars parser.Variables) error {
	debug.Debug("[app] ValidateVariables: starting variable validation")

	if ignJson == nil {
		debug.Debug("[app] ValidateVariables: ignJson is nil")
		return NewValidationError("ignJson is nil", nil)
	}

	if vars == nil {
		debug.Debug("[app] ValidateVariables: variables are nil")
		return NewValidationError("variables are nil", nil)
	}

	debug.DebugValue("[app] Number of variable definitions", len(ignJson.Variables))

	var missingVars []string

	for name, varDef := range ignJson.Variables {
		// Skip if not required
		if !varDef.Required {
			debug.Debug("[app] Variable '%s': not required, skipping", name)
			continue
		}

		debug.Debug("[app] Variable '%s': validating required variable", name)

		// Check if variable exists
		value, exists := vars.Get(name)
		if !exists {
			debug.Debug("[app] Variable '%s': missing", name)
			missingVars = append(missingVars, name)
			continue
		}

		// Check if string variable is empty
		if varDef.Type == model.VarTypeString {
			if strVal, ok := value.(string); ok && strings.TrimSpace(strVal) == "" {
				debug.Debug("[app] Variable '%s': empty string value", name)
				missingVars = append(missingVars, name)
			} else {
				debug.Debug("[app] Variable '%s': valid", name)
			}
		} else {
			debug.Debug("[app] Variable '%s': valid (non-string type)", name)
		}
	}

	if len(missingVars) > 0 {
		debug.Debug("[app] ValidateVariables: validation failed, missing %d variables", len(missingVars))
		return NewValidationError(
			fmt.Sprintf("missing required variables: %s", strings.Join(missingVars, ", ")),
			nil,
		)
	}

	debug.Debug("[app] ValidateVariables: all required variables validated successfully")
	return nil
}
