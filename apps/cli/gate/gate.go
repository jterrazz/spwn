// Package gate hosts the `spwn gate ...` subcommands. The bare
// `spwn gate` command renders cobra's help; concrete actions live
// under start/stop/status/logs/restart.
package gate

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/packages/gate"
)

// Cmd is the parent for `spwn gate ...`. Wired into the root command
// in apps/cli/root.go.
var Cmd = &cobra.Command{
	Use:   "gate",
	Short: "Manage the host-side credential broker (start/stop/status/logs/restart)",
	Long: `Manage the host-side credential broker.

The gate is a long-running container that holds OAuth credentials,
hosts upstream MCP servers, and exposes them to world containers as
authenticated MCP endpoints. World containers never see credentials —
they get tiny CLI wrappers that route through the gate.

Auto-started on first ` + "`spwn up`" + `; explicit lifecycle commands let you
inspect, restart, or troubleshoot.`,
}

var startRebuild bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Build the gate image (if missing) and start the container",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		var err error
		if startRebuild {
			err = gate.EnsureRunningRebuild(ctx, cmd.OutOrStderr())
		} else {
			err = gate.EnsureRunning(ctx, cmd.OutOrStderr())
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ gate running on 127.0.0.1:"+gate.HostPort)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop and remove the gate container (image stays on disk)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := gate.Stop(context.Background()); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ gate stopped")
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop + start the gate container; --rebuild forces a fresh image build first",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		if err := gate.Stop(ctx); err != nil {
			return err
		}
		var err error
		if startRebuild {
			err = gate.EnsureRunningRebuild(ctx, cmd.OutOrStderr())
		} else {
			err = gate.EnsureRunning(ctx, cmd.OutOrStderr())
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ gate restarted")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the gate is running",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		st, err := gate.Status(context.Background())
		if err != nil {
			return err
		}
		switch st {
		case "running":
			fmt.Fprintln(cmd.OutOrStdout(), "  ✓ running on 127.0.0.1:"+gate.HostPort)
		case "stopped":
			fmt.Fprintln(cmd.OutOrStdout(), "  · stopped (container exists)   ·  spwn gate start")
		case "missing":
			fmt.Fprintln(cmd.OutOrStdout(), "  · not installed   ·  spwn gate start")
		}
		return nil
	},
}

var logsFollow bool
var logsTail int

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream gate logs (docker logs)",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		c := gate.LogsCmd(context.Background(), logsFollow, logsTail)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	startCmd.Flags().BoolVar(&startRebuild, "rebuild", false, "Force a fresh image build (use after a spwn binary upgrade)")
	restartCmd.Flags().BoolVar(&startRebuild, "rebuild", false, "Force a fresh image build before restart")

	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 50, "Number of lines from the tail (0 = all)")

	Cmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd, logsCmd)
}
