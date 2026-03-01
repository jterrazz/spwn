package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/mind"
)

var exportAgentCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export a Mind as tar.gz archive",
	Args:  cobra.ExactArgs(1),
	RunE:  runExportAgent,
}

var (
	exportOutput  string
	exportExclude []string
)

func init() {
	exportAgentCmd.Flags().StringVarP(&exportOutput, "output", "o", ".", "Output directory")
	exportAgentCmd.Flags().StringArrayVar(&exportExclude, "exclude", nil, "Layers to exclude (repeatable)")
}

func runExportAgent(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := mind.Validate(name); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Exporting agent %q...\n", name)
	fmt.Println()

	archivePath, err := mind.Export(name, exportOutput, exportExclude)
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ Written to %s\n", archivePath)
	fmt.Println()
	fmt.Printf("  Import with: universe agent spawn <universe-id> --agent %s --import %s\n", name, archivePath)
	fmt.Println()

	return nil
}
