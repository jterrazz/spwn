package cli

import (
	"fmt"
	"io"
	"os"

	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/architect"
	"spwn.sh/apps/cli/auth"
	"spwn.sh/apps/cli/example"
	"spwn.sh/apps/cli/get"
	"spwn.sh/apps/cli/logs"
	"spwn.sh/apps/cli/organization"
	"spwn.sh/apps/cli/profile"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/snap"
	"spwn.sh/apps/cli/team"
	"spwn.sh/apps/cli/tool"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/apps/cli/web"
	"spwn.sh/apps/cli/world"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

// Version is set by goreleaser via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn — create realities for things that can think",
	Long: `spwn creates isolated Docker environments for AI agents.
Each world has physics (constants, laws, tools),
and a Mind (persistent agent identity).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isArchitectMode() {
			return validateArchitectCommand(cmd)
		}
		startVersionCheck()
		if err := runMigrations(); err != nil {
			return err
		}
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
	rootCmd.SetHelpFunc(customHelp)

	// Top-level aliases — shortcuts for the 80% cases
	rootCmd.AddCommand(world.UpCmd)      // spwn up
	rootCmd.AddCommand(world.LsCmd)      // spwn ls
	rootCmd.AddCommand(agent.TalkTopCmd) // spwn talk

	// Additional top-level shortcuts
	rootCmd.AddCommand(world.DownCmd)

	// Command groups — entities
	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)

	// Command groups — building blocks
	rootCmd.AddCommand(tool.Cmd)
	rootCmd.AddCommand(skill.Cmd)
	rootCmd.AddCommand(profile.Cmd)

	// Command groups — coordination
	rootCmd.AddCommand(snap.Cmd)
	rootCmd.AddCommand(team.Cmd)
	rootCmd.AddCommand(organization.Cmd)

	// Command groups — system
	rootCmd.AddCommand(auth.Cmd)
	rootCmd.AddCommand(architect.Cmd)
	rootCmd.AddCommand(example.Cmd)
	rootCmd.AddCommand(web.Cmd)
	rootCmd.AddCommand(get.Cmd)
	rootCmd.AddCommand(logs.Cmd)
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

// printHelpCmd prints a command entry indented one level under its section
// header. If desc is empty, the name is printed without trailing padding.
func printHelpCmd(w io.Writer, name, desc string) {
	if desc == "" {
		fmt.Fprintf(w, "%s%s\n", ui.Indent, ui.ColorizeHelpName(name))
		return
	}
	fmt.Fprintf(w, "%s%s %s\n", ui.Indent, ui.PadVisible(ui.ColorizeHelpName(name), ui.HelpColWidth), ui.Faint(desc))
}

// customHelp renders grouped, structured help for the root command.
// For subcommands, it falls back to MinimalHelp (flush-left, no padding).
func customHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "spwn" {
		ui.MinimalHelp(cmd, args)
		return
	}

	w := cmd.OutOrStdout()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s %s\n", ui.Strong("⬡ spwn"), ui.Faint("— the building blocks of agent intelligence"))
	fmt.Fprintln(w)

	// Quick Start — the 30-second path
	fmt.Fprintf(w, "%s\n", ui.Strong("Quick Start:"))
	printHelpCmd(w, "spwn agent new neo", "Create an agent")
	printHelpCmd(w, "spwn agent add neo --tool @spwn/python", "")
	printHelpCmd(w, "spwn up --agent neo -w .", "Spawn a world")
	printHelpCmd(w, "spwn talk neo", "Talk to it")
	fmt.Fprintln(w)

	// Entities — the things you create
	fmt.Fprintf(w, "%s\n", ui.Strong("Entities:"))
	printHelpCmd(w, "agent", "Composed minds "+ui.Faint("(new, ls, show, add, fork, dream, sleep)"))
	printHelpCmd(w, "world", "Runtime instances "+ui.Faint("(up, ls, show, down, enter, snap)"))
	fmt.Fprintln(w)

	// Building blocks — the things you compose agents from
	fmt.Fprintf(w, "%s\n", ui.Strong("Building blocks:"))
	printHelpCmd(w, "tool", "Reusable tool packs "+ui.Faint("(ls, show, install)"))
	printHelpCmd(w, "skill", "Reusable skill files "+ui.Faint("(ls, new, edit, show)"))
	printHelpCmd(w, "profile", "Reusable personality templates "+ui.Faint("(ls, new, edit)"))
	fmt.Fprintln(w)

	// Shortcuts — the 80% top-level aliases
	fmt.Fprintf(w, "%s\n", ui.Strong("Shortcuts:"))
	printHelpCmd(w, "up", "Spawn a world "+ui.Faint("(alias: world up)"))
	printHelpCmd(w, "ls", "List active worlds "+ui.Faint("(alias: world ls)"))
	printHelpCmd(w, "talk <name>", "Talk to a running agent "+ui.Faint("(alias: agent talk)"))
	printHelpCmd(w, "down <id>", "Destroy a world")
	fmt.Fprintln(w)

	// Coordination — multi-agent + orchestration
	fmt.Fprintf(w, "%s\n", ui.Strong("Coordination:"))
	printHelpCmd(w, "architect", "Always-on orchestration daemon")
	printHelpCmd(w, "web", "Open the local web UI")
	printHelpCmd(w, "logs", "System event log "+ui.Faint("(--world, --agent, --type)"))
	fmt.Fprintln(w)

	// System
	fmt.Fprintf(w, "%s\n", ui.Strong("System:"))
	printHelpCmd(w, "init", "Create default configs")
	printHelpCmd(w, "status", "Show running state")
	printHelpCmd(w, "auth", "Manage credentials")
	printHelpCmd(w, "upgrade", "Update the CLI")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Faint("Run \"spwn <command> --help\" for details."))
	fmt.Fprintln(w)
}
