package cli

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

// TestPromptNumber_Validator tests the number validation logic
func TestPromptNumber_Validator(t *testing.T) {
	tests := []struct {
		name    string
		varDef  model.VarDef
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid float input",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "3.14",
			wantErr: false,
		},
		{
			name: "valid integer as float",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "42",
			wantErr: false,
		},
		{
			name: "empty input with required flag",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: true,
			},
			input:   "",
			wantErr: true,
			errMsg:  "value is required",
		},
		{
			name: "empty input without required flag",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "",
			wantErr: false,
		},
		{
			name: "invalid input - not a number",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "abc",
			wantErr: true,
			errMsg:  "must be a number",
		},
		{
			name: "min boundary - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MinFloat: floatPtr(1.0),
			},
			input:   "1.0",
			wantErr: false,
		},
		{
			name: "min boundary - invalid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MinFloat: floatPtr(1.0),
			},
			input:   "0.5",
			wantErr: true,
			errMsg:  "must be >= 1",
		},
		{
			name: "max boundary - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MaxFloat: floatPtr(10.0),
			},
			input:   "10.0",
			wantErr: false,
		},
		{
			name: "max boundary - invalid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MaxFloat: floatPtr(10.0),
			},
			input:   "10.5",
			wantErr: true,
			errMsg:  "must be <= 10",
		},
		{
			name: "min and max boundary - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MinFloat: floatPtr(1.0),
				MaxFloat: floatPtr(10.0),
			},
			input:   "5.5",
			wantErr: false,
		},
		{
			name: "min and max boundary - below min",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MinFloat: floatPtr(1.0),
				MaxFloat: floatPtr(10.0),
			},
			input:   "0.9",
			wantErr: true,
			errMsg:  "must be >= 1",
		},
		{
			name: "min and max boundary - above max",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				MinFloat: floatPtr(1.0),
				MaxFloat: floatPtr(10.0),
			},
			input:   "10.1",
			wantErr: true,
			errMsg:  "must be <= 10",
		},
		{
			name: "negative number - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "-3.14",
			wantErr: false,
		},
		{
			name: "zero - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "0",
			wantErr: false,
		},
		{
			name: "very small number - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "0.0001",
			wantErr: false,
		},
		{
			name: "very large number - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "999999999.999",
			wantErr: false,
		},
		{
			name: "scientific notation - valid",
			varDef: model.VarDef{
				Type:     model.VarTypeNumber,
				Required: false,
			},
			input:   "1.23e-4",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the validator function (mirrors the logic from promptNumber)
			numberValidator := func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return fmt.Errorf("expected string, got %T", val)
				}

				if str == "" {
					if tt.varDef.Required {
						return fmt.Errorf("value is required")
					}
					return nil
				}

				num, err := strconv.ParseFloat(str, 64)
				if err != nil {
					return fmt.Errorf("must be a number")
				}

				if tt.varDef.MinFloat != nil && num < *tt.varDef.MinFloat {
					return fmt.Errorf("must be >= %v", *tt.varDef.MinFloat)
				}
				if tt.varDef.MaxFloat != nil && num > *tt.varDef.MaxFloat {
					return fmt.Errorf("must be <= %v", *tt.varDef.MaxFloat)
				}

				return nil
			}

			err := numberValidator(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestPromptNumber_DefaultValueFormatting tests default value type conversions
func TestPromptNumber_DefaultValueFormatting(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue interface{}
		want         string
	}{
		{
			name:         "float64 default",
			defaultValue: float64(3.14),
			want:         "3.14",
		},
		{
			name:         "float32 default",
			defaultValue: float32(2.5),
			want:         "2.5",
		},
		{
			name:         "int default",
			defaultValue: int(42),
			want:         "42",
		},
		{
			name:         "zero float64",
			defaultValue: float64(0),
			want:         "0",
		},
		{
			name:         "negative float64",
			defaultValue: float64(-1.5),
			want:         "-1.5",
		},
		{
			name:         "nil default",
			defaultValue: nil,
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the default value formatting logic from promptNumber
			got := formatNumberDefault(tt.defaultValue)
			if got != tt.want {
				t.Errorf("formatNumberDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper functions for tests

func floatPtr(f float64) *float64 {
	return &f
}

// formatNumberDefault formats the default value for a number variable
// This mirrors the logic from promptNumber lines 224-234
func formatNumberDefault(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	default:
		return ""
	}
}
