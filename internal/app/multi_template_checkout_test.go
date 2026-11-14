package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestMultiTemplateCheckout_AddsTemplateAndRemoveRestoresHiddenFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"common.txt": "alpha\n",
	})
	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"common.txt": "beta\n",
		"beta.txt":   "beta-only\n",
	})

	checkoutTemplate(t, alphaPath, false)
	assertFileContent(t, "common.txt", "alpha\n")

	checkoutTemplate(t, betaPath, false)
	assertFileContent(t, "common.txt", "beta\n")
	assertFileContent(t, "beta.txt", "beta-only\n")

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to load project state: %v", err)
	}
	if state.Version != model.IgnProjectMetadataVersionV2 {
		t.Fatalf("state version = %d, want v2", state.Version)
	}
	if got := len(state.Templates); got != 2 {
		t.Fatalf("template count = %d, want 2", got)
	}
	if state.Templates[0].ID != "alpha-1" || state.Templates[1].ID != "beta-2" {
		t.Fatalf("template ids = %q, %q; want alpha-1, beta-2", state.Templates[0].ID, state.Templates[1].ID)
	}

	result, err := Remove(context.Background(), RemoveOptions{
		TemplateIDs: []string{"beta-2"},
		OutputDir:   ".",
	})
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if result.TemplatesRemoved != 1 {
		t.Fatalf("TemplatesRemoved = %d, want 1", result.TemplatesRemoved)
	}

	assertFileContent(t, "common.txt", "alpha\n")
	if _, err := os.Stat("beta.txt"); !os.IsNotExist(err) {
		t.Fatalf("beta.txt should be removed, stat err: %v", err)
	}

	state, err = config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to reload project state: %v", err)
	}
	if got := len(state.Templates); got != 1 {
		t.Fatalf("remaining template count = %d, want 1", got)
	}
	if state.Templates[0].ID != "alpha-1" {
		t.Fatalf("remaining template id = %q, want alpha-1", state.Templates[0].ID)
	}
}

func TestMultiTemplateCheckout_MigratesV1OnSecondCheckout(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"second.txt": "second\n",
	})

	if err := os.Mkdir(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign: %v", err)
	}
	writeJSONFile(t, filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile), &model.IgnConfig{
		Template: model.TemplateSource{URL: "./alpha-template"},
		Hash:     strings.Repeat("a", 64),
		Metadata: &model.FileMetadata{
			TemplateName: "alpha",
		},
	})
	writeJSONFile(t, filepath.Join(model.IgnConfigDir, model.IgnVarFile), &model.IgnVarJson{
		Variables: map[string]interface{}{"name": "alpha"},
	})
	writeJSONFile(t, filepath.Join(model.IgnConfigDir, model.IgnManifestFile), &model.IgnManifest{
		Files: []string{"first.txt"},
	})

	checkoutTemplate(t, betaPath, true)

	var ignConfig model.IgnConfig
	readJSONFile(t, filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile), &ignConfig)
	if ignConfig.Version != model.IgnProjectMetadataVersionV2 {
		t.Fatalf("ign.json version = %d, want v2", ignConfig.Version)
	}
	if got := len(ignConfig.Templates); got != 2 {
		t.Fatalf("template count = %d, want 2", got)
	}
	if ignConfig.Templates[0].ID != "alpha-1" || ignConfig.Templates[1].ID != "beta-2" {
		t.Fatalf("template ids = %q, %q; want alpha-1, beta-2", ignConfig.Templates[0].ID, ignConfig.Templates[1].ID)
	}
}

func TestMultiTemplateCheckout_DoesNotManageSkippedUnmanagedFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templatePath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"existing.txt": "template\n",
	})
	if err := os.WriteFile("existing.txt", []byte("user\n"), 0644); err != nil {
		t.Fatalf("failed to write unmanaged file: %v", err)
	}

	checkoutTemplate(t, templatePath, false)

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to load project state: %v", err)
	}
	if got := len(state.Templates[0].Files); got != 0 {
		t.Fatalf("managed files count = %d, want 0 for skipped unmanaged file", got)
	}

	if _, err := Remove(context.Background(), RemoveOptions{All: true, OutputDir: "."}); err != nil {
		t.Fatalf("Remove all failed: %v", err)
	}
	assertFileContent(t, "existing.txt", "user\n")
}

func TestMultiTemplateRemove_ReplayDoesNotOverwriteSkippedUnmanagedFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"alpha.txt": "alpha\n",
	})
	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"beta.txt": "beta\n",
		"user.txt": "template\n",
	})
	if err := os.WriteFile("user.txt", []byte("user\n"), 0644); err != nil {
		t.Fatalf("failed to write unmanaged file: %v", err)
	}

	checkoutTemplate(t, alphaPath, false)
	checkoutTemplate(t, betaPath, true)
	assertFileContent(t, "user.txt", "user\n")

	if _, err := Remove(context.Background(), RemoveOptions{
		TemplateIDs: []string{"alpha-1"},
		OutputDir:   ".",
	}); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	assertFileContent(t, "user.txt", "user\n")
	assertFileContent(t, "beta.txt", "beta\n")
	if _, err := os.Stat("alpha.txt"); !os.IsNotExist(err) {
		t.Fatalf("alpha.txt should be removed, stat err: %v", err)
	}
}

