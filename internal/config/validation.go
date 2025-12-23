package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

// Validate validates the global configuration.
func Validate(config *Config) error {
	loader := NewLoader()
	return loader.Validate(config)
}

// ValidateIgnJson validates template configuration (ign.json).
func ValidateIgnJson(ign *model.IgnJson) error {
	if ign == nil {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "", "ign.json cannot be nil")
	}

	// Validate required fields
	if ign.Name == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "name", "template name is required")
	}
	if ign.Version == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "version", "template version is required")
	}

	// Validate template name format (lowercase, hyphens, underscores, alphanumeric)
	namePattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-_]*$`)
	if !namePattern.MatchString(ign.Name) {
		return NewConfigErrorWithField(
			ConfigValidationFailed,
			"ign.json",
			"name",
			"template name must start with lowercase letter or digit and contain only lowercase letters, digits, hyphens, and underscores",
		)
	}

	// Validate version format (basic semver check)
	versionPattern := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)
	if !versionPattern.MatchString(ign.Version) {
		return NewConfigErrorWithField(
			ConfigValidationFailed,
			"ign.json",
			"version",
			fmt.Sprintf("invalid version format: %s (expected semantic version like 1.0.0)", ign.Version),
		)
	}

	// Validate variables
	if err := validateVariables(ign.Variables); err != nil {
		return err
	}

	// Validate settings if present
	if ign.Settings != nil {
		if ign.Settings.MaxIncludeDepth < 0 {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				"settings.max_include_depth",
				"max include depth cannot be negative",
			)
		}
	}

	return nil
}

// ValidateIgnVarJson validates build configuration (ign-var.json).
// ign-var.json now contains only variables, which are validated against
// the template's ign.json during generation, so this is a simple nil check.
func ValidateIgnVarJson(ignVar *model.IgnVarJson) error {
	if ignVar == nil {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign-var.json", "", "ign-var.json cannot be nil")
	}

	// Variables can be empty (will be validated against template's ign.json during generation)
	return nil
}

// ValidateIgnConfig validates user project configuration file (.ign/ign.json).
// This validates the project's configuration file which contains template source information,
// NOT the template's own ign.json metadata file. For template metadata validation, see ValidateIgnJson.
func ValidateIgnConfig(ignConfig *model.IgnConfig) error {
	if ignConfig == nil {
		return NewConfigErrorWithField(ConfigValidationFailed, ".ign/ign.json", "", "project configuration (.ign/ign.json) cannot be nil")
	}

	// Validate template source
	if ignConfig.Template.URL == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, ".ign/ign.json", "template.url", "template URL is required")
	}

	// Validate URL format
	if err := validateTemplateURL(ignConfig.Template.URL); err != nil {
		return NewConfigErrorWithField(ConfigValidationFailed, ".ign/ign.json", "template.url", err.Error())
	}

	// Hash should be present (calculated during checkout)
	if ignConfig.Hash == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, ".ign/ign.json", "hash", "template hash is required")
	}

	// Validate hash format (must be valid SHA256: 64 hexadecimal characters)
	if !isValidSHA256Hash(ignConfig.Hash) {
		return NewConfigErrorWithField(ConfigValidationFailed, ".ign/ign.json", "hash",
			"hash must be a valid SHA256 string (64 hexadecimal characters)")
	}

	return nil
}

// validateVariables validates the variables section of ign.json.
func validateVariables(variables map[string]model.VarDef) error {
	for name, varDef := range variables {
		// Validate variable name format
		namePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
		if !namePattern.MatchString(name) {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s", name),
				"variable name must start with a letter and contain only letters, digits, underscores, and hyphens",
			)
		}

		// Validate variable type
		if err := validateVarType(varDef.Type); err != nil {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s.type", name),
				err.Error(),
			)
		}

		// Validate description
		if varDef.Description == "" {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s.description", name),
				"variable description is required",
			)
		}

		// Validate default value type matches
		if varDef.Default != nil {
			if err := validateValueType(varDef.Default, varDef.Type); err != nil {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.default", name),
					fmt.Sprintf("default value type mismatch: %v", err),
				)
			}
		}

		// Validate example value type matches
		if varDef.Example != nil {
			if err := validateValueType(varDef.Example, varDef.Type); err != nil {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.example", name),
					fmt.Sprintf("example value type mismatch: %v", err),
				)
			}
		}

		// Validate pattern for string types
		if varDef.Pattern != "" && varDef.Type != model.VarTypeString {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s.pattern", name),
				"pattern can only be specified for string variables",
			)
		}

		// Validate pattern is valid regex
		if varDef.Pattern != "" {
			if _, err := regexp.Compile(varDef.Pattern); err != nil {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.pattern", name),
					fmt.Sprintf("invalid regex pattern: %v", err),
				)
			}

			// Validate default value against pattern
			if varDef.Default != nil {
				if str, ok := varDef.Default.(string); ok {
					matched, err := regexp.MatchString(varDef.Pattern, str)
					if err != nil {
						return NewConfigErrorWithField(
							ConfigValidationFailed,
							"ign.json",
							fmt.Sprintf("variables.%s.pattern", name),
							fmt.Sprintf("error matching pattern: %v", err),
						)
					}
					if !matched {
						return NewConfigErrorWithField(
							ConfigValidationFailed,
							"ign.json",
							fmt.Sprintf("variables.%s.default", name),
							fmt.Sprintf("default value %q does not match pattern %q", str, varDef.Pattern),
						)
					}
				}
			}

			// Validate example value against pattern
			if varDef.Example != nil {
				if str, ok := varDef.Example.(string); ok {
					matched, err := regexp.MatchString(varDef.Pattern, str)
					if err != nil {
						return NewConfigErrorWithField(
							ConfigValidationFailed,
							"ign.json",
							fmt.Sprintf("variables.%s.pattern", name),
							fmt.Sprintf("error matching pattern: %v", err),
						)
					}
					if !matched {
						return NewConfigErrorWithField(
							ConfigValidationFailed,
							"ign.json",
							fmt.Sprintf("variables.%s.example", name),
							fmt.Sprintf("example value %q does not match pattern %q", str, varDef.Pattern),
						)
					}
				}
			}
		}

		// Validate min/max for integer and number types
		if (varDef.Min != nil || varDef.Max != nil) && varDef.Type != model.VarTypeInt && varDef.Type != model.VarTypeNumber {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s", name),
				"min/max can only be specified for integer and number variables",
			)
		}

		// Validate min <= max
		if varDef.Min != nil && varDef.Max != nil && *varDef.Min > *varDef.Max {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s", name),
				fmt.Sprintf("min (%v) cannot be greater than max (%v)", *varDef.Min, *varDef.Max),
			)
		}

		// Validate default value against min/max constraints for integers and numbers
		if varDef.Default != nil && (varDef.Type == model.VarTypeInt || varDef.Type == model.VarTypeNumber) {
			floatVal := toFloat64(varDef.Default)
			if floatVal == nil {
				// Invariant: toFloat64 should never return nil for values that passed validateValueType.
				// This check enforces strict validation to prevent silent constraint skipping in case
				// of unexpected runtime type coercion failures (e.g., interface{} containing unsupported numeric types).
				// If this error occurs, it indicates either:
				// 1. A bug in validateValueType allowing non-numeric types for int/number variables, or
				// 2. Runtime type corruption of the Default value after validation
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.default", name),
					fmt.Sprintf("default value cannot be converted to numeric type"),
				)
			}
			if varDef.Min != nil && *floatVal < *varDef.Min {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.default", name),
					fmt.Sprintf("default value %v is less than min %v", *floatVal, *varDef.Min),
				)
			}
			if varDef.Max != nil && *floatVal > *varDef.Max {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.default", name),
					fmt.Sprintf("default value %v is greater than max %v", *floatVal, *varDef.Max),
				)
			}
		}

		// Validate example value against min/max constraints for integers and numbers
		if varDef.Example != nil && (varDef.Type == model.VarTypeInt || varDef.Type == model.VarTypeNumber) {
			floatVal := toFloat64(varDef.Example)
			if floatVal == nil {
				// Invariant: toFloat64 should never return nil for values that passed validateValueType.
				// This check enforces strict validation to prevent silent constraint skipping in case
				// of unexpected runtime type coercion failures (e.g., interface{} containing unsupported numeric types).
				// If this error occurs, it indicates either:
				// 1. A bug in validateValueType allowing non-numeric types for int/number variables, or
				// 2. Runtime type corruption of the Example value after validation
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.example", name),
					fmt.Sprintf("example value cannot be converted to numeric type"),
				)
			}
			if varDef.Min != nil && *floatVal < *varDef.Min {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.example", name),
					fmt.Sprintf("example value %v is less than min %v", *floatVal, *varDef.Min),
				)
			}
			if varDef.Max != nil && *floatVal > *varDef.Max {
				return NewConfigErrorWithField(
					ConfigValidationFailed,
					"ign.json",
					fmt.Sprintf("variables.%s.example", name),
					fmt.Sprintf("example value %v is greater than max %v", *floatVal, *varDef.Max),
				)
			}
		}
	}

	return nil
}

// validateVarType validates that a variable type is valid.
func validateVarType(typ model.VarType) error {
	switch typ {
	case model.VarTypeString, model.VarTypeInt, model.VarTypeNumber, model.VarTypeBool:
		return nil
	default:
		return fmt.Errorf("invalid variable type: %s (must be string, int, number, or bool)", typ)
	}
}

// validateValueType validates that a value matches the expected type.
func validateValueType(value interface{}, expectedType model.VarType) error {
	switch expectedType {
	case model.VarTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case model.VarTypeInt:
		// JSON unmarshals numbers as float64
		switch v := value.(type) {
		case int, int32, int64, float64:
			return nil
		default:
			return fmt.Errorf("expected int, got %T", v)
		}
	case model.VarTypeNumber:
		// JSON unmarshals numbers as float64
		switch v := value.(type) {
		case float32, float64, int, int32, int64:
			return nil
		default:
			return fmt.Errorf("expected number, got %T", v)
		}
	case model.VarTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	}
	return nil
}

// toFloat64 converts a value to float64 pointer if it's a numeric type.
func toFloat64(value interface{}) *float64 {
	switch v := value.(type) {
	case float64:
		return &v
	case float32:
		f := float64(v)
		return &f
	case int:
		f := float64(v)
		return &f
	case int32:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	default:
		return nil
	}
}

// validateTemplateURL validates that a template URL is in a supported format.
// Supports:
//   - Full URLs: https://github.com/owner/repo, git@github.com:owner/repo.git
//   - GitHub shorthand: github.com/owner/repo, github:owner/repo
//   - Local paths: /path/to/template, ./relative/path
func validateTemplateURL(templateURL string) error {
	templateURL = strings.TrimSpace(templateURL)
	if templateURL == "" {
		return fmt.Errorf("template URL cannot be empty")
	}

	// Check for github: shorthand
	if strings.HasPrefix(templateURL, "github:") {
		// Format: github:owner/repo
		return nil
	}

	// Check for git@ SSH format
	if strings.HasPrefix(templateURL, "git@") {
		// Format: git@github.com:owner/repo.git
		return nil
	}

	// Check for local filesystem path
	if strings.HasPrefix(templateURL, "/") || strings.HasPrefix(templateURL, "./") || strings.HasPrefix(templateURL, "../") {
		// Local path (absolute or relative)
		return nil
	}

	// Check for github.com shorthand (without scheme)
	if strings.HasPrefix(templateURL, "github.com/") {
		// Format: github.com/owner/repo
		return nil
	}

	// Try parsing as URL
	parsedURL, err := url.Parse(templateURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check for valid scheme
	if parsedURL.Scheme != "" && parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q (supported: https, http, git@, github:, or local path)", parsedURL.Scheme)
	}

	// If we got here, it's either a valid URL or a format we recognize
	return nil
}
