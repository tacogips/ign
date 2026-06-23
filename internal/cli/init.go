package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
	templatedefaults "github.com/tacogips/ign/internal/template/defaults"
)

var (
	initRef   string
	initForce bool
	initVars  []string
)

var initCmd = &cobra.Command{
	Use:   "init <url-or-path>",
	Short: "Initialize ign configuration from a template",
	Long: `Initialize ign configuration from a template source.

The init command creates .ign/ign.json and .ign/ign-var.json. Template variables
can be supplied non-interactively with --var key=value. Missing variables are
prompted interactively.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initRef, FlagRef, "r", "main", DescRef)
	initCmd.Flags().BoolVarP(&initForce, FlagForce, "f", false, "Backup existing config and reinitialize")
	initCmd.Flags().StringArrayVarP(&initVars, FlagVar, "V", nil, DescVar)
}

func runInit(cmd *cobra.Command, args []string) error {
	url := args[0]
	if err := ValidateVariableAssignmentSyntax(initVars); err != nil {
		printErrorMsg(fmt.Sprintf("Variable parsing failed: %v", err))
		return err
	}

	configDir := ".ign"
	configExists := false

	if _, err := os.Stat(configDir); err == nil {
		configExists = true
		if !initForce {
			err := fmt.Errorf("configuration already exists at %s (use --force to reinitialize)", configDir)
			printErrorMsg(err.Error())
			return err
		}
		printWarning("Force mode enabled - will backup existing configuration")
	}

	githubToken := getGitHubToken("")

	printInfo(fmt.Sprintf("Template: %s", url))
	if initRef != "main" {
		printInfo(fmt.Sprintf("Reference: %s", initRef))
	}

	prepResult, err := app.PrepareCheckout(cmd.Context(), app.PrepareCheckoutOptions{
		URL:             url,
		Ref:             initRef,
		Force:           initForce,
		ConfigExists:    configExists,
		GitHubToken:     githubToken,
		SkipConfigSetup: true,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Initialization failed: %v", err))
		return err
	}

	resolvedIgnJSON := templatedefaults.ResolveIgnJSON(prepResult.IgnJson, ".")
	providedVars, err := ParseVariableAssignments(initVars, resolvedIgnJSON.Variables)
	if err != nil {
		printErrorMsg(fmt.Sprintf("Variable parsing failed: %v", err))
		return err
	}

	vars, err := PromptForVariablesWithProvided(resolvedIgnJSON, providedVars)
	if err != nil {
		printErrorMsg(fmt.Sprintf("Variable collection failed: %v", err))
		return err
	}

	if err := app.PrepareCheckoutConfigDir(configExists); err != nil {
		printErrorMsg(fmt.Sprintf("Initialization failed: %v", err))
		return err
	}

	if err := app.CompleteInit(cmd.Context(), app.CompleteInitOptions{
		PrepareResult: prepResult,
		Variables:     vars,
		GeneratedBy:   "ign init",
	}); err != nil {
		printErrorMsg(fmt.Sprintf("Initialization failed: %v", err))
		return err
	}

	printSuccess("Configuration initialized successfully")
	printInfo("Configuration saved to: .ign/ign.json, .ign/ign-var.json")
	return nil
}
