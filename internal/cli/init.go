package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <url-or-path>",
	Short: "Initialize configuration from template",
	Long: `Create .ign-config/ign-var.json configuration from a template.

URL Formats:
  - Full HTTPS: https://github.com/owner/repo
  - Short form: github.com/owner/repo
  - Owner/repo: owner/repo
  - With path: github.com/owner/repo/templates/go-basic
  - Git SSH: git@github.com:owner/repo.git
  - Local path: ./my-local-template or /absolute/path

Examples:
  ign init github.com/owner/repo
  ign init github.com/owner/repo/templates/go-basic
  ign init github.com/owner/repo --ref v1.2.0
  ign init ./my-local-template
  ign init github.com/owner/repo --force`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

// Init command flags
var (
	initRef   string
	initForce bool
)

func init() {
	// Flags for init
	initCmd.Flags().StringVarP(&initRef, "ref", "r", "main", "Git branch, tag, or commit SHA")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Backup existing config and reinitialize")
}

func runInit(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Check if .ign-config already exists
	if _, err := os.Stat(".ign-config"); err == nil && !initForce {
		printInfo("Configuration already exists at .ign-config")
		printInfo("(use --force to backup and reinitialize)")
		return nil
	}

	printInfo(fmt.Sprintf("Initializing configuration from: %s", url))
	if initRef != "main" {
		printInfo(fmt.Sprintf("Reference: %s", initRef))
	}

	if initForce {
		printWarning("Force mode enabled - will backup existing configuration")
	}

	// Get GitHub token from environment or config
	githubToken := getGitHubToken("")

	// Call app layer
	err := app.Init(cmd.Context(), app.InitOptions{
		URL:         url,
		Ref:         initRef,
		Force:       initForce,
		GitHubToken: githubToken,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Initialization failed: %v", err))
		return err
	}

	printSuccess("Created: .ign-config/ign-var.json")
	printInfo("")
	printInfo("Next steps:")
	printInfo("  1. Edit .ign-config/ign-var.json to set variable values")
	printInfo("  2. Run: ign checkout ./my-project")

	return nil
}
