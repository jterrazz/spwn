// Package dependency implements the install/uninstall verbs behind
// `spwn install` and `spwn uninstall`. It mutates agent.yaml lists
// (scoped to one agent via --agent, or across the whole project by
// default) and keeps spwn.lock pinned.
//
// Reference kinds (delegated to packages/dependency):
//
//   - spwn:<name>                built-in catalog (compiled into the binary)
//   - github:<owner>/<repo>      third-party (planned — git tags as versions)
//   - skill/<name>               local skill (./spwn/skills/<name>.md)
//   - tool/<name>                local tool (./spwn/tools/<name>/)
//   - hook/<name>                local hook (./spwn/hooks/<name>.sh)
//
// Local refs (skill/, tool/, hook/) attach the in-repo block to the
// named agent; they require --agent because bolting a local onto
// every agent by default is almost never what the user wants.
package dependency

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/refs"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
)

// installCmd and uninstallCmd are retained so the dependency_test
// suite can drive the install/uninstall flow directly without going
// through the top-level root command.
var installAgentFilter string

var installCmd = &cobra.Command{
	Use:   "install <ref>",
	Short: "Install a dependency into the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args[0], installAgentFilter)
	},
}

var uninstallAgentFilter string

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <ref>",
	Aliases: []string{"rm", "remove"},
	Short:   "Uninstall a dependency from the project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUninstall(cmd, args[0], uninstallAgentFilter)
	},
}

func init() {
	installCmd.Flags().StringVar(&installAgentFilter, "agent", "", "Target a single agent instead of every agent in the project")
	uninstallCmd.Flags().StringVar(&uninstallAgentFilter, "agent", "", "Target a single agent instead of every agent in the project")
}

