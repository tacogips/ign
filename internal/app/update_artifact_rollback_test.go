package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestCompleteUpdateRestoresManifestAfterConfigPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	setupTestTemplate(t, root, testHash1)

	ignDir := filepath.Join(root, model.IgnConfigDir)
	manifestPath := filepath.Join(ignDir, model.IgnManifestFile)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{"old-managed.txt"}}); err != nil {
		t.Fatalf("save original manifest: %v", err)
	}
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}

	ignConfigPath := filepath.Join(ignDir, model.IgnProjectConfigFile)
	configBefore, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("read original ign.json: %v", err)
	}
	originalSave := saveIgnConfigWithResult
	saveIgnConfigWithResult = func(path string, value *model.IgnConfig) (config.AtomicWriteResult, error) {
		result, saveErr := originalSave(path, value)
		if saveErr != nil {
			return result, saveErr
		}
		return result, errors.New("injected post-rename config persistence failure")
	}
	t.Cleanup(func() { saveIgnConfigWithResult = originalSave })

	template := &model.Template{
		Config: model.IgnJson{Name: "rollback-test", Version: "1.0.0", Hash: testHash2, Variables: map[string]model.VarDef{}},
		Files:  []model.TemplateFile{{Path: "generated.txt", Content: []byte("generated")}},
	}
	prep := &PrepareUpdateResult{
		Template:      template,
		IgnJson:       &template.Config,
		ExistingVars:  map[string]interface{}{},
		CurrentHash:   testHash1,
		NewHash:       testHash2,
		HashChanged:   true,
		IgnConfigPath: ignConfigPath,
		IgnVarPath:    filepath.Join(ignDir, model.IgnVarFile),
		IgnConfig:     &model.IgnConfig{Template: model.TemplateSource{URL: "./template", Ref: "main"}, Hash: testHash1},
	}

	_, err = CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
		Overwrite:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "failed to save ign.json") {
		t.Fatalf("CompleteUpdate error = %v, want ign.json save failure", err)
	}
	assertArtifactRollbackFileContent(t, manifestPath, manifestBefore)
	assertArtifactRollbackFileContent(t, ignConfigPath, configBefore)
	if _, err := os.Lstat(filepath.Join(root, "generated.txt")); !os.IsNotExist(err) {
		t.Fatalf("generated file = %v; want rollback removal", err)
	}
}

func TestCompleteUpdateConfigOnlyRestoresConfigAfterVarPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	setupTestTemplate(t, root, testHash1)

	ignDir := filepath.Join(root, model.IgnConfigDir)
	ignConfigPath := filepath.Join(ignDir, model.IgnProjectConfigFile)
	configBefore, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("read original ign.json: %v", err)
	}
	ignVarPath := filepath.Join(ignDir, model.IgnVarFile)
	if err := os.Remove(ignVarPath); err != nil {
		t.Fatalf("remove ign-var.json fixture: %v", err)
	}
	if err := os.Mkdir(ignVarPath, 0755); err != nil {
		t.Fatalf("create ign-var.json failure fixture: %v", err)
	}

	template := &model.Template{Config: model.IgnJson{Name: "rollback-test", Version: "1.0.0", Hash: testHash1, Variables: map[string]model.VarDef{}}}
	prep := &PrepareUpdateResult{
		Template:             template,
		IgnJson:              &template.Config,
		ExistingVars:         map[string]interface{}{},
		CurrentHash:          testHash1,
		NewHash:              testHash1,
		IgnConfigPath:        ignConfigPath,
		IgnVarPath:           ignVarPath,
		IgnConfig:            &model.IgnConfig{Template: model.TemplateSource{URL: "./template", Ref: "main"}, Hash: testHash1},
		PreviousRef:          "main",
		RequestedRef:         "v2.0.0",
		RefOverrideRequested: true,
		RefChanged:           true,
	}

	_, err = CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep,
		NewVariables:  map[string]interface{}{},
		OutputDir:     root,
	})
	if err == nil || !strings.Contains(err.Error(), "failed to save ign-var.json") {
		t.Fatalf("CompleteUpdate error = %v, want ign-var.json save failure", err)
	}
	assertArtifactRollbackFileContent(t, ignConfigPath, configBefore)
}

