package cli

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/tacogips/ign/internal/template/model"
)

// PromptForVariables interactively prompts the user for variable values.
// Returns a map of variable names to their values.
func PromptForVariables(ignJson *model.IgnJson) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	if len(ignJson.Variables) == 0 {
		return vars, nil
	}

	// Sort variable names for consistent ordering
	varNames := make([]string, 0, len(ignJson.Variables))
	for name := range ignJson.Variables {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	fmt.Println()
	fmt.Println("Please provide values for template variables:")
	fmt.Println()

	for _, name := range varNames {
		varDef := ignJson.Variables[name]

		value, err := promptForVariable(name, varDef)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for variable %q: %w", name, err)
		}

		vars[name] = value
	}

	return vars, nil
}

// promptForVariable prompts for a single variable based on its type.
func promptForVariable(name string, varDef model.VarDef) (interface{}, error) {
	// Build help text for '?' display
	// Use Help field if provided, otherwise fall back to Description
	help := varDef.Help
	if help == "" {
		help = varDef.Description
	}
	if varDef.Example != nil {
		help += fmt.Sprintf(" (example: %v)", varDef.Example)
	}

	switch varDef.Type {
	case model.VarTypeString:
		return promptString(name, varDef, help)
	case model.VarTypeInt:
		return promptInt(name, varDef, help)
	case model.VarTypeNumber:
		return promptNumber(name, varDef, help)
	case model.VarTypeBool:
		return promptBool(name, varDef, help)
	default:
		// Default to string for unknown types
		return promptString(name, varDef, help)
	}
}

// promptString prompts for a string variable.
func promptString(name string, varDef model.VarDef, help string) (string, error) {
	var result string

	// Build message with description displayed by default
	message := name
	if varDef.Description != "" {
		message += " - " + varDef.Description
	}
	if varDef.Required {
		message += " (required)"
	}

	// Get default value
	defaultVal := ""
	if varDef.Default != nil {
		if s, ok := varDef.Default.(string); ok {
			defaultVal = s
		}
	}

	prompt := &survey.Input{
		Message: message,
		Default: defaultVal,
		Help:    help,
	}

	// Build validators
	var validators []survey.Validator
	if varDef.Required {
		validators = append(validators, survey.Required)
	}
	if varDef.Pattern != "" {
		validators = append(validators, matchPattern(varDef.Pattern, "value must match pattern: "+varDef.Pattern))
	}

	opts := []survey.AskOpt{}
	if len(validators) > 0 {
		opts = append(opts, survey.WithValidator(survey.ComposeValidators(validators...)))
	}

	if err := survey.AskOne(prompt, &result, opts...); err != nil {
		return "", err
	}

	return result, nil
}

// promptInt prompts for an integer variable.
func promptInt(name string, varDef model.VarDef, help string) (int, error) {
	var result string

	// Build message with description displayed by default
	message := name
	if varDef.Description != "" {
		message += " - " + varDef.Description
	}
	if varDef.Required {
		message += " (required)"
	}
	if varDef.Min != nil || varDef.Max != nil {
		if varDef.Min != nil && varDef.Max != nil {
			message += fmt.Sprintf(" [%d-%d]", *varDef.Min, *varDef.Max)
		} else if varDef.Min != nil {
			message += fmt.Sprintf(" [>=%d]", *varDef.Min)
		} else {
			message += fmt.Sprintf(" [<=%d]", *varDef.Max)
		}
	}

	// Get default value
	defaultVal := ""
	if varDef.Default != nil {
		switch v := varDef.Default.(type) {
		case int:
			defaultVal = strconv.Itoa(v)
		case float64:
			defaultVal = strconv.Itoa(int(v))
		}
	}

	prompt := &survey.Input{
		Message: message,
		Default: defaultVal,
		Help:    help,
	}

	// Create validator for integer
	intValidator := func(val interface{}) error {
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", val)
		}

		if str == "" {
			if varDef.Required {
				return fmt.Errorf("value is required")
			}
			return nil
		}

		num, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("must be an integer")
		}

		if varDef.Min != nil && num < *varDef.Min {
			return fmt.Errorf("must be >= %d", *varDef.Min)
		}
		if varDef.Max != nil && num > *varDef.Max {
			return fmt.Errorf("must be <= %d", *varDef.Max)
		}

		return nil
	}

	if err := survey.AskOne(prompt, &result, survey.WithValidator(intValidator)); err != nil {
		return 0, err
	}

	if result == "" {
		return 0, nil
	}

	return strconv.Atoi(result)
}

