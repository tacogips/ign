package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/template/model"
)

func writeTemplateWithRequiredVariable(t *testing.T, dir string, name string) string {
	t.Helper()

	templateDir := filepath.Join(dir, name)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}

	templateConfig := `{
  "name": "` + name + `",
  "version": "1.0.0",
  "variables": {
    "project_name": {
      "type": "string",
      "required": true
    }
  },
  "hash": "` + strings.Repeat("a", 64) + `"
}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(templateConfig), 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("@ign-var:project_name@"), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	return templateDir
}

func TestRunCheckoutNonInteractiveMissingVariableDoesNotCreateConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := checkoutRef
	origForce := checkoutForce
	origDryRun := checkoutDryRun
	origVerbose := checkoutVerbose
	origVars := checkoutVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		checkoutRef = origRef
		checkoutForce = origForce
		checkoutDryRun = origDryRun
		checkoutVerbose = origVerbose
		checkoutVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	checkoutRef = "main"
	checkoutForce = false
	checkoutDryRun = false
	checkoutVerbose = false
	checkoutVars = nil
	promptInputIsTerminal = func() bool { return false }

	err := runCheckout(&cobra.Command{}, []string{templateDir, "."})
	if err == nil {
		t.Fatal("runCheckout expected non-interactive prompt error")
	}
	errText := err.Error()
	for _, want := range []string{"require a TTY", "--var key=value", "-V key=value", "project_name"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want %q", errText, want)
		}
	}
	if _, statErr := os.Stat(model.IgnConfigDir); !os.IsNotExist(statErr) {
		t.Fatalf(".ign should not be created after prompt preflight failure: %v", statErr)
	}
	if _, statErr := os.Stat("README.md"); !os.IsNotExist(statErr) {
		t.Fatalf("output should not be generated after prompt preflight failure: %v", statErr)
	}
}

func TestRunCheckoutNonInteractiveAllVariablesProvidedSucceeds(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := checkoutRef
	origForce := checkoutForce
	origDryRun := checkoutDryRun
	origVerbose := checkoutVerbose
	origVars := checkoutVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		checkoutRef = origRef
		checkoutForce = origForce
		checkoutDryRun = origDryRun
		checkoutVerbose = origVerbose
		checkoutVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	checkoutRef = "main"
	checkoutForce = false
	checkoutDryRun = false
	checkoutVerbose = false
	checkoutVars = []string{"project_name=my-app"}
	promptInputIsTerminal = func() bool { return false }

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runCheckout(cmd, []string{templateDir, "."}); err != nil {
		t.Fatalf("runCheckout returned error: %v", err)
	}

	generated, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read generated output: %v", err)
	}
	if string(generated) != "my-app" {
		t.Fatalf("README.md = %q, want supplied variable value", generated)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnVarFile)); err != nil {
		t.Fatalf("ign-var.json should be created for supplied non-interactive checkout: %v", err)
	}
}

func TestRunCheckoutInvalidFileVariableDoesNotBackupExistingConfig(t *testing.T) {
	tempDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	originalIgnConfig := []byte("existing ign config")
	originalIgnVars := []byte("existing ign vars")
	if err := os.WriteFile(ignConfigPath, originalIgnConfig, 0644); err != nil {
		t.Fatalf("failed to write existing ign config: %v", err)
	}
	if err := os.WriteFile(ignVarPath, originalIgnVars, 0644); err != nil {
		t.Fatalf("failed to write existing ign vars: %v", err)
	}

	templateDir := filepath.Join(tempDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}
	templateConfig := `{
  "name": "file-var-template",
  "version": "1.0.0",
  "variables": {
    "license_text": {
      "type": "string",
      "required": true
    }
  },
  "hash": "` + strings.Repeat("a", 64) + `"
}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(templateConfig), 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("@ign-var:license_text@"), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	origRef := checkoutRef
	origForce := checkoutForce
	origDryRun := checkoutDryRun
	origVerbose := checkoutVerbose
	origVars := checkoutVars
	defer func() {
		checkoutRef = origRef
		checkoutForce = origForce
		checkoutDryRun = origDryRun
		checkoutVerbose = origVerbose
		checkoutVars = origVars
	}()

	checkoutRef = "main"
	checkoutForce = true
	checkoutDryRun = false
	checkoutVerbose = false
	checkoutVars = []string{"license_text=@file:missing.txt"}

	err = runCheckout(&cobra.Command{}, []string{templateDir, "."})
	if err == nil {
		t.Fatalf("runCheckout expected missing @file variable error")
	}

	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile+".bk1")); !os.IsNotExist(err) {
		t.Fatalf("ign.json backup was created before variable validation completed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnVarFile+".bk1")); !os.IsNotExist(err) {
		t.Fatalf("ign-var.json backup was created before variable validation completed: %v", err)
	}

	gotIgnConfig, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read existing ign config: %v", err)
	}
	if string(gotIgnConfig) != string(originalIgnConfig) {
		t.Fatalf("ign config changed, got %q want %q", gotIgnConfig, originalIgnConfig)
	}

	gotIgnVars, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read existing ign vars: %v", err)
	}
	if string(gotIgnVars) != string(originalIgnVars) {
		t.Fatalf("ign vars changed, got %q want %q", gotIgnVars, originalIgnVars)
	}
}

