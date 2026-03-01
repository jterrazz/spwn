package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Stream agent output from a running universe",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

var (
	logsNoFollow bool
	logsTail     string
)

func init() {
	logsCmd.Flags().BoolVar(&logsNoFollow, "no-follow", false, "Print current logs and exit")
	logsCmd.Flags().StringVarP(&logsTail, "n", "n", "100", "Number of lines to show")

	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	universeID := args[0]

	arch, err := newArchitect()
	if err != nil {
		return err
	}

	reader, err := arch.Logs(cmd.Context(), universeID, !logsNoFollow, logsTail)
	if err != nil {
		return fmt.Errorf("error: universe %s not found.\nRun 'universe list' to see available universes", universeID)
	}
	defer reader.Close()

	// Docker multiplexes stdout/stderr with 8-byte headers.
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	if err != nil {
		// Fallback to raw copy if stdcopy fails (e.g., TTY mode).
		io.Copy(os.Stdout, reader)
	}

	return nil
}
