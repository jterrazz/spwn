package blueprint

import (
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultBlueprintHelp func(*cobra.Command, []string)

// Cmd is the parent command for blueprint operations.
var Cmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Universe knowledge base — the single source of truth",
	Long:  `The blueprint is the knowledge base for your spwn universe. The Architect maintains it as the single source of truth for projects, architecture, decisions, and team structure.`,
	RunE:  runOverview,
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all files in the blueprint",
	RunE:  runLs,
}

var showCmd = &cobra.Command{
	Use:   "show <path>",
	Short: "Show the contents of a blueprint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all blueprint files",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	defaultBlueprintHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(blueprintHelp)

	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(searchCmd)
}

func runOverview(cmd *cobra.Command, args []string) error {
	s := newStepper(cmd)

	// Ensure blueprint exists
	if err := universe.InitBlueprint(); err != nil {
		return s.FailHint("Blueprint", err, "")
	}

	content, err := universe.ReadBlueprintFile("overview.md")
	if err != nil {
		return s.FailHint("Blueprint", err, "Run 'spwn blueprint ls' to see available files")
	}

	s.Blank()
	fmt.Fprint(cmd.OutOrStdout(), content)
	s.Blank()

	return nil
}

func runLs(cmd *cobra.Command, args []string) error {
	s := newStepper(cmd)

	// Ensure blueprint exists
	if err := universe.InitBlueprint(); err != nil {
		return s.FailHint("Blueprint", err, "")
	}

	files, err := universe.ListBlueprintFiles()
	if err != nil {
		return s.FailHint("Blueprint", err, "")
	}

	if len(files) == 0 {
		s.Blank()
		s.Info("Blueprint:", "empty")
		s.Blank()
		return nil
	}

	s.Blank()
	fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ui.Strong("Blueprint Files:"))
	s.Blank()

	for _, f := range files {
		sizeStr := formatSize(f.Size)
		fmt.Fprintf(cmd.OutOrStdout(), "    %s %s\n",
			ui.PadVisible(ui.ColorizeHelpName(f.Path), 44),
			ui.Faint(sizeStr))
	}

	s.Blank()
	fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ui.Faint("Use \"spwn blueprint show <path>\" to read a file."))
	s.Blank()

	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	s := newStepper(cmd)

	content, err := universe.ReadBlueprintFile(args[0])
	if err != nil {
		return s.FailHint("Blueprint", err, "Run 'spwn blueprint ls' to see available files")
	}

	s.Blank()
	fmt.Fprint(cmd.OutOrStdout(), content)
	s.Blank()

	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	s := newStepper(cmd)

	results, err := universe.SearchBlueprint(args[0])
	if err != nil {
		return s.FailHint("Blueprint", err, "")
	}

	if len(results) == 0 {
		s.Blank()
		s.Info("Search:", fmt.Sprintf("no results for %q", args[0]))
		s.Blank()
		return nil
	}

	s.Blank()
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n",
		ui.Strong("Search results for"),
		ui.Strong(fmt.Sprintf("%q", args[0])))
	s.Blank()

	for path, lines := range results {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ui.ColorizeHelpName(path))
		for _, line := range lines {
			fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", strings.TrimSpace(line))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
}

func blueprintHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "blueprint" {
		if defaultBlueprintHelp != nil {
			defaultBlueprintHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ blueprint")+" "+ui.Faint("— universe knowledge base"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "blueprint", Desc: "Show the overview (default)"},
				{Name: "ls", Desc: "List all files in the blueprint"},
				{Name: "show <path>", Desc: "Show the contents of a file"},
				{Name: "search <query>", Desc: "Search across all files"},
			}},
		},
		"spwn blueprint [command]",
		"Use \"spwn blueprint <command> --help\" for more information.",
	)
}
