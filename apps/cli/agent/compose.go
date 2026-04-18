package agent

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/catalog"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/dependency"
)

// ── agent add / remove ─────────────────────────────────────────────────────
//
// Composition commands for attaching reusable blocks (tools, skills) to an
// agent. These edit ~/.spwn/agents/<name>/agent.yaml directly.

// composeDeps is filled by any of --dep/--deps/--skill/--tool/--hook.
// All four flags append to the same list and share identical resolution
// (bare name → spwn:<name>; explicit scheme passes through). The
// semantic flavour of each flag is purely a docs/discoverability hint —
// `--skill qmd` and `--dep qmd` behave identically, but the former reads
// better in tutorials and scripts.
var (
	composeDeps    []string
	composeSkills  []string
	composeTools   []string
	composeHooks   []string
	composeRemoves []string
)

func init() {
	addCmd.Flags().StringArrayVar(&composeDeps, "dep", nil, "Dependency ref to add (repeatable, e.g. --dep python)")
	addCmd.Flags().StringArrayVar(&composeDeps, "deps", nil, "Plural alias for --dep")
	addCmd.Flags().StringArrayVar(&composeSkills, "skill", nil, "Skill ref to add (bare name resolves to catalog; use skill:<name> for local)")
	addCmd.Flags().StringArrayVar(&composeTools, "tool", nil, "Tool ref to add (bare name resolves to catalog; use tool:<name> for local)")
	addCmd.Flags().StringArrayVar(&composeHooks, "hook", nil, "Hook ref to add (bare name resolves to catalog; use hook:<name> for local)")
	Cmd.AddCommand(addCmd)

	removeCmd.Flags().StringArrayVar(&composeRemoves, "dep", nil, "Dependency ref to remove (repeatable)")
	removeCmd.Flags().StringArrayVar(&composeRemoves, "deps", nil, "Plural alias for --dep")
	Cmd.AddCommand(removeCmd)

	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(getCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Add dependencies to an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Compose an agent by attaching catalog entries or local blocks.

Bare names resolve to the spwn: catalog ("--dep qmd" adds spwn:qmd).
Locals must use the explicit skill:/tool:/hook: scheme.

Examples:
  spwn agent add neo --dep python
  spwn agent add neo --dep spwn:unix --dep spwn:git
  spwn agent add neo --dep skill:focus --dep tool:my-parser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		combined := make([]string, 0,
			len(composeDeps)+len(composeSkills)+len(composeTools)+len(composeHooks))
		combined = append(combined, composeDeps...)
		combined = append(combined, composeSkills...)
		combined = append(combined, composeTools...)
		combined = append(combined, composeHooks...)

		if len(combined) == 0 {
			return fmt.Errorf("nothing to add.\nPass at least one of --dep / --skill / --tool / --hook")
		}

		if err := agent.ValidateMind(name); err != nil {
			return err
		}

		// Resolve every ref through the shared CLI resolver: bare
		// names become spwn:<name>, explicit schemes pass through,
		// everything else errors out with a catalog-aware hint. The
		// resolved form is what we persist to agent.yaml — manifests
		// stay scheme-only.
		catalogNames := catalogToolNames()
		resolved := make([]string, len(combined))
		for i, p := range combined {
			r, err := dependency.ResolveCLI(p, catalogNames)
			if err != nil {
				return err
			}
			// `spwn:<name>` refs still need to exist in the catalog —
			// ResolveCLI only confirms grammar for explicit schemes.
			base, _ := dependency.SplitVersion(r)
			parsed := dependency.ParseRef(base)
			if parsed.Kind == dependency.KindSpwnBuiltin && !knownComposeRef(r) {
				return unknownComposeRefError("dependency", p)
			}
			resolved[i] = r
		}

		s := ui.New()
		s.Blank()
		s.Info("Agent:", name)

		for _, p := range resolved {
			if err := agent.AddDependency(name, p); err != nil {
				return fmt.Errorf("add dependency %q: %w", p, err)
			}
			s.Done("+ dep", p)
		}

		s.Blank()
		s.Success("Composition updated.")
		s.Info("Manifest:", agent.ManifestPath(name))
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <agent-name>",
	Short: "Remove dependencies from an agent",
	Args:  cobra.ExactArgs(1),
	Long: `Remove dependencies from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --dep X' removes just that entry.

Examples:
  spwn agent remove neo --dep spwn:python
  spwn agent remove neo --deps spwn:mempalace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if len(composeRemoves) == 0 {
			return fmt.Errorf("nothing to remove.\nPass at least one --dep")
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
		for _, p := range composeRemoves {
			if !hasString(preflight.Deps, p) {
				return fmt.Errorf("dependency %q is not attached to agent %q — nothing to remove", p, name)
			}
		}

		s := ui.New()
		s.Blank()
		s.Info("Agent:", name)

		for _, p := range composeRemoves {
			if err := agent.RemoveDependency(name, p); err != nil {
				return fmt.Errorf("remove dependency %q: %w", p, err)
			}
			s.Done("- dep", p)
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
only the composition (dependencies, skills) and identity ship.

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
// reference matches a built-in dependency in the catalog. Used by
// `agent add` to preflight --dep refs before they hit agent.yaml.
func knownComposeRef(ref string) bool {
	name := stripVersion(ref)
	for _, t := range catalog.All {
		if t.Name() == name {
			return true
		}
	}
	return false
}

// catalogToolNames returns the bare names (without the spwn: prefix)
// of every tool-shaped entry in the catalog. Used as the bare-name
// lookup set for dependency.ResolveCLI — "--dep qmd" only resolves
// when "qmd" is in this list.
func catalogToolNames() []string {
	out := make([]string, 0, len(catalog.All))
	for _, t := range catalog.All {
		// catalog.All keys are canonical spwn:<name>. Trim the prefix
		// so ResolveCLI can match against bare input.
		name := strings.TrimPrefix(t.Name(), "spwn:")
		out = append(out, name)
	}
	return out
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
// user passes --dep with a reference the catalog does not know about.
// The "known:" list mirrors what `spwn check` shows so the two
// commands never disagree.
func unknownComposeRefError(kind, ref string) error {
	known := make([]string, 0, len(catalog.All))
	for _, t := range catalog.All {
		known = append(known, t.Name())
	}
	sort.Strings(known)
	return fmt.Errorf("%s %q does not exist.\nknown: %s",
		kind, ref, strings.Join(known, ", "))
}