func TestMultiTemplateCheckout_UsesStoredOutputRootWhenOutputPathOmitted(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"alpha.txt": "alpha\n",
	})
	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"beta.txt": "beta\n",
	})

	checkoutTemplateToOutput(t, alphaPath, "project", false, true)

	resolvedOutput, err := ResolveCheckoutOutputDir(".", false, false)
	if err != nil {
		t.Fatalf("ResolveCheckoutOutputDir failed: %v", err)
	}
	if resolvedOutput != "project" {
		t.Fatalf("resolved output = %q, want project", resolvedOutput)
	}

	checkoutTemplateToOutput(t, betaPath, resolvedOutput, true, false)

	assertFileContent(t, filepath.Join("project", "alpha.txt"), "alpha\n")
	assertFileContent(t, filepath.Join("project", "beta.txt"), "beta\n")

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to load project state: %v", err)
	}
	if state.OutputRoot != "project" {
		t.Fatalf("output root = %q, want project", state.OutputRoot)
	}
	if got := len(state.Templates); got != 2 {
		t.Fatalf("template count = %d, want 2", got)
	}
}

func TestMultiTemplateCheckout_RejectsExplicitMismatchedOutputRoot(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"alpha.txt": "alpha\n",
	})
	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"beta.txt": "beta\n",
	})

	checkoutTemplateToOutput(t, alphaPath, "project", false, true)

	prep, err := PrepareCheckout(context.Background(), PrepareCheckoutOptions{
		URL:          betaPath,
		ConfigExists: true,
	})
	if err != nil {
		t.Fatalf("PrepareCheckout failed: %v", err)
	}
	if _, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult:     prep,
		Variables:         map[string]interface{}{},
		OutputDir:         ".",
		OutputDirExplicit: true,
	}); err == nil {
		t.Fatal("CompleteCheckout succeeded with explicit mismatched output root")
	}
}

func TestMultiTemplateUpdate_UpdatesAllTemplateRecords(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"alpha.txt": "alpha-v1\n",
	})
	betaPath := writeTestTemplate(t, tempDir, "beta", strings.Repeat("b", 64), map[string]string{
		"beta.txt": "beta-v1\n",
	})

	checkoutTemplate(t, alphaPath, false)
	checkoutTemplate(t, betaPath, true)

	updateTestTemplate(t, filepath.Join(tempDir, "alpha-template"), "alpha", strings.Repeat("c", 64), map[string]string{
		"alpha.txt": "alpha-v2\n",
	})
	updateTestTemplateWithVars(t, filepath.Join(tempDir, "beta-template"), "beta", strings.Repeat("d", 64), map[string]string{
		"beta.txt": "beta-@ign-var:suffix@\n",
	}, map[string]model.VarDef{
		"suffix": {
			Type:     model.VarTypeString,
			Required: true,
		},
	})

	prep, err := PrepareUpdate(context.Background(), UpdateOptions{
		OutputDir: ".",
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("PrepareUpdate failed: %v", err)
	}
	if !prep.HashChanged {
		t.Fatal("PrepareUpdate HashChanged = false, want true")
	}
	if got := prep.TemplateUpdates[1].NewVars; len(got) != 1 || got[0] != "suffix" {
		t.Fatalf("second template NewVars = %v, want [suffix]", got)
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariablesByTemplateID: map[string]map[string]interface{}{
			"beta-2": {"suffix": "v2"},
		},
		OutputDir: ".",
		Overwrite: true,
	}); err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	assertFileContent(t, "alpha.txt", "alpha-v2\n")
	assertFileContent(t, "beta.txt", "beta-v2\n")

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to load project state: %v", err)
	}
	if state.Templates[0].Hash != strings.Repeat("c", 64) {
		t.Fatalf("alpha hash not updated: %s", state.Templates[0].Hash)
	}
	if state.Templates[1].Hash != strings.Repeat("d", 64) {
		t.Fatalf("beta hash not updated: %s", state.Templates[1].Hash)
	}
}

func TestMultiTemplateUpdate_SelectiveOverwriteRespectsIgnoreForManagedPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"keep.txt": "v1\n",
	})
	checkoutTemplate(t, alphaPath, false)

	updateTestTemplate(t, filepath.Join(tempDir, "alpha-template"), "alpha", strings.Repeat("c", 64), map[string]string{
		"keep.txt":                   "v2\n",
		model.IgnOverwriteIgnoreFile: "keep.txt\n",
	})

	prep, err := PrepareUpdate(context.Background(), UpdateOptions{
		OutputDir:     ".",
		Overwrite:     true,
		OverwriteMode: generator.OverwriteSelective,
	})
	if err != nil {
		t.Fatalf("PrepareUpdate failed: %v", err)
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     ".",
		Overwrite:     true,
		OverwriteMode: generator.OverwriteSelective,
	}); err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	assertFileContent(t, "keep.txt", "v1\n")
}

