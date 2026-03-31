package cli

import (
	"fmt"
	"io"

	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/claw"
	"spwn.sh/apps/cli/observatory"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/apps/cli/world"
	"github.com/spf13/cobra"
)

// Version is set by goreleaser via ldflags.
var Version = "dev"

var (
	jsonOutput bool
	quiet      bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn — create realities for things that can think",
	Long: `spwn creates isolated Docker environments for AI agents.
Each world has physics (constants, laws, elements),
and a Mind (persistent agent identity).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureDefaults()
	},
}

func init() {
	rootCmd.Version = Version
	defaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(customHelp)

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)
	rootCmd.AddCommand(claw.Cmd)
	rootCmd.AddCommand(observatory.Cmd)
	rootCmd.AddCommand(skill.Cmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for documentation generation.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// printHelpCmd prints a command name and description in the help output.
func printHelpCmd(w io.Writer, name, desc string) {
	fmt.Fprintf(w, "    %-28s %s\n", name, ui.Faint(desc))
}

// defaultHelpFunc stores Cobra's original help function so subcommands
// can fall back to it.
var defaultHelpFunc func(*cobra.Command, []string)

// customHelp renders grouped, structured help for the root command.
// For subcommands, it falls back to Cobra's default help.
func customHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "spwn" {
		if defaultHelpFunc != nil {
			defaultHelpFunc(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s %s\n", ui.Strong("⬡ spwn"), ui.Faint("— the framework for artificial life"))
	fmt.Fprintln(w)

	// Quick Start
	fmt.Fprintf(w, "  %s\n", ui.Strong("Quick Start:"))
	printHelpCmd(w, "spwn init", "First-time setup")
	printHelpCmd(w, "spwn agent init neo", "Create an agent")
	printHelpCmd(w, "spwn world --agent neo -w .", "Spawn a world")
	printHelpCmd(w, "spwn agent talk neo", "Talk to the agent")
	fmt.Fprintln(w)

	// World
	fmt.Fprintf(w, "  %s\n", ui.Strong("World:"))
	printHelpCmd(w, "world", "Spawn a world")
	printHelpCmd(w, "world list", "List active worlds")
	printHelpCmd(w, "world inspect <id>", "Show world details")
	printHelpCmd(w, "world destroy <id>", "Destroy a world")
	printHelpCmd(w, "world logs <id>", "Stream agent output")
	printHelpCmd(w, "world attach <id>", "Interactive shell")
	printHelpCmd(w, "world snapshot <id>", "Save world state")
	printHelpCmd(w, "world snapshots", "List snapshots")
	printHelpCmd(w, "world restore <snap>", "Restore from snapshot")
	printHelpCmd(w, "world send <id>", "Send message between agents")
	printHelpCmd(w, "world inbox <id>", "Show messages")
	printHelpCmd(w, "world watch <id>", "Watch for new messages")
	fmt.Fprintln(w)

	// Agent
	fmt.Fprintf(w, "  %s\n", ui.Strong("Agent:"))
	printHelpCmd(w, "agent", "Spawn an agent into a world")
	printHelpCmd(w, "agent init <name>", "Create a new agent")
	printHelpCmd(w, "agent list", "List all agents")
	printHelpCmd(w, "agent inspect <name>", "Show agent details")
	printHelpCmd(w, "agent talk <name>", "Talk to a running agent")
	printHelpCmd(w, "agent delete <name>", "Remove an agent")
	printHelpCmd(w, "agent export <name>", "Export as tar.gz")
	printHelpCmd(w, "agent fork <src> <dst>", "Clone an agent")
	printHelpCmd(w, "agent journal <name>", "View session history")
	printHelpCmd(w, "agent sessions <name>", "View saved sessions")
	printHelpCmd(w, "agent mind <name>", "Show Mind directory tree")
	printHelpCmd(w, "agent stats <name>", "Show agent statistics")
	printHelpCmd(w, "agent reflect <name>", "Analyze journal")
	printHelpCmd(w, "agent sleep <name>", "Archive stale knowledge")
	fmt.Fprintln(w)

	// System
	fmt.Fprintf(w, "  %s\n", ui.Strong("System:"))
	printHelpCmd(w, "init", "First-time setup")
	printHelpCmd(w, "status", "Full environment overview")
	printHelpCmd(w, "doctor", "Diagnose environment issues")
	printHelpCmd(w, "upgrade", "Upgrade to latest version")
	printHelpCmd(w, "claw", "Orchestration daemon")
	printHelpCmd(w, "observatory", "Visual dashboard")
	printHelpCmd(w, "skill", "Manage skills")
	fmt.Fprintln(w)

	// Flags
	fmt.Fprintf(w, "  %s\n", ui.Strong("Flags:"))
	printHelpCmd(w, "--json", "Output as JSON")
	printHelpCmd(w, "-q, --quiet", "Suppress output")
	printHelpCmd(w, "-v, --verbose", "Debug info")
	printHelpCmd(w, "--version", "Show version")
	fmt.Fprintln(w)
}
