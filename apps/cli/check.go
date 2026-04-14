package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/manifest"
)

func init() {
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "Exit non-zero on warnings, not just errors")
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Emit results as structured JSON on stdout")
	rootCmd.AddCommand(checkCmd)
}

var (
	checkStrict bool
	checkJSON   bool
)

// checkReport is the CLI-owned JSON schema for `spwn check`. It's
// intentionally decoupled from the internal validate.Issue type so the
// JSON contract can evolve independently of the rule engine internals.
type checkReport struct {
	Valid        bool          `json:"valid"`
	ManifestPath string        `json:"manifestPath"`
	Summary      checkSummary  `json:"summary"`
	Issues       []checkIssue  `json:"issues"`
}

type checkSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

type checkIssue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

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

		errors := filter(issues, manifest.LevelError)
		warnings := filter(issues, manifest.LevelWarning)
		infos := filter(issues, manifest.LevelInfo)

		if checkJSON {
			report := buildCheckReport(p.ManifestPath, errors, warnings, infos)
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			if err := enc.Encode(report); err != nil {
				return fmt.Errorf("encode json: %w", err)
			}
			if len(errors) > 0 || (checkStrict && len(warnings) > 0) {
				os.Exit(1)
			}
			return nil
		}

		if len(issues) == 0 {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  %s  %s\n", ui.Green("✓"), ui.Strong("Project is valid"))
			fmt.Fprintf(out, "     %s\n", ui.Faint(p.ManifestPath))
			fmt.Fprintln(out)
			return nil
		}

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

func buildCheckReport(manifestPath string, errors, warnings, infos []manifest.Issue) checkReport {
	issues := make([]checkIssue, 0, len(errors)+len(warnings)+len(infos))
	for _, group := range [][]manifest.Issue{errors, warnings, infos} {
		for _, i := range group {
			issues = append(issues, checkIssue{
				Level:   i.Level.String(),
				Path:    i.Path,
				Message: i.Message,
				Hint:    i.Hint,
			})
		}
	}
	return checkReport{
		Valid:        len(errors) == 0,
		ManifestPath: manifestPath,
		Summary: checkSummary{
			Errors:   len(errors),
			Warnings: len(warnings),
			Info:     len(infos),
		},
		Issues: issues,
	}
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
