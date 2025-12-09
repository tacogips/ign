package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate project from build configuration",
	Long: `Generate project files from template using .ign-build/ign-var.json.

This command reads the build configuration created by "ign build init",
fetches the template, processes template directives, and generates
the project files in the specified output directory.

Examples:
  ign init
  ign init --output ./my-project
  ign init --output ./my-project --overwrite
  ign init --config ./custom-build/ign-var.json --output ./output
  ign init --dry-run`,
	RunE: runInit,
}

// Init command flags
var (
	initOutput    string
	initOverwrite bool
	initConfig    string
	initDryRun    bool
	initVerbose   bool
)

func init() {
	// Flags for init
	initCmd.Flags().StringVarP(&initOutput, "output", "o", ".", "Output directory for generated project")
	initCmd.Flags().BoolVarP(&initOverwrite, "overwrite", "w", false, "Overwrite existing files")
	initCmd.Flags().StringVarP(&initConfig, "config", "c", ".ign-build/ign-var.json", "Path to variable config file")
	initCmd.Flags().BoolVarP(&initDryRun, "dry-run", "d", false, "Show what would be generated without writing files")
	initCmd.Flags().BoolVarP(&initVerbose, "verbose", "v", false, "Show detailed processing information")
}

func runInit(cmd *cobra.Command, args []string) error {
	if initDryRun {
		printInfo("[DRY RUN] Would generate project from template")
	} else {
		printInfo("Generating project from template...")
	}

	printInfo(fmt.Sprintf("Config: %s", initConfig))
	printInfo(fmt.Sprintf("Output: %s", initOutput))

	if initOverwrite {
		printWarning("Overwrite mode enabled - existing files will be replaced")
	}

	// Get GitHub token from environment
	githubToken := getGitHubToken("")

	// Call app layer
	result, err := app.Init(cmd.Context(), app.InitOptions{
		OutputDir:   initOutput,
		ConfigPath:  initConfig,
		Overwrite:   initOverwrite,
		DryRun:      initDryRun,
		Verbose:     initVerbose,
		GitHubToken: githubToken,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Initialization failed: %v", err))
		return err
	}

	// Print results
	if initDryRun {
		printInfo("")
		printInfo("[DRY RUN] Files to create:")
		for _, file := range result.Files {
			printInfo(fmt.Sprintf("  - %s", file))
		}
		printInfo("")
		printInfo("No files written (dry run).")
	} else {
		printSuccess("Project generated successfully")
		printInfo("")
		printInfo("Summary:")
		printInfo(fmt.Sprintf("  Created: %d files", result.FilesCreated))
		if result.FilesSkipped > 0 {
			printInfo(fmt.Sprintf("  Skipped: %d files (already exist)", result.FilesSkipped))
		}
		if result.FilesOverwritten > 0 {
			printInfo(fmt.Sprintf("  Overwritten: %d files", result.FilesOverwritten))
		}

		// Print any non-fatal errors
		if len(result.Errors) > 0 {
			printWarning(fmt.Sprintf("%d errors occurred during generation:", len(result.Errors)))
			for _, e := range result.Errors {
				printWarning(fmt.Sprintf("  - %v", e))
			}
		}

		printInfo(fmt.Sprintf("\nProject ready at: %s", initOutput))
	}

	return nil
}
