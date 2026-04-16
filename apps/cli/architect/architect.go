package architect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	arch "spwn.sh/packages/architect"
	"spwn.sh/packages/platform"
	"github.com/spf13/cobra"
)

var defaultArchitectHelp func(*cobra.Command, []string)

// Cmd is the parent command for Architect operations.
var Cmd = &cobra.Command{
	Use:   "architect",
	Short: "Your always-on world builder",
	Long:  `The Architect is your always-on world builder. It manages worlds and orchestrates artificial life.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Architect daemon",
	Long: `Start the Architect daemon in a Docker container.

The Architect runs the spwn binary inside a long-lived container with the
host's Docker socket mounted (DooD - Docker-outside-of-Docker), allowing it
to create and manage world containers as siblings.

The container mounts:
  /var/run/docker.sock    Docker daemon access (sibling containers, not nested)
  ~/.spwn/                Shared configuration and state

The Architect's identity is defined in /world/ARCHITECT.md inside the container,
which describes its capabilities and role as the always-on world builder.`,
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
	Short: "Show Architect status - worlds, agents",
	Long:  `Query Docker to show whether the Architect container is running, its uptime, and active worlds.`,
	RunE:  runStatus,
}

var talkCmd = &cobra.Command{
	Use:   "talk [message]",
	Short: "Talk to the Architect - ask it to manage worlds and agents",
	Long: `Send a message to the Architect (Claude Code running inside Docker).

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive Claude session.

Examples:
  spwn architect talk "list all agents"
  spwn architect talk "create a new agent called neo"
  spwn architect talk                    # interactive mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTalk,
}

var talkOutputFormat string

func init() {
	defaultArchitectHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(architectHelp)

	talkCmd.Flags().StringVar(&talkOutputFormat, "output-format", "", "Output format: text (default) or stream-json")
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(talkCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := ui.New()

	s.Blank()
	s.Start("Starting Architect (building image if needed)...")

	// Resolve image (allow override via env var for testing)
	imageOverride := os.Getenv("SPWN_ARCHITECT_IMAGE")

	// Friendly labels for the stable spawn events. Anything not in
	// this map is shown verbatim - keeps unknown future events
	// readable rather than crashing the stepper.
	labels := map[string]string{
		"docker_check":       "Connecting to Docker",
		"cleanup":            "Cleaning up old container",
		"image_resolve":      "Resolving image",
		"image_building":     "Building architect image",
		"image_ready":        "Image ready",
		"credentials_sync":   "Syncing credentials",
		"host_files":         "Preparing host files",
		"container_creating": "Creating container",
		"container_starting": "Starting container",
		"ready":              "Architect is ready",
	}

	id, err := arch.StartDaemonWithOpts(ctx, arch.StartDaemonOpts{
		ImageOverride: imageOverride,
		LogWriter:     cmd.ErrOrStderr(),
		OnProgress: func(event, detail string) {
			label, ok := labels[event]
			if !ok {
				label = event
			}
			s.Info(label, detail)
		},
	})
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
		case strings.Contains(msg, "source tree"):
			return s.FailHint("Image build", err,
				"Run from the spwn source directory, or build manually: make build-architect-image")
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

	s.Blank()
	fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Status: spwn architect status"))
	fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Stop:   spwn architect stop"))

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := ui.New()

	s.Blank()
	s.Start("Stopping Architect...")

	err := arch.StopDaemon(ctx)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not running"):
			s.Blank()
			s.Info("Architect:", "not running")
			s.Blank()
			return nil
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
	s := ui.New()

	info, err := arch.GetDaemonStatus(ctx)
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
		s.Info("Org:", info.OrgName)
	}

	s.Blank()

	return nil
}

func runTalk(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := ui.New()

	// Check if architect container is running. If not, auto-start it.
	info, err := arch.GetDaemonStatus(ctx)
	if err != nil || !info.Running {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  Architect not running. Starting...\n")
		imageOverride := os.Getenv("SPWN_ARCHITECT_IMAGE")
		_, startErr := arch.StartDaemon(ctx, imageOverride, cmd.ErrOrStderr())
		if startErr != nil {
			if strings.Contains(startErr.Error(), "already running") {
				// Race condition - it started between check and start, that's fine
			} else {
				return s.FailHint("Architect", fmt.Errorf("failed to auto-start architect: %w", startErr),
					"Start manually with: spwn architect start")
			}
		}
		time.Sleep(2 * time.Second) // Wait for container to be ready
		fmt.Fprintf(cmd.ErrOrStderr(), "  ✓ Architect started\n\n")
	}

	message := ""
	if len(args) > 0 {
		message = args[0]
	}

	// Check the container exists and is responsive
	checkCmd := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", platform.ArchitectContainerName())
	checkOut, err := checkCmd.Output()
	if err != nil || strings.TrimSpace(string(checkOut)) != "true" {
		return s.FailHint("Container", fmt.Errorf("architect container is not running"),
			"Start it with: spwn architect start")
	}

	// Get docker exec args from universe package
	dockerArgs, err := arch.TalkExecArgs(message)
	if err != nil {
		return s.FailHint("Talk", err, "")
	}

	// If stream-json output requested, swap --print for --output-format stream-json --verbose
	if talkOutputFormat == "stream-json" && message != "" {
		// Replace --print with --output-format stream-json --verbose in the args
		filtered := make([]string, 0, len(dockerArgs))
		for _, a := range dockerArgs {
			if a == "--print" {
				continue
			}
			filtered = append(filtered, a)
		}
		filtered = append(filtered, "--output-format", "stream-json", "--verbose")
		dockerArgs = filtered
	}

	if message != "" {
		if talkOutputFormat == "stream-json" {
			// Stream mode: pipe stdout directly for real-time output
			execCmd := exec.Command("docker", dockerArgs...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("architect exec failed: %w", err)
			}
		} else {
			// One-shot text mode
			s.Blank()
			s.Info("Architect:", "thinking...")
			s.Blank()

			execCmd := exec.Command("docker", dockerArgs...)
			output, err := execCmd.CombinedOutput()
			if err != nil {
				if len(output) > 0 {
					fmt.Fprint(os.Stdout, string(output))
				}
				return fmt.Errorf("architect exec failed: %w", err)
			}
			fmt.Fprint(os.Stdout, string(output))
		}
	} else {
		// Interactive mode
		s.Blank()
		s.Info("Architect:", "entering interactive session")
		s.Info("Container:", platform.ArchitectContainerName())
		s.Blank()

		execCmd := exec.Command("docker", dockerArgs...)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("interactive session: %w", err)
		}
	}

	return nil
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
		ui.Strong("⬡ architect")+" "+ui.Faint("- your always-on world builder"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "start", Desc: "Start the Architect daemon"},
				{Name: "stop", Desc: "Stop the Architect daemon"},
				{Name: "status", Desc: "Show status and active worlds"},
				{Name: "talk", Desc: "Talk to the Architect (Claude Code)"},
				{Name: "logs", Desc: "Show the Architect's event log"},
			}},
		},
		"spwn architect [command]",
		"Use \"spwn architect <command> --help\" for more information.",
	)
}
