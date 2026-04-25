package auth

import (
	"fmt"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	"spwn.sh/packages/auth/daemon"
)

// daemonCmd is the parent for `spwn auth daemon …`. The bare form
// renders cobra's help — concrete actions live in the subcommands.
//
// Why a separate command group rather than top-level: keeping it
// under `auth` keeps the surface area narrow ("everything about
// credentials lives here") and lets future credential-management
// daemons (key rotation, vault sync, …) slot in without inventing
// new top-level verbs.
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Background MCP credential refresher (install/start/stop/status/uninstall/run)",
	Long: `Background MCP credential refresher.

Wires the spwn binary into your OS init system (launchd on macOS,
systemd-user on Linux, SCM on Windows) so OAuth tokens for MCP
providers (Notion, …) get refreshed periodically — even when no
spwn command is running and no agent is alive.

Without this daemon, tokens still get refreshed on every spwn up /
spwn agent talk / spwn auth event. The daemon plugs the gap for
long-running agent sessions and idle desktops.`,
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Register the refresh daemon with your OS init system and start it",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		if err := s.Install(); err != nil {
			return fmt.Errorf("install: %w", err)
		}
		if err := s.Start(); err != nil {
			// Install succeeded; surface the start failure but don't
			// roll back — the user can `spwn auth daemon start` again
			// once the underlying issue is fixed.
			return fmt.Errorf("installed, but failed to start: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ refresh daemon installed and started")
		fmt.Fprintln(cmd.OutOrStdout(), "    runs every 15 minutes; logs via your OS init system")
		fmt.Fprintln(cmd.OutOrStdout(), "    macOS: ~/Library/Logs/spwn-auth-refresh.log (when configured)")
		fmt.Fprintln(cmd.OutOrStdout(), "    linux: journalctl --user -u sh.spwn.auth-refresh")
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop and unregister the refresh daemon",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		// Stop is best-effort — Uninstall handles the not-running case.
		_ = s.Stop()
		if err := s.Uninstall(); err != nil {
			return fmt.Errorf("uninstall: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ refresh daemon uninstalled")
		return nil
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start an installed refresh daemon",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		if err := s.Start(); err != nil {
			return fmt.Errorf("start: %w (try `spwn auth daemon install` first)", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ refresh daemon started")
		return nil
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running refresh daemon (leaves it installed)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		if err := s.Stop(); err != nil {
			return fmt.Errorf("stop: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ refresh daemon stopped")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the refresh daemon is installed and running",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		st, err := s.Status()
		if err != nil {
			// kardianos returns ErrNotInstalled here on the major
			// platforms — translate to a friendlier message.
			fmt.Fprintln(cmd.OutOrStdout(), "  · not installed   ·  spwn auth daemon install")
			return nil
		}
		switch st {
		case service.StatusRunning:
			fmt.Fprintln(cmd.OutOrStdout(), "  ✓ running")
		case service.StatusStopped:
			fmt.Fprintln(cmd.OutOrStdout(), "  · installed, stopped   ·  spwn auth daemon start")
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "  ? unknown state")
		}
		return nil
	},
}

// daemonRunCmd is what the OS init system actually invokes. Hidden
// from help because users almost never call it directly — they use
// install/start/stop. When invoked from a terminal it still works:
// it blocks running the ticker until Ctrl+C, useful for foreground
// debugging.
var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the refresh ticker in the foreground (used by the OS init system)",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		s, _, err := daemon.New(0)
		if err != nil {
			return err
		}
		// service.Run blocks until the process is signaled (SIGTERM
		// from the init system, Ctrl+C from a terminal). It calls
		// Program.Start internally, then Program.Stop on shutdown.
		return s.Run()
	},
}

func init() {
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRunCmd)
	Cmd.AddCommand(daemonCmd)
}
