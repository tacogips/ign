package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/app"
)

func TestVarsCmd_FlagRegistration(t *testing.T) {
	if rootCmd.CommandPath() == "" {
		t.Fatal("root command is not initialized")
	}
	if varsCmd.Flags().Lookup("json") == nil {
		t.Fatal("vars --json flag is not registered")
	}
	if varsCmd.Flags().Lookup("unset") == nil {
		t.Fatal("vars --unset flag is not registered")
	}
}

func TestPrintVarsTable(t *testing.T) {
	result := &app.VarsResult{
		DeclarationsAvailable: true,
		Rows: []app.VarsRow{
			{Name: "project_name", Type: "string", Required: true, Default: "demo", Current: "current", HasCurrent: true, Description: "Project name", Declared: true},
		},
	}

	var out bytes.Buffer
	if err := printVarsTable(&out, result); err != nil {
		t.Fatalf("printVarsTable returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"NAME", "TYPE", "REQUIRED", "DEFAULT", "CURRENT", "DESCRIPTION", "project_name", "current"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output %q does not contain %q", got, want)
		}
	}
}

func TestPrintVarsJSONIsScriptSafe(t *testing.T) {
	result := &app.VarsResult{
		DeclarationsAvailable: false,
		Rows: []app.VarsRow{
			{Name: "project_name", Current: "demo", HasCurrent: true},
		},
		UnsetCount: 0,
	}

	var out bytes.Buffer
	if err := printVarsJSON(&out, result); err != nil {
		t.Fatalf("printVarsJSON returned error: %v", err)
	}

	var decoded app.VarsResult
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output did not parse: %v\n%s", err, out.String())
	}
	if decoded.Rows[0].Name != "project_name" {
		t.Fatalf("decoded rows = %#v", decoded.Rows)
	}
}

func TestVarsUnsetRequiredError(t *testing.T) {
	result := app.FilterUnsetVars(&app.VarsResult{
		Rows: []app.VarsRow{
			{Name: "project_name", Required: true, Unset: true, Declared: true},
			{Name: "port", Required: false, HasCurrent: true, Declared: true},
		},
		UnsetCount:         1,
		RequiredUnsetCount: 1,
	})

	if result.RequiredUnsetCount != 1 {
		t.Fatalf("RequiredUnsetCount = %d, want 1", result.RequiredUnsetCount)
	}
}
