package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

// templateCmd represents the template command group
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Template management commands",
	Long: `Manage templates, including validation and checking.

The template command group provides utilities for working with templates,
such as checking template syntax and validating directives.`,
}

// templateCheckCmd represents the template check command
var templateCheckCmd = &cobra.Command{
	Use:   "check [PATH]",
	Short: "Validate template files for syntax errors",
	Long: `Validate template files for correct format and syntax errors.

The check command scans template files for @ign- directives and validates
their syntax without processing them. It reports any errors found with
file paths and line numbers.

If PATH is not specified, the current directory is checked.

Examples:
  ign template check
  ign template check ./templates
  ign template check template.txt
  ign template check -r
  ign template check -r -v`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemplateCheck,
}

// Template check command flags
var (
	templateCheckRecursive bool
	templateCheckVerbose   bool
)

func init() {
	// Add check as subcommand of template
	templateCmd.AddCommand(templateCheckCmd)

	// Flags for template check
	templateCheckCmd.Flags().BoolVarP(&templateCheckRecursive, "recursive", "r", false, "Recursively check subdirectories")
	templateCheckCmd.Flags().BoolVarP(&templateCheckVerbose, "verbose", "v", false, "Show detailed validation info")
}

func runTemplateCheck(cmd *cobra.Command, args []string) error {
	// Default to current directory if no path specified
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	printInfo(fmt.Sprintf("Checking templates in: %s", path))
	if templateCheckRecursive {
		printInfo("Mode: Recursive")
	}
	if templateCheckVerbose {
		printInfo("Verbosity: Enabled")
	}
	printSeparator()

	// Call app layer
	result, err := app.CheckTemplate(cmd.Context(), app.CheckTemplateOptions{
		Path:      path,
		Recursive: templateCheckRecursive,
		Verbose:   templateCheckVerbose,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Template check failed: %v", err))
		return err
	}

	// Display results
	printInfo("")
	printHeader("Check Results")

	if result.FilesChecked == 0 {
		printWarning("No template files found (files containing @ign- directives)")
		return nil
	}

	printInfo(fmt.Sprintf("Files checked: %d", result.FilesChecked))

	if result.FilesWithErrors > 0 {
		printInfo(fmt.Sprintf("Files with errors: %d", result.FilesWithErrors))
		printSeparator()
		printHeader("Errors Found")

		for _, checkErr := range result.Errors {
			if checkErr.Line > 0 {
				printErrorMsg(fmt.Sprintf("%s:%d - %s", checkErr.File, checkErr.Line, checkErr.Message))
			} else {
				printErrorMsg(fmt.Sprintf("%s - %s", checkErr.File, checkErr.Message))
			}
			if checkErr.Directive != "" && templateCheckVerbose {
				printVerbose(templateCheckVerbose, fmt.Sprintf("  Directive: %s", checkErr.Directive))
			}
		}

		printSeparator()
		printErrorMsg(fmt.Sprintf("Validation failed: %d file(s) with errors", result.FilesWithErrors))

		// Exit with error code
		os.Exit(1)
	} else {
		printSuccess("All templates are valid")
	}

	return nil
}
