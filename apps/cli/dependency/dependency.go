// Package dependency implements the project-wide install/uninstall
// logic behind `spwn install` and `spwn uninstall`. It mutates every
// agent's agent.yaml#dependencies list and keeps spwn.lock pinned.
//
// Reference kinds (delegated to packages/dependency):
//
//   - @spwn/<name>               built-in catalog (compiled into the binary)
//   - github.com/<owner>/<repo>  third-party (planned — git tags as versions)
//   - <bare-name>                local authoring (./spwn/tools/<name>/)
//
// Bare names are never installed — they are authored in place. The
// install verb rejects them with a hint pointing at the local flow.
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

	ref, version := dependency.SplitVersion(raw)
	parsed := dependency.ParseRef(ref)
	switch parsed.Kind {
	case dependency.KindLocal:
		return fmt.Errorf("%q is a bare name — local dependencies are authored in place, not installed. "+
			"Create ./spwn/tools/%s/tool.yaml for a full dependency or ./spwn/tools/%s.md for a bare skill",
			ref, ref, ref)
	case dependency.KindRegistry:
		return fmt.Errorf("%q targets @%s/%s — remote registries are not yet supported. "+
			"Use spwn:<name> for built-in dependencies, or author a local one under ./spwn/tools/",
			raw, parsed.Owner, parsed.Name)
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

	ref, _ := dependency.SplitVersion(raw)
	parsed := dependency.ParseRef(ref)
	if parsed.Kind == dependency.KindRegistry {
		return fmt.Errorf("%q is a registry ref; nothing to uninstall", raw)
	}
	if parsed.Kind == dependency.KindLocal {
		return fmt.Errorf("%q is a bare name — delete ./spwn/tools/%s/ (or %s.md) by hand to remove it", ref, ref, ref)
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

// catalogLookup checks whether `ref` is a known @spwn/* dependency.
// The catalog list is supplied by the parent CLI — this package
// doesn't import spwn.sh/catalog to avoid a cycle.
var catalogLookup func(ref string) bool

// SetCatalogLookup wires the built-in catalog into the install verbs.
// Called from the CLI entrypoint to avoid a direct catalog import.
func SetCatalogLookup(f func(ref string) bool) {
	catalogLookup = f
}

func catalogHas(ref string) bool {
	if catalogLookup == nil {
		return true // permissive when no catalog wired — fallback for tests
	}
	return catalogLookup(ref)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
