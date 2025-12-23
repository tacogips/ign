package cli

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
	"github.com/tacogips/ign/internal/template/model"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [output-path]",
	Short: "Update project from template changes",
	Long: `Update project files when the template has changed.

This command checks if the template has been updated since the last checkout:
1. Fetches the template from the stored URL in .ign/ign.json
2. Compares the template hash to detect changes
3. If new variables are added, prompts for their values
4. Regenerates project files with the updated template

Requirements:
  - .ign/ign.json must exist (created by 'ign checkout')
  - .ign/ign-var.json must exist (stores variable values)

If the template has not changed (same hash), no action is taken.

Examples:
  ign update                     # Update in current directory
  ign update ./my-project        # Update to specific directory
  ign update --dry-run           # Preview changes without writing
  ign update --force             # Overwrite existing files`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpdate,
}

// Update command flags
var (
	updateForce   bool
	updateDryRun  bool
	updateVerbose bool
)

func init() {
	// Flags for update
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Overwrite existing files")
	updateCmd.Flags().BoolVarP(&updateDryRun, "dry-run", "d", false, "Show what would be generated without writing files")
	updateCmd.Flags().BoolVarP(&updateVerbose, "verbose", "v", false, "Show detailed processing information")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Output path defaults to current directory
	outputPath := "."
	if len(args) > 0 {
		outputPath = args[0]
	}

	// Get GitHub token from environment
	githubToken := getGitHubToken("")

	printInfo("Checking for template updates...")

	// Prepare update - fetch template and check for changes
	prepResult, err := app.PrepareUpdate(cmd.Context(), app.UpdateOptions{
		OutputDir:   outputPath,
		Overwrite:   updateForce,
		DryRun:      updateDryRun,
		Verbose:     updateVerbose,
		GitHubToken: githubToken,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Update preparation failed: %v", err))
		return err
	}

	// Show template info
	printInfo(fmt.Sprintf("Template: %s", prepResult.IgnConfig.Template.URL))
	if prepResult.IgnConfig.Template.Ref != "" && prepResult.IgnConfig.Template.Ref != "main" {
		printInfo(fmt.Sprintf("Reference: %s", prepResult.IgnConfig.Template.Ref))
	}
	printSeparator()

	// Check if hash changed
	if !prepResult.HashChanged {
		printSuccess("Template is up to date (no changes detected)")
		return nil
	}

	printInfo("Template has been updated")
	printInfo(fmt.Sprintf("  Previous hash: %s", truncateHash(prepResult.CurrentHash)))
	printInfo(fmt.Sprintf("  New hash:      %s", truncateHash(prepResult.NewHash)))

	// Show variable changes
	if len(prepResult.RemovedVars) > 0 {
		printSeparator()
		printWarning("The following variables have been removed from the template:")
		for _, name := range prepResult.RemovedVars {
			printInfo(fmt.Sprintf("  - %s", name))
		}
	}

	// Prompt for new variables if any
	var newVarValues map[string]interface{}
	if len(prepResult.NewVars) > 0 {
		printSeparator()
		printInfo("New variables have been added to the template:")

		// Get variable definitions for new variables
		newVarDefs := app.GetNewVariableDefinitions(prepResult)

		// Separate variables into those needing prompt and those with defaults
		varsNeedingPrompt := app.FilterVariablesForPrompt(newVarDefs)

		// Show variables with defaults
		for name, varDef := range newVarDefs {
			if _, needsPrompt := varsNeedingPrompt[name]; !needsPrompt {
				printInfo(fmt.Sprintf("  + %s (default: %v)", name, varDef.Default))
			}
		}

		// Prompt for variables that need input
		if len(varsNeedingPrompt) > 0 {
			printInfo("")
			printInfo("Please provide values for the following new variables:")
			promptedVars, err := PromptForNewVariables(varsNeedingPrompt)
			if err != nil {
				printErrorMsg(fmt.Sprintf("Failed to collect variable values: %v", err))
				return err
			}
			newVarValues = app.ApplyDefaults(newVarDefs, promptedVars)
		} else {
			// All new variables have defaults
			newVarValues = app.ApplyDefaults(newVarDefs, nil)
		}
	}

	// Complete update
	printSeparator()
	if updateDryRun {
		printInfo("[DRY RUN] Would regenerate project from template")
	} else {
		printInfo("Regenerating project from template...")
	}

	result, err := app.CompleteUpdate(cmd.Context(), app.CompleteUpdateOptions{
		PrepareResult: prepResult,
		NewVariables:  newVarValues,
		OutputDir:     outputPath,
		Overwrite:     updateForce,
		DryRun:        updateDryRun,
		Verbose:       updateVerbose,
	})

	if err != nil {
		printErrorMsg(fmt.Sprintf("Update failed: %v", err))
		return err
	}

	// Print results
	if updateDryRun {
		printUpdateDryRunPatch(result)
	} else {
		printSuccess("Project updated successfully")
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
		printInfo("Configuration updated: .ign/ign.json, .ign/ign-var.json")
		printInfo(fmt.Sprintf("Project ready at: %s", outputPath))
	}

	return nil
}

// truncateHash truncates a hash for display purposes.
func truncateHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:8] + "..." + hash[len(hash)-8:]
}

