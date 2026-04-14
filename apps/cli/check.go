package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/manifest"
)

func init() {
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "Exit non-zero on warnings, not just errors")
	rootCmd.AddCommand(checkCmd)
}

var checkStrict bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate the project tree against spwn.yaml",
	Long: `Walks up from the current directory looking for spwn.yaml, then runs
every validation rule against the project. Reports issues grouped by
severity. Exits non-zero when errors are found (or warnings, with
--strict).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}

		p, err := manifest.Find(cwd)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		if p == nil {
			return fmt.Errorf("no spwn.yaml found in %s or any parent directory.\nRun `spwn init` to create one", cwd)
		}

		issues := manifest.Validate(p, manifest.ValidateOpts{
			BuiltinTools:      catalogToolNames(),
			SupportedRuntimes: supportedRuntimes(),
		})
		out := cmd.OutOrStdout()

		if len(issues) == 0 {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  %s  %s\n", ui.Green("✓"), ui.Strong("Project is valid"))
			fmt.Fprintf(out, "     %s\n", ui.Faint(p.ManifestPath))
			fmt.Fprintln(out)
			return nil
		}

		errors := filter(issues, manifest.LevelError)
		warnings := filter(issues, manifest.LevelWarning)
		infos := filter(issues, manifest.LevelInfo)

		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s\n", ui.Faint(p.ManifestPath))
		fmt.Fprintln(out)

		printGroup(out, "errors", errors, ui.Red)
		printGroup(out, "warnings", warnings, ui.Yellow)
		printGroup(out, "info", infos, ui.Faint)

		fmt.Fprintf(out, "  %d error(s), %d warning(s), %d info\n",
			len(errors), len(warnings), len(infos))
		fmt.Fprintln(out)

		if len(errors) > 0 || (checkStrict && len(warnings) > 0) {
			os.Exit(1)
		}
		return nil
	},
}

func filter(issues []manifest.Issue, level manifest.Level) []manifest.Issue {
	var out []manifest.Issue
	for _, i := range issues {
		if i.Level == level {
			out = append(out, i)
		}
	}
	return out
}

func printGroup(out interface{ Write([]byte) (int, error) }, label string, issues []manifest.Issue, color func(string) string) {
	if len(issues) == 0 {
		return
	}
	fmt.Fprintf(out, "  %s\n", color(label))
	for _, i := range issues {
		fmt.Fprintf(out, "    %s  %s\n", color("•"), i.Message)
		fmt.Fprintf(out, "       %s\n", ui.Faint(i.Path))
		if i.Hint != "" {
			fmt.Fprintf(out, "       %s %s\n", ui.Faint("→"), ui.Faint(i.Hint))
		}
	}
	fmt.Fprintln(out)
}
