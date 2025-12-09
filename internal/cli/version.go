package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for ign.

Examples:
  ign version
  ign version --short
  ign version --json`,
	RunE: runVersion,
}

// Version command flags
var (
	versionShort bool
	versionJSON  bool
)

func init() {
	// Flags for version
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Show version number only")
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Output as JSON")
}

// VersionInfo contains version information
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := VersionInfo{
		Version:   Version,
		GoVersion: runtime.Version(),
		Commit:    GitCommit,
		BuildDate: BuildDate,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if versionShort {
		fmt.Println(info.Version)
		return nil
	}

	if versionJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Normal output
	fmt.Printf("ign version %s\n", info.Version)
	fmt.Printf("Built with: %s\n", info.GoVersion)
	fmt.Printf("Commit: %s\n", info.Commit)
	fmt.Printf("Build date: %s\n", info.BuildDate)
	fmt.Printf("OS/Arch: %s/%s\n", info.OS, info.Arch)

	return nil
}
