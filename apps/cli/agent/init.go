package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/packages/project"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/ids"
)

var initTeam string

func init() {
	initCmd.Flags().StringVar(&initTeam, "team", "", "Assign agent to a team (slug)")
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new agent with a 6-layer Mind",
	Long: `Create a new agent with the 6-layer Mind structure. If no name is
provided, a random name is picked from a curated dictionary.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)

		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = ids.RandomAgentName()
		}

		// Reject names that would shadow `spwn agent <subcommand>`.
		// Caught here, before InitMind writes anything to disk, so
		// the user never sees a half-created agent directory.
		if project.IsReservedAgentName(name) {
			return fmt.Errorf("agent name %q is reserved (collides with `spwn agent %s`). Reserved names: %s",
				name, name, strings.Join(project.ReservedAgentNames(), ", "))
		}

		s.Blank()
		s.Start(fmt.Sprintf("Creating agent %q...", name))
		_, err := agent.InitMind(name)
		if err != nil {
			hint := "Check that ~/.spwn/agents/ is writable"
			if strings.Contains(err.Error(), "already exists") {
				hint = fmt.Sprintf("Run \"spwn agent rm %s\" first, or choose a different name", name)
			}
			return s.FailHint("Agent creation failed", err, hint)
		}

		s.Done("Created agent", name)
		s.Done("Created profile", "profile.md")

		// Auto-world: when a spwn project is active, also add a
		// single-agent world entry to spwn.yaml so the agent is
		// immediately deployable via `spwn up` or `spwn agent <name>`.
		if cwd, werr := os.Getwd(); werr == nil {
			if p, perr := project.Find(cwd); perr == nil && p != nil {
				if addErr := project.AddAgentToManifest(p.ManifestPath, name); addErr != nil {
					s.Warn("Warning", fmt.Sprintf("could not add world to spwn.yaml: %v", addErr))
				} else {
					s.Done("Added world to spwn.yaml", name)
				}
			}
		}

		// Assign team if provided
		if initTeam != "" {
			if err := agent.SetAgentTeam(name, initTeam); err != nil {
				s.Warn("Warning", fmt.Sprintf("could not set team: %v", err))
			} else {
				s.Done("Team", initTeam)
			}
		}

		s.Blank()
		s.Success(fmt.Sprintf("Spawn with: spwn up --agent %s", name))
		s.Blank()

		return nil
	},
}
