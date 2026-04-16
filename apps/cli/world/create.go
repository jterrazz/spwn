package world

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/project"
)

var (
	createAgents     []string
	createWorkspaces []string
)

// createCmd is `spwn world create <name>` — declare a new world by
// appending a worlds.<name> entry to spwn.yaml. Pure config write,
// nothing touches Docker. Pair with `spwn world start <name>` to
// actually spawn it.
var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Declare a new world in spwn.yaml",
	Long: `Append a worlds.<name> entry to the project's spwn.yaml.

This is a pure config write - no container is created. Once declared,
spawn the world with "spwn world start <name>" or "spwn world <name>".

The agents listed via --agent must already exist on disk under
spwn/agents/<name>/. Use "spwn agent create" first if they don't.`,
	Example: `  spwn world create matrix --agent neo --agent trinity
  spwn world create alignment --agent clippy --workspace data=./data`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New()
		name := args[0]

		p, err := cliproject.Require()
		if err != nil {
			return s.FailHint("No project", err, "Run \"spwn init\" first")
		}

		if err := project.AddWorld(p.ManifestPath, name, project.AddWorldOpts{
			Agents:     createAgents,
			Workspaces: createWorkspaces,
		}); err != nil {
			return s.FailHint("Create failed", err, "")
		}

		s.Blank()
		s.Done("Declared world", name)
		if len(createAgents) > 0 {
			s.Info("Agents", fmt.Sprintf("%v", createAgents))
		}
		s.Blank()
		s.Success(fmt.Sprintf("Start with: spwn world %s", name))
		s.Blank()
		return nil
	},
}

// rmCmd is `spwn world rm <name>` — remove a world entry from
// spwn.yaml. Errors if the world is currently running (caller must
// `world stop` first).
var rmCmd = &cobra.Command{
	Use:     "rm <name>",
	Aliases: []string{"remove"},
	Short:   "Remove a world declaration from spwn.yaml",
	Long: `Remove the worlds.<name> entry from spwn.yaml.

This only edits config - it does NOT stop a running container. If
the world is currently running, stop it first with "spwn world stop
<name>".

The agents listed by the world stay on disk; their minds are
preserved. Other worlds may still reference them.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New()
		name := args[0]

		p, err := cliproject.Require()
		if err != nil {
			return s.FailHint("No project", err, "Run \"spwn init\" first")
		}

		if err := project.RemoveWorld(p.ManifestPath, name); err != nil {
			if errors.Is(err, project.ErrWorldNotFound) {
				return s.FailHint("Not found",
					fmt.Errorf("no world named %q in spwn.yaml", name),
					"Run \"spwn world ls\" to see declared worlds")
			}
			return s.FailHint("Remove failed", err, "")
		}

		s.Blank()
		s.Done("Removed world", name)
		s.Blank()
		return nil
	},
}

func init() {
	createCmd.Flags().StringArrayVarP(&createAgents, "agent", "a", nil, "Agent name (repeatable). Must already exist under spwn/agents/")
	createCmd.Flags().StringArrayVarP(&createWorkspaces, "workspace", "w", nil, `Workspace mount. Forms: "path", "name=path", "name=path:ro"`)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(rmCmd)
}
