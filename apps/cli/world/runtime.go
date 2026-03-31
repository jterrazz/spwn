package world

import (
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(runtimeListCmd)
}

var runtimeListCmd = &cobra.Command{
	Use:   "runtimes",
	Short: "List available agent runtimes and claws",
	RunE: func(cmd *cobra.Command, args []string) error {
		runtimes := universe.ListRuntimes()
		claws := universe.ListClaws()

		t := ui.NewTable(ui.ModeNormal, "NAME", "TYPE", "BASE IMAGE")
		for _, name := range runtimes {
			df, _ := universe.GenerateRuntimeDockerfile(name)
			baseImage := extractBaseImage(df)
			t.AddRow(name, "runtime", baseImage)
		}
		for _, name := range claws {
			t.AddRow(name, "claw", "\u2014")
		}
		t.Render()

		fmt.Fprintf(cmd.ErrOrStderr(), "  %d runtime(s), %d claw(s)\n\n", len(runtimes), len(claws))
		return nil
	},
}

// extractBaseImage parses the FROM line from a Dockerfile string.
func extractBaseImage(dockerfile string) string {
	for _, line := range strings.Split(dockerfile, "\n") {
		if strings.HasPrefix(line, "FROM ") {
			return strings.TrimPrefix(line, "FROM ")
		}
	}
	return "\u2014"
}
