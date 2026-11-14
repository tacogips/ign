package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

// ParseVariableAssignments parses repeatable key=value CLI variable assignments.
func ParseVariableAssignments(assignments []string, varDefs map[string]model.VarDef) (map[string]interface{}, error) {
	if err := ValidateVariableAssignmentSyntax(assignments); err != nil {
		return nil, err
	}

	vars := make(map[string]interface{}, len(assignments))
	for _, assignment := range assignments {
		name, rawValue, _ := strings.Cut(assignment, "=")
		name = strings.TrimSpace(name)

		varDef, ok := varDefs[name]
		if !ok {
			return nil, fmt.Errorf("unknown template variable %q", name)
		}

		value, err := parseVariableValue(name, rawValue, varDef)
		if err != nil {
			return nil, err
		}
		vars[name] = value
	}
	return vars, nil
}

// ValidateVariableAssignmentSyntax validates key=value syntax without template metadata.
func ValidateVariableAssignmentSyntax(assignments []string) error {
	for _, assignment := range assignments {
		name, _, ok := strings.Cut(assignment, "=")
		if !ok {
			return fmt.Errorf("invalid variable assignment %q: expected key=value", assignment)
		}
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("invalid variable assignment %q: variable name cannot be empty", assignment)
		}
	}
	return nil
}

func parseVariableValue(name string, rawValue string, varDef model.VarDef) (interface{}, error) {
	switch varDef.Type {
	case model.VarTypeInt:
		value, err := strconv.Atoi(rawValue)
		if err != nil {
			return nil, fmt.Errorf("variable %q must be an integer: %w", name, err)
		}
		if err := validateNumericRange(name, float64(value), varDef); err != nil {
			return nil, err
		}
		return value, nil
	case model.VarTypeNumber:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return nil, fmt.Errorf("variable %q must be a number: %w", name, err)
		}
		if err := validateNumericRange(name, value, varDef); err != nil {
			return nil, err
		}
		return value, nil
	case model.VarTypeBool:
		value, err := strconv.ParseBool(rawValue)
		if err != nil {
			return nil, fmt.Errorf("variable %q must be a boolean: %w", name, err)
		}
		return value, nil
	case model.VarTypeString, "":
		if varDef.Required && strings.TrimSpace(rawValue) == "" {
			return nil, fmt.Errorf("variable %q is required", name)
		}
		if varDef.Pattern != "" {
			matched, err := regexp.MatchString(varDef.Pattern, rawValue)
			if err != nil {
				return nil, fmt.Errorf("variable %q has invalid pattern %q: %w", name, varDef.Pattern, err)
			}
			if !matched {
				return nil, fmt.Errorf("variable %q must match pattern: %s", name, varDef.Pattern)
			}
		}
		return rawValue, nil
	default:
		return rawValue, nil
	}
}

func validateNumericRange(name string, value float64, varDef model.VarDef) error {
	if varDef.Min != nil && value < *varDef.Min {
		return fmt.Errorf("variable %q must be >= %v", name, *varDef.Min)
	}
	if varDef.Max != nil && value > *varDef.Max {
		return fmt.Errorf("variable %q must be <= %v", name, *varDef.Max)
	}
	return nil
}
