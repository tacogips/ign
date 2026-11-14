package app

import (
	"fmt"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

// NormalizeTemplateURL normalizes a template URL to a consistent format.
// Handles various URL formats (full URL, short form, owner/repo, etc.).
func NormalizeTemplateURL(url string) string {
	url = strings.TrimSpace(url)

	// Check for file:// URLs first
	if strings.HasPrefix(url, "file://") {
		return url
	}

	// Check for absolute paths (UNIX-style)
	if strings.HasPrefix(url, "/") {
		return url
	}

	// Check for local paths (before modifying URL)
	if strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") {
		return url
	}

	// If it's already a full URL or git@ format, return as-is
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "git@") {
		return url
	}

	// If it starts with github.com/, prepend https://
	if strings.HasPrefix(url, "github.com/") {
		return "https://" + url
	}

	// If it's owner/repo format (contains / but not github.com), assume GitHub
	if strings.Contains(url, "/") {
		return "https://github.com/" + url
	}

	// Otherwise, return as-is (might be local path)
	return url
}

// ValidateOutputDir validates that the output directory path is safe.
func ValidateOutputDir(path string) error {
	if path == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("output directory cannot contain '..'")
	}

	return nil
}

// CreateEmptyVariablesMap creates an initial variables map from IgnJson variable definitions.
// Declared defaults are preserved as authored so runtime-only placeholders can be resolved later.
func CreateEmptyVariablesMap(ignJson *model.IgnJson) map[string]interface{} {
	if ignJson == nil {
		return map[string]interface{}{}
	}

	vars := mergeVariableDefaults(ignJson.Variables, nil)

	for name, varDef := range ignJson.Variables {
		if _, ok := vars[name]; ok {
			continue
		}

		// Fall back to the type-appropriate zero value when no default was resolved.
		switch varDef.Type {
		case model.VarTypeString:
			vars[name] = ""
		case model.VarTypeInt:
			vars[name] = 0
		case model.VarTypeBool:
			vars[name] = false
		default:
			vars[name] = ""
		}
	}

	return vars
}

func mergeVariableDefaults(varDefs map[string]model.VarDef, providedVars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(providedVars)+len(varDefs))

	for name, value := range providedVars {
		result[name] = value
	}

	for name, varDef := range varDefs {
		if _, provided := result[name]; !provided && varDef.Default != nil {
			result[name] = varDef.Default
		}
	}

	return result
}

// FormatVariableTip creates a helpful tip message for a variable.
// Suggests using @file: for string variables without defaults.
func FormatVariableTip(name string, varDef model.VarDef) string {
	if varDef.Type == model.VarTypeString && varDef.Default == nil {
		return "Tip: You can use @file:filename.txt to load content from a file"
	}
	return ""
}

// CountVariablesByType counts variables by type in an IgnJson.
func CountVariablesByType(ignJson *model.IgnJson) (stringCount int, intCount int, boolCount int) {
	if ignJson == nil {
		return 0, 0, 0
	}

	for _, varDef := range ignJson.Variables {
		switch varDef.Type {
		case model.VarTypeString:
			stringCount++
		case model.VarTypeInt:
			intCount++
		case model.VarTypeBool:
			boolCount++
		}
	}
	return
}
