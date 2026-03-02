package agent

import (
	"fmt"
	"os"

	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

var (
	exportOutput  string
	exportExclude []string
)

func init() {
	exportCmd.Flags().StringVar(&exportOutput, "output", ".", "Output directory")
	exportCmd.Flags().StringSliceVar(&exportExclude, "exclude", nil, "Exclude layers (e.g., journal, sessions)")
	Cmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export <agent-name>",
	Short: "Export an agent as tar.gz",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Validate output directory
		if _, err := os.Stat(exportOutput); err != nil {
			return fmt.Errorf("error: output directory %q not found", exportOutput)
		}

		fmt.Printf("\n  Exporting agent %s...\n\n", name)

		archivePath, err := mind.Export(name, exportOutput, exportExclude)
		if err != nil {
			return fmt.Errorf("error: export failed.\n%w", err)
		}

		fmt.Printf("  ✓ Written to %s\n\n", archivePath)
		return nil
	},
}
