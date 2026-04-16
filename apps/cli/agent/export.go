package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
)

var (
	exportOutput  string
	exportExclude []string
)

func init() {
	exportCmd.Flags().StringVar(&exportOutput, "output", ".", "Output directory")
	exportCmd.Flags().StringSliceVar(&exportExclude, "exclude", nil, "Exclude layers (e.g., journal, sessions)")
	Cmd.AddCommand(exportCmd)
	ui.MarkExperimental(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export <agent-name>",
	Short: "Export an agent as tar.gz",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := ui.New()

		// Validate output directory
		if _, err := os.Stat(exportOutput); err != nil {
			return fmt.Errorf("output directory %q not found", exportOutput)
		}

		s.Blank()
		s.Start(fmt.Sprintf("Exporting agent %s...", name))

		archivePath, err := agent.ExportMind(name, exportOutput, exportExclude)
		if err != nil {
			return s.FailHint("Export failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent ls\"", name))
		}

		s.Done("Exported", archivePath)
		s.Blank()
		return nil
	},
}
