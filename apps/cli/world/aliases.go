package world

import (
	"github.com/spf13/cobra"
)

// UpCmd is the top-level alias for `spwn world` (spawn a world).
var UpCmd = &cobra.Command{
	Use:     "up",
	Short:   "Spawn a world — an isolated reality for agents",
	Long:    Cmd.Long,
	Example: `  spwn up --agent neo -w .                  Single agent in current dir
  spwn up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn up -c acme --agent neo -w ~/project   Named config + workspace`,
	RunE: Cmd.RunE,
}

// DownCmd is the top-level alias for spwn down.
var DownCmd = &cobra.Command{
	Use:   "down [world-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
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

// InspectTopCmd is the top-level alias for spwn inspect (kept for back-compat).
var InspectTopCmd = &cobra.Command{
	Use:   "inspect <world-id>",
	Short: "Show world details, physics, and agent status",
	Args:  cobra.ExactArgs(1),
	RunE:  inspectCmd.RunE,
}

func init() {
	// Copy flags from world Cmd to UpCmd — stay in sync with world.go
	UpCmd.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	UpCmd.Flags().StringArrayVarP(&spawnAgents, "agent", "a", nil, "Agent name (repeatable; first agent becomes chief in multi-agent worlds)")
	UpCmd.Flags().StringArrayVarP(&spawnWorkspaces, "workspace", "w", nil, `Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.`)
	UpCmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	UpCmd.Flags().BoolVarP(&spawnInteractive, "interactive", "i", false, "Attach to agent interactively")
	UpCmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	UpCmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge tool from Host: "source:as:cap1,cap2"`)
	UpCmd.Flags().StringVar(&spawnRuntime, "runtime", "claude-code", "Agent runtime")

	// Copy --all flag for DownCmd
	DownCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")

	// Copy flags for logs
	LogsTopCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	LogsTopCmd.Flags().IntVarP(&logsTail, "n", "n", 100, "Number of lines to show")
}
