package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/manifest"
)

func init() {
	buildCmd.Flags().BoolVar(&buildSkipValidate, "skip-validate", false, "Build even if spwn check finds errors")
	rootCmd.AddCommand(buildCmd)
}

var buildSkipValidate bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Flatten the project into a reproducible build artifact",
	Long: `Flatten the project into .spwn/build/ — every agent file, the world
config, and a normalized manifest, all copied into one self-contained
tree that spwn up can consume directly.

Runs spwn check first (unless --skip-validate is set). Errors abort
the build; warnings are printed but not blocking.`,
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

		out := cmd.OutOrStdout()

		if !buildSkipValidate {
			issues := manifest.Validate(p, manifest.ValidateOpts{
				BuiltinTools:      catalogToolNames(),
				SupportedRuntimes: supportedRuntimes(),
			})
			if manifest.HasErrors(issues) {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "  %s Build blocked — project has validation errors.\n", ui.Red("✗"))
				fmt.Fprintln(out, "  Run `spwn check` to see them, or pass --skip-validate.")
				fmt.Fprintln(out)
				os.Exit(1)
			}
		}

		result, err := manifest.Build(p)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s  %s\n", ui.Green("✓"), ui.Strong("Build complete"))
		fmt.Fprintf(out, "     %s\n", ui.Faint(result.Dir))
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %d file(s), %d agent(s), world=%s\n",
			result.FileCount, len(result.Agents), result.World)
		fmt.Fprintln(out)

		return nil
	},
}
