package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestPrepareUpdateLoadsConfigFromOutputDirWhenCallerCwdDiffers(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "generated")
	callerDir := filepath.Join(root, "caller")
	if err := os.MkdirAll(callerDir, 0755); err != nil {
		t.Fatal(err)
	}

	templateDir := writeUpdateOutputRootTemplate(t, root, testHash2)
	writeProjectConfigAt(t, projectDir, templateDir, "main", map[string]interface{}{"project_name": "from-output"})

	t.Chdir(callerDir)

	result, err := PrepareUpdate(context.Background(), UpdateOptions{OutputDir: projectDir})
	if err != nil {
		t.Fatalf("PrepareUpdate returned error: %v", err)
	}
	if result.IgnConfigPath != filepath.Join(projectDir, model.IgnConfigDir, model.IgnProjectConfigFile) {
		t.Fatalf("IgnConfigPath = %q, want output-rooted path", result.IgnConfigPath)
	}
	if result.IgnVarPath != filepath.Join(projectDir, model.IgnConfigDir, model.IgnVarFile) {
		t.Fatalf("IgnVarPath = %q, want output-rooted path", result.IgnVarPath)
	}
	if result.ExistingVars["project_name"] != "from-output" {
		t.Fatalf("ExistingVars = %#v, want vars loaded from output dir", result.ExistingVars)
	}
}

func TestPrepareUpdateMissingProjectChecksOutputDirWhenCallerCwdHasIgn(t *testing.T) {
	root := t.TempDir()
	callerDir := filepath.Join(root, "caller")
	if err := os.MkdirAll(filepath.Join(callerDir, model.IgnConfigDir), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(callerDir)

	_, err := PrepareUpdate(context.Background(), UpdateOptions{OutputDir: filepath.Join(root, "missing-project")})
	if err == nil {
		t.Fatal("PrepareUpdate succeeded for missing output project")
	}
	if !strings.Contains(err.Error(), "update requires prior checkout") {
		t.Fatalf("PrepareUpdate error = %v, want prior checkout diagnostic", err)
	}
}

func TestCompleteUpdateUsesOutputDirConfigRootForFileVariables(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "generated")
	callerDir := filepath.Join(root, "caller")
	if err := os.MkdirAll(callerDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeProjectConfigAt(t, projectDir, "./template", "main", map[string]interface{}{"secret": "@file:secret.txt"})
	if err := os.WriteFile(filepath.Join(projectDir, model.IgnConfigDir, "secret.txt"), []byte("output-secret"), 0644); err != nil {
		t.Fatal(err)
	}

	template := &model.Template{
		Config: model.IgnJson{
			Name:    "output-root",
			Version: "1.0.0",
			Hash:    testHash2,
			Variables: map[string]model.VarDef{
				"secret": {Required: true},
			},
		},
		Files: []model.TemplateFile{{Path: "secret.txt", Content: []byte("@ign-var:secret@"), Mode: 0644}},
	}
	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{"secret": "@file:secret.txt"},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: filepath.Join(projectDir, model.IgnConfigDir, model.IgnProjectConfigFile),
		IgnVarPath:    filepath.Join(projectDir, model.IgnConfigDir, model.IgnVarFile),
		IgnConfig:     &model.IgnConfig{Template: model.TemplateSource{URL: "./template", Ref: "main"}, Hash: testHash1},
	}

	t.Chdir(callerDir)

	result, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     projectDir,
		Overwrite:     true,
	})
	if err != nil {
		t.Fatalf("CompleteUpdate returned error: %v", err)
	}
	if result.FilesOverwritten+result.FilesCreated == 0 {
		t.Fatal("CompleteUpdate did not write generated file")
	}
	content, err := os.ReadFile(filepath.Join(projectDir, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "output-secret" {
		t.Fatalf("generated content = %q, want output-rooted @file value", content)
	}
}

func writeUpdateOutputRootTemplate(t *testing.T, root string, hash string) string {
	t.Helper()
	templateDir := filepath.Join(root, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := `{"name":"output-root","version":"1.0.0","hash":"` + hash + `","variables":{"project_name":{"type":"string","required":true}}}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("@ign-var:project_name@"), 0644); err != nil {
		t.Fatal(err)
	}
	return templateDir
}

func writeProjectConfigAt(t *testing.T, projectDir string, templateURL string, ref string, vars map[string]interface{}) {
	t.Helper()
	ignDir := filepath.Join(projectDir, model.IgnConfigDir)
	if err := os.MkdirAll(ignDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveIgnConfig(filepath.Join(ignDir, model.IgnProjectConfigFile), &model.IgnConfig{
		Template: model.TemplateSource{URL: templateURL, Ref: ref},
		Hash:     testHash1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveIgnVarJson(filepath.Join(ignDir, model.IgnVarFile), &model.IgnVarJson{Variables: vars}); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveIgnManifest(filepath.Join(ignDir, model.IgnManifestFile), &model.IgnManifest{Files: []string{}}); err != nil {
		t.Fatal(err)
	}
}
