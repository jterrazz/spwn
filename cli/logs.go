package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var (
	logsNoFollow bool
	logsTail     int
)

func init() {
	logsCmd.Flags().BoolVar(&logsNoFollow, "no-follow", false, "Print current logs and exit")
	logsCmd.Flags().IntVarP(&logsTail, "n", "n", 100, "Number of lines to show")

	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <universe-id>",
	Short: "Stream agent output from a running universe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]

		arc, err := newArchitect()
		if err != nil {
			return err
		}

		tail := fmt.Sprintf("%d", logsTail)
		reader, err := arc.Logs(ctx, universeID, !logsNoFollow, tail)
		if err != nil {
			return fmt.Errorf("error: cannot stream logs for %s.\n%w", universeID, err)
		}
		defer reader.Close()

		stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		if logsNoFollow {
			io.Copy(os.Stdout, reader)
		}

		return nil
	},
}
