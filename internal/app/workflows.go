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

	// Check for local paths first (before modifying URL)
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

// CreateEmptyVariablesMap creates an empty variables map from IgnJson variable definitions.
// Sets all values to appropriate zero values or empty strings.
func CreateEmptyVariablesMap(ignJson *model.IgnJson) map[string]interface{} {
	vars := make(map[string]interface{})

	for name, varDef := range ignJson.Variables {
		// Use default value if provided
		if varDef.Default != nil {
			vars[name] = varDef.Default
			continue
		}

		// Otherwise use type-appropriate zero/empty value
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

// FormatVariableTip creates a helpful tip message for a variable.
// Suggests using @file: for string variables without defaults.
func FormatVariableTip(name string, varDef model.VarDef) string {
	if varDef.Type == model.VarTypeString && varDef.Default == nil {
		return fmt.Sprintf("Tip: You can use @file:filename.txt to load content from a file")
	}
	return ""
}

// CountVariablesByType counts variables by type in an IgnJson.
func CountVariablesByType(ignJson *model.IgnJson) (strings int, ints int, bools int) {
	for _, varDef := range ignJson.Variables {
		switch varDef.Type {
		case model.VarTypeString:
			strings++
		case model.VarTypeInt:
			ints++
		case model.VarTypeBool:
			bools++
		}
	}
	return
}
