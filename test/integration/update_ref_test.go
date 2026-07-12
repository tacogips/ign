package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestUpdateRefRetargetsTemplateAndPersistsRef(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := app.Init(context.Background(), app.InitOptions{URL: templatePath}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	seedSimpleTemplateVars(t)

	prep, err := app.PrepareUpdate(context.Background(), app.UpdateOptions{
		OutputDir: ".",
		TargetRef: "v2.0.0",
	})
	if err != nil {
		t.Fatalf("PrepareUpdate failed: %v", err)
	}
	if !prep.RefChanged {
		t.Fatal("RefChanged = false, want true")
	}

	if _, err := app.CompleteUpdate(context.Background(), app.CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     ".",
	}); err != nil {
		t.Fatalf("CompleteUpdate failed: %v", err)
	}

	loaded, err := config.LoadIgnConfig(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile))
	if err != nil {
		t.Fatalf("failed to load ign config: %v", err)
	}
	if loaded.Template.Ref != "v2.0.0" {
		t.Fatalf("stored ref = %q, want v2.0.0", loaded.Template.Ref)
	}
}

func TestUpdateRefDryRunDoesNotPersistRef(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := copyFixtureToTemp(t, "simple-template", tempDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := app.Init(context.Background(), app.InitOptions{URL: templatePath}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	seedSimpleTemplateVars(t)

	prep, err := app.PrepareUpdate(context.Background(), app.UpdateOptions{
		OutputDir: ".",
		TargetRef: "v2.0.0",
	})
	if err != nil {
		t.Fatalf("PrepareUpdate failed: %v", err)
	}

	if _, err := app.CompleteUpdate(context.Background(), app.CompleteUpdateOptions{
		PrepareResult: prep,
		OutputDir:     ".",
		DryRun:        true,
	}); err != nil {
		t.Fatalf("CompleteUpdate dry-run failed: %v", err)
	}

	loaded, err := config.LoadIgnConfig(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile))
	if err != nil {
		t.Fatalf("failed to load ign config: %v", err)
	}
	if loaded.Template.Ref != "" {
		t.Fatalf("stored ref = %q, want unchanged empty ref", loaded.Template.Ref)
	}
}

func TestUpdateRefCLIUpdatesGeneratedFilesAndPersistsRef(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "template")
	writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
		Hash:  updateRefHash1,
		Files: map[string]string{"README.md": "version one\n"},
	})
	ignBinary := buildIgnBinary(t)

	withWorkingDir(t, tempDir, func() {
		if err := app.Init(context.Background(), app.InitOptions{URL: "./template"}); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := app.Checkout(context.Background(), app.CheckoutOptions{OutputDir: "."}); err != nil {
			t.Fatalf("Checkout failed: %v", err)
		}

		writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
			Hash:  updateRefHash2,
			Files: map[string]string{"README.md": "version two\n"},
		})

		result := runIgnCLI(t, ignBinary, tempDir, "update", "--ref", "v2.0.0", "--overwrite-all", "--yes")
		if result.exitCode != 0 {
			t.Fatalf("ign update --ref exit = %d, stderr = %s", result.exitCode, result.stderr)
		}

		if got := readFileString(t, filepath.Join(tempDir, "README.md")); got != "version two\n" {
			t.Fatalf("README.md = %q, want updated template content", got)
		}
		assertStoredUpdateRef(t, "v2.0.0")
	})
}

func TestUpdateRefCLIIdenticalContentPersistsRefWithoutRewritingFiles(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "template")
	writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
		Hash:  updateRefHash1,
		Files: map[string]string{"README.md": "stable content\n"},
	})
	ignBinary := buildIgnBinary(t)

	withWorkingDir(t, tempDir, func() {
		if err := app.Init(context.Background(), app.InitOptions{URL: "./template"}); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := app.Checkout(context.Background(), app.CheckoutOptions{OutputDir: "."}); err != nil {
			t.Fatalf("Checkout failed: %v", err)
		}

		readmePath := filepath.Join(tempDir, "README.md")
		oldTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
		if err := os.Chtimes(readmePath, oldTime, oldTime); err != nil {
			t.Fatalf("failed to set README.md mtime: %v", err)
		}
		beforeInfo, err := os.Stat(readmePath)
		if err != nil {
			t.Fatalf("failed to stat README.md: %v", err)
		}

		result := runIgnCLI(t, ignBinary, tempDir, "update", "--ref", "v2.0.0")
		if result.exitCode != 0 {
			t.Fatalf("ign update --ref exit = %d, stderr = %s", result.exitCode, result.stderr)
		}

		afterInfo, err := os.Stat(readmePath)
		if err != nil {
			t.Fatalf("failed to stat README.md after update: %v", err)
		}
		if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
			t.Fatalf("README.md mtime changed: before %s after %s", beforeInfo.ModTime(), afterInfo.ModTime())
		}
		if got := readFileString(t, readmePath); got != "stable content\n" {
			t.Fatalf("README.md = %q, want unchanged content", got)
		}
		assertStoredUpdateRef(t, "v2.0.0")
	})
}