// RunInstall parses the ref, rejects bare/registry refs with crisp
// hints, mutates each targeted agent's agent.yaml, and updates the
// lockfile.
//
// agentFilter narrows scope: empty means "every agent in the project"
// (npm-style), a non-empty value means "only this agent" (poetry-
// group-style). The lockfile is project-scoped either way — it
// tracks the set of refs that are pinned somewhere in the project,
// not which agents carry them.
func RunInstall(cmd *cobra.Command, raw, agentFilter string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	// Run the raw input through the shared CLI resolver so bare names
	// auto-promote to spwn:<name> ("spwn install qmd" → "spwn install
	// spwn:qmd"). Explicit schemes pass through unchanged. Manifests
	// always receive the canonical scheme-form.
	resolved, rerr := refs.ResolveCLI(raw, knownCatalogNames())
	if rerr != nil {
		return rerr
	}

	ref, version := refs.SplitVersion(resolved)
	parsed := refs.ParseRef(ref)
	switch parsed.Kind {
	case refs.KindLocalSkill, refs.KindLocalTool, refs.KindLocalHook:
		// Local refs ARE installable (they attach the in-repo block
		// To an agent's manifest), but only when a specific agent is
		// Named — bolting a local skill onto every agent in the
		// Project is almost never what the user wants. Without
		// --agent, point them at the flag.
		if agentFilter == "" {
			return fmt.Errorf("%q is a local ref — pass --agent <name> to attach it to one agent, or author a new one with `spwn skill new %s` / `spwn/tools/%s/` / `spwn/hooks/%s.sh`",
				ref, parsed.Name, parsed.Name, parsed.Name)
		}
		// Validate the target file/dir actually exists. Installing a
		// ref to a missing local block lets the user accumulate broken
		// state quietly (the failure only surfaces at `spwn up` time).
		// Catch it at install time with an actionable hint.
		if res := refs.ResolveSkill(p.Root, parsed, nil, false); res == refs.ResolveNotFound {
			switch parsed.Kind {
			case refs.KindLocalSkill:
				return fmt.Errorf("skill/%s not found at spwn/skills/%s.md — create it first with `spwn skill new %s`",
					parsed.Name, parsed.Name, parsed.Name)
			case refs.KindLocalTool:
				return fmt.Errorf("tool/%s not found at spwn/tools/%s/ — scaffold a tool.yaml there first",
					parsed.Name, parsed.Name)
			case refs.KindLocalHook:
				return fmt.Errorf("hook/%s not found at spwn/hooks/%s.sh — create the hook script first",
					parsed.Name, parsed.Name)
			}
		}
	case refs.KindRegistry:
		return fmt.Errorf("%q targets github:%s/%s — remote registries are not yet supported. "+
			"Use spwn:<name> for built-in dependencies, or author a local one under ./spwn/tools/",
			raw, parsed.Owner, parsed.Name)
	case refs.KindInvalid:
		return fmt.Errorf("%q is not a valid dependency ref — use spwn:<name> (for built-ins), "+
			"github:<owner>/<repo> (for remote deps), skill/<name>, tool/<name>, or hook/<name>",
			raw)
	}

	// Catalog-existence check applies only to spwn:<name> refs; local
	// Schemes are authored in-repo so the catalog lookup doesn't gate
	// Them.
	if parsed.Kind == refs.KindSpwnBuiltin && !catalogHas(ref) {
		return fmt.Errorf("unknown builtin %q — see the catalog for available dependencies", ref)
	}

	targets, terr := filterAgents(p.Agents, agentFilter)
	if terr != nil {
		return terr
	}
	mutated := 0
	for _, a := range targets {
		if err := agent.AddDependency(a.Name, ref); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := dependency.LoadLockfileOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Add(ref, dependency.LockEntry{
		Version: version,
		Source:  dependency.SourceBuiltin,
	})
	if err := dependency.SaveLockfile(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("installed "+ref))
	fmt.Fprintf(out, "     %d agent%s updated, %s pinned\n",
		mutated, plural(mutated), dependency.LockFileName)

	// Catalog tools that declare a gate: section in their tool.yaml
	// (cookies / mcp.entry) need to land under ~/.spwn/gate/tools/
	// for the gate daemon to pick them up. Do this transparently as
	// part of `spwn install` so users don't have to copy files by
	// hand. Lockfile + agent.yaml are already updated; this is purely
	// the "make the gate aware" half.
	if parsed.Kind == refs.KindSpwnBuiltin {
		gateToolsRoot, err := platform.GateToolsDir()
		if err == nil {
			gateSlugs, gerr := dependency.CopyGateTools(parsed.Name, gateToolsRoot)
			if gerr != nil {
				fmt.Fprintf(out, "  %s  warning: gate-tool sync: %v\n", ui.Strong("!"), gerr)
			} else if len(gateSlugs) > 0 {
				fmt.Fprintf(out, "  %s  gate tool(s) installed: %s\n", ui.Green("\u2713"), strings.Join(gateSlugs, ", "))
				fmt.Fprintf(out, "     restart the gate to load: %s\n", ui.Strong("spwn gate restart"))
			}
		}
	}
	return nil
}

// RunUninstall mirrors RunInstall: removes the ref from each
// targeted agent's agent.yaml, and from the lockfile when no agent
// still carries the ref.
//
// agentFilter narrows scope the same way as RunInstall: empty
// targets every agent, non-empty targets one.
func RunUninstall(cmd *cobra.Command, raw, agentFilter string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	// Symmetry with install: run the input through the CLI resolver
	// so `spwn uninstall python` finds `spwn:python` in the manifest
	// just like `spwn install python` would have written it there.
	// Explicit schemes pass through unchanged.
	resolved, rerr := refs.ResolveCLI(raw, knownCatalogNames())
	if rerr != nil {
		return rerr
	}

	ref, _ := refs.SplitVersion(resolved)
	parsed := refs.ParseRef(ref)
	if parsed.Kind == refs.KindRegistry {
		return fmt.Errorf("%q is a registry ref; nothing to uninstall", raw)
	}
	if parsed.Kind == refs.KindInvalid {
		return fmt.Errorf("%q is not a valid dependency ref — use spwn:<name>, github:<owner>/<repo>, skill/<name>, tool/<name>, or hook/<name>", raw)
	}
	// Local refs are authorable in-repo but also attachable via
	// `install skill/foo --agent mark` — so uninstall accepts them
	// Symmetrically. The underlying file is left alone.

	targets, terr := filterAgents(p.Agents, agentFilter)
	if terr != nil {
		return terr
	}
	mutated := 0
	for _, a := range targets {
		if err := agent.RemoveDependency(a.Name, ref); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := dependency.LoadLockfileOrEmpty(p.Root)
	if err != nil {
		return err
	}
	// Only drop the lockfile entry when no agent still carries the
	// Ref. This matters for scoped uninstalls: `uninstall python
	// --agent mark` shouldn't nuke python's lockfile pin if dylan
	// Still depends on it.
	if !anyAgentCarriesRef(p.Agents, ref) {
		lock.Remove(ref)
	}
	if err := dependency.SaveLockfile(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("uninstalled "+ref))
	fmt.Fprintf(out, "     %d agent%s updated\n", mutated, plural(mutated))
	return nil
}

// filterAgents narrows the project's agent list to the set targeted
// by an install/uninstall run. An empty filter returns every agent
// that exists on disk; a non-empty filter returns exactly the named
// agent or errors with a list of valid names so the user can
// correct a typo.
func filterAgents(all []project.AgentRef, filter string) ([]project.AgentRef, error) {
	available := make([]project.AgentRef, 0, len(all))
	for _, a := range all {
		if a.Exists {
			available = append(available, a)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no agents declared in this project — add one with `spwn agent new`")
	}
	if filter == "" {
		return available, nil
	}
	for _, a := range available {
		if a.Name == filter {
			return []project.AgentRef{a}, nil
		}
	}
	names := make([]string, len(available))
	for i, a := range available {
		names[i] = a.Name
	}
	return nil, fmt.Errorf("agent %q is not in this project.\nknown: %s", filter, strings.Join(names, ", "))
}

// anyAgentCarriesRef reports whether any agent still carries the
// given ref in its agent.yaml after a scoped uninstall. Used to
// decide whether the lockfile pin should be dropped.
func anyAgentCarriesRef(all []project.AgentRef, ref string) bool {
	for _, a := range all {
		if !a.Exists {
			continue
		}
		m, err := agent.LoadManifest(a.Name)
		if err != nil || m == nil {
			continue
		}
		for _, d := range m.Deps {
			if d == ref {
				return true
			}
		}
	}
	return false
}

// findProject is an alias over cliproject.Require so the install
// verbs stay concise. The canonical walker lives in apps/cli/cliproject.
func findProject() (*project.Project, error) {
	return cliproject.Require()
}

// catalogLookup checks whether `ref` is a known spwn:* dependency.
// The catalog list is supplied by the parent CLI — this package
// doesn't import spwn.sh/catalog to avoid a cycle.
var (
	catalogLookup func(ref string) bool
	catalogNames  func() []string
)

// SetCatalogLookup wires the built-in catalog into the install verbs.
// Called from the CLI entrypoint to avoid a direct catalog import.
// Pass both a membership predicate and a names-list supplier so the
// bare-name resolver can surface a known-list hint on miss.
func SetCatalogLookup(has func(ref string) bool, names func() []string) {
	catalogLookup = has
	catalogNames = names
}

func catalogHas(ref string) bool {
	if catalogLookup == nil {
		return true // permissive when no catalog wired — fallback for tests
	}
	return catalogLookup(ref)
}

func knownCatalogNames() []string {
	if catalogNames == nil {
		return nil
	}
	return catalogNames()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
