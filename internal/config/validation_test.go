package config

import (
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestValidateIgnJson(t *testing.T) {
	t.Run("valid ign.json", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:        "test-template",
			Version:     "1.0.0",
			Description: "Test",
			Variables: map[string]model.VarDef{
				"project_name": {
					Type:        model.VarTypeString,
					Description: "Project name",
					Required:    true,
				},
				"port": {
					Type:        model.VarTypeInt,
					Description: "Port number",
					Default:     8080,
				},
			},
		}

		if err := ValidateIgnJson(ign); err != nil {
			t.Errorf("Valid ign.json should pass validation: %v", err)
		}
	})

	t.Run("nil ign.json", func(t *testing.T) {
		if err := ValidateIgnJson(nil); err == nil {
			t.Error("Expected error for nil ign.json")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		ign := &model.IgnJson{
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("missing version", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "test",
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for missing version")
		}
	})

	t.Run("empty name string", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("empty version string", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "test",
			Version:   "",
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for empty version")
		}
	})

	t.Run("invalid name format", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "Test-Template", // uppercase not allowed
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for invalid name format")
		}
	})

	t.Run("invalid version format", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "test",
			Version:   "1.0", // not semver
			Variables: map[string]model.VarDef{},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for invalid version format")
		}
	})

	t.Run("negative max include depth", func(t *testing.T) {
		ign := &model.IgnJson{
			Name:      "test",
			Version:   "1.0.0",
			Variables: map[string]model.VarDef{},
			Settings: &model.TemplateSettings{
				MaxIncludeDepth: -1,
			},
		}
		if err := ValidateIgnJson(ign); err == nil {
			t.Error("Expected error for negative max include depth")
		}
	})
}

