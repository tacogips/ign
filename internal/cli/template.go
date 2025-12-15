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

// templateNewCmd represents the template new command
var templateNewCmd = &cobra.Command{
	Use:   "new [PATH]",
	Short: "Create a new template scaffold",
	Long: `Create a new template directory with scaffold files.

The new command creates a template directory structure with:
- ign.json: Template configuration with variable definitions
- README.md: Template documentation with ign directive examples
- example.txt: Example file demonstrating ign directives

If PATH is not specified, creates the template in ./my-template.

Examples:
  ign template new
  ign template new ./my-template
  ign template new ./my-go-app --type go
  ign template new --force ./existing-dir`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemplateNew,
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

// templateCollectVarsCmd represents the template collect-vars command
var templateCollectVarsCmd = &cobra.Command{
	Use:   "collect-vars [PATH]",
	Short: "Collect variables from templates and update ign.json",
	Long: `Scan template files for @ign-var: and @ign-if: directives and
automatically update ign.json with the collected variable definitions.

This command helps keep ign.json in sync with the actual variables used
in your template files.

If PATH is not specified, the current directory is used.

Examples:
  ign template collect-vars
  ign template collect-vars ./my-template
  ign template collect-vars -r           # Recursive scan
  ign template collect-vars --dry-run    # Preview changes
  ign template collect-vars --merge      # Only add new variables`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemplateCollectVars,
}

// Template new command flags
var (
	templateNewType  string
	templateNewForce bool
)

// Template check command flags
var (
	templateCheckRecursive bool
	templateCheckVerbose   bool
)

// Template auto-collect-vars command flags
var (
	templateCollectRecursive bool
	templateCollectDryRun    bool
	templateCollectMerge     bool
)

func init() {
	// Add subcommands to template
	templateCmd.AddCommand(templateNewCmd)
	templateCmd.AddCommand(templateCheckCmd)
	templateCmd.AddCommand(templateCollectVarsCmd)

	// Flags for template new
	templateNewCmd.Flags().StringVarP(&templateNewType, "type", "t", "default", "Scaffold type to use (e.g., default, go, web)")
	templateNewCmd.Flags().BoolVarP(&templateNewForce, "force", "f", false, "Overwrite existing files")

	// Flags for template check
	templateCheckCmd.Flags().BoolVarP(&templateCheckRecursive, "recursive", "r", false, "Recursively check subdirectories")
	templateCheckCmd.Flags().BoolVarP(&templateCheckVerbose, "verbose", "v", false, "Show detailed validation info")

	// Flags for template collect-vars
	templateCollectVarsCmd.Flags().BoolVarP(&templateCollectRecursive, "recursive", "r", false, "Recursively scan subdirectories")
	templateCollectVarsCmd.Flags().BoolVar(&templateCollectDryRun, "dry-run", false, "Preview changes without writing")
	templateCollectVarsCmd.Flags().BoolVar(&templateCollectMerge, "merge", false, "Only add new variables, preserve existing")
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

func runTemplateNew(cmd *cobra.Command, args []string) error {
	// Default path if not specified
	path := "./my-template"
	if len(args) > 0 {
		path = args[0]
	}

	printInfo(fmt.Sprintf("Creating new template at: %s", path))
	printInfo(fmt.Sprintf("Scaffold type: %s", templateNewType))
	if templateNewForce {
		printWarning("Force mode: existing files will be overwritten")
	}
	printSeparator()

	// Get available types for error message
	availableTypes, err := app.AvailableScaffoldTypes()
	if err != nil {
		printErrorMsg(fmt.Sprintf("Failed to list scaffold types: %v", err))
		return err
	}

	// Show available types in verbose mode
	printInfo(fmt.Sprintf("Available scaffold types: %v", availableTypes))

	// Call app layer
	result, err := app.NewTemplate(cmd.Context(), app.NewTemplateOptions{
		Path:  path,
		Type:  templateNewType,
		Force: templateNewForce,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Failed to create template: %v", err))
		return err
	}

	// Display results
	printSeparator()
	printHeader("Template Created")
	printSuccess(fmt.Sprintf("Created at: %s", result.Path))
	printInfo(fmt.Sprintf("Files created: %d", result.FilesCreated))

	printSeparator()
	printHeader("Files")
	for _, file := range result.Files {
		printInfo(fmt.Sprintf("  %s", file))
	}

	printSeparator()
	printInfo("Next steps:")
	printInfo(fmt.Sprintf("  1. cd %s", path))
	printInfo("  2. Edit ign.json to customize template variables")
	printInfo("  3. Add your template files with @ign- directives")
	printInfo("  4. Run 'ign template check' to validate")

	return nil
}

func runTemplateCollectVars(cmd *cobra.Command, args []string) error {
	// Default to current directory if no path specified
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	printInfo(fmt.Sprintf("Scanning templates in: %s", path))
	if templateCollectRecursive {
		printInfo("Mode: Recursive")
	}
	if templateCollectDryRun {
		printWarning("Dry-run mode: no files will be modified")
	}
	if templateCollectMerge {
		printInfo("Merge mode: only adding new variables")
	}
	printSeparator()

	// Call app layer
	result, err := app.CollectVars(cmd.Context(), app.CollectVarsOptions{
		Path:      path,
		Recursive: templateCollectRecursive,
		DryRun:    templateCollectDryRun,
		Merge:     templateCollectMerge,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Failed to collect variables: %v", err))
		return err
	}

	// Display results
	printHeader("Scan Results")
	printInfo(fmt.Sprintf("Files scanned: %d", result.FilesScanned))
	printInfo(fmt.Sprintf("Variables found: %d", len(result.Variables)))

	if len(result.Variables) == 0 {
		printWarning("No variables found in template files")
		return nil
	}

	printSeparator()
	printHeader("Variables")
	for name, v := range result.Variables {
		typeStr := string(v.Type)
		if typeStr == "" {
			typeStr = "string"
		}
		reqStr := ""
		if v.Required {
			reqStr = " (required)"
		} else if v.HasDefault {
			reqStr = fmt.Sprintf(" (default: %v)", v.Default)
		}
		printInfo(fmt.Sprintf("  %s: %s%s", name, typeStr, reqStr))
	}

	if len(result.NewVars) > 0 {
		printSeparator()
		printHeader("New Variables")
		for _, name := range result.NewVars {
			printSuccess(fmt.Sprintf("  + %s", name))
		}
	}

	if len(result.UpdatedVars) > 0 {
		printSeparator()
		printHeader("Updated Variables")
		for _, name := range result.UpdatedVars {
			printInfo(fmt.Sprintf("  ~ %s", name))
		}
	}

	printSeparator()
	if templateCollectDryRun {
		printWarning(fmt.Sprintf("Would update: %s", result.IgnJsonPath))
	} else if result.Updated {
		printSuccess(fmt.Sprintf("Updated: %s", result.IgnJsonPath))
	}

	return nil
}
