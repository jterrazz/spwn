package agent

import (
	"fmt"

	"github.com/spf13/cobra"
	"spwn.sh/packages/agent"
)

// ── agent add / remove ─────────────────────────────────────────────────────
//
// Composition commands for attaching reusable blocks (tools, skills, profile)
// to an agent. These edit ~/.spwn/agents/<name>/agent.yaml directly.

var (
	composeTools    []string
	composePlugins  []string
	composeSkills   []string
	composeProfile  string
	composeClearPro bool
)

func init() {
	addCmd.Flags().StringArrayVar(&composeTools, "tool", nil, "Tool pack to add (repeatable, e.g. @spwn/python)")
	addCmd.Flags().StringArrayVar(&composePlugins, "plugin", nil, "Plugin pack to add (repeatable, e.g. @spwn/mempalace)")
	addCmd.Flags().StringArrayVar(&composeSkills, "skill", nil, "Skill to add (repeatable)")
	addCmd.Flags().StringVar(&composeProfile, "profile", "", "Profile template to apply")
	Cmd.AddCommand(addCmd)

	removeCmd.Flags().StringArrayVar(&composeTools, "tool", nil, "Tool pack to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composePlugins, "plugin", nil, "Plugin pack to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composeSkills, "skill", nil, "Skill to remove (repeatable)")
	removeCmd.Flags().BoolVar(&composeClearPro, "profile", false, "Clear the agent's profile attachment")
	Cmd.AddCommand(removeCmd)

	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Add tools, skills, or a profile to an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Compose an agent by attaching reusable blocks.

Examples:
  spwn agent add neo --tool @spwn/python
  spwn agent add neo --plugin @spwn/mempalace
  spwn agent add neo --skill paper-reading --skill refactoring
  spwn agent add neo --profile researcher
  spwn agent add neo --tool @spwn/unix --tool @spwn/git --profile dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composeTools) == 0 && len(composePlugins) == 0 && len(composeSkills) == 0 && composeProfile == "" {
			return fmt.Errorf("nothing to add.\nPass at least one of --tool, --plugin, --skill, or --profile")
		}

		// Verify the agent exists before touching the manifest.
		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		s := newStepper(cmd)
		s.Blank()
		s.Info("Agent:", name)

		for _, t := range composeTools {
			if err := agent.AddTool(name, t); err != nil {
				return fmt.Errorf("add tool %q: %w", t, err)
			}
			s.Done("+ tool", t)
		}
		for _, p := range composePlugins {
			if err := agent.AddPlugin(name, p); err != nil {
				return fmt.Errorf("add plugin %q: %w", p, err)
			}
			s.Done("+ plugin", p)
		}
		for _, sk := range composeSkills {
			if err := agent.AddSkill(name, sk); err != nil {
				return fmt.Errorf("add skill %q: %w", sk, err)
			}
			s.Done("+ skill", sk)
		}
		if composeProfile != "" {
			if err := agent.SetProfile(name, composeProfile); err != nil {
				return fmt.Errorf("set profile %q: %w", composeProfile, err)
			}
			s.Done("+ profile", composeProfile)
		}

		s.Blank()
		s.Success("Composition updated.")
		s.Info("Manifest:", agent.ManifestPath(name))
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <agent-name>",
	Short: "Remove tools, skills, or profile from an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Remove composable blocks from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --tool X' removes just that block.

Examples:
  spwn agent remove neo --tool @spwn/python
  spwn agent remove neo --plugin @spwn/mempalace
  spwn agent remove neo --skill paper-reading
  spwn agent remove neo --profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composeTools) == 0 && len(composePlugins) == 0 && len(composeSkills) == 0 && !composeClearPro {
			return fmt.Errorf("nothing to remove.\nPass at least one of --tool, --plugin, --skill, or --profile")
		}

		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		s := newStepper(cmd)
		s.Blank()
		s.Info("Agent:", name)

		for _, t := range composeTools {
			if err := agent.RemoveTool(name, t); err != nil {
				return fmt.Errorf("remove tool %q: %w", t, err)
			}
			s.Done("- tool", t)
		}
		for _, p := range composePlugins {
			if err := agent.RemovePlugin(name, p); err != nil {
				return fmt.Errorf("remove plugin %q: %w", p, err)
			}
			s.Done("- plugin", p)
		}
		for _, sk := range composeSkills {
			if err := agent.RemoveSkill(name, sk); err != nil {
				return fmt.Errorf("remove skill %q: %w", sk, err)
			}
			s.Done("- skill", sk)
		}
		if composeClearPro {
			if err := agent.ClearProfile(name); err != nil {
				return fmt.Errorf("clear profile: %w", err)
			}
			s.Done("- profile", "cleared")
		}

		s.Blank()
		s.Success("Composition updated.")
		s.Info("Manifest:", agent.ManifestPath(name))
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish <agent-name>",
	Short: "Publish an agent to the registry (memory stripped)",
	Args:  cobra.ExactArgs(1),
	Long: `Publish an agent to the community registry for others to pull.

Memory (journal, knowledge, sessions) is stripped before publishing -
only the composition (tools, skills, profile) and core identity ship.

Not yet implemented - tracks the registry (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Fprintf(cmd.OutOrStderr(), "publish %q: not yet implemented.\n", name)
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <agent-ref>",
	Short: "Install a shared agent from the registry",
	Args:  cobra.ExactArgs(1),
	Long: `Install a shared agent from the community registry into
./spwn/agents/<name>/.

The installed agent starts with a fresh memory but inherits the
full composition from its published form.

Not yet implemented - tracks the registry port (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		fmt.Fprintf(cmd.OutOrStderr(), "get %q: not yet implemented.\n", ref)
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}
