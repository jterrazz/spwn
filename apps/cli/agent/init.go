package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"

	"spwn.sh/packages/project"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/platform"
)

var (
	initTeam  string
	initForce bool
)

func init() {
	initCmd.Flags().StringVar(&initTeam, "team", "", "Assign agent to a team (slug)")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Re-scaffold any missing Mind files without complaining if the agent already exists")
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:     "create [name]",
	Aliases: []string{"new"},
	Short:   "Create a new agent (SOUL.md + 3-layer Mind)",
	Long: `Create a new agent with a SOUL.md at the agent root and the
three Mind layer directories (skills/playbooks/journal). Knowledge is
world-scoped, not agent-scoped — it lives at /world/knowledge/ when a
world opts in via spwn.yaml's worlds.<name>.knowledge key, which resolves
to a host path under the project root. If no name is provided, a random
name is picked from a curated dictionary.

With --force, an existing agent's Mind is re-scaffolded: any missing
files are recreated and the command exits zero even if the agent
already exists.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New()

		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = platform.RandomAgentName()
		}

		// Reject empty / invalid names before touching disk. The
		// same slug regex the manifest enforces for world names is
		// applied here so we never leave a half-created agent
		// directory that `spwn check` cannot reconcile.
		if name == "" {
			return fmt.Errorf("agent name is required")
		}
		if !project.IsValidAgentName(name) {
			return fmt.Errorf("agent name %q is invalid — must match ^[a-z][a-z0-9-]*$ (lowercase letters, digits, and dashes; must start with a letter)", name)
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
			if initForce && strings.Contains(err.Error(), "already exists") {
				// --force: re-scaffold any missing files over the
				// existing Mind and proceed as if the create had
				// succeeded.
				if ferr := agent.RepairMind(name); ferr != nil {
					return s.FailHint("Force re-scaffold failed", ferr,
						"Check that the agent directory is writable")
				}
				s.Done("Re-scaffolded agent", name)
				s.Blank()
				s.Success(fmt.Sprintf("Spawn with: spwn up --agent %s", name))
				s.Blank()
				return nil
			}
			hint := "Check that ~/.spwn/agents/ is writable"
			if strings.Contains(err.Error(), "already exists") {
				hint = fmt.Sprintf("Run \"spwn agent rm %s\" first, or pass --force to re-scaffold", name)
			}
			return s.FailHint("Agent creation failed", err, hint)
		}

		s.Done("Created agent", name)
		s.Done("Created soul", "SOUL.md")

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
			if err := project.SetAgentTeam(name, initTeam); err != nil {
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
