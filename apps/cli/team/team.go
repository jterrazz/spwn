package team

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
)

// Cmd is the top-level `spwn team` command.
var Cmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams - create, list, edit, and remove agent groups",
}

var (
	newColor string
	newDesc  string
)

func init() {
	newCmd.Flags().StringVar(&newColor, "color", "", "Accent color - hex (#8B5CF6) or name (purple, blue, red, amber, emerald)")
	newCmd.Flags().StringVar(&newDesc, "description", "", "Short description")

	Cmd.AddCommand(newCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(assignCmd)
	Cmd.AddCommand(membersCmd)

	// The team subsystem is in active design - the manifest, storage layout,
	// and CLI surface are all subject to change. Mark every entry point.
	ui.MarkExperimental(Cmd)
	ui.MarkExperimental(newCmd)
	ui.MarkExperimental(lsCmd)
	ui.MarkExperimental(editCmd)
	ui.MarkExperimental(rmCmd)
	ui.MarkExperimental(assignCmd)
	ui.MarkExperimental(membersCmd)
}

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
}

// ── spwn team new ──

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new team",
	Example: `  spwn team new "Matrix Ops" --color "#8B5CF6"
  spwn team new infra --color purple`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		name := strings.TrimSpace(args[0])
		if name == "" {
			return fmt.Errorf("team name is required")
		}
		slug := agent.Slugify(name)
		if slug == "" {
			return fmt.Errorf("team name %q has no valid characters — use letters, digits, and dashes", args[0])
		}

		t := agent.Team{
			Slug:        slug,
			Name:        name,
			Color:       newColor,
			Description: newDesc,
		}
		if err := agent.CreateTeam(t); err != nil {
			return s.FailHint("Team creation failed", err, "Choose a different name")
		}

		s.Blank()
		s.Done("Created team", fmt.Sprintf("%s (%s)", name, slug))
		s.Blank()
		return nil
	},
}

// ── spwn team ls ──

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all teams",
	RunE: func(cmd *cobra.Command, args []string) error {
		teams, err := agent.ListTeams()
		if err != nil {
			return err
		}
		if len(teams) == 0 {
			fmt.Println("No teams yet. Create one with: spwn team new \"Team Name\"")
			return nil
		}
		for _, t := range teams {
			members, _ := agent.TeamMembers(t.Slug)
			line := fmt.Sprintf("  ●  %-20s %d agent(s)", t.Name, len(members))
			if len(members) > 0 {
				line += "   " + strings.Join(members, ", ")
			}
			fmt.Println(line)
		}
		return nil
	},
}

// ── spwn team edit ──

var editCmd = &cobra.Command{
	Use:   "edit <slug>",
	Short: "Edit a team's metadata",
	Example: `  spwn team edit matrix-ops --color "#A855F7"
  spwn team edit infra --description "Infrastructure & ops"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		slug := args[0]

		t, err := agent.GetTeam(slug)
		if err != nil {
			return s.FailHint("Team not found", err, "Run \"spwn team ls\" to see all teams")
		}

		changed := false
		if f := cmd.Flag("color"); f != nil && f.Changed {
			t.Color = newColor
			changed = true
		}
		if f := cmd.Flag("description"); f != nil && f.Changed {
			t.Description = newDesc
			changed = true
		}

		if !changed {
			fmt.Printf("  %s (%s)\n", t.Name, t.Slug)
			if t.Color != "" {
				fmt.Printf("     color: %s\n", t.Color)
			}
			if t.Description != "" {
				fmt.Printf("     %s\n", t.Description)
			}
			return nil
		}

		if err := agent.UpdateTeam(*t); err != nil {
			return err
		}
		s.Blank()
		s.Done("Updated", t.Name)
		s.Blank()
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&newColor, "color", "", "Accent color - hex or name")
	editCmd.Flags().StringVar(&newDesc, "description", "", "Short description")
}

// ── spwn team rm ──

var rmCmd = &cobra.Command{
	Use:     "rm <slug>",
	Aliases: []string{"remove", "delete"},
	Short:   "Delete a team (agents become solo)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		slug := args[0]

		if err := agent.DeleteTeam(slug); err != nil {
			return s.FailHint("Delete failed", err, "Run \"spwn team ls\" to see teams")
		}

		s.Blank()
		s.Done("Deleted team", slug)
		s.Info("Note:", "Agents that referenced this team are now solo")
		s.Blank()
		return nil
	},
}

// ── spwn team assign (shorthand for spwn agent team) ──

var assignCmd = &cobra.Command{
	Use:   "assign <agent-name> <team-slug>",
	Short: "Assign an agent to a team (or --clear to remove)",
	Example: `  spwn team assign neo matrix-ops
  spwn team assign qa --clear`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		agentName := args[0]
		teamSlug := ""

		clear, _ := cmd.Flags().GetBool("clear")
		if !clear && len(args) > 1 {
			teamSlug = args[1]
		}

		if err := agent.SetAgentTeam(agentName, teamSlug); err != nil {
			return s.FailHint("Assign failed", err, "Check agent exists")
		}

		s.Blank()
		if teamSlug == "" {
			s.Done("Cleared team", agentName)
		} else {
			s.Done("Assigned", fmt.Sprintf("%s → %s", agentName, teamSlug))
		}
		s.Blank()
		return nil
	},
}

func init() {
	assignCmd.Flags().Bool("clear", false, "Remove agent from team")
}

// ── spwn team members ──

var membersCmd = &cobra.Command{
	Use:     "members <slug>",
	Short:   "List agents in a team",
	Example: `  spwn team members matrix-ops`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		members, err := agent.TeamMembers(slug)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			fmt.Printf("No agents in team %q\n", slug)
			return nil
		}
		for _, name := range members {
			fmt.Println("  " + name)
		}
		return nil
	},
}
