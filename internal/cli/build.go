package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build configuration management",
	Long: `Manage build configurations for template-based project generation.

The build command group handles initialization of build configurations
that specify which template to use and what variables to set.`,
}

// buildInitCmd represents the build init command
var buildInitCmd = &cobra.Command{
	Use:   "init [URL]",
	Short: "Create build configuration from template",
	Long: `Create a new .ign-build/ directory with ign-var.json for a template.

URL Formats:
  - Full HTTPS: https://github.com/owner/repo
  - Short form: github.com/owner/repo
  - Owner/repo: owner/repo
  - With path: github.com/owner/repo/templates/go-basic
  - Git SSH: git@github.com:owner/repo.git

Examples:
  ign build init github.com/owner/repo
  ign build init github.com/owner/repo/templates/go-basic
  ign build init github.com/owner/repo --ref v1.2.0
  ign build init github.com/owner/repo --output ./my-config`,
	Args: cobra.ExactArgs(1),
	RunE: runBuildInit,
}

// Build init command flags
var (
	buildInitOutput string
	buildInitRef    string
	buildInitConfig string
	buildInitForce  bool
)

func init() {
	// Add init as subcommand of build
	buildCmd.AddCommand(buildInitCmd)

	// Flags for build init
	buildInitCmd.Flags().StringVarP(&buildInitOutput, "output", "o", ".ign-build", "Output directory for build config")
	buildInitCmd.Flags().StringVarP(&buildInitRef, "ref", "r", "main", "Git branch, tag, or commit SHA")
	buildInitCmd.Flags().StringVarP(&buildInitConfig, "config", "c", "", "Path to global config file")
	buildInitCmd.Flags().BoolVarP(&buildInitForce, "force", "f", false, "Overwrite existing .ign-build directory")
}

func runBuildInit(cmd *cobra.Command, args []string) error {
	url := args[0]

	printInfo(fmt.Sprintf("Initializing build configuration from: %s", url))
	if buildInitRef != "main" {
		printInfo(fmt.Sprintf("Reference: %s", buildInitRef))
	}
	printInfo(fmt.Sprintf("Output directory: %s", buildInitOutput))

	if buildInitForce {
		printWarning("Force mode enabled - will overwrite existing configuration")
	}

	// Get GitHub token from environment or config
	githubToken := getGitHubToken(buildInitConfig)

	// Call app layer
	err := app.BuildInit(cmd.Context(), app.BuildInitOptions{
		URL:         url,
		OutputDir:   buildInitOutput,
		Ref:         buildInitRef,
		Force:       buildInitForce,
		Config:      buildInitConfig,
		GitHubToken: githubToken,
		IgnVersion:  Version,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Build initialization failed: %v", err))
		return err
	}

	printSuccess(fmt.Sprintf("Created: %s/ign-var.json", buildInitOutput))
	printInfo("")
	printInfo("Next steps:")
	printInfo("  1. Edit .ign-build/ign-var.json to set variable values")
	printInfo("  2. Run: ign init --output ./my-project")

	return nil
}

// getGitHubToken retrieves GitHub token from environment or gh CLI.
// Priority: GITHUB_TOKEN env > GH_TOKEN env > gh auth token command
func getGitHubToken(configPath string) string {
	// Try environment variables first (highest priority)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}

	// Try gh CLI auth token (uses gh's secure credential storage)
	// Only attempt if gh command is available
	if _, err := exec.LookPath("gh"); err == nil {
		cmd := exec.Command("gh", "auth", "token")
		output, err := cmd.Output()
		if err == nil {
			token := strings.TrimSpace(string(output))
			if token != "" {
				return token
			}
		}
	}

	return ""
}
