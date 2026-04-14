// Package tool implements the `spwn tool` command group - management of
// reusable tool packs (e.g. @spwn/unix, @spwn/python). Tool packs are
// first-class composable blocks that agents stack into their compositions.
//
// Tools are stubs for now - the full implementation requires a tool
// registry. Planned for a future release.
package tool

import (
	"fmt"

	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

// Cmd is the root `spwn tool` command group.
var Cmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage reusable tool packs (e.g. @spwn/unix, @spwn/python)",
	Long: `Tool packs are composable building blocks that agents plug into their worlds.

Attach one to an agent with:
  spwn agent add <agent> --tool <pack>

If a tool isn't in an agent's composition, it's physically absent from that
agent's world - not forbidden, absent.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(publishCmd)

	Cmd.SetHelpFunc(toolHelp)
}

func toolHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "tool" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ tool")+" "+ui.Faint("- reusable tool packs for agents"),
		[]ui.HelpGroup{
			{Title: "Manage", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List installed tool packs"},
				{Name: "show <pack>", Desc: "Inspect a tool pack"},
				{Name: "rm <pack>", Desc: "Uninstall a tool pack"},
			}},
			{Title: "Registry", Commands: []ui.HelpEntry{
				{Name: "search <query>", Desc: "Search the registry " + ui.Faint("[planned]")},
				{Name: "get <pack>", Desc: "Install a shared pack " + ui.Faint("[planned]")},
				{Name: "publish <path>", Desc: "Publish a pack " + ui.Faint("[planned]")},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn tool ls", Desc: "See every built-in pack"},
				{Name: "spwn agent add neo --tool @spwn/python", Desc: ""},
			}},
		},
		"spwn tool [command]",
		"",
	)
}

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List installed tool packs",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStderr(), "Built-in tool packs:")
		for _, t := range []struct{ name, desc string }{
			{"@spwn/unix", "bash, grep, sed, awk, find, xargs, curl, wget"},
			{"@spwn/git", "git"},
			{"@spwn/node", "node, npm, npx"},
			{"@spwn/python", "python3, pip3"},
			{"@spwn/build", "make, gcc, g++"},
		} {
			fmt.Fprintf(cmd.OutOrStderr(), "  %-16s  %s\n", t.name, t.desc)
		}
		fmt.Fprintln(cmd.OutOrStderr())
		fmt.Fprintln(cmd.OutOrStderr(), "Registry-backed listings are planned for a future release.")
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <tool-pack>",
	Short: "Inspect a tool pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "tool %q: inspection not yet wired.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "Built-in packs are listed with 'spwn tool ls'.")
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the tool registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "search %q: the tool registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <tool-pack>",
	Short: "Install a tool pack from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "install %q: the tool registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "Built-in packs (@spwn/*) are always available - no install needed.")
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:     "rm <tool-pack>",
	Aliases: []string{"remove", "uninstall"},
	Short:   "Remove an installed tool pack",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "rm %q: no tools are yet installed locally.\n", args[0])
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish <path>",
	Short: "Publish a tool pack to the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "publish %q: the tool registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}
