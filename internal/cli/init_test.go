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

func TestRunInitNonInteractiveMissingVariableDoesNotCreateConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := initRef
	origForce := initForce
	origVars := initVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		initRef = origRef
		initForce = origForce
		initVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	initRef = "main"
	initForce = false
	initVars = nil
	promptInputIsTerminal = func() bool { return false }

	err := runInit(&cobra.Command{}, []string{templateDir})
	if err == nil {
		t.Fatal("runInit expected non-interactive prompt error")
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
}

func TestRunInitNonInteractiveAllVariablesProvidedSucceeds(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	templateDir := writeTemplateWithRequiredVariable(t, tempDir, "template")

	origRef := initRef
	origForce := initForce
	origVars := initVars
	origPromptInputIsTerminal := promptInputIsTerminal
	defer func() {
		initRef = origRef
		initForce = origForce
		initVars = origVars
		promptInputIsTerminal = origPromptInputIsTerminal
	}()

	initRef = "main"
	initForce = false
	initVars = []string{"project_name=my-app"}
	promptInputIsTerminal = func() bool { return false }

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runInit(cmd, []string{templateDir}); err != nil {
		t.Fatalf("runInit returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnProjectConfigFile)); err != nil {
		t.Fatalf("ign.json should be created for supplied non-interactive init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(model.IgnConfigDir, model.IgnVarFile)); err != nil {
		t.Fatalf("ign-var.json should be created for supplied non-interactive init: %v", err)
	}
	if _, statErr := os.Stat("README.md"); !os.IsNotExist(statErr) {
		t.Fatalf("init should not generate project output: %v", statErr)
	}
}
