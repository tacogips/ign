package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestInspectVars_MergesDeclarationsAndCurrentValues(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := writeVarsTemplate(t, tempDir, map[string]model.VarDef{
		"enabled": {
			Type:        model.VarTypeBool,
			Description: "Enable feature",
			Required:    true,
		},
		"project_name": {
			Type:        model.VarTypeString,
			Description: "Project name",
			Required:    true,
			Default:     "demo",
		},
		"port": {
			Type:        model.VarTypeInt,
			Description: "Port",
			Default:     8080,
		},
	})
	writeProjectConfig(t, templateDir, "main", map[string]interface{}{
		"enabled": false,
		"extra":   "local-only",
		"port":    0,
	})

	result, err := InspectVars(context.Background(), VarsOptions{})
	if err != nil {
		t.Fatalf("InspectVars returned error: %v", err)
	}

	if !result.DeclarationsAvailable {
		t.Fatalf("DeclarationsAvailable = false, err = %v", result.DeclarationError)
	}
	if result.UnsetCount != 1 {
		t.Fatalf("UnsetCount = %d, want 1", result.UnsetCount)
	}
	if result.RequiredUnsetCount != 1 {
		t.Fatalf("RequiredUnsetCount = %d, want 1", result.RequiredUnsetCount)
	}

	rows := varsRowsByName(result.Rows)
	if rows["enabled"].Unset {
		t.Fatalf("enabled false value should count as set")
	}
	if rows["port"].Unset {
		t.Fatalf("port zero value should count as set")
	}
	if !rows["project_name"].Unset {
		t.Fatalf("project_name without current value should be unset")
	}
	if !rows["extra"].Declared && rows["extra"].Current != "local-only" {
		t.Fatalf("extra row = %#v, want local-only current value", rows["extra"])
	}
}

func TestInspectVars_FetchFailureReturnsLocalOnlyRows(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	writeProjectConfig(t, "./missing-template", "main", map[string]interface{}{
		"project_name": "demo",
	})

	result, err := InspectVars(context.Background(), VarsOptions{})
	if err != nil {
		t.Fatalf("InspectVars returned error: %v", err)
	}
	if result.DeclarationsAvailable {
		t.Fatalf("DeclarationsAvailable = true, want false")
	}
	if result.DeclarationError == nil {
		t.Fatalf("DeclarationError = nil, want fetch error")
	}
	if len(result.Rows) != 1 || result.Rows[0].Name != "project_name" || !result.Rows[0].HasCurrent {
		t.Fatalf("Rows = %#v, want local project_name row", result.Rows)
	}
}

func TestInspectVars_MissingConfigReturnsCheckoutStyleError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	_, err := InspectVars(context.Background(), VarsOptions{})
	if err == nil {
		t.Fatal("InspectVars expected missing config error")
	}
	if err.Error() == "" {
		t.Fatal("InspectVars returned empty error")
	}
}

func varsRowsByName(rows []VarsRow) map[string]VarsRow {
	result := make(map[string]VarsRow, len(rows))
	for _, row := range rows {
		result[row.Name] = row
	}
	return result
}

func writeVarsTemplate(t *testing.T, tempDir string, variables map[string]model.VarDef) string {
	t.Helper()

	templateDir := filepath.Join(tempDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}
	ignJSON := &model.IgnJson{
		Name:      "vars-template",
		Version:   "1.0.0",
		Hash:      testHash1,
		Variables: variables,
	}
	data, err := json.MarshalIndent(ignJSON, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), data, 0644); err != nil {
		t.Fatalf("failed to save template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}
	return "./template"
}

func writeProjectConfig(t *testing.T, templateURL, ref string, vars map[string]interface{}) {
	t.Helper()

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign: %v", err)
	}
	ignConfig := &model.IgnConfig{
		Template: model.TemplateSource{URL: templateURL, Ref: ref},
		Hash:     testHash1,
	}
	if err := config.SaveIgnConfig(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile), ignConfig); err != nil {
		t.Fatalf("failed to save ign config: %v", err)
	}
	if err := config.SaveIgnVarJson(filepath.Join(model.IgnConfigDir, model.IgnVarFile), &model.IgnVarJson{Variables: vars}); err != nil {
		t.Fatalf("failed to save ign vars: %v", err)
	}
}
