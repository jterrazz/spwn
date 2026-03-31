package world

import (
	"github.com/spf13/cobra"
)

// UpCmd is the top-level alias for `spwn world` (spawn a world).
var UpCmd = &cobra.Command{
	Use:     "up",
	Short:   "Spawn a world — an isolated reality for agents",
	Long:    Cmd.Long,
	Example: `  spwn up -w .                    Spawn with current directory
  spwn up -c acme -w ~/project   Named config + workspace
  spwn up --governor morpheus     With a governor agent`,
	RunE: Cmd.RunE,
}

// DownCmd is the top-level alias for spwn down.
var DownCmd = &cobra.Command{
	Use:   "down <world-id>",
	Short: "Destroy a world",
	Args:  cobra.ExactArgs(1),
	RunE:  destroyCmd.RunE,
}

// LsCmd is the top-level alias for spwn ls.
var LsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List active worlds",
	RunE:  listCmd.RunE,
}

// LogsTopCmd is the top-level alias for spwn logs.
var LogsTopCmd = &cobra.Command{
	Use:   "logs <world-id>",
	Short: "Stream agent output from a running world",
	Args:  cobra.ExactArgs(1),
	RunE:  logsCmd.RunE,
}

// AttachTopCmd is the top-level alias for spwn attach.
var AttachTopCmd = &cobra.Command{
	Use:   "attach <world-id>",
	Short: "Open interactive session into a running world",
	Args:  cobra.ExactArgs(1),
	RunE:  attachCmd.RunE,
}

// InspectTopCmd is the top-level alias for spwn inspect.
var InspectTopCmd = &cobra.Command{
	Use:   "inspect <world-id>",
	Short: "Show world details, physics, and agent status",
	Args:  cobra.ExactArgs(1),
	RunE:  inspectCmd.RunE,
}

func init() {
	// Copy flags from world Cmd to UpCmd
	UpCmd.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	UpCmd.Flags().StringVarP(&spawnAgent, "agent", "a", "default", "Agent name")
	UpCmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	UpCmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	UpCmd.Flags().BoolVarP(&spawnInteractive, "interactive", "i", false, "Attach to agent interactively")
	UpCmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	UpCmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge element from Host: "source:as:cap1,cap2"`)
	UpCmd.Flags().StringVar(&spawnGovernor, "governor", "", "Governor agent for this world")
	UpCmd.Flags().StringVar(&spawnRuntime, "runtime", "claude-code", "Agent runtime")

	// Copy flags for logs
	LogsTopCmd.Flags().BoolVar(&logsNoFollow, "no-follow", false, "Print current logs and exit")
	LogsTopCmd.Flags().IntVarP(&logsTail, "n", "n", 100, "Number of lines to show")
}
