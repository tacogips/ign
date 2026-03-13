package cli

import "testing"

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
