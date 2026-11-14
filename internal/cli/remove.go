package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

var (
	removeAll     bool
	removeOutput  string
	removeVerbose bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <template-id>...",
	Short: "Remove one or more checked-out templates",
	Long: `Remove one or more checked-out templates from the local project.

Removing a template deletes its generated files, removes its .ign metadata, and
replays remaining templates in checkout order so overwritten files can be
restored from earlier templates.

Examples:
  ign remove go-basic-1
  ign remove go-basic-1 lint-2
  ign remove --all`,
	Args: func(cmd *cobra.Command, args []string) error {
		if removeAll {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("requires at least one template id or --all")
		}
		return nil
	},
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Remove all checked-out templates and delete .ign")
	removeCmd.Flags().StringVarP(&removeOutput, "output", "o", ".", "Project output path; must match the stored .ign output root")
	removeCmd.Flags().BoolVarP(&removeVerbose, "verbose", "v", false, "Show detailed replay information")
}

func runRemove(cmd *cobra.Command, args []string) error {
	githubToken := getGitHubToken("")
	result, err := app.Remove(cmd.Context(), app.RemoveOptions{
		TemplateIDs: args,
		All:         removeAll,
		OutputDir:   removeOutput,
		GitHubToken: githubToken,
		Verbose:     removeVerbose,
	})
	if err != nil {
		printErrorMsg(fmt.Sprintf("Remove failed: %v", err))
		return err
	}

	if removeAll {
		printSuccess("All templates removed")
	} else {
		printSuccess("Template removed successfully")
	}
	printInfo("")
	printInfo("Summary:")
	printInfo(fmt.Sprintf("  Templates removed: %d", result.TemplatesRemoved))
	printInfo(fmt.Sprintf("  Files removed: %d", result.FilesRemoved))
	if result.FilesMissing > 0 {
		printInfo(fmt.Sprintf("  Files already missing: %d", result.FilesMissing))
	}
	if result.FilesReplayed > 0 || result.FilesOverwritten > 0 {
		printInfo(fmt.Sprintf("  Replayed created: %d", result.FilesReplayed))
		printInfo(fmt.Sprintf("  Replayed overwritten: %d", result.FilesOverwritten))
	}
	return nil
}
