package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tacogips/ign/internal/app"
)

var (
	varsJSON  bool
	varsUnset bool
)

var errRequiredVarsUnset = errors.New("required template variables are unset")

var varsCmd = &cobra.Command{
	Use:   "vars",
	Short: "Inspect template variables and current values",
	Long: `Inspect template variable declarations and current values from .ign/ign-var.json.

The default output is a table with NAME, TYPE, REQUIRED, DEFAULT, CURRENT, and
DESCRIPTION columns. Use --json for scripting and --unset to show only variables
without a current value.`,
	Args: cobra.NoArgs,
	RunE: runVars,
}

func init() {
	varsCmd.Flags().BoolVar(&varsJSON, "json", false, "Print variables as JSON")
	varsCmd.Flags().BoolVar(&varsUnset, "unset", false, "Show only variables without a current value")
}

func runVars(cmd *cobra.Command, args []string) error {
	result, err := app.InspectVars(cmd.Context(), app.VarsOptions{
		GitHubToken: getGitHubToken(""),
	})
	if err != nil {
		return err
	}

	if varsUnset {
		result = app.FilterUnsetVars(result)
	}

	if !globalQuiet {
		if varsJSON {
			if result.DeclarationError != nil {
				printVarsWarning(cmd.ErrOrStderr(), result.DeclarationError)
			}
			if err := printVarsJSON(cmd.OutOrStdout(), result); err != nil {
				return err
			}
		} else {
			if result.DeclarationError != nil {
				printVarsWarning(cmd.ErrOrStderr(), result.DeclarationError)
			}
			if err := printVarsTable(cmd.OutOrStdout(), result); err != nil {
				return err
			}
		}
	}

	if varsUnset && result.RequiredUnsetCount > 0 {
		return errRequiredVarsUnset
	}

	return nil
}

func printVarsWarning(w io.Writer, err error) {
	if err == nil || globalQuiet {
		return
	}
	fmt.Fprintf(w, "Warning: template declarations unavailable; showing local values only: %v\n", err)
}

func printVarsJSON(w io.Writer, result *app.VarsResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printVarsTable(w io.Writer, result *app.VarsResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tTYPE\tREQUIRED\tDEFAULT\tCURRENT\tDESCRIPTION"); err != nil {
		return err
	}
	for _, row := range result.Rows {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Name,
			row.Type,
			formatBool(row.Required),
			formatVarsValue(row.Default, row.Default != nil),
			formatVarsValue(row.Current, row.HasCurrent),
			strings.TrimSpace(row.Description),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func formatVarsValue(value interface{}, hasValue bool) string {
	if !hasValue {
		return ""
	}
	if value == nil {
		return "null"
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
