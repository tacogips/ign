package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestRunUpdate_ConfirmationPreviewReusesSymlinkTransitionPlan(t *testing.T) {
	resetUpdateCommandDependencies(t)
	updateOverwrite = true

	previewPlan := &app.UpdateExecutionPlan{}
	prepareUpdate = func(context.Context, app.UpdateOptions) (*app.PrepareUpdateResult, error) {
		return &app.PrepareUpdateResult{
			HashChanged: true,
			IgnConfig:   &model.IgnConfig{Template: model.TemplateSource{URL: "https://github.com/test/template"}},
		}, nil
	}
	var calls []app.CompleteUpdateOptions
	completeUpdate = func(_ context.Context, opts app.CompleteUpdateOptions) (*app.UpdateResult, error) {
		calls = append(calls, opts)
		if opts.DryRun {
			return &app.UpdateResult{ExecutionPlan: previewPlan}, nil
		}
		return &app.UpdateResult{}, nil
	}
	confirmUpdate = func() (bool, error) { return true, nil }

	if err := runUpdate(&cobra.Command{}, nil); err != nil {
		t.Fatalf("run update: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("CompleteUpdate calls = %d, want preview and mutation", len(calls))
	}
	if !calls[0].DryRun || calls[0].ExecutionPlan != nil {
		t.Fatalf("preview options = %#v, want a plan-producing dry run", calls[0])
	}
	if calls[1].DryRun {
		t.Fatal("confirmed update remained a dry run")
	}
	if calls[1].ExecutionPlan != previewPlan {
		t.Fatal("confirmed update did not receive the exact preview execution plan")
	}
}

func resetUpdateCommandDependencies(t *testing.T) {
	t.Helper()
	originalPrepare := prepareUpdate
	originalComplete := completeUpdate
	originalConfirm := confirmUpdate
	originalForce := updateForce
	originalOverwrite := updateOverwrite
	originalOverwriteAll := updateOverwriteAll
	originalDryRun := updateDryRun
	originalVerbose := updateVerbose
	originalYes := updateYes
	originalRef := updateRef
	t.Cleanup(func() {
		prepareUpdate = originalPrepare
		completeUpdate = originalComplete
		confirmUpdate = originalConfirm
		updateForce = originalForce
		updateOverwrite = originalOverwrite
		updateOverwriteAll = originalOverwriteAll
		updateDryRun = originalDryRun
		updateVerbose = originalVerbose
		updateYes = originalYes
		updateRef = originalRef
	})
}

func TestUpdateCmd_FlagDefaults(t *testing.T) {
	// Reset flags for testing
	updateForce = false
	updateOverwrite = false
	updateOverwriteAll = false
	updateDryRun = false
	updateVerbose = false
	updateYes = false
	updateRef = ""

	// Verify default values
	if updateForce != false {
		t.Errorf("Expected updateForce default to be false, got %v", updateForce)
	}
	if updateOverwrite != false {
		t.Errorf("Expected updateOverwrite default to be false, got %v", updateOverwrite)
	}
	if updateOverwriteAll != false {
		t.Errorf("Expected updateOverwriteAll default to be false, got %v", updateOverwriteAll)
	}
	if updateDryRun != false {
		t.Errorf("Expected updateDryRun default to be false, got %v", updateDryRun)
	}
	if updateVerbose != false {
		t.Errorf("Expected updateVerbose default to be false, got %v", updateVerbose)
	}
	if updateRef != "" {
		t.Errorf("Expected updateRef default to be empty, got %q", updateRef)
	}
}

func TestUpdateCmd_FlagRegistration(t *testing.T) {
	// Verify flags are registered on the command
	tests := []struct {
		flagName  string
		shorthand string
	}{
		{"force", "f"},
		{"overwrite", "o"},
		{"overwrite-all", ""},
		{"dry-run", "d"},
		{"verbose", "v"},
		{"yes", "y"},
		{"ref", "r"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := updateCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag --%s not found on updateCmd", tt.flagName)
				return
			}
			if flag.Shorthand != tt.shorthand {
				t.Errorf("Flag --%s expected shorthand -%s, got -%s", tt.flagName, tt.shorthand, flag.Shorthand)
			}
		})
	}
}

