package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/version"
)

// Alias version variables for compatibility
var (
	Version   = version.Version
	GitCommit = version.GitCommit
	BuildDate = version.BuildDate
)

// Global flags
var (
	globalNoColor bool
	globalQuiet   bool
	globalDebug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ign",
	Short: "Project template initialization tool",
	Long: `ign is a CLI tool for initializing projects from templates.

Use "ign checkout <url> [output-path]" to:
  1. Create .ign directory with configuration
  2. Interactively prompt for template variables
  3. Generate project files from the template

Templates are fetched from GitHub repositories and can include variables,
conditionals, and file inclusions for flexible project generation.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set debug mode
		debug.SetDebug(globalDebug)
		debug.SetNoColor(globalNoColor)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&globalQuiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&globalDebug, FlagDebug, false, DescDebug)

	// Add subcommands
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(versionCmd)
}

// printError prints an error message to stderr
func printError(err error) {
	if globalQuiet {
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
