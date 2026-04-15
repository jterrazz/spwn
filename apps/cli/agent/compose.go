package agent

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/catalog/plugins"
	"spwn.sh/catalog/tools"
)

// ── agent add / remove ─────────────────────────────────────────────────────
//
// Composition commands for attaching reusable blocks (tools, skills) to an
// agent. These edit ~/.spwn/agents/<name>/agent.yaml directly.

var (
	composeTools   []string
	composePlugins []string
	composeSkills  []string
)

func init() {
	addCmd.Flags().StringArrayVar(&composeTools, "tool", nil, "Tool pack to add (repeatable, e.g. @spwn/python)")
	addCmd.Flags().StringArrayVar(&composePlugins, "plugin", nil, "Plugin pack to add (repeatable, e.g. @spwn/mempalace)")
	addCmd.Flags().StringArrayVar(&composeSkills, "skill", nil, "Skill to add (repeatable)")
	Cmd.AddCommand(addCmd)

	removeCmd.Flags().StringArrayVar(&composeTools, "tool", nil, "Tool pack to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composePlugins, "plugin", nil, "Plugin pack to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composeSkills, "skill", nil, "Skill to remove (repeatable)")
	Cmd.AddCommand(removeCmd)

	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Add tools or skills to an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Compose an agent by attaching reusable blocks.

Examples:
  spwn agent add neo --tool @spwn/python
  spwn agent add neo --plugin @spwn/mempalace
  spwn agent add neo --skill paper-reading --skill refactoring
  spwn agent add neo --tool @spwn/unix --tool @spwn/git`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composeTools) == 0 && len(composePlugins) == 0 && len(composeSkills) == 0 {
			return fmt.Errorf("nothing to add.\nPass at least one of --tool, --plugin, or --skill")
		}

		// Verify the agent exists before touching the manifest.
		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		// Pre-flight every --tool / --plugin ref against the catalog
		// so we never write an unknown pack to agent.yaml — otherwise
		// `agent add` silently breaks `spwn check` with a cryptic
		// "tool does not exist" (Finding #12). Symmetric with the
		// preflight on `agent remove`.
		for _, t := range composeTools {
			if !knownComposeRef(t) {
				return unknownComposeRefError("tool", t)
			}
		}
		for _, p := range composePlugins {
			if !knownComposeRef(p) {
				return unknownComposeRefError("plugin", p)
			}
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

		s.Blank()
		s.Success("Composition updated.")
		s.Info("Manifest:", agent.ManifestPath(name))
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <agent-name>",
	Short: "Remove tools or skills from an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Remove composable blocks from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --tool X' removes just that block.

Examples:
  spwn agent remove neo --tool @spwn/python
  spwn agent remove neo --plugin @spwn/mempalace
  spwn agent remove neo --skill paper-reading`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composeTools) == 0 && len(composePlugins) == 0 && len(composeSkills) == 0 {
			return fmt.Errorf("nothing to remove.\nPass at least one of --tool, --plugin, or --skill")
		}

		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		// Load the manifest once so we can pre-flight every requested
		// removal: if the user passes a tool / plugin / skill that
		// isn't actually attached, we refuse instead of printing a
		// misleading green checkmark on a no-op.
		preflight, err := agent.LoadManifest(name)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		hasString := func(list []string, target string) bool {
			for _, v := range list {
				if v == target {
					return true
				}
			}
			return false
		}
		for _, t := range composeTools {
			if !hasString(preflight.Tools, t) {
				return fmt.Errorf("tool %q is not attached to agent %q — nothing to remove", t, name)
			}
		}
		for _, p := range composePlugins {
			if !hasString(preflight.Plugins, p) {
				return fmt.Errorf("plugin %q is not attached to agent %q — nothing to remove", p, name)
			}
		}
		for _, sk := range composeSkills {
			if !hasString(preflight.Skills, sk) {
				return fmt.Errorf("skill %q is not attached to agent %q — nothing to remove", sk, name)
			}
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
only the composition (tools, skills) and identity ship.

Not yet implemented - tracks the registry (planned).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return notImplemented(fmt.Sprintf("agent publish %q", name),
			"The registry is planned for a future release.")
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
		return notImplemented(fmt.Sprintf("agent get %q", ref),
			"The registry is planned for a future release.")
	},
}

// notImplemented writes a structured "not yet implemented" message to
// stderr and returns a DisplayedError that carries exit code 2 —
// the dedicated "feature unavailable" code used across the CLI so
// scripts can distinguish a missing feature from a runtime failure.
func notImplemented(what, detail string) error {
	fmt.Fprintf(os.Stderr, "\n  %s %s: not yet implemented\n", ui.Red("✗"), what)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(detail))
	}
	fmt.Fprintln(os.Stderr)
	return &notImplementedError{what: what}
}

// notImplementedError carries the exit-2 ("feature unavailable")
// signal back to the process entry point while keeping the
// already-rendered banner untouched (it embeds DisplayedError so
// root.Execute skips its generic "Error:" line).
type notImplementedError struct {
	what string
}

func (e *notImplementedError) Error() string {
	return fmt.Sprintf("%s: not yet implemented", e.what)
}

// ExitCode returns 2 so cmd/spwn/main.go forwards the signal to
// os.Exit. The CLI reserves exit code 2 for "planned but not yet
// implemented" features; exit 1 stays for runtime failures.
func (e *notImplementedError) ExitCode() int { return 2 }

// knownComposeRef reports whether the given @scope/name[@version]
// reference matches a built-in tool or plugin in the catalog. Used
// by `agent add` to preflight --tool and --plugin refs before they
// hit agent.yaml.
func knownComposeRef(ref string) bool {
	pack := stripVersion(ref)
	for _, t := range tools.All {
		if t.Name() == pack {
			return true
		}
	}
	for _, p := range plugins.All {
		if p.Name() == pack {
			return true
		}
	}
	return false
}

// stripVersion drops the "@version" suffix from an @scope/name@version
// reference. Mirrors packages/project/internal/validate/validate.go's
// splitToolVersion but kept local so the CLI doesn't depend on the
// validator's internals.
func stripVersion(ref string) string {
	if !strings.HasPrefix(ref, "@") {
		if idx := strings.LastIndex(ref, "@"); idx > 0 {
			return ref[:idx]
		}
		return ref
	}
	rest := ref[1:]
	if idx := strings.LastIndex(rest, "@"); idx >= 0 {
		return "@" + rest[:idx]
	}
	return ref
}

// unknownComposeRefError formats the refusal message shown when the
// user passes --tool or --plugin with a reference the catalog does
// not know about. The "known:" list mirrors what `spwn check` shows
// so the two commands never disagree.
func unknownComposeRefError(kind, ref string) error {
	known := make([]string, 0, len(tools.All)+len(plugins.All))
	for _, t := range tools.All {
		known = append(known, t.Name())
	}
	for _, p := range plugins.All {
		known = append(known, p.Name())
	}
	sort.Strings(known)
	return fmt.Errorf("%s %q does not exist.\nknown: %s",
		kind, ref, strings.Join(known, ", "))
}

