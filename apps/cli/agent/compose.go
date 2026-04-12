package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ── agent add ──────────────────────────────────────────────────────────────
//
// Composition commands for attaching reusable blocks (tools, skills, profiles)
// to an agent. Stubs for now — the full implementation requires wiring to the
// agent/tool/skill/profile registries.

var (
	addToolFlag    []string
	addSkillFlag   []string
	addProfileFlag string
)

func init() {
	addCmd.Flags().StringArrayVar(&addToolFlag, "tool", nil, "Tool pack to add (repeatable, e.g. @spwn/python)")
	addCmd.Flags().StringArrayVar(&addSkillFlag, "skill", nil, "Skill to add (repeatable)")
	addCmd.Flags().StringVar(&addProfileFlag, "profile", "", "Profile template to apply")
	Cmd.AddCommand(addCmd)

	removeCmd.Flags().StringArrayVar(&addToolFlag, "tool", nil, "Tool pack to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&addSkillFlag, "skill", nil, "Skill to remove (repeatable)")
	removeCmd.Flags().StringVar(&addProfileFlag, "profile", "", "Remove profile (clears agent's profile)")
	Cmd.AddCommand(removeCmd)

	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(pullCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Add tools, skills, or a profile to an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Compose an agent by attaching reusable blocks.

Examples:
  spwn agent add neo --tool @spwn/python
  spwn agent add neo --skill paper-reading --skill refactoring
  spwn agent add neo --profile researcher
  spwn agent add neo --tool @spwn/unix --tool @spwn/git --profile dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(addToolFlag) == 0 && len(addSkillFlag) == 0 && addProfileFlag == "" {
			return fmt.Errorf("nothing to add.\nPass at least one of --tool, --skill, or --profile")
		}
		// TODO: wire to agent.yaml composition. Currently a stub that reports intent.
		fmt.Fprintf(cmd.OutOrStderr(), "agent compose %q:\n", name)
		for _, t := range addToolFlag {
			fmt.Fprintf(cmd.OutOrStderr(), "  + tool:    %s\n", t)
		}
		for _, s := range addSkillFlag {
			fmt.Fprintf(cmd.OutOrStderr(), "  + skill:   %s\n", s)
		}
		if addProfileFlag != "" {
			fmt.Fprintf(cmd.OutOrStderr(), "  + profile: %s\n", addProfileFlag)
		}
		fmt.Fprintln(cmd.OutOrStderr())
		fmt.Fprintln(cmd.OutOrStderr(), "Note: agent.yaml composition is not yet wired.")
		fmt.Fprintln(cmd.OutOrStderr(), "      This command will edit the manifest in a future release.")
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:     "remove <agent-name>",
	Aliases: []string{"unadd"},
	Short:   "Remove tools, skills, or profile from an agent",
	Args:    cobra.ExactArgs(1),
	Long: `Remove composable blocks from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent rm <name> --tool X' removes just that block.

Examples:
  spwn agent remove neo --tool @spwn/python
  spwn agent remove neo --skill paper-reading
  spwn agent remove neo --profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(addToolFlag) == 0 && len(addSkillFlag) == 0 && addProfileFlag == "" {
			return fmt.Errorf("nothing to remove.\nPass at least one of --tool, --skill, or --profile")
		}
		// TODO: wire to agent.yaml composition. Currently a stub that reports intent.
		fmt.Fprintf(cmd.OutOrStderr(), "agent compose %q:\n", name)
		for _, t := range addToolFlag {
			fmt.Fprintf(cmd.OutOrStderr(), "  - tool:    %s\n", t)
		}
		for _, s := range addSkillFlag {
			fmt.Fprintf(cmd.OutOrStderr(), "  - skill:   %s\n", s)
		}
		if addProfileFlag != "" {
			fmt.Fprintf(cmd.OutOrStderr(), "  - profile: %s\n", addProfileFlag)
		}
		fmt.Fprintln(cmd.OutOrStderr())
		fmt.Fprintln(cmd.OutOrStderr(), "Note: agent.yaml composition is not yet wired.")
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish <agent-name>",
	Short: "Publish an agent to the registry (memory stripped)",
	Args:  cobra.ExactArgs(1),
	Long: `Publish an agent to the community registry for others to pull.

Memory (journal, knowledge, sessions) is stripped before publishing —
only the composition (tools, skills, profile) and core identity ship.

Not yet implemented — tracks the registry port (coming in Epoch 10).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Fprintf(cmd.OutOrStderr(), "publish %q: not yet implemented.\n", name)
		fmt.Fprintln(cmd.OutOrStderr(), "The registry port is planned for Epoch 10 (Marketplace).")
		return nil
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull <agent-ref>",
	Short: "Pull a shared agent from the registry",
	Args:  cobra.ExactArgs(1),
	Long: `Install a shared agent from the community registry.

The pulled agent starts with a fresh memory but inherits the full
composition from its published form.

Not yet implemented — tracks the registry port (coming in Epoch 10).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		fmt.Fprintf(cmd.OutOrStderr(), "pull %q: not yet implemented.\n", ref)
		fmt.Fprintln(cmd.OutOrStderr(), "The registry port is planned for Epoch 10 (Marketplace).")
		return nil
	},
}
