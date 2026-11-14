package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestSaveProjectState_RestoresPreviousMetadataSetOnReplaceFailure(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), model.IgnConfigDir)
	original := projectStateForSaveTest("alpha-1", strings.Repeat("a", 64), "alpha.txt")
	next := projectStateForSaveTest("beta-1", strings.Repeat("b", 64), "beta.txt")

	if err := SaveProjectState(configDir, original); err != nil {
		t.Fatalf("SaveProjectState original failed: %v", err)
	}

	originalRename := renameStagedProjectStateFile
	t.Cleanup(func() {
		renameStagedProjectStateFile = originalRename
	})
	installCount := 0
	renameStagedProjectStateFile = func(oldpath, newpath string) error {
		installCount++
		if installCount == 2 {
			return errors.New("forced replace failure")
		}
		return originalRename(oldpath, newpath)
	}

	if err := SaveProjectState(configDir, next); err == nil {
		t.Fatal("SaveProjectState succeeded, want forced replace failure")
	}

	loaded, err := LoadProjectState(configDir)
	if err != nil {
		t.Fatalf("LoadProjectState after failed save failed: %v", err)
	}
	if got := len(loaded.Templates); got != 1 {
		t.Fatalf("template count after failed save = %d, want 1", got)
	}
	tmpl := loaded.Templates[0]
	if tmpl.ID != "alpha-1" || tmpl.Hash != strings.Repeat("a", 64) || strings.Join(tmpl.Files, ",") != "alpha.txt" {
		t.Fatalf("metadata after failed save = id:%q hash:%q files:%v, want original alpha metadata", tmpl.ID, tmpl.Hash, tmpl.Files)
	}
}

func TestLoadProjectState_RejectsInvalidV2OutputRoot(t *testing.T) {
	for _, tc := range []struct {
		name       string
		outputRoot string
	}{
		{name: "absolute", outputRoot: filepath.Join(string(filepath.Separator), "tmp", "project")},
		{name: "traversal", outputRoot: "../project"},
		{name: "not_normalized", outputRoot: "project/../other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configDir := filepath.Join(t.TempDir(), model.IgnConfigDir)
			writeV2ProjectMetadataForTest(t, configDir, tc.outputRoot)

			if _, err := LoadProjectState(configDir); err == nil {
				t.Fatalf("LoadProjectState succeeded with output_root %q", tc.outputRoot)
			}
		})
	}
}

func projectStateForSaveTest(id string, hash string, file string) *ProjectState {
	return &ProjectState{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: ".",
		Templates: []ProjectTemplateState{
			{
				ID:    id,
				Order: 1,
				Template: model.TemplateSource{
					URL: "./template",
				},
				Hash:      hash,
				Variables: map[string]interface{}{"name": id},
				Files:     []string{file},
			},
		},
	}
}

func writeV2ProjectMetadataForTest(t *testing.T, configDir string, outputRoot string) {
	t.Helper()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	ignConfig := &model.IgnConfig{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: outputRoot,
		Template:   model.TemplateSource{URL: "./template"},
		Hash:       strings.Repeat("a", 64),
		Templates: []model.TemplateCheckoutRecord{
			{
				ID:       "alpha-1",
				Order:    1,
				Template: model.TemplateSource{URL: "./template"},
				Hash:     strings.Repeat("a", 64),
			},
		},
	}
	ignVar := &model.IgnVarJson{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: outputRoot,
		Variables:  map[string]interface{}{},
		Templates: []model.IgnVarTemplateRecord{
			{ID: "alpha-1", Variables: map[string]interface{}{}},
		},
	}
	manifest := &model.IgnManifest{
		Version:    model.IgnProjectMetadataVersionV2,
		OutputRoot: outputRoot,
		Files:      []string{"alpha.txt"},
		Templates: []model.IgnManifestTemplateRecord{
			{ID: "alpha-1", Order: 1, Files: []string{"alpha.txt"}},
		},
	}

	writeJSONForProjectStateTest(t, filepath.Join(configDir, model.IgnProjectConfigFile), ignConfig)
	writeJSONForProjectStateTest(t, filepath.Join(configDir, model.IgnVarFile), ignVar)
	writeJSONForProjectStateTest(t, filepath.Join(configDir, model.IgnManifestFile), manifest)
}

func writeJSONForProjectStateTest(t *testing.T, path string, value interface{}) {
	t.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
