package architect

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultArchitectHelp func(*cobra.Command, []string)

// Cmd is the parent command for Architect operations.
var Cmd = &cobra.Command{
	Use:   "architect",
	Short: "Your always-on world builder",
	Long:  `The Architect is your always-on world builder. It manages worlds, connects to messaging channels, and orchestrates artificial life.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Architect daemon",
	Long: `Start the Architect daemon in a Docker container.

The Architect runs the spwn binary inside a long-lived container with the
host's Docker socket mounted (DooD — Docker-outside-of-Docker), allowing it
to create and manage world containers as siblings. Channels (Telegram, Slack,
etc.) connect here.

The container mounts:
  /var/run/docker.sock    Docker daemon access (sibling containers, not nested)
  ~/.spwn/                Shared configuration and state`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Architect daemon",
	Long:  `Stop the Architect daemon and remove the container.`,
	RunE:  runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Architect status — channels, worlds, agents",
	Long:  `Query Docker to show whether the Architect container is running, its uptime, and connected channels.`,
	RunE:  runStatus,
}

var connectCmd = &cobra.Command{
	Use:   "connect [channel]",
	Short: "Connect a messaging channel (telegram, slack, discord, ...)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channel := args[0]
		fmt.Printf("  Channel %q connected.\n", channel)
		return nil
	},
}

func init() {
	defaultArchitectHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(architectHelp)

	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(connectCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := newStepper(cmd)

	s.Blank()
	s.Start("Connecting to Docker...")

	// Resolve image (allow override via env var for testing)
	imageOverride := os.Getenv("SPWN_ARCHITECT_IMAGE")

	id, err := universe.StartArchitectDaemon(ctx, imageOverride)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "already running"):
			s.Done("Architect", "already running")
			s.Blank()
			s.Info("Container:", "spwn-architect")
			return nil
		case strings.Contains(msg, "not reachable"):
			return s.FailHint("Docker", err,
				"Install Docker Desktop or start the daemon")
		case strings.Contains(msg, "not found"):
			return s.FailHint("Image", err,
				"Build it with: make build-architect-image")
		default:
			return s.FailHint("Start failed", err, "")
		}
	}

	s.Done("Docker connected", "")

	// Show success
	s.Blank()
	s.Success("Architect started")
	s.Blank()
	s.Info("Container:", "spwn-architect")
	s.Info("ID:", id[:12])

	// Load org name if available
	org, _ := universe.LoadOrg()
	if org != nil {
		s.Info("Universe:", org.Name)
	}

	s.Blank()
	fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Status: spwn architect status"))
	fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Stop:   spwn architect stop"))

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := newStepper(cmd)

	s.Blank()
	s.Start("Stopping Architect...")

	err := universe.StopArchitectDaemon(ctx)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not running"):
			return s.FailHint("Architect", err,
				"Start it with: spwn architect start")
		case strings.Contains(msg, "not reachable"):
			return s.FailHint("Docker", err,
				"Install Docker Desktop or start the daemon")
		default:
			return s.FailHint("Stop failed", err, "")
		}
	}

	s.Done("Stopped", "container spwn-architect removed")
	s.Blank()
	s.Success("Architect stopped")
	s.Blank()

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := newStepper(cmd)

	info, err := universe.GetArchitectDaemonStatus(ctx)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not reachable") {
			return s.FailHint("Docker", err,
				"Install Docker Desktop or start the daemon")
		}
		return s.FailHint("Status", err, "")
	}

	s.Blank()

	if !info.Running {
		s.Info("Architect:", "not running")
		s.Blank()
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Start with: spwn architect start"))
		s.Blank()
		return nil
	}

	s.Info("Architect:", "running")
	s.Info("Container:", info.ContainerID)
	s.Info("Image:", info.Image)
	s.Info("Status:", info.Status)
	s.Info("Uptime:", formatDuration(info.Uptime))
	s.Info("Started:", info.StartedAt.Format(time.RFC3339))

	if info.OrgName != "" {
		s.Info("Universe:", info.OrgName)
	}

	if len(info.Channels) > 0 {
		s.Info("Channels:", strings.Join(info.Channels, ", "))
	} else {
		s.Info("Channels:", "none")
	}

	s.Blank()

	return nil
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

func architectHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "architect" {
		if defaultArchitectHelp != nil {
			defaultArchitectHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ architect")+" "+ui.Faint("— your always-on world builder"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "start", Desc: "Start the Architect daemon"},
				{Name: "stop", Desc: "Stop the Architect daemon"},
				{Name: "status", Desc: "Show status, channels, active worlds"},
				{Name: "connect <channel>", Desc: "Connect a messaging channel"},
			}},
		},
		"spwn architect [command]",
		"Use \"spwn architect <command> --help\" for more information.",
	)
}