// PromptForNewVariables prompts for values of new variables.
func PromptForNewVariables(varDefs map[string]model.VarDef) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	if len(varDefs) == 0 {
		return vars, nil
	}

	// Sort variable names for consistent ordering
	varNames := make([]string, 0, len(varDefs))
	for name := range varDefs {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	fmt.Println()

	for _, name := range varNames {
		varDef := varDefs[name]

		value, err := promptForVariable(name, varDef)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for variable %q: %w", name, err)
		}

		vars[name] = value
	}

	return vars, nil
}

// PromptForVariablesSubset prompts for a subset of variables from IgnJson.
func PromptForVariablesSubset(ignJson *model.IgnJson, varNames []string) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	if len(varNames) == 0 {
		return vars, nil
	}

	// Sort for consistent ordering
	sort.Strings(varNames)

	fmt.Println()
	fmt.Println("Please provide values for new template variables:")
	fmt.Println()

	for _, name := range varNames {
		varDef, ok := ignJson.Variables[name]
		if !ok {
			continue // Skip if variable not found
		}

		value, err := promptForVariable(name, varDef)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for variable %q: %w", name, err)
		}

		vars[name] = value
	}

	return vars, nil
}

// ConfirmUpdate prompts the user to confirm the update operation.
func ConfirmUpdate(prepResult *app.PrepareUpdateResult) (bool, error) {
	var confirm bool
	prompt := &survey.Confirm{
		Message: "Do you want to proceed with the update?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return false, err
	}
	return confirm, nil
}

// printUpdateDryRunPatch outputs dry-run results in unified diff format.
func printUpdateDryRunPatch(result *app.UpdateResult) {
	// Print summary header
	fmt.Println("# DRY RUN - No files will be written")
	fmt.Println("#")

	// Print variable changes
	if len(result.NewVariables) > 0 || len(result.RemovedVariables) > 0 {
		fmt.Println("# Variable changes:")
		for _, name := range result.NewVariables {
			fmt.Printf("#   + %s (new)\n", name)
		}
		for _, name := range result.RemovedVariables {
			fmt.Printf("#   - %s (removed)\n", name)
		}
		fmt.Println("#")
	}

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
			fmt.Printf("# SKIP: %s (file exists, use --force to overwrite)\n\n", f.Path)
			continue
		}

		if f.WouldOverwrite {
			fmt.Printf("# OVERWRITE: %s\n", f.Path)
		}
		fmt.Printf("--- /dev/null\n")
		fmt.Printf("+++ %s\n", f.Path)

		lines := countLinesForUpdate(f.Content)
		if lines == 0 {
			fmt.Println("@@ -0,0 +0,0 @@")
		} else {
			fmt.Printf("@@ -0,0 +1,%d @@\n", lines)
		}

		printPatchContentForUpdate(f.Content)
		fmt.Println()
	}
}

// countLinesForUpdate counts the number of lines in content.
func countLinesForUpdate(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte{'\n'})
	if len(content) > 0 && content[len(content)-1] != '\n' {
		count++
	}
	return count
}

// printPatchContentForUpdate prints file content with + prefix for each line.
func printPatchContentForUpdate(content []byte) {
	if len(content) == 0 {
		return
	}

	if isBinaryContentForUpdate(content) {
		fmt.Println("+[binary file]")
		return
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}
		fmt.Printf("+%s\n", line)
	}
}

// isBinaryContentForUpdate checks if content appears to be binary.
func isBinaryContentForUpdate(content []byte) bool {
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