func TestMultiTemplateUpdate_PreservesUnchangedManagedFilesInManifest(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"same.txt":    "same\n",
		"changed.txt": "v1\n",
	})
	checkoutTemplate(t, alphaPath, false)

	updateTestTemplate(t, filepath.Join(tempDir, "alpha-template"), "alpha", strings.Repeat("c", 64), map[string]string{
		"same.txt":    "same\n",
		"changed.txt": "v2\n",
	})

	prep, err := PrepareUpdate(context.Background(), UpdateOptions{
		OutputDir: ".",
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("PrepareUpdate failed: %v", err)
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     ".",
		Overwrite:     true,
	}); err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	state, err := config.LoadProjectState(model.IgnConfigDir)
	if err != nil {
		t.Fatalf("failed to load project state: %v", err)
	}
	if got := strings.Join(state.Templates[0].Files, ","); got != "changed.txt,same.txt" {
		t.Fatalf("managed files = %q, want changed.txt,same.txt", got)
	}

	if _, err := Remove(context.Background(), RemoveOptions{All: true, OutputDir: "."}); err != nil {
		t.Fatalf("Remove all failed: %v", err)
	}
	if _, err := os.Stat("same.txt"); !os.IsNotExist(err) {
		t.Fatalf("same.txt should be removed, stat err: %v", err)
	}
	if _, err := os.Stat("changed.txt"); !os.IsNotExist(err) {
		t.Fatalf("changed.txt should be removed, stat err: %v", err)
	}
}

func TestRewindRejectsMismatchedV2OutputRoot(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	alphaPath := writeTestTemplate(t, tempDir, "alpha", strings.Repeat("a", 64), map[string]string{
		"alpha.txt": "alpha\n",
	})
	checkoutTemplateToOutput(t, alphaPath, "project", false, true)

	if _, err := Rewind(context.Background(), RewindOptions{OutputDir: "."}); err == nil {
		t.Fatal("Rewind succeeded with mismatched output root")
	}
	assertFileContent(t, filepath.Join("project", "alpha.txt"), "alpha\n")
	if _, err := os.Stat(model.IgnConfigDir); err != nil {
		t.Fatalf(".ign should be preserved, stat err: %v", err)
	}
}

func checkoutTemplate(t *testing.T, templatePath string, configExists bool) {
	t.Helper()

	checkoutTemplateToOutput(t, templatePath, ".", configExists, false)
}

func checkoutTemplateToOutput(t *testing.T, templatePath string, outputDir string, configExists bool, outputDirExplicit bool) {
	t.Helper()

	prep, err := PrepareCheckout(context.Background(), PrepareCheckoutOptions{
		URL:          templatePath,
		ConfigExists: configExists,
	})
	if err != nil {
		t.Fatalf("PrepareCheckout failed: %v", err)
	}
	if _, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult:     prep,
		Variables:         map[string]interface{}{},
		OutputDir:         outputDir,
		OutputDirExplicit: outputDirExplicit,
	}); err != nil {
		t.Fatalf("CompleteCheckout failed: %v", err)
	}
}

func updateTestTemplate(t *testing.T, templateDir, name, hash string, files map[string]string) {
	t.Helper()

	updateTestTemplateWithVars(t, templateDir, name, hash, files, map[string]model.VarDef{})
}

func updateTestTemplateWithVars(t *testing.T, templateDir, name, hash string, files map[string]string, vars map[string]model.VarDef) {
	t.Helper()

	writeJSONFile(t, filepath.Join(templateDir, model.IgnTemplateConfigFile), &model.IgnJson{
		Name:      name,
		Version:   "1.0.0",
		Variables: vars,
		Hash:      hash,
	})
	for path, content := range files {
		fullPath := filepath.Join(templateDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to update template file %s: %v", path, err)
		}
	}
}

func writeTestTemplate(t *testing.T, baseDir, name, hash string, files map[string]string) string {
	t.Helper()

	templateDir := filepath.Join(baseDir, name+"-template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}
	writeJSONFile(t, filepath.Join(templateDir, model.IgnTemplateConfigFile), &model.IgnJson{
		Name:      name,
		Version:   "1.0.0",
		Variables: map[string]model.VarDef{},
		Hash:      hash,
	})
	for path, content := range files {
		fullPath := filepath.Join(templateDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write template file %s: %v", path, err)
		}
	}
	return "./" + filepath.Base(templateDir)
}

func writeJSONFile(t *testing.T, path string, value interface{}) {
	t.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func readJSONFile(t *testing.T, path string, target interface{}) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}
