package get

import (
	"fmt"

	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

var defaultGetHelp func(*cobra.Command, []string)

// Cmd is the parent command for marketplace package management.
var Cmd = &cobra.Command{
	Use:   "get",
	Short: "Install from the marketplace",
	Long:  `Install packages from the marketplace — skills, worlds, packs, and adapters.`,
}

var installCmd = &cobra.Command{
	Use:   "install [name]",
	Short: "Install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("  Installing package %q...\n", args[0])
		fmt.Println("  (Not yet implemented — coming in Epoch 10)")
		return nil
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  No packages installed.")
		fmt.Println("  Run 'spwn get install <name>' to add one.")
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("  Searching for %q...\n", args[0])
		fmt.Println("  (Not yet implemented — coming in Epoch 10)")
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Remove a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("  Removing package %q...\n", args[0])
		return nil
	},
}

func init() {
	defaultGetHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(getHelp)

	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(rmCmd)
}

func getHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "get" {
		if defaultGetHelp != nil {
			defaultGetHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ get")+" "+ui.Faint("— install from the marketplace"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{"install <name>", "Install a package"},
				{"ls", "List installed packages"},
				{"search <query>", "Search the marketplace"},
				{"rm <name>", "Remove a package"},
			}},
		},
		"spwn get [command]",
		"Use \"spwn get <command> --help\" for more information.",
	)
}
