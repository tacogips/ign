package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/model"
)

func TestSwitchCmd_FlagRegistration(t *testing.T) {
	tests := []struct {
		flagName  string
		shorthand string
	}{
		{"ref", "r"},
		{"force", "f"},
		{"verbose", "v"},
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
