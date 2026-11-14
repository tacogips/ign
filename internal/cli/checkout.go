package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout <path>",
	Short: "Generate project from configuration",
	Long: `Generate project files from template using .ign-config/ign-var.json.

This command reads the configuration created by "ign init",
fetches the template, processes template directives, and generates
the project files in the specified output directory.

Examples:
  ign checkout .
  ign checkout ./my-project
  ign checkout ./my-project --force
  ign checkout ./my-project --dry-run
  ign checkout ./my-project --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckout,
}

// Checkout command flags
var (
	checkoutForce   bool
	checkoutDryRun  bool
	checkoutVerbose bool
)

func init() {
	// Flags for checkout
	checkoutCmd.Flags().BoolVarP(&checkoutForce, "force", "f", false, "Overwrite existing files")
	checkoutCmd.Flags().BoolVarP(&checkoutDryRun, "dry-run", "d", false, "Show what would be generated without writing files")
	checkoutCmd.Flags().BoolVarP(&checkoutVerbose, "verbose", "v", false, "Show detailed processing information")
}

func runCheckout(cmd *cobra.Command, args []string) error {
	outputPath := args[0]

	if checkoutDryRun {
		printInfo("[DRY RUN] Would generate project from template")
	} else {
		printInfo("Generating project from template...")
	}

	printInfo(fmt.Sprintf("Config: .ign-config/ign-var.json"))
	printInfo(fmt.Sprintf("Output: %s", outputPath))

	if checkoutForce {
		printWarning("Force mode enabled - existing files will be replaced")
	}

	// Get GitHub token from environment
	githubToken := getGitHubToken("")

	// Call app layer
	result, err := app.Checkout(cmd.Context(), app.CheckoutOptions{
		OutputDir:   outputPath,
		Overwrite:   checkoutForce,
		DryRun:      checkoutDryRun,
		Verbose:     checkoutVerbose,
		GitHubToken: githubToken,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}

	// Print results
	if checkoutDryRun {
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

		printInfo(fmt.Sprintf("\nProject ready at: %s", outputPath))
	}

	return nil
}
