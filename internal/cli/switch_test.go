package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func writeTemplateWithoutHash(t *testing.T, dir string) string {
	t.Helper()

	templateDir := filepath.Join(dir, "template-without-hash")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template directory: %v", err)
	}

	ignTemplate := `{
  "name": "broken-template",
  "version": "1.0.0",
  "description": "template missing hash",
  "variables": {}
}`
	if err := os.WriteFile(filepath.Join(templateDir, model.IgnTemplateConfigFile), []byte(ignTemplate), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", model.IgnTemplateConfigFile, err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("new template"), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	return templateDir
}

func TestSwitchCmd_FlagRegistration(t *testing.T) {
	tests := []struct {
		flagName  string
		shorthand string
	}{
		{"ref", "r"},
		{"force", "f"},
		{"verbose", "v"},
		{FlagVar, "V"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := switchCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag --%s not found on switchCmd", tt.flagName)
			}
			if flag.Shorthand != tt.shorthand {
				t.Fatalf("flag --%s expected shorthand -%s, got -%s", tt.flagName, tt.shorthand, flag.Shorthand)
			}
		})
	}
}

func TestRootCmd_IncludesRewindAndSwitch(t *testing.T) {
	if rootCmd.Commands() == nil {
		t.Fatal("root command should have subcommands")
	}

	if _, _, err := rootCmd.Find([]string{"rewind"}); err != nil {
		t.Fatalf("root command should include rewind: %v", err)
	}
	if _, _, err := rootCmd.Find([]string{"switch"}); err != nil {
		t.Fatalf("root command should include switch: %v", err)
	}
}

func TestRunSwitch_PreservesCurrentProjectWhenPreparationFails(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	switchRef = "main"
	switchForce = false
	switchVerbose = false
	switchCmd.SetContext(context.Background())

	generatedFile := filepath.Join(tempDir, "generated.txt")
	if err := os.WriteFile(generatedFile, []byte("existing project"), 0644); err != nil {
		t.Fatalf("failed to create generated file: %v", err)
	}

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}
	if err := config.SaveIgnManifest(
		filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile),
		&model.IgnManifest{Files: []string{generatedFile}},
	); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	err := runSwitch(switchCmd, []string{filepath.Join(tempDir, "missing-template")})
	if err == nil {
		t.Fatal("runSwitch should fail for an invalid template path")
	}

	if _, statErr := os.Stat(generatedFile); statErr != nil {
		t.Fatalf("existing generated file should be preserved when switch preparation fails: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, model.IgnConfigDir)); statErr != nil {
		t.Fatalf(".ign should be preserved when switch preparation fails: %v", statErr)
	}
}

func TestRunSwitch_PreservesCurrentProjectWhenValidationFails(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	switchRef = "main"
	switchForce = false
	switchVerbose = false
	switchCmd.SetContext(context.Background())

	generatedFile := filepath.Join(tempDir, "generated.txt")
	if err := os.WriteFile(generatedFile, []byte("existing project"), 0644); err != nil {
		t.Fatalf("failed to create generated file: %v", err)
	}

	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}
	if err := config.SaveIgnManifest(
		filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile),
		&model.IgnManifest{Files: []string{generatedFile}},
	); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	templateDir := writeTemplateWithoutHash(t, tempDir)

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runSwitch(cmd, []string{templateDir})
	if err == nil {
		t.Fatal("runSwitch should fail when the new template fails preflight validation")
	}

	if _, statErr := os.Stat(generatedFile); statErr != nil {
		t.Fatalf("existing generated file should be preserved when switch validation fails: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, model.IgnConfigDir)); statErr != nil {
		t.Fatalf(".ign should be preserved when switch validation fails: %v", statErr)
	}
}

func TestRunSwitchNonInteractiveMissingVariablePreservesCurrentProject(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	generatedFile := filepath.Join(tempDir, "generated.txt")
	if err := os.WriteFile(generatedFile, []byte("existing project"), 0644); err != nil {
		t.Fatalf("failed to create generated file: %v", err)
	}
	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}
	if err := config.SaveIgnManifest(
		filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile),
		&model.IgnManifest{Files: []string{generatedFile}},
	); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := switchRef
	origForce := switchForce
	origVerbose := switchVerbose
	origVars := switchVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		switchRef = origRef
		switchForce = origForce
		switchVerbose = origVerbose
		switchVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	switchRef = "main"
	switchForce = false
	switchVerbose = false
	switchVars = nil
	promptInputIsTerminal = func() bool { return false }

	err := runSwitch(switchCmd, []string{templateDir})
	if err == nil {
		t.Fatal("runSwitch expected non-interactive prompt error")
	}
	errText := err.Error()
	for _, want := range []string{"require a TTY", "--var key=value", "-V key=value", "project_name"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want %q", errText, want)
		}
	}
	if got, readErr := os.ReadFile(generatedFile); readErr != nil || string(got) != "existing project" {
		t.Fatalf("existing generated file should be preserved, content=%q err=%v", got, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, model.IgnConfigDir)); statErr != nil {
		t.Fatalf(".ign should be preserved when prompt preflight fails: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, "README.md")); !os.IsNotExist(statErr) {
		t.Fatalf("new template output should not be generated after prompt preflight failure: %v", statErr)
	}
}

func TestRunSwitchNonInteractiveAllVariablesProvidedSucceeds(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	generatedFile := filepath.Join(tempDir, "generated.txt")
	if err := os.WriteFile(generatedFile, []byte("existing project"), 0644); err != nil {
		t.Fatalf("failed to create generated file: %v", err)
	}
	if err := os.MkdirAll(model.IgnConfigDir, 0755); err != nil {
		t.Fatalf("failed to create .ign directory: %v", err)
	}
	if err := config.SaveIgnManifest(
		filepath.Join(tempDir, model.IgnConfigDir, model.IgnManifestFile),
		&model.IgnManifest{Files: []string{generatedFile}},
	); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := switchRef
	origForce := switchForce
	origVerbose := switchVerbose
	origVars := switchVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		switchRef = origRef
		switchForce = origForce
		switchVerbose = origVerbose
		switchVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	switchRef = "main"
	switchForce = false
	switchVerbose = false
	switchVars = []string{"project_name=my-app"}
	promptInputIsTerminal = func() bool { return false }

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runSwitch(cmd, []string{templateDir}); err != nil {
		t.Fatalf("runSwitch returned error: %v", err)
	}

	if _, statErr := os.Stat(generatedFile); !os.IsNotExist(statErr) {
		t.Fatalf("old generated file should be removed during successful switch: %v", statErr)
	}
	generated, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read generated output: %v", err)
	}
	if string(generated) != "my-app" {
		t.Fatalf("README.md = %q, want supplied variable value", generated)
	}
}
