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
	logsFollow bool
	logsTail   int
)

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsTail, "n", "n", 100, "Number of lines to show")

	Cmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <world-id>",
	Short: "Show agent output from a world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		tail := fmt.Sprintf("%d", logsTail)
		reader, err := arc.Logs(ctx, worldID, logsFollow, tail)
		if err != nil {
			return fmt.Errorf("cannot stream logs for %s: %w", worldID, err)
		}
		defer reader.Close()

		stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		if !logsFollow {
			io.Copy(os.Stdout, reader)
		}

		return nil
	},
}
