package world

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/pkg/stdcopy"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	logsNoFollow bool
	logsTail     int
)

func init() {
	logsCmd.Flags().BoolVar(&logsNoFollow, "no-follow", false, "Print current logs and exit")
	logsCmd.Flags().IntVarP(&logsTail, "n", "n", 100, "Number of lines to show")

	Cmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <world-id>",
	Short: "Stream agent output from a running world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		tail := fmt.Sprintf("%d", logsTail)
		reader, err := arc.Logs(ctx, worldID, !logsNoFollow, tail)
		if err != nil {
			return fmt.Errorf("error: cannot stream logs for %s.\n%w", worldID, err)
		}
		defer reader.Close()

		stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		if logsNoFollow {
			io.Copy(os.Stdout, reader)
		}

		return nil
	},
}