// promptNumber prompts for a floating-point number variable.
func promptNumber(name string, varDef model.VarDef, help string) (float64, error) {
	var result string

	// Build message with description displayed by default
	message := name
	if varDef.Description != "" {
		message += " - " + varDef.Description
	}
	if varDef.Required {
		message += " (required)"
	}
	if varDef.MinFloat != nil || varDef.MaxFloat != nil {
		if varDef.MinFloat != nil && varDef.MaxFloat != nil {
			message += fmt.Sprintf(" [%v-%v]", *varDef.MinFloat, *varDef.MaxFloat)
		} else if varDef.MinFloat != nil {
			message += fmt.Sprintf(" [>=%v]", *varDef.MinFloat)
		} else {
			message += fmt.Sprintf(" [<=%v]", *varDef.MaxFloat)
		}
	}

	// Get default value
	defaultVal := ""
	if varDef.Default != nil {
		switch v := varDef.Default.(type) {
		case float64:
			defaultVal = strconv.FormatFloat(v, 'f', -1, 64)
		case float32:
			defaultVal = strconv.FormatFloat(float64(v), 'f', -1, 64)
		case int:
			defaultVal = strconv.FormatFloat(float64(v), 'f', -1, 64)
		}
	}

	prompt := &survey.Input{
		Message: message,
		Default: defaultVal,
		Help:    help,
	}

	// Create validator for number
	numberValidator := func(val interface{}) error {
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", val)
		}

		if str == "" {
			if varDef.Required {
				return fmt.Errorf("value is required")
			}
			return nil
		}

		num, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}

		if varDef.MinFloat != nil && num < *varDef.MinFloat {
			return fmt.Errorf("must be >= %v", *varDef.MinFloat)
		}
		if varDef.MaxFloat != nil && num > *varDef.MaxFloat {
			return fmt.Errorf("must be <= %v", *varDef.MaxFloat)
		}

		return nil
	}

	if err := survey.AskOne(prompt, &result, survey.WithValidator(numberValidator)); err != nil {
		return 0, err
	}

	// Note: For non-required number variables, empty input is treated as 0.
	// This means we cannot distinguish between "user entered 0" and "user entered nothing".
	// Both cases return 0. This is consistent with promptInt behavior.
	if result == "" {
		return 0, nil
	}

	return strconv.ParseFloat(result, 64)
}

// promptBool prompts for a boolean variable.
func promptBool(name string, varDef model.VarDef, help string) (bool, error) {
	var result bool

	// Build message with description displayed by default
	message := name
	if varDef.Description != "" {
		message += " - " + varDef.Description
	}
	if varDef.Required {
		message += " (required)"
	}

	// Get default value
	defaultVal := false
	if varDef.Default != nil {
		if b, ok := varDef.Default.(bool); ok {
			defaultVal = b
		}
	}

	prompt := &survey.Confirm{
		Message: message,
		Default: defaultVal,
		Help:    help,
	}

	if err := survey.AskOne(prompt, &result); err != nil {
		return false, err
	}

	return result, nil
}

// matchPattern creates a survey validator for regex pattern matching.
func matchPattern(pattern string, message string) survey.Validator {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return func(val interface{}) error {
			return fmt.Errorf("invalid pattern: %s", pattern)
		}
	}
	return func(val interface{}) error {
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
		if !re.MatchString(str) {
			return fmt.Errorf("%s", message)
		}
		return nil
	}
}