func TestValidateVariables(t *testing.T) {
	t.Run("valid variables", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"project_name": {
				Type:        model.VarTypeString,
				Description: "Project name",
				Pattern:     "^[a-z][a-z0-9-]*$",
			},
			"port": {
				Type:        model.VarTypeInt,
				Description: "Port",
				Default:     8080,
				Min:         floatPtr(1024),
				Max:         floatPtr(65535),
			},
			"enable_feature": {
				Type:        model.VarTypeBool,
				Description: "Enable feature",
				Default:     false,
			},
		}

		if err := validateVariables(vars); err != nil {
			t.Errorf("Valid variables should pass validation: %v", err)
		}
	})

	t.Run("invalid variable name", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"123invalid": { // starts with number
				Type:        model.VarTypeString,
				Description: "Invalid",
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for invalid variable name")
		}
	})

	t.Run("missing description", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"var_name": {
				Type: model.VarTypeString,
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for missing description")
		}
	})

	t.Run("invalid variable type", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"var_name": {
				Type:        model.VarType("invalid"),
				Description: "Test",
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for invalid variable type")
		}
	})

	t.Run("default value type mismatch", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"port": {
				Type:        model.VarTypeInt,
				Description: "Port",
				Default:     "8080", // string instead of int
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for default value type mismatch")
		}
	})

	t.Run("example value type mismatch", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"enabled": {
				Type:        model.VarTypeBool,
				Description: "Enabled",
				Example:     "true", // string instead of bool
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for example value type mismatch")
		}
	})

	t.Run("pattern on non-string variable", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"port": {
				Type:        model.VarTypeInt,
				Description: "Port",
				Pattern:     "^[0-9]+$",
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for pattern on non-string variable")
		}
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"name": {
				Type:        model.VarTypeString,
				Description: "Name",
				Pattern:     "[invalid(regex",
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})

	t.Run("min/max on non-numeric variable", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"name": {
				Type:        model.VarTypeString,
				Description: "Name",
				Min:         floatPtr(1),
				Max:         floatPtr(10),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for min/max on non-numeric variable")
		}
	})

	t.Run("min greater than max", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"port": {
				Type:        model.VarTypeInt,
				Description: "Port",
				Min:         floatPtr(10000),
				Max:         floatPtr(1000),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for min > max")
		}
	})

	t.Run("valid number variable", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate_limit": {
				Type:        model.VarTypeNumber,
				Description: "Rate limit per second",
				Default:     1.5,
				Example:     2.5,
				Min:         floatPtr(0.1),
				Max:         floatPtr(100.0),
			},
		}
		if err := validateVariables(vars); err != nil {
			t.Errorf("Valid number variable should pass validation: %v", err)
		}
	})

	t.Run("min greater than max for number", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate": {
				Type:        model.VarTypeNumber,
				Description: "Rate",
				Min:         floatPtr(10.0),
				Max:         floatPtr(1.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for min > max")
		}
	})

	t.Run("number default below min", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate": {
				Type:        model.VarTypeNumber,
				Description: "Rate",
				Default:     0.5,
				Min:         floatPtr(1.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for default value below min")
		}
	})

	t.Run("number default above max", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate": {
				Type:        model.VarTypeNumber,
				Description: "Rate",
				Default:     15.0,
				Max:         floatPtr(10.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for default value above max")
		}
	})

	t.Run("number example below min", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate": {
				Type:        model.VarTypeNumber,
				Description: "Rate",
				Example:     0.5,
				Min:         floatPtr(1.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for example value below min")
		}
	})

	t.Run("number example above max", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate": {
				Type:        model.VarTypeNumber,
				Description: "Rate",
				Example:     15.0,
				Max:         floatPtr(10.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for example value above max")
		}
	})

	t.Run("number variable with int default below min", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"count": {
				Type:        model.VarTypeNumber,
				Description: "Count",
				Default:     0,
				Min:         floatPtr(1.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for int default value below min")
		}
	})

	t.Run("number variable with int default above max", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"count": {
				Type:        model.VarTypeNumber,
				Description: "Count",
				Default:     100,
				Max:         floatPtr(10.0),
			},
		}
		if err := validateVariables(vars); err == nil {
			t.Error("Expected error for int default value above max")
		}
	})

	t.Run("number variable with int default in valid range", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"count": {
				Type:        model.VarTypeNumber,
				Description: "Count",
				Default:     5,
				Min:         floatPtr(1.0),
				Max:         floatPtr(10.0),
			},
		}
		if err := validateVariables(vars); err != nil {
			t.Errorf("Valid int default value should pass validation: %v", err)
		}
	})

	t.Run("number variable with both default and example in valid range", func(t *testing.T) {
		vars := map[string]model.VarDef{
			"rate_limit": {
				Type:        model.VarTypeNumber,
				Description: "Rate limit per second",
				Default:     1.5,
				Example:     2.5,
				Min:         floatPtr(0.1),
				Max:         floatPtr(100.0),
			},
		}
		if err := validateVariables(vars); err != nil {
			t.Errorf("Valid number variable with both default and example should pass validation: %v", err)
		}
	})
}

func TestValidateIgnVarJson(t *testing.T) {
	t.Run("valid ign-var.json", func(t *testing.T) {
		ignVar := &model.IgnVarJson{
			Variables: map[string]interface{}{
				"project_name": "test",
			},
		}

		if err := ValidateIgnVarJson(ignVar); err != nil {
			t.Errorf("Valid ign-var.json should pass validation: %v", err)
		}
	})

	t.Run("nil ign-var.json", func(t *testing.T) {
		if err := ValidateIgnVarJson(nil); err == nil {
			t.Error("Expected error for nil ign-var.json")
		}
	})

	t.Run("empty variables", func(t *testing.T) {
		ignVar := &model.IgnVarJson{
			Variables: map[string]interface{}{},
		}
		// Empty variables should be valid (will be validated against template ign.json later)
		if err := ValidateIgnVarJson(ignVar); err != nil {
			t.Errorf("Empty variables should be valid: %v", err)
		}
	})
}

