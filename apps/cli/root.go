package cli

import (
	"fmt"
	"io"
	"os"

	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/auth"
	"spwn.sh/apps/cli/architect"
	"spwn.sh/apps/cli/msg"
	"spwn.sh/apps/cli/observatory"
	"spwn.sh/apps/cli/profile"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/snap"
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
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isGodMode() {
			return validateGodCommand(cmd)
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
	rootCmd.AddCommand(observatory.Cmd)
	rootCmd.AddCommand(skill.Cmd)
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
	printHelpCmd(w, "spwn init", "First-time setup")
	printHelpCmd(w, "spwn up --agent neo -w .", "Spawn a world with an agent")
	printHelpCmd(w, "spwn agent talk neo", "Talk to the agent")
	fmt.Fprintln(w)

	// World shortcuts
	fmt.Fprintf(w, "  %s\n", ui.Strong("World:"))
	printHelpCmd(w, "up", "Spawn a world "+ui.Faint("(alias: world)"))
	printHelpCmd(w, "down <id>", "Destroy a world")
	printHelpCmd(w, "ls", "List active worlds")
	printHelpCmd(w, "logs <id>", "Stream agent output")
	printHelpCmd(w, "attach <id>", "Open interactive shell")
	printHelpCmd(w, "inspect <id>", "Show world details")
	fmt.Fprintln(w)

	// Command groups
	fmt.Fprintf(w, "  %s\n", ui.Strong("Agents & Profiles:"))
	printHelpCmd(w, "agent", "Create and manage agents "+ui.Faint("(new, ls, rm, talk, fork, export)"))
	printHelpCmd(w, "profile <name>", "View/edit agent profile "+ui.Faint("(purpose, traits, skills, journal)"))
	printHelpCmd(w, "msg", "Agent messaging "+ui.Faint("(send, inbox, watch)"))
	printHelpCmd(w, "snap", "World snapshots "+ui.Faint("(save, ls, restore, rm)"))
	fmt.Fprintln(w)

	// Platform
	fmt.Fprintf(w, "  %s\n", ui.Strong("Platform:"))
	printHelpCmd(w, "architect", "Orchestration daemon "+ui.Faint("(start, stop, status, connect)"))
	printHelpCmd(w, "observatory", "Visual dashboard")
	printHelpCmd(w, "skill", "Manage agent skills "+ui.Faint("(list, install, remove)"))
	fmt.Fprintln(w)

	// System
	fmt.Fprintf(w, "  %s\n", ui.Strong("System:"))
	printHelpCmd(w, "init", "First-time setup")
	printHelpCmd(w, "status", "Environment overview")
	printHelpCmd(w, "auth", "Manage credentials "+ui.Faint("(login, logout, token)"))
	printHelpCmd(w, "doctor", "Diagnose issues")
	printHelpCmd(w, "upgrade", "Upgrade to latest version")
	fmt.Fprintln(w)

	// Flags
	fmt.Fprintf(w, "  %s\n", ui.Strong("Flags:"))
	printHelpFlag(w, "--json", "Output as JSON")
	printHelpFlag(w, "-q, --quiet", "Suppress output")
	printHelpFlag(w, "-v, --verbose", "Debug info")
	printHelpFlag(w, "--version", "Show version")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %s\n", ui.Faint("Use \"spwn <command> --help\" for more information about a command."))
	fmt.Fprintln(w)
}
