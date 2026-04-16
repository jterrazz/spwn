package agent

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/catalog/packs"
	"spwn.sh/packages/agent"
)

// ── agent add / remove ─────────────────────────────────────────────────────
//
// Composition commands for attaching reusable blocks (tools, skills) to an
// agent. These edit ~/.spwn/agents/<name>/agent.yaml directly.

var composePacks []string

func init() {
	addCmd.Flags().StringArrayVar(&composePacks, "pack", nil, "Pack ref to add (repeatable, e.g. @spwn/python)")
	addCmd.Flags().StringArrayVar(&composePacks, "packs", nil, "Plural alias for --plugin")
	Cmd.AddCommand(addCmd)

	removeCmd.Flags().StringArrayVar(&composePacks, "pack", nil, "Pack ref to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composePacks, "packs", nil, "Plural alias for --plugin")
	Cmd.AddCommand(removeCmd)

	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Add packs to an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Compose an agent by attaching packs.

Examples:
  spwn agent add neo --pack @spwn/python
  spwn agent add neo --packs @spwn/unix --packs @spwn/git
  spwn agent add neo --pack @spwn/unix --pack @spwn/git`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composePacks) == 0 {
			return fmt.Errorf("nothing to add.\nPass at least one --plugin")
		}

		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		// Pre-flight every catalog ref against the catalog so we never
		// write an unknown plugin to agent.yaml. Bare-name local refs
		// are skipped here — they resolve against the project tree at
		// build time, not the catalog.
		for _, p := range composePacks {
			if strings.HasPrefix(p, "@") && !knownComposeRef(p) {
				return unknownComposeRefError("pack", p)
			}
		}

		s := newStepper(cmd)
		s.Blank()
		s.Info("Agent:", name)

		for _, p := range composePacks {
			if err := agent.AddPack(name, p); err != nil {
				return fmt.Errorf("add pack %q: %w", p, err)
			}
			s.Done("+ pack", p)
		}

		s.Blank()
		s.Success("Composition updated.")
		s.Info("Manifest:", agent.ManifestPath(name))
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <agent-name>",
	Short: "Remove packs from an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Remove packs from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --plugin X' removes just that entry.

Examples:
  spwn agent remove neo --pack @spwn/python
  spwn agent remove neo --packs @spwn/mempalace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composePacks) == 0 {
			return fmt.Errorf("nothing to remove.\nPass at least one --plugin")
		}

		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		// Preflight: every ref must currently be attached, otherwise
		// the removal silently no-ops and the user doesn't notice the
		// typo.
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
		for _, p := range composePacks {
			if !hasString(preflight.Plugins, p) {
				return fmt.Errorf("pack %q is not attached to agent %q — nothing to remove", p, name)
			}
		}

		s := newStepper(cmd)
		s.Blank()
		s.Info("Agent:", name)

		for _, p := range composePacks {
			if err := agent.RemovePack(name, p); err != nil {
				return fmt.Errorf("remove pack %q: %w", p, err)
			}
			s.Done("- pack", p)
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
// reference matches a built-in pack in the catalog. Used by
// `agent add` to preflight --pack refs before they
// hit agent.yaml.
func knownComposeRef(ref string) bool {
	pack := stripVersion(ref)
	for _, t := range packs.All {
		if t.Name() == pack {
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
// user passes --plugin with a reference the catalog does
// not know about. The "known:" list mirrors what `spwn check` shows
// so the two commands never disagree.
func unknownComposeRefError(kind, ref string) error {
	known := make([]string, 0, len(packs.All))
	for _, t := range packs.All {
		known = append(known, t.Name())
	}
	sort.Strings(known)
	return fmt.Errorf("%s %q does not exist.\nknown: %s",
		kind, ref, strings.Join(known, ", "))
}

