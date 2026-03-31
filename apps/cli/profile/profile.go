package profile

import (
	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

var defaultProfileHelp func(*cobra.Command, []string)

func init() {
	defaultProfileHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(profileHelp)
}

func profileHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "profile" {
		if defaultProfileHelp != nil {
			defaultProfileHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ profile")+" "+ui.Faint("— view and edit an agent's character sheet"),
		[]ui.HelpGroup{
			{Title: "Identity", Commands: []ui.HelpEntry{
				{"purpose", "Why the agent exists"},
				{"traits", "Core principles and character"},
				{"persona", "Personality and role"},
				{"bonds", "Relationships and trust"},
			}},
			{Title: "Capabilities", Commands: []ui.HelpEntry{
				{"skills", "Learned capabilities"},
				{"playbooks", "Step-by-step procedures"},
			}},
			{Title: "Memory", Commands: []ui.HelpEntry{
				{"knowledge", "Facts and context"},
				{"journal", "Session history"},
				{"sessions", "Saved session state"},
			}},
			{Title: "Evolution", Commands: []ui.HelpEntry{
				{"reflect", "Promote patterns to playbooks"},
				{"sleep", "Consolidate and prune memory"},
			}},
			{Title: "Config", Commands: []ui.HelpEntry{
				{"edit", "Open profile.yaml in $EDITOR"},
				{"tier", "View/change tier"},
				{"engine", "View/change runtime engine"},
			}},
		},
		"spwn profile <name>          Show full character sheet\n    spwn profile <name> [aspect]",
		"Use \"spwn profile <name> <aspect> --help\" for more information.",
	)
}

// Cmd is the profile command group.
var Cmd = &cobra.Command{
	Use:   "profile <name> [subcommand]",
	Short: "View and edit agent profiles",
	Long:  `View and edit an agent's character sheet — identity, skills, memory, and configuration.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show full character sheet
		return cmd.Help()
	},
}