func TestUpdateRefCLIDryRunPreviewsAndDoesNotPersistRef(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "template")
	writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
		Hash:  updateRefHash1,
		Files: map[string]string{"README.md": "version one\n"},
	})
	ignBinary := buildIgnBinary(t)

	withWorkingDir(t, tempDir, func() {
		if err := app.Init(context.Background(), app.InitOptions{URL: "./template"}); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := app.Checkout(context.Background(), app.CheckoutOptions{OutputDir: "."}); err != nil {
			t.Fatalf("Checkout failed: %v", err)
		}

		writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
			Hash:  updateRefHash2,
			Files: map[string]string{"README.md": "version two\n"},
		})

		result := runIgnCLI(t, ignBinary, tempDir, "update", "--ref", "v2.0.0", "--dry-run")
		if result.exitCode != 0 {
			t.Fatalf("ign update --ref --dry-run exit = %d, stderr = %s", result.exitCode, result.stderr)
		}
		if !strings.Contains(result.stdout, "[DRY RUN]") {
			t.Fatalf("dry-run stdout = %q, want [DRY RUN] preview", result.stdout)
		}
		if got := readFileString(t, filepath.Join(tempDir, "README.md")); got != "version one\n" {
			t.Fatalf("README.md = %q, want unchanged dry-run content", got)
		}
		assertStoredUpdateRef(t, "")
	})
}

func TestUpdateRefCLIOverwriteRespectsOverwriteIgnore(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "template")
	writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
		Hash:            updateRefHash1,
		OverwriteIgnore: "protected.txt\n",
		Files: map[string]string{
			"generated.txt": "generated v1\n",
			"protected.txt": "protected v1\n",
		},
	})
	ignBinary := buildIgnBinary(t)

	withWorkingDir(t, tempDir, func() {
		if err := app.Init(context.Background(), app.InitOptions{URL: "./template"}); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := app.Checkout(context.Background(), app.CheckoutOptions{OutputDir: "."}); err != nil {
			t.Fatalf("Checkout failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "protected.txt"), []byte("user-owned\n"), 0644); err != nil {
			t.Fatalf("failed to write user-owned protected file: %v", err)
		}

		writeUpdateRefTemplate(t, templatePath, updateRefTemplateSpec{
			Hash:            updateRefHash2,
			OverwriteIgnore: "protected.txt\n",
			Files: map[string]string{
				"generated.txt": "generated v2\n",
				"protected.txt": "protected v2\n",
			},
		})

		result := runIgnCLI(t, ignBinary, tempDir, "update", "--ref", "v2.0.0", "--overwrite", "--yes")
		if result.exitCode != 0 {
			t.Fatalf("ign update --ref --overwrite --yes exit = %d, stderr = %s", result.exitCode, result.stderr)
		}
		if got := readFileString(t, filepath.Join(tempDir, "generated.txt")); got != "generated v2\n" {
			t.Fatalf("generated.txt = %q, want overwritten generated content", got)
		}
		if got := readFileString(t, filepath.Join(tempDir, "protected.txt")); got != "user-owned\n" {
			t.Fatalf("protected.txt = %q, want user-owned content preserved", got)
		}
		assertStoredUpdateRef(t, "v2.0.0")
	})
}

func seedSimpleTemplateVars(t *testing.T) {
	t.Helper()

	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	ignVar := loadIgnVar(t, ignVarPath)
	if ignVar.Variables == nil {
		ignVar.Variables = map[string]interface{}{}
	}
	ignVar.Variables["project_name"] = "demo"
	ignVar.Variables["port"] = 8080
	ignVar.Variables["enable_feature"] = false
	if err := config.SaveIgnVarJson(ignVarPath, &ignVar); err != nil {
		t.Fatalf("failed to save ign-var.json: %v", err)
	}
}

const (
	updateRefHash1 = "1111111111111111111111111111111111111111111111111111111111111111"
	updateRefHash2 = "2222222222222222222222222222222222222222222222222222222222222222"
)

type updateRefTemplateSpec struct {
	Hash            string
	OverwriteIgnore string
	Files           map[string]string
}

func writeUpdateRefTemplate(t *testing.T, templatePath string, spec updateRefTemplateSpec) {
	t.Helper()

	if err := os.RemoveAll(templatePath); err != nil {
		t.Fatalf("failed to reset template directory: %v", err)
	}
	if err := os.MkdirAll(templatePath, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}

	ignJSON := model.IgnJson{
		Name:      "update-ref-template",
		Version:   "1.0.0",
		Hash:      spec.Hash,
		Variables: map[string]model.VarDef{},
	}
	data, err := json.MarshalIndent(ignJSON, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatePath, model.IgnTemplateConfigFile), data, 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}

	if spec.OverwriteIgnore != "" {
		if err := os.WriteFile(filepath.Join(templatePath, model.IgnOverwriteIgnoreFile), []byte(spec.OverwriteIgnore), 0644); err != nil {
			t.Fatalf("failed to write overwrite ignore: %v", err)
		}
	}

	for name, content := range spec.Files {
		path := filepath.Join(templatePath, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create template file directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write template file %s: %v", name, err)
		}
	}
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()
	fn()
}

func assertStoredUpdateRef(t *testing.T, want string) {
	t.Helper()

	loaded, err := config.LoadIgnConfig(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile))
	if err != nil {
		t.Fatalf("failed to load ign config: %v", err)
	}
	if loaded.Template.Ref != want {
		t.Fatalf("stored ref = %q, want %q", loaded.Template.Ref, want)
	}
}