func TestValidateIgnConfig(t *testing.T) {
	t.Run("valid ign.json", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{
				URL: "github.com/owner/repo",
				Ref: "v1.0.0",
			},
			Hash: "abc123def456789012345678901234567890123456789012345678901234abcd",
		}

		if err := ValidateIgnConfig(ignConfig); err != nil {
			t.Errorf("Valid ign.json should pass validation: %v", err)
		}
	})

	t.Run("nil ign.json", func(t *testing.T) {
		if err := ValidateIgnConfig(nil); err == nil {
			t.Error("Expected error for nil ign.json")
		}
	})

	t.Run("missing template URL", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{},
			Hash:     "abc123",
		}
		if err := ValidateIgnConfig(ignConfig); err == nil {
			t.Error("Expected error for missing template URL")
		}
	})

	t.Run("missing hash", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{
				URL: "github.com/owner/repo",
			},
			Hash: "",
		}
		if err := ValidateIgnConfig(ignConfig); err == nil {
			t.Error("Expected error for missing hash")
		}
	})

	t.Run("empty template URL", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{URL: ""},
			Hash:     "abc123def456789012345678901234567890123456789012345678901234abcd",
		}
		if err := ValidateIgnConfig(ignConfig); err == nil {
			t.Error("Expected error for empty URL")
		}
	})

	t.Run("invalid hash format - too short", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{
				URL: "github.com/owner/repo",
			},
			Hash: "abc123",
		}
		if err := ValidateIgnConfig(ignConfig); err == nil {
			t.Error("Expected error for invalid hash format")
		}
	})

	t.Run("invalid hash format - non-hex characters", func(t *testing.T) {
		ignConfig := &model.IgnConfig{
			Template: model.TemplateSource{
				URL: "github.com/owner/repo",
			},
			Hash: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		}
		if err := ValidateIgnConfig(ignConfig); err == nil {
			t.Error("Expected error for invalid hash format with non-hex characters")
		}
	})
}

func TestValidateVarType(t *testing.T) {
	tests := []struct {
		name    string
		varType model.VarType
		wantErr bool
	}{
		{"string type", model.VarTypeString, false},
		{"int type", model.VarTypeInt, false},
		{"number type", model.VarTypeNumber, false},
		{"bool type", model.VarTypeBool, false},
		{"invalid type", model.VarType("invalid"), true},
		{"empty type", model.VarType(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVarType(tt.varType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVarType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateValueType(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expectedType model.VarType
		wantErr      bool
	}{
		{"string match", "hello", model.VarTypeString, false},
		{"int match", 42, model.VarTypeInt, false},
		{"float as int", 42.0, model.VarTypeInt, false},
		{"bool match", true, model.VarTypeBool, false},
		{"number match float64", 3.14, model.VarTypeNumber, false},
		{"number match int", 42, model.VarTypeNumber, false},
		{"string mismatch", "hello", model.VarTypeInt, true},
		{"int mismatch", 42, model.VarTypeString, true},
		{"bool mismatch", true, model.VarTypeString, true},
		{"string as number mismatch", "3.14", model.VarTypeNumber, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValueType(tt.value, tt.expectedType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateValueType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	t.Run("error without field", func(t *testing.T) {
		err := NewConfigError(ConfigNotFound, "config.json", "file not found")
		if err.Error() == "" {
			t.Error("ConfigError.Error() returned empty string")
		}
		if err.File != "config.json" {
			t.Errorf("Expected file=config.json, got %s", err.File)
		}
	})

	t.Run("error with field", func(t *testing.T) {
		err := NewConfigErrorWithField(ConfigValidationFailed, "ign.json", "name", "name is required")
		errStr := err.Error()
		if errStr == "" {
			t.Error("ConfigError.Error() returned empty string")
		}
		// Should contain field name
		if err.Field != "name" {
			t.Errorf("Expected field=name, got %s", err.Field)
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := NewConfigError(ConfigInvalid, "test.json", "test error")
		err := NewConfigErrorWithCause(ConfigValidationFailed, "ign.json", "validation failed", cause)

		if err.Unwrap() != cause {
			t.Error("Unwrap() should return the cause")
		}
	})
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}
