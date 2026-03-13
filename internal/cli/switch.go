package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

var (
	switchRef     string
	switchForce   bool
	switchVerbose bool
)

var switchCmd = &cobra.Command{
	Use:   "switch <url-or-path> [output-path]",
	Short: "Replace the current template with a new one",
	Long: `Replace the current checked-out template with a new template.

This command is equivalent to:
1. ign rewind
2. ign checkout <new-template>

It removes files created by the current template, deletes .ign, then initializes
and generates the project from the new template.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSwitch,
}

func init() {
	switchCmd.Flags().StringVarP(&switchRef, "ref", "r", "main", "Git branch, tag, or commit SHA")
	switchCmd.Flags().BoolVarP(&switchForce, "force", "f", false, "Overwrite existing files when applying the new template")
	switchCmd.Flags().BoolVarP(&switchVerbose, "verbose", "v", false, "Show detailed processing information")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	url := args[0]
	outputPath := "."
	if len(args) > 1 {
		outputPath = args[1]
	}

	githubToken := getGitHubToken("")

	printInfo("Removing current template output...")
	if _, err := app.Rewind(cmd.Context(), app.RewindOptions{
		OutputDir:   outputPath,
		GitHubToken: githubToken,
	}); err != nil {
		printErrorMsg(fmt.Sprintf("Switch failed during rewind: %v", err))
		return err
	}

	printInfo(fmt.Sprintf("Template: %s", url))
	if switchRef != "main" {
		printInfo(fmt.Sprintf("Reference: %s", switchRef))
	}
	printInfo(fmt.Sprintf("Output: %s", outputPath))

	prepResult, err := app.PrepareCheckout(cmd.Context(), app.PrepareCheckoutOptions{
		URL:          url,
		Ref:          switchRef,
		Force:        false,
		ConfigExists: false,
		GitHubToken:  githubToken,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Switch preparation failed: %v", err))
		return err
	}

	vars, err := PromptForVariables(prepResult.IgnJson)
	if err != nil {
		printErrorMsg(fmt.Sprintf("Variable collection failed: %v", err))
		return err
	}

	result, err := app.CompleteCheckout(cmd.Context(), app.CompleteCheckoutOptions{
		PrepareResult: prepResult,
		Variables:     vars,
		OutputDir:     outputPath,
		Overwrite:     switchForce,
		Verbose:       switchVerbose,
		GitHubToken:   githubToken,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Switch failed: %v", err))
		return err
	}

	printSuccess("Template switched successfully")
	printInfo("")
	printInfo("Summary:")
	printInfo(fmt.Sprintf("  Created: %d files", result.FilesCreated))
	if result.FilesSkipped > 0 {
		printInfo(fmt.Sprintf("  Skipped: %d files (already exist)", result.FilesSkipped))
	}
	if result.FilesOverwritten > 0 {
		printInfo(fmt.Sprintf("  Overwritten: %d files", result.FilesOverwritten))
	}
	if len(result.Errors) > 0 {
		printWarning(fmt.Sprintf("%d errors occurred during generation:", len(result.Errors)))
		for _, e := range result.Errors {
			printWarning(fmt.Sprintf("  - %v", e))
		}
	}

	return nil
}
