// Package dependency implements the project-wide install/uninstall
// logic behind `spwn install` and `spwn uninstall`. It mutates every
// agent's agent.yaml#dependencies list and keeps spwn.lock pinned.
//
// Reference kinds (delegated to packages/dependency):
//
//   - spwn:<name>                built-in catalog (compiled into the binary)
//   - github:<owner>/<repo>      third-party (planned — git tags as versions)
//   - skill:<name>               local skill (./spwn/skills/<name>.md)
//   - tool:<name>                local tool (./spwn/tools/<name>/)
//   - hook:<name>                local hook (./spwn/hooks/<name>.sh)
//
// Local refs (skill:/tool:/hook:) are never installed — they are
// authored in place. The install verb rejects them with a hint
// pointing at the local authoring flow.
package dependency

import (
	"fmt"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/project"
)

// installCmd and uninstallCmd are retained so the dependency_test
// suite can drive the install/uninstall flow directly without going
// through the top-level root command.
var installCmd = &cobra.Command{
	Use:   "install <ref>",
	Short: "Install a dependency into the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args[0])
	},
}

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <ref>",
	Aliases: []string{"rm", "remove"},
	Short:   "Uninstall a dependency from the project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUninstall(cmd, args[0])
	},
}

// RunInstall parses the ref, rejects bare/registry refs with crisp
// hints, mutates every agent.yaml in the project, and updates the
// lockfile.
func RunInstall(cmd *cobra.Command, raw string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	// Run the raw input through the shared CLI resolver so bare names
	// auto-promote to spwn:<name> ("spwn install qmd" → "spwn install
	// spwn:qmd"). Explicit schemes pass through unchanged. Manifests
	// always receive the canonical scheme-form.
	resolved, rerr := dependency.ResolveCLI(raw, knownCatalogNames())
	if rerr != nil {
		return rerr
	}

	ref, version := dependency.SplitVersion(resolved)
	parsed := dependency.ParseRef(ref)
	switch parsed.Kind {
	case dependency.KindLocalSkill, dependency.KindLocalTool, dependency.KindLocalHook:
		return fmt.Errorf("%q is a local ref — local dependencies are authored in place, not installed. "+
			"Use `spwn skill new %s` (for skill: refs), drop a directory at ./spwn/tools/%s/ (for tool: refs), "+
			"or author ./spwn/hooks/%s.sh (for hook: refs) instead",
			ref, parsed.Name, parsed.Name, parsed.Name)
	case dependency.KindRegistry:
		return fmt.Errorf("%q targets github:%s/%s — remote registries are not yet supported. "+
			"Use spwn:<name> for built-in dependencies, or author a local one under ./spwn/tools/",
			raw, parsed.Owner, parsed.Name)
	case dependency.KindInvalid:
		return fmt.Errorf("%q is not a valid dependency ref — use spwn:<name> (for built-ins), "+
			"github:<owner>/<repo> (for remote deps), skill:<name>, tool:<name>, or hook:<name>",
			raw)
	}

	if !catalogHas(ref) {
		return fmt.Errorf("unknown builtin %q — see the catalog for available dependencies", ref)
	}

	agents := p.Agents
	if len(agents) == 0 {
		return fmt.Errorf("no agents declared in this project — add one with `spwn agent new`")
	}
	mutated := 0
	for _, a := range agents {
		if !a.Exists {
			continue
		}
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
	return nil
}

// RunUninstall mirrors RunInstall: removes the ref from every
// agent.yaml and from the lockfile.
func RunUninstall(cmd *cobra.Command, raw string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	// Symmetry with install: run the input through the CLI resolver
	// so `spwn uninstall python` finds `spwn:python` in the manifest
	// just like `spwn install python` would have written it there.
	// Explicit schemes pass through unchanged.
	resolved, rerr := dependency.ResolveCLI(raw, knownCatalogNames())
	if rerr != nil {
		return rerr
	}

	ref, _ := dependency.SplitVersion(resolved)
	parsed := dependency.ParseRef(ref)
	if parsed.Kind == dependency.KindRegistry {
		return fmt.Errorf("%q is a registry ref; nothing to uninstall", raw)
	}
	if dependency.IsLocalKind(parsed.Kind) {
		return fmt.Errorf("%q is a local ref — delete the underlying file (spwn/skills/%s.md, spwn/tools/%s/, or spwn/hooks/%s.sh) by hand to remove it", ref, parsed.Name, parsed.Name, parsed.Name)
	}
	if parsed.Kind == dependency.KindInvalid {
		return fmt.Errorf("%q is not a valid dependency ref — use spwn:<name>, github:<owner>/<repo>, skill:<name>, tool:<name>, or hook:<name>", raw)
	}

	mutated := 0
	for _, a := range p.Agents {
		if !a.Exists {
			continue
		}
		if err := agent.RemoveDependency(a.Name, ref); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := dependency.LoadLockfileOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Remove(ref)
	if err := dependency.SaveLockfile(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("uninstalled "+ref))
	fmt.Fprintf(out, "     %d agent%s updated\n", mutated, plural(mutated))
	return nil
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