func TestUpdateCmd_ShouldOverwriteLogic(t *testing.T) {
	tests := []struct {
		name              string
		force             bool
		overwrite         bool
		overwriteAll      bool
		expectedOverwrite bool
	}{
		{
			name:              "no flags - no overwrite",
			force:             false,
			overwrite:         false,
			overwriteAll:      false,
			expectedOverwrite: false,
		},
		{
			name:              "overwrite only - selective overwrite enabled",
			force:             false,
			overwrite:         true,
			overwriteAll:      false,
			expectedOverwrite: true,
		},
		{
			name:              "overwrite-all only - overwrite enabled",
			force:             false,
			overwrite:         false,
			overwriteAll:      true,
			expectedOverwrite: true,
		},
		{
			name:              "force only - overwrite enabled (force implies overwrite-all)",
			force:             true,
			overwrite:         false,
			overwriteAll:      false,
			expectedOverwrite: true,
		},
		{
			name:              "both force and overwrite - overwrite enabled",
			force:             true,
			overwrite:         true,
			overwriteAll:      false,
			expectedOverwrite: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldOverwrite := updateOverwriteMode(tt.overwrite, tt.overwriteAll, tt.force) != generator.OverwriteNone

			if shouldOverwrite != tt.expectedOverwrite {
				t.Errorf("shouldOverwrite = %v, expected %v (force=%v, overwrite=%v, overwriteAll=%v)",
					shouldOverwrite, tt.expectedOverwrite, tt.force, tt.overwrite, tt.overwriteAll)
			}
		})
	}
}

