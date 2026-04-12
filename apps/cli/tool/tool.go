// Package tool implements the `spwn tool` command group — management of
// reusable tool packs (e.g. @spwn/unix, @spwn/python). Tool packs are
// first-class composable blocks that agents stack into their compositions.
//
// Tools are stubs for now — the full implementation requires a tool
// registry port. Coming in Epoch 10 (Marketplace).
package tool

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the root `spwn tool` command group.
var Cmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage reusable tool packs (e.g. @spwn/unix, @spwn/python)",
	Long: `Tool packs are composable building blocks that agents plug into their worlds.

Each tool pack bundles one or more binaries (bash, grep, python3, etc.) and any
accompanying skills that ship with the tool. If a tool isn't in an agent's
composition, it's physically absent from that agent's world.

Tools are installed from the registry and stacked into agents via
"spwn agent add <name> --tool <pack>".`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(publishCmd)
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
		fmt.Fprintln(cmd.OutOrStderr(), "Registry (remote) listings are not yet wired — coming in Epoch 10.")
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
		fmt.Fprintln(cmd.OutOrStderr(), "Registry port is planned for Epoch 10 (Marketplace).")
		return nil
	},
}

var installCmd = &cobra.Command{
	Use:   "install <tool-pack>",
	Short: "Install a tool pack from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "install %q: the tool registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "Built-in packs (@spwn/*) are always available — no install needed.")
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
		fmt.Fprintln(cmd.OutOrStderr(), "Registry port is planned for Epoch 10 (Marketplace).")
		return nil
	},
}
