package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/architect"
	"spwn.sh/apps/cli/auth"
	"spwn.sh/apps/cli/inspect"
	"spwn.sh/apps/cli/logs"
	"spwn.sh/apps/cli/organization"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/snap"
	"spwn.sh/apps/cli/team"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/apps/cli/web"
	"spwn.sh/apps/cli/world"
	"spwn.sh/packages/update"
)

// Version is set by goreleaser via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn - create realities for things that can think",
	Long: `spwn creates isolated Docker environments for AI agents.
Each world has its own rules (network, filesystem, tools) and a
Mind (persistent agent identity).`,
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
		discoverProject()
		return ensureDefaults()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		printUpgradeHint()
	},
}

func init() {
	rootCmd.Version = Version
	// Sync CLI version to the shared base package so the web UI can use it
	update.CLIVersion = Version
	rootCmd.SetHelpFunc(customHelp)

	// Top-level world shortcuts.
	//
	// `spwn ls` is intentionally the agent-centric smart view when a
	// project is active — the legacy world-centric list is kept as
	// `spwn world ls`. The smart command's RunE already falls back
	// to the global view when no spwn.yaml is discovered, so we can
	// just point the top-level alias at it.
	rootCmd.AddCommand(world.UpCmd)
	rootCmd.AddCommand(smartLsCmd())
	rootCmd.AddCommand(world.DownCmd)

	// Command groups - entities
	world.Cmd.AddCommand(snap.Cmd) // snap lives under `spwn world snap`
	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)

	// Building blocks — install/uninstall at top level (like `go get`).
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(uninstallCmd())
	rootCmd.AddCommand(skill.Cmd)

	// Command groups - coordination
	rootCmd.AddCommand(team.Cmd)
	rootCmd.AddCommand(organization.Cmd)

	// Command groups - introspection
	rootCmd.AddCommand(inspect.Cmd)

	// Command groups - system
	rootCmd.AddCommand(auth.Cmd)
	rootCmd.AddCommand(architect.Cmd)
	rootCmd.AddCommand(web.Cmd)
	rootCmd.AddCommand(logs.Cmd)
}

// Execute runs the root command.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		// Don't re-print errors already shown by a Stepper — or
		// by a custom error type that rendered its own banner (e.g.
		// notImplementedError, which implements ExitCoder).
		var displayed *ui.DisplayedError
		var coded ExitCoder
		if !errors.As(err, &displayed) && !errors.As(err, &coded) {
			fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", ui.Red("Error:"), err)
		}
	}
	return err
}

// ExitCoder is implemented by errors that want to set a non-default
// process exit code. The spwn binary entry point inspects the returned
// error and, if it satisfies this interface, forwards ExitCode() to
// os.Exit. Unknown errors default to exit 1.
type ExitCoder interface {
	ExitCode() int
}

// GetRootCmd returns the root command for documentation generation.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// smartLsCmd wraps agent.LsCmd as a top-level `spwn ls` shortcut.
// We wrap rather than re-register the same *cobra.Command instance
// because cobra only allows a command to have a single parent.
func smartLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "Agent-centric status (running / stopped / orphan)",
		RunE:  agent.LsCmd.RunE,
	}
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
	fmt.Fprintf(w, "%s %s\n", ui.Strong("⬡ spwn"), ui.Faint("- the building blocks of agent intelligence"))
	fmt.Fprintln(w)

	// Quick Start - the 30-second path (new compose-style grammar)
	fmt.Fprintf(w, "%s\n", ui.Strong("Quick Start:"))
	printHelpCmd(w, "spwn init", "Scaffold a new project")
	printHelpCmd(w, "spwn agent new neo", "Create an agent (auto-creates a single-agent world)")
	printHelpCmd(w, "spwn up", "Bring up every world in spwn.yaml")
	printHelpCmd(w, "spwn agent neo", "Start the world that contains neo")
	fmt.Fprintln(w)

	// Entities - the things you create
	fmt.Fprintf(w, "%s\n", ui.Strong("Entities:"))
	printHelpCmd(w, "agent", "Composed minds "+ui.Faint("(new, ls, start, stop, inspect, add, fork, talk)"))
	printHelpCmd(w, "world", "Deployable groupings "+ui.Faint("(start, stop, ls, inspect, enter, snap)"))
	fmt.Fprintln(w)

	// Building blocks - the things you compose agents from
	fmt.Fprintf(w, "%s\n", ui.Strong("Building blocks:"))
	printHelpCmd(w, "install <ref>", "Install a dependency "+ui.Faint("(spwn:unix, github.com/...)"))
	printHelpCmd(w, "uninstall <ref>", "Remove a dependency")
	printHelpCmd(w, "skill", "Author skill files "+ui.Faint("(new, edit, show, rm, ls)"))
	fmt.Fprintln(w)

	// Shortcuts (compose-style: no-arg = all worlds in spwn.yaml)
	fmt.Fprintf(w, "%s\n", ui.Strong("Shortcuts:"))
	printHelpCmd(w, "up [name]", "Start every world (or one) "+ui.Faint("(alias: world start)"))
	printHelpCmd(w, "ls", "Agent-centric status "+ui.Faint("(running, deployed, orphan)"))
	printHelpCmd(w, "down [name]", "Stop every world (or one) "+ui.Faint("(alias: world stop)"))
	fmt.Fprintln(w)

	// Coordination - multi-agent + orchestration
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