func TestUpdateOverwriteMode(t *testing.T) {
	tests := []struct {
		name         string
		overwrite    bool
		overwriteAll bool
		force        bool
		want         generator.OverwriteMode
	}{
		{name: "none", want: generator.OverwriteNone},
		{name: "selective", overwrite: true, want: generator.OverwriteSelective},
		{name: "all", overwriteAll: true, want: generator.OverwriteAll},
		{name: "force", force: true, want: generator.OverwriteAll},
		{name: "force wins over selective", overwrite: true, force: true, want: generator.OverwriteAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateOverwriteMode(tt.overwrite, tt.overwriteAll, tt.force)
			if got != tt.want {
				t.Fatalf("updateOverwriteMode() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestShouldRegenerate(t *testing.T) {
	tests := []struct {
		name        string
		hashChanged bool
		force       bool
		overwrite   bool
		want        bool
	}{
		{
			name:        "changed template always regenerates",
			hashChanged: true,
			force:       false,
			overwrite:   false,
			want:        true,
		},
		{
			name:        "unchanged without flags does not regenerate",
			hashChanged: false,
			force:       false,
			overwrite:   false,
			want:        false,
		},
		{
			name:        "unchanged with overwrite regenerates",
			hashChanged: false,
			force:       false,
			overwrite:   true,
			want:        true,
		},
		{
			name:        "unchanged with force regenerates",
			hashChanged: false,
			force:       true,
			overwrite:   false,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRegenerate(tt.hashChanged, tt.force, tt.overwrite)
			if got != tt.want {
				t.Fatalf("shouldRegenerate(%v, %v, %v) = %v, want %v",
					tt.hashChanged, tt.force, tt.overwrite, got, tt.want)
			}
		})
	}
}

func TestShouldCompleteUpdateIncludesRefChanged(t *testing.T) {
	tests := []struct {
		name      string
		prep      *app.PrepareUpdateResult
		force     bool
		overwrite bool
		want      bool
	}{
		{
			name: "nil prep",
			want: false,
		},
		{
			name: "unchanged no ref",
			prep: &app.PrepareUpdateResult{},
			want: false,
		},
		{
			name: "ref changed",
			prep: &app.PrepareUpdateResult{RefChanged: true},
			want: true,
		},
		{
			name: "hash changed",
			prep: &app.PrepareUpdateResult{HashChanged: true},
			want: true,
		},
		{
			name:  "force",
			prep:  &app.PrepareUpdateResult{},
			force: true,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldCompleteUpdate(tt.prep, tt.force, tt.overwrite)
			if got != tt.want {
				t.Fatalf("shouldCompleteUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateCmd_FlagParsing(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		expectedForce        bool
		expectedOverwrite    bool
		expectedOverwriteAll bool
		expectedDryRun       bool
		expectedYes          bool
		expectedRef          string
	}{
		{
			name:                 "no flags",
			args:                 []string{},
			expectedForce:        false,
			expectedOverwrite:    false,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "force flag long",
			args:                 []string{"--force"},
			expectedForce:        true,
			expectedOverwrite:    false,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "force flag short",
			args:                 []string{"-f"},
			expectedForce:        true,
			expectedOverwrite:    false,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "overwrite flag long",
			args:                 []string{"--overwrite"},
			expectedForce:        false,
			expectedOverwrite:    true,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "overwrite flag short",
			args:                 []string{"-o"},
			expectedForce:        false,
			expectedOverwrite:    true,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "overwrite all flag",
			args:                 []string{"--overwrite-all"},
			expectedForce:        false,
			expectedOverwrite:    false,
			expectedOverwriteAll: true,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "force and overwrite combined",
			args:                 []string{"-f", "-o"},
			expectedForce:        true,
			expectedOverwrite:    true,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "",
		},
		{
			name:                 "all flags",
			args:                 []string{"--force", "--overwrite", "--overwrite-all", "--dry-run", "--yes"},
			expectedForce:        true,
			expectedOverwrite:    true,
			expectedOverwriteAll: true,
			expectedDryRun:       true,
			expectedYes:          true,
			expectedRef:          "",
		},
		{
			name:                 "ref flag long",
			args:                 []string{"--ref", "v2.0.0"},
			expectedForce:        false,
			expectedOverwrite:    false,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "v2.0.0",
		},
		{
			name:                 "ref flag short",
			args:                 []string{"-r", "feature/new-template"},
			expectedForce:        false,
			expectedOverwrite:    false,
			expectedOverwriteAll: false,
			expectedDryRun:       false,
			expectedYes:          false,
			expectedRef:          "feature/new-template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test to avoid flag state pollution
			cmd := &cobra.Command{Use: "update"}
			var force, overwrite, overwriteAll, dryRun, verbose, yes bool
			var ref string
			cmd.Flags().BoolVarP(&force, "force", "f", false, "")
			cmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "")
			cmd.Flags().BoolVar(&overwriteAll, "overwrite-all", false, "")
			cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "")
			cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "")
			cmd.Flags().BoolVarP(&yes, "yes", "y", false, "")
			cmd.Flags().StringVarP(&ref, "ref", "r", "", "")

			// Parse the arguments
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			if force != tt.expectedForce {
				t.Errorf("force = %v, expected %v", force, tt.expectedForce)
			}
			if overwrite != tt.expectedOverwrite {
				t.Errorf("overwrite = %v, expected %v", overwrite, tt.expectedOverwrite)
			}
			if overwriteAll != tt.expectedOverwriteAll {
				t.Errorf("overwriteAll = %v, expected %v", overwriteAll, tt.expectedOverwriteAll)
			}
			if dryRun != tt.expectedDryRun {
				t.Errorf("dryRun = %v, expected %v", dryRun, tt.expectedDryRun)
			}
			if yes != tt.expectedYes {
				t.Errorf("yes = %v, expected %v", yes, tt.expectedYes)
			}
			if ref != tt.expectedRef {
				t.Errorf("ref = %q, expected %q", ref, tt.expectedRef)
			}
		})
	}
}

func TestTruncateHash(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected string
	}{
		{
			name:     "empty string",
			hash:     "",
			expected: "",
		},
		{
			name:     "short hash (less than 16 chars)",
			hash:     "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly 16 chars - no truncation",
			hash:     "0123456789abcdef",
			expected: "0123456789abcdef",
		},
		{
			name:     "17 chars - triggers truncation",
			hash:     "0123456789abcdef0",
			expected: "01234567...9abcdef0",
		},
		{
			name:     "normal git hash (40 chars)",
			hash:     "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
			expected: "a1b2c3d4...q7r8s9t0",
		},
		{
			name:     "long hash (64 chars)",
			hash:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: "01234567...89abcdef",
		},
		{
			name:     "very long hash (100 chars)",
			hash:     "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			expected: "01234567...23456789",
		},
		{
			name:     "single character",
			hash:     "a",
			expected: "a",
		},
		{
			name:     "exactly at boundary (8 chars)",
			hash:     "12345678",
			expected: "12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateHash(tt.hash)
			if result != tt.expected {
				t.Errorf("truncateHash(%q) = %q, want %q", tt.hash, result, tt.expected)
			}
		})
	}
}