func TestRunCheckoutExistingConfigWithoutForceReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign: %v", err)
	}

	origForce := checkoutForce
	origVars := checkoutVars
	defer func() {
		checkoutForce = origForce
		checkoutVars = origVars
	}()
	checkoutForce = false
	checkoutVars = nil

	err := runCheckout(&cobra.Command{}, []string{"github.com/owner/repo"})
	if err == nil {
		t.Fatal("runCheckout expected existing config error")
	}
	if !strings.Contains(err.Error(), "configuration already exists") {
		t.Fatalf("error = %v, want existing configuration message", err)
	}
}

func TestRunCheckoutInvalidTemplateHashDoesNotBackupExistingConfig(t *testing.T) {
	tempDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	originalIgnConfig := []byte("existing ign config")
	originalIgnVars := []byte("existing ign vars")
	if err := os.WriteFile(ignConfigPath, originalIgnConfig, 0644); err != nil {
		t.Fatalf("failed to write existing ign config: %v", err)
	}
	if err := os.WriteFile(ignVarPath, originalIgnVars, 0644); err != nil {
		t.Fatalf("failed to write existing ign vars: %v", err)
	}

	origRef := checkoutRef
	origForce := checkoutForce
	origDryRun := checkoutDryRun
	origVerbose := checkoutVerbose
	origVars := checkoutVars
	defer func() {
		checkoutRef = origRef
		checkoutForce = origForce
		checkoutDryRun = origDryRun
		checkoutVerbose = origVerbose
		checkoutVars = origVars
	}()

	checkoutRef = "main"
	checkoutForce = true
	checkoutDryRun = false
	checkoutVerbose = false
	checkoutVars = nil

	err = runCheckout(&cobra.Command{}, []string{writeTemplateWithoutHash(t, tempDir), "."})
	if err == nil {
		t.Fatalf("runCheckout expected missing template hash error")
	}

	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile+".bk1")); !os.IsNotExist(err) {
		t.Fatalf("ign.json backup was created before template hash validation completed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnVarFile+".bk1")); !os.IsNotExist(err) {
		t.Fatalf("ign-var.json backup was created before template hash validation completed: %v", err)
	}

	gotIgnConfig, err := os.ReadFile(ignConfigPath)
	if err != nil {
		t.Fatalf("failed to read existing ign config: %v", err)
	}
	if string(gotIgnConfig) != string(originalIgnConfig) {
		t.Fatalf("ign config changed, got %q want %q", gotIgnConfig, originalIgnConfig)
	}

	gotIgnVars, err := os.ReadFile(ignVarPath)
	if err != nil {
		t.Fatalf("failed to read existing ign vars: %v", err)
	}
	if string(gotIgnVars) != string(originalIgnVars) {
		t.Fatalf("ign vars changed, got %q want %q", gotIgnVars, originalIgnVars)
	}
}

func TestRunCheckoutForceReusesPreparedFileVariableAfterBackup(t *testing.T) {
	tempDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	ignConfigPath := filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(model.IgnConfigDir, model.IgnVarFile)
	if err := os.WriteFile(ignConfigPath, []byte("existing ign config"), 0644); err != nil {
		t.Fatalf("failed to write existing ign config: %v", err)
	}
	existingFileVariableContent := "content read before backup"
	if err := os.WriteFile(ignVarPath, []byte(existingFileVariableContent), 0644); err != nil {
		t.Fatalf("failed to write existing ign vars: %v", err)
	}

	templateDir := filepath.Join(tempDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}
	templateConfig := `{
  "name": "file-var-template",
  "version": "1.0.0",
  "variables": {
    "license_text": {
      "type": "string",
      "required": true
    }
  },
  "hash": "` + strings.Repeat("a", 64) + `"
}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(templateConfig), 0644); err != nil {
		t.Fatalf("failed to write template config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("@ign-var:license_text@"), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	origRef := checkoutRef
	origForce := checkoutForce
	origDryRun := checkoutDryRun
	origVerbose := checkoutVerbose
	origVars := checkoutVars
	defer func() {
		checkoutRef = origRef
		checkoutForce = origForce
		checkoutDryRun = origDryRun
		checkoutVerbose = origVerbose
		checkoutVars = origVars
	}()

	checkoutRef = "main"
	checkoutForce = true
	checkoutDryRun = false
	checkoutVerbose = false
	checkoutVars = []string{"license_text=@file:" + model.IgnVarFile}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := runCheckout(cmd, []string{templateDir, "."}); err != nil {
		t.Fatalf("runCheckout failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnVarFile+".bk1")); err != nil {
		t.Fatalf("ign-var.json backup was not created: %v", err)
	}

	generated, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read generated README: %v", err)
	}
	if string(generated) != existingFileVariableContent {
		t.Fatalf("generated README = %q, want %q", generated, existingFileVariableContent)
	}
}
