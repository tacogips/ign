package cli

import (
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestParseVariableAssignments(t *testing.T) {
	min := 1000.0
	max := 9000.0
	varDefs := map[string]model.VarDef{
		"name": {
			Type:     model.VarTypeString,
			Required: true,
			Pattern:  `^[a-z][a-z0-9-]*$`,
		},
		"port": {
			Type: model.VarTypeInt,
			Min:  &min,
			Max:  &max,
		},
		"ratio": {
			Type: model.VarTypeNumber,
		},
		"enabled": {
			Type: model.VarTypeBool,
		},
	}

	got, err := ParseVariableAssignments([]string{
		"name=my-app",
		"port=8080",
		"ratio=0.75",
		"enabled=true",
		"name=my-other-app",
	}, varDefs)
	if err != nil {
		t.Fatalf("ParseVariableAssignments() returned error: %v", err)
	}

	if got["name"] != "my-other-app" {
		t.Fatalf("name = %v, want last repeated value", got["name"])
	}
	if got["port"] != 8080 {
		t.Fatalf("port = %v, want 8080", got["port"])
	}
	if got["ratio"] != 0.75 {
		t.Fatalf("ratio = %v, want 0.75", got["ratio"])
	}
	if got["enabled"] != true {
		t.Fatalf("enabled = %v, want true", got["enabled"])
	}
}

func TestParseVariableAssignments_Invalid(t *testing.T) {
	varDefs := map[string]model.VarDef{
		"name":    {Type: model.VarTypeString, Required: true},
		"port":    {Type: model.VarTypeInt},
		"enabled": {Type: model.VarTypeBool},
	}

	tests := []struct {
		name        string
		assignments []string
	}{
		{name: "missing equals", assignments: []string{"name"}},
		{name: "empty name", assignments: []string{"=value"}},
		{name: "unknown variable", assignments: []string{"missing=value"}},
		{name: "empty required string", assignments: []string{"name="}},
		{name: "invalid int", assignments: []string{"port=abc"}},
		{name: "invalid bool", assignments: []string{"enabled=maybe"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseVariableAssignments(tt.assignments, varDefs); err == nil {
				t.Fatalf("ParseVariableAssignments() expected error")
			}
		})
	}
}

func TestValidateVariableAssignmentSyntax(t *testing.T) {
	tests := []struct {
		name        string
		assignments []string
		wantErr     bool
	}{
		{name: "empty", assignments: nil},
		{name: "single valid", assignments: []string{"name=value"}},
		{name: "value contains equals", assignments: []string{"name=value=with=equals"}},
		{name: "missing equals", assignments: []string{"name"}, wantErr: true},
		{name: "empty name", assignments: []string{"=value"}, wantErr: true},
		{name: "blank name", assignments: []string{"   =value"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableAssignmentSyntax(tt.assignments)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateVariableAssignmentSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPromptForVariablesWithProvided_AllProvided(t *testing.T) {
	ignJSON := &model.IgnJson{
		Variables: map[string]model.VarDef{
			"name": {Type: model.VarTypeString, Required: true},
			"port": {Type: model.VarTypeInt, Required: true},
		},
	}

	got, err := PromptForVariablesWithProvided(ignJSON, map[string]interface{}{
		"name": "my-app",
		"port": 8080,
	})
	if err != nil {
		t.Fatalf("PromptForVariablesWithProvided() returned error: %v", err)
	}
	if got["name"] != "my-app" || got["port"] != 8080 {
		t.Fatalf("PromptForVariablesWithProvided() = %v", got)
	}
}