func TestCompleteUpdateConfigOnlyRestoresCommittedConfigAfterReportedFailure(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	setupTestTemplate(t, root, testHash1)

	ignDir := filepath.Join(root, model.IgnConfigDir)
	ignConfigPath := filepath.Join(ignDir, model.IgnProjectConfigFile)
	before, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("read original ign.json: %v", err)
	}
	originalSave := saveIgnConfigWithResult
	saveIgnConfigWithResult = func(path string, value *model.IgnConfig) (config.AtomicWriteResult, error) {
		result, saveErr := originalSave(path, value)
		if saveErr != nil {
			return result, saveErr
		}
		return result, errors.New("injected post-rename persistence failure")
	}
	t.Cleanup(func() { saveIgnConfigWithResult = originalSave })

	template := &model.Template{Config: model.IgnJson{Name: "rollback-test", Version: "1.0.0", Hash: testHash1, Variables: map[string]model.VarDef{}}}
	prep := &PrepareUpdateResult{
		Template: template, IgnJson: &template.Config, ExistingVars: map[string]interface{}{}, CurrentHash: testHash1, NewHash: testHash1,
		IgnConfigPath: ignConfigPath, IgnVarPath: filepath.Join(ignDir, model.IgnVarFile),
		IgnConfig:   &model.IgnConfig{Template: model.TemplateSource{URL: "./template", Ref: "main"}, Hash: testHash1},
		PreviousRef: "main", RequestedRef: "v2.0.0", RefOverrideRequested: true, RefChanged: true,
	}

	_, err = CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root})
	if err == nil || !strings.Contains(err.Error(), "failed to save ign.json") {
		t.Fatalf("CompleteUpdate error = %v, want committed ign.json save failure", err)
	}
	assertArtifactRollbackFileContent(t, ignConfigPath, before)
}

func TestCompleteUpdateConfigOnlyRetainsConcurrentArtifactReplacement(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	setupTestTemplate(t, root, testHash1)

	ignDir := filepath.Join(root, model.IgnConfigDir)
	ignConfigPath := filepath.Join(ignDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(ignDir, model.IgnVarFile)
	if err := os.Remove(ignVarPath); err != nil {
		t.Fatalf("remove ign-var.json fixture: %v", err)
	}
	if err := os.Mkdir(ignVarPath, 0755); err != nil {
		t.Fatalf("create ign-var.json failure fixture: %v", err)
	}
	originalSave := saveIgnConfigWithResult
	saveIgnConfigWithResult = func(path string, value *model.IgnConfig) (config.AtomicWriteResult, error) {
		result, saveErr := originalSave(path, value)
		if saveErr == nil {
			saveErr = os.WriteFile(path, []byte("concurrent artifact"), 0644)
		}
		return result, saveErr
	}
	t.Cleanup(func() { saveIgnConfigWithResult = originalSave })

	template := &model.Template{Config: model.IgnJson{Name: "rollback-test", Version: "1.0.0", Hash: testHash1, Variables: map[string]model.VarDef{}}}
	prep := &PrepareUpdateResult{
		Template: template, IgnJson: &template.Config, ExistingVars: map[string]interface{}{}, CurrentHash: testHash1, NewHash: testHash1,
		IgnConfigPath: ignConfigPath, IgnVarPath: ignVarPath,
		IgnConfig:   &model.IgnConfig{Template: model.TemplateSource{URL: "./template", Ref: "main"}, Hash: testHash1},
		PreviousRef: "main", RequestedRef: "v2.0.0", RefOverrideRequested: true, RefChanged: true,
	}

	_, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root})
	if err == nil || !strings.Contains(err.Error(), "failed to save ign-var.json") {
		t.Fatalf("CompleteUpdate error = %v, want ign-var.json save failure", err)
	}
	assertArtifactRollbackFileContent(t, ignConfigPath, []byte("concurrent artifact"))
}

func assertArtifactRollbackFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}
