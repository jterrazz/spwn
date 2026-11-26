package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/authentication"
	"spwn.sh/apps/cli/automation"
	"spwn.sh/apps/cli/gate"
	"spwn.sh/apps/cli/logs"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/apps/cli/world"
	"spwn.sh/packages/upgrade"
)

// Version is set by goreleaser via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn - run AI agents in isolated worlds you declare",
	Long: `The spwn command is Agents, the Spwn app that runs one agent, or a
team of agents, in isolated Docker worlds. Each world has its own
rules (network, filesystem, tools), and each agent keeps its Mind
(its identity and memory) from one world to the next.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrations(); err != nil {
			return err
		}
		discoverProject()
		return nil
	},
}

func init() {
	rootCmd.Version = Version
	upgrade.CLIVersion = Version
	rootCmd.SetHelpFunc(customHelp)

	rootCmd.AddCommand(world.UpCmd)
	rootCmd.AddCommand(world.DownCmd)

	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)

	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(uninstallCmd())
	rootCmd.AddCommand(skill.Cmd)

	rootCmd.AddCommand(automation.Cmd)

	rootCmd.AddCommand(authentication.Cmd)
	rootCmd.AddCommand(gate.Cmd)
	rootCmd.AddCommand(logs.Cmd)
}

// Execute runs the root command.
//
// Wraps cobra's lifecycle so the upgrade-banner middleware always
// Fires even on error returns, subcommand-help short-circuits, or
// Handlers that os.Exit without returning. The pre-hook kicks off the
// Version check (synchronous cache read + async refresh); the deferred
// Post-hook renders the banner, and skips it when the user just ran
// `spwn upgrade` (the upgrade command prints its own progress, stacking
// A banner on top would be redundant noise).
func Execute() error {
	startVersionCheck()
	defer func() {
		if ranUpgradeCommand() {
			return
		}
		printUpgradeHint()
	}()

	err := rootCmd.Execute()
	if err != nil {
		var displayed *ui.DisplayedError
		var coded ExitCoder
		if !errors.As(err, &displayed) && !errors.As(err, &coded) {
			fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", ui.Red("Error:"), err)
		}
	}
	return err
}

// ExitCoder is implemented by spwn-side errors that want to set a
// non-default process exit code. The marker method distinguishes
// spwn's own wrappers from stdlib errors that happen to expose an
// ExitCode(), notably os/exec's *ExitError, which implements
// ExitCode via *os.ProcessState. Without the marker, any docker/
// subcommand failure would be treated as a "silent" spwn-managed
// exit and the Execute() error banner would be suppressed, hiding
// real failures from the user.
type ExitCoder interface {
	ExitCode() int
	IsSpwnExitCoder()
}

// GetRootCmd returns the root command for documentation generation.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func printHelpCmd(w io.Writer, name, desc string) {
	if desc == "" {
		fmt.Fprintf(w, "%s%s\n", ui.Indent, ui.ColorizeHelpName(name))
		return
	}
	fmt.Fprintf(w, "%s%s %s\n", ui.Indent, ui.PadVisible(ui.ColorizeHelpName(name), ui.HelpColWidth), ui.Faint(desc))
}

func customHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "spwn" {
		ui.MinimalHelp(cmd, args)
		return
	}

	w := cmd.OutOrStdout()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s %s\n", ui.Strong("⬡ spwn"), ui.Faint("- the building blocks of agent intelligence"))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("Quick Start:"))
	printHelpCmd(w, "spwn init", "Scaffold a new project")
	printHelpCmd(w, "station agents create neo", "Create an agent")
	printHelpCmd(w, "spwn up", "Bring up every world in spwn.yaml")
	printHelpCmd(w, "spwn agent neo", "Start the world that contains neo")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("Entities:"))
	printHelpCmd(w, "agent", "Composed minds "+ui.Faint("(talk, logs)"))
	printHelpCmd(w, "world", "Deployable groupings "+ui.Faint("(list, inspect, enter)"))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("Building blocks:"))
	printHelpCmd(w, "install <ref>", "Install a dependency "+ui.Faint("(spwn:unix, github.com/...)"))
	printHelpCmd(w, "uninstall <ref>", "Remove a dependency")
	printHelpCmd(w, "skill", "Author skill files "+ui.Faint("(new, edit, show, rm, ls)"))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("Worlds:"))
	printHelpCmd(w, "up [name]", "Start every world (or one)")
	printHelpCmd(w, "down [name]", "Stop every world (or one)")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("Coordination:"))
	printHelpCmd(w, "logs", "System event log "+ui.Faint("(--world, --agent, --type)"))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Strong("System:"))
	printHelpCmd(w, "init", "Create default configs")
	printHelpCmd(w, "auth", "Manage credentials")
	printHelpCmd(w, "upgrade", "Update the CLI")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%s\n", ui.Faint("Run \"spwn <command> --help\" for details."))
	fmt.Fprintln(w)
}
