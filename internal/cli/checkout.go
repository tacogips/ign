package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout <url-or-path> [output-path]",
	Short: "Initialize and generate project from template",
	Long: `Initialize configuration and generate project files from a template.

This command combines initialization and checkout into a single step:
1. If .ign doesn't exist, creates it and prompts for variables interactively
2. Fetches the template and generates project files

If .ign already exists, the command will error unless --force is specified.
With --force, existing configuration is backed up and reinitialized.

URL Formats:
  - Full HTTPS: https://github.com/owner/repo
  - Short form: github.com/owner/repo
  - Owner/repo: owner/repo
  - With path: github.com/owner/repo/templates/go-basic
  - Git SSH: git@github.com:owner/repo.git
  - Local path: ./my-local-template or /absolute/path

Examples:
  ign checkout github.com/owner/repo
  ign checkout github.com/owner/repo ./my-project
  ign checkout github.com/owner/repo --ref v1.2.0
  ign checkout ./my-local-template ./output
  ign checkout github.com/owner/repo --force
  ign checkout github.com/owner/repo --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runCheckout,
}

// Checkout command flags
var (
	checkoutRef     string
	checkoutForce   bool
	checkoutDryRun  bool
	checkoutVerbose bool
)

func init() {
	// Flags for checkout
	checkoutCmd.Flags().StringVarP(&checkoutRef, "ref", "r", "main", "Git branch, tag, or commit SHA")
	checkoutCmd.Flags().BoolVarP(&checkoutForce, "force", "f", false, "Backup and reinitialize existing config, overwrite files")
	checkoutCmd.Flags().BoolVarP(&checkoutDryRun, "dry-run", "d", false, "Show what would be generated without writing files")
	checkoutCmd.Flags().BoolVarP(&checkoutVerbose, "verbose", "v", false, "Show detailed processing information")
}

func runCheckout(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Output path defaults to current directory
	outputPath := "."
	if len(args) > 1 {
		outputPath = args[1]
	}

	configDir := ".ign"
	configExists := false

	// Check if .ign already exists
	if _, err := os.Stat(configDir); err == nil {
		configExists = true
		if !checkoutForce {
			printInfo("Configuration already exists at .ign")
			printInfo("(use --force to backup and reinitialize)")
			return nil
		}
		printWarning("Force mode enabled - will backup existing configuration")
	}

	// Get GitHub token from environment
	githubToken := getGitHubToken("")

	// Call app layer for initialization phase
	printInfo(fmt.Sprintf("Template: %s", url))
	if checkoutRef != "main" {
		printInfo(fmt.Sprintf("Reference: %s", checkoutRef))
	}
	printInfo(fmt.Sprintf("Output: %s", outputPath))

	// Prepare template and get variable definitions
	prepResult, err := app.PrepareCheckout(cmd.Context(), app.PrepareCheckoutOptions{
		URL:          url,
		Ref:          checkoutRef,
		Force:        checkoutForce,
		ConfigExists: configExists,
		GitHubToken:  githubToken,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Preparation failed: %v", err))
		return err
	}

	// Prompt for variables interactively
	vars, err := PromptForVariables(prepResult.IgnJson)
	if err != nil {
		printErrorMsg(fmt.Sprintf("Variable collection failed: %v", err))
		return err
	}

	// Complete checkout with collected variables
	if checkoutDryRun {
		printInfo("[DRY RUN] Would generate project from template")
	} else {
		printInfo("Generating project from template...")
	}

	result, err := app.CompleteCheckout(cmd.Context(), app.CompleteCheckoutOptions{
		PrepareResult: prepResult,
		Variables:     vars,
		OutputDir:     outputPath,
		Overwrite:     checkoutForce,
		DryRun:        checkoutDryRun,
		Verbose:       checkoutVerbose,
		GitHubToken:   githubToken,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}

	// Print results
	if checkoutDryRun {
		// Output patch format to stdout
		printDryRunPatch(result)
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

		printInfo("")
		printInfo("Configuration saved to: .ign/ign-var.json")
		printInfo(fmt.Sprintf("Project ready at: %s", outputPath))
	}

	return nil
}

// printDryRunPatch outputs dry-run results in unified diff (patch) format.
func printDryRunPatch(result *app.CheckoutResult) {
	// Print summary header
	fmt.Println("# DRY RUN - No files will be written")
	fmt.Println("#")

	// Print directories that would be created
	if len(result.Directories) > 0 {
		fmt.Println("# Directories to create:")
		for _, dir := range result.Directories {
			fmt.Printf("#   mkdir -p %s\n", dir)
		}
		fmt.Println("#")
	}

	// Print summary statistics
	fmt.Printf("# Summary: %d files to create", result.FilesCreated)
	if result.FilesOverwritten > 0 {
		fmt.Printf(", %d to overwrite", result.FilesOverwritten)
	}
	if result.FilesSkipped > 0 {
		fmt.Printf(", %d to skip (already exist)", result.FilesSkipped)
	}
	fmt.Println()
	fmt.Println()

	// Print each file in patch format
	for _, f := range result.DryRunFiles {
		if f.WouldSkip {
			// Show skipped files as comments
			fmt.Printf("# SKIP: %s (file exists, use --force to overwrite)\n\n", f.Path)
			continue
		}

		// Print unified diff header
		if f.WouldOverwrite {
			fmt.Printf("# OVERWRITE: %s\n", f.Path)
		}
		fmt.Printf("--- /dev/null\n")
		fmt.Printf("+++ %s\n", f.Path)

		// Count lines for the hunk header
		lines := countLines(f.Content)
		if lines == 0 {
			fmt.Println("@@ -0,0 +0,0 @@")
		} else {
			fmt.Printf("@@ -0,0 +1,%d @@\n", lines)
		}

		// Print content with + prefix for each line
		printPatchContent(f.Content)
		fmt.Println()
	}
}

// countLines counts the number of lines in content.
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte{'\n'})
	// If content doesn't end with newline, there's one more line
	if len(content) > 0 && content[len(content)-1] != '\n' {
		count++
	}
	return count
}

// printPatchContent prints file content with + prefix for each line.
func printPatchContent(content []byte) {
	if len(content) == 0 {
		return
	}

	// Check if content is likely binary
	if isBinaryContent(content) {
		fmt.Println("+[binary file]")
		return
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		// Skip the last empty line if content ends with newline
		if i == len(lines)-1 && line == "" {
			continue
		}
		fmt.Printf("+%s\n", line)
	}
}

// isBinaryContent checks if content appears to be binary.
func isBinaryContent(content []byte) bool {
	// Check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}
