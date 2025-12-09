package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Build-time variables set via ldflags
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Global flags
var (
	globalNoColor bool
	globalQuiet   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ign",
	Short: "Project template initialization tool",
	Long: `ign is a CLI tool for initializing projects from templates.

It provides a two-step workflow:
  1. "ign build init" - Creates build configuration from a template
  2. "ign init" - Generates project files using the build configuration

Templates are fetched from GitHub repositories and can include variables,
conditionals, and file inclusions for flexible project generation.`,
	SilenceUsage:  true,
	SilenceErrors: true,
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

	// Add subcommands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
}

// printError prints an error message to stderr
func printError(err error) {
	if globalQuiet {
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
