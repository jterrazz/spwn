package cli

import (
	"fmt"
	"io"
	"os"

	"spwn.sh/apps/cli/activity"
	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/architect"
	"spwn.sh/apps/cli/auth"
	"spwn.sh/apps/cli/dash"
	"spwn.sh/apps/cli/get"
	"spwn.sh/apps/cli/knowledge"
	"spwn.sh/apps/cli/msg"
	"spwn.sh/apps/cli/profile"
	"spwn.sh/apps/cli/snap"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/apps/cli/world"
	"spwn.sh/core/foundation"
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
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isArchitectMode() {
			return validateArchitectCommand(cmd)
		}
		startVersionCheck()
		return ensureDefaults()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		printUpgradeHint()
	},
}

func init() {
	rootCmd.Version = Version
	// Sync CLI version to the shared foundation package so observatory can use it
	foundation.Version = Version
	defaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(customHelp)

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	// Top-level aliases for world commands
	rootCmd.AddCommand(world.UpCmd)
	rootCmd.AddCommand(world.DownCmd)
	rootCmd.AddCommand(world.LsCmd)
	rootCmd.AddCommand(world.LogsTopCmd)
	rootCmd.AddCommand(world.AttachTopCmd)
	rootCmd.AddCommand(world.InspectTopCmd)

	// Command groups
	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)
	rootCmd.AddCommand(profile.Cmd)
	rootCmd.AddCommand(msg.Cmd)
	rootCmd.AddCommand(snap.Cmd)
	rootCmd.AddCommand(auth.Cmd)
	rootCmd.AddCommand(architect.Cmd)
	rootCmd.AddCommand(knowledge.Cmd)
	rootCmd.AddCommand(dash.Cmd)
	rootCmd.AddCommand(get.Cmd)
	rootCmd.AddCommand(activity.Cmd)
}

// Execute runs the root command.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		// Don't re-print errors already shown by a Stepper
		if _, ok := err.(*ui.DisplayedError); !ok {
			fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", ui.Red("Error:"), err)
		}
	}
	return err
}

// GetRootCmd returns the root command for documentation generation.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// printHelpCmd prints a command name and description in the help output.
func printHelpCmd(w io.Writer, name, desc string) {
	fmt.Fprintf(w, "    %s %s\n", ui.PadVisible(ui.ColorizeHelpName(name), 28), ui.Faint(desc))
}

// printHelpFlag prints a flag and description in the help output.
func printHelpFlag(w io.Writer, flag, desc string) {
	fmt.Fprintf(w, "    %s %s\n", ui.PadVisible(ui.Yellow(flag), 28), ui.Faint(desc))
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
	fmt.Fprintf(w, "  %s %s\n", ui.Strong("⬡ spwn"), ui.Faint("— the control plane for AI agents"))
	fmt.Fprintln(w)

	// Quick Start
	fmt.Fprintf(w, "  %s\n", ui.Strong("Quick Start:"))
	printHelpCmd(w, "spwn architect start", "Start the world builder daemon")
	printHelpCmd(w, "spwn agent new neo", "Create an agent")
	printHelpCmd(w, "spwn up --agent neo -w .", "Spawn a world")
	printHelpCmd(w, "spwn agent talk neo", "Talk to the agent")
	printHelpCmd(w, "spwn dash open", "Open the visual dashboard")
	fmt.Fprintln(w)

	// Orchestration
	fmt.Fprintf(w, "  %s\n", ui.Strong("Orchestration:"))
	printHelpCmd(w, "architect", "Your always-on world builder "+ui.Faint("(start, stop, status, connect)"))
	printHelpCmd(w, "knowledge", "Universe knowledge base "+ui.Faint("(ls, show, search)"))
	printHelpCmd(w, "dash", "Visual dashboard "+ui.Faint("(start, open)"))
	printHelpCmd(w, "activity", "View recent activity across worlds and agents")
	fmt.Fprintln(w)

	// World
	fmt.Fprintf(w, "  %s\n", ui.Strong("World:"))
	printHelpCmd(w, "up", "Spawn a world")
	printHelpCmd(w, "down <id>", "Destroy a world")
	printHelpCmd(w, "ls", "List active worlds")
	printHelpCmd(w, "logs <id>", "Stream agent output")
	printHelpCmd(w, "attach <id>", "Interactive shell")
	printHelpCmd(w, "inspect <id>", "World details and physics")
	printHelpCmd(w, "snap", "Snapshots "+ui.Faint("(save, ls, restore, rm)"))
	fmt.Fprintln(w)

	// Agent
	fmt.Fprintf(w, "  %s\n", ui.Strong("Agent:"))
	printHelpCmd(w, "agent", "Lifecycle "+ui.Faint("(new, ls, rm, talk, inspect, fork, export)"))
	printHelpCmd(w, "agent dream <name>", "Analyze experience, discover patterns, promote playbooks")
	printHelpCmd(w, "agent sleep <name>", "Shutdown — save state, consolidate, archive")
	printHelpCmd(w, "profile <name>", "Character sheet — the passport, not the person")
	printHelpCmd(w, "msg", "Messaging "+ui.Faint("(send, inbox, watch)"))
	fmt.Fprintln(w)

	// System
	fmt.Fprintf(w, "  %s\n", ui.Strong("System:"))
	printHelpCmd(w, "init · status · auth · doctor · upgrade", "")
	fmt.Fprintln(w)

	// Flags
	fmt.Fprintf(w, "  %s\n", ui.Strong("Flags:"))
	printHelpFlag(w, "--json · -q/--quiet · -v/--verbose · --version", "")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %s\n", ui.Faint("Use \"spwn <command> --help\" for more information about a command."))
	fmt.Fprintln(w)
}
