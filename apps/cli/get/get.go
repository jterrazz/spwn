package get

import (
	"fmt"
	"os"

	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

// notImplementedError mirrors the one in apps/cli/agent/compose.go:
// it renders a structured "not yet implemented" banner to stderr
// and carries exit code 2 so scripts can distinguish a missing
// feature from a runtime failure.
type notImplementedError struct{ what string }

func (e *notImplementedError) Error() string { return fmt.Sprintf("%s: not yet implemented", e.what) }
func (e *notImplementedError) ExitCode() int { return 2 }

func notImplemented(what, detail string) error {
	fmt.Fprintf(os.Stderr, "\n  %s %s: not yet implemented\n", ui.Red("\u2717"), what)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(detail))
	}
	fmt.Fprintln(os.Stderr)
	return &notImplementedError{what: what}
}

var defaultGetHelp func(*cobra.Command, []string)

// Cmd is the parent command for marketplace package management.
var Cmd = &cobra.Command{
	Use:    "get",
	Short:  "Install from the marketplace",
	Long:   `Install packages from the marketplace - skills, worlds, packs, and adapters.`,
	Hidden: true,
}

var installCmd = &cobra.Command{
	Use:   "install [name]",
	Short: "Install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(fmt.Sprintf("get install %q", args[0]),
			"The marketplace is planned for a future release.")
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
		return notImplemented(fmt.Sprintf("get search %q", args[0]),
			"The marketplace is planned for a future release.")
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
		ui.Strong("⬡ get")+" "+ui.Faint("- install from the marketplace"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "install <name>", Desc: "Install a package"},
				{Name: "ls", Desc: "List installed packages"},
				{Name: "search <query>", Desc: "Search the marketplace"},
				{Name: "rm <name>", Desc: "Remove a package"},
			}},
		},
		"spwn get [command]",
		"Use \"spwn get <command> --help\" for more information.",
	)
}
