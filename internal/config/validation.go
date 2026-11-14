package config

import (
	"fmt"
	"regexp"

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

// ValidateIgnConfig validates configuration file (ign.json in .ign directory).
func ValidateIgnConfig(ignConfig *model.IgnConfig) error {
	if ignConfig == nil {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "", "ign.json cannot be nil")
	}

	// Validate template source
	if ignConfig.Template.URL == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "template.url", "template URL is required")
	}

	// Hash should be present (calculated during checkout)
	if ignConfig.Hash == "" {
		return NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "hash", "template hash is required")
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
		}

		// Validate min/max for integer types
		if (varDef.Min != nil || varDef.Max != nil) && varDef.Type != model.VarTypeInt {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s", name),
				"min/max can only be specified for integer variables",
			)
		}

		// Validate min <= max
		if varDef.Min != nil && varDef.Max != nil && *varDef.Min > *varDef.Max {
			return NewConfigErrorWithField(
				ConfigValidationFailed,
				"ign.json",
				fmt.Sprintf("variables.%s", name),
				fmt.Sprintf("min (%d) cannot be greater than max (%d)", *varDef.Min, *varDef.Max),
			)
		}
	}

	return nil
}

// validateVarType validates that a variable type is valid.
func validateVarType(typ model.VarType) error {
	switch typ {
	case model.VarTypeString, model.VarTypeInt, model.VarTypeBool:
		return nil
	default:
		return fmt.Errorf("invalid variable type: %s (must be string, int, or bool)", typ)
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
	case model.VarTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	}
	return nil
}
