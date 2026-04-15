// Package tool implements the `spwn package` command group — managing
// reusable packages (tools, plugins, skills unified under one concept).
//
// Packages are declared in each agent's agent.yaml#packages list and
// pinned in the project's spwn.lock.yaml. The shape is deliberately
// npm-ish:
//
//   - @spwn/<name> is a catalog package compiled into the spwn binary.
//     `spwn package install @spwn/unix` adds it to every agent's
//     agent.yaml and records the pin in the lockfile.
//   - <bare-name> is a local package authored under
//     spwn/packages/<name>/ (directory form) or spwn/packages/<name>.md
//     (bare-markdown skill form). The install verb rejects bare names
//     with a hint — they are not "installed", they are authored.
//   - @<owner>/<name> (owner != spwn) is a future community-registry
//     ref, currently rejected as unsupported.
package tool

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/project"
	"spwn.sh/packages/project/lockfile"
	"spwn.sh/packages/project/refs"
)

// Cmd is the root `spwn package` command group.
var Cmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"pkg"},
	Short:   "Manage reusable packages (e.g. @spwn/unix, @spwn/mempalace)",
	Long: `Packages are the unified building blocks that agents plug into their worlds:
tools, plugins, and skills all share one schema.

Install a catalog package into the project's agents + lockfile with:
  spwn package install @spwn/python

Remove it with:
  spwn package uninstall @spwn/python

List what's installed with:
  spwn package ls

Local packages authored under spwn/packages/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)

	Cmd.SetHelpFunc(packageHelp)
}

func packageHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "package" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ package")+" "+ui.Faint("- reusable packages for agents"),
		[]ui.HelpGroup{
			{Title: "Manage", Commands: []ui.HelpEntry{
				{Name: "install <ref>", Desc: "Add a package to every agent + lockfile"},
				{Name: "uninstall <ref>", Desc: "Remove a package from every agent + lockfile"},
				{Name: "ls", Desc: "List installed packages"},
				{Name: "show <pack>", Desc: "Inspect a package"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn package install @spwn/python", Desc: ""},
				{Name: "spwn pkg install @spwn/mempalace", Desc: "Short alias"},
				{Name: "spwn package ls", Desc: "What's pinned in the lockfile"},
			}},
		},
		"spwn package [command]",
		"",
	)
}

// lsCmd reads the project lockfile and prints what's pinned.
var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := findProject()
		if err != nil {
			return err
		}
		lock, err := lockfile.LoadOrEmpty(p.Root)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		refs := lock.Refs()
		if len(refs) == 0 {
			fmt.Fprintln(out, "No packages installed.")
			fmt.Fprintln(out, "Install one with 'spwn package install @spwn/<name>'.")
			return nil
		}
		fmt.Fprintln(out, "Installed packages:")
		for _, r := range refs {
			e := lock.Packages[r]
			version := e.Version
			if version == "" {
				version = "-"
			}
			fmt.Fprintf(out, "  %-24s  %s  %s\n", r, version, e.Source)
		}
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <package>",
	Short: "Inspect a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := findProject()
		if err != nil {
			return err
		}
		lock, err := lockfile.LoadOrEmpty(p.Root)
		if err != nil {
			return err
		}
		ref := args[0]
		e, ok := lock.Packages[ref]
		if !ok {
			return fmt.Errorf("%q is not recorded in %s", ref, lockfile.FileName)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n  version: %s\n  source:  %s\n", ref, e.Version, e.Source)
		return nil
	},
}

// installCmd adds a ref to every agent's agent.yaml#packages and
// pins it in the project lockfile. Bare names are rejected with a
// pointer to the local-package authoring flow.
var installCmd = &cobra.Command{
	Use:   "install <ref>",
	Short: "Install a package into the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args[0])
	},
}

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <ref>",
	Aliases: []string{"rm", "remove"},
	Short:   "Uninstall a package from the project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUninstall(cmd, args[0])
	},
}

// RunInstall parses the ref, rejects bare/registry refs with crisp
// hints, mutates every agent.yaml in the project, and updates the
// lockfile. Exported so sibling CLI commands (if any land in the
// future) can reuse the logic.
func RunInstall(cmd *cobra.Command, raw string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	pack, version := refs.SplitVersion(raw)
	ref := refs.Parse(pack)
	switch ref.Kind {
	case refs.KindLocal:
		return fmt.Errorf("%q is a bare name — local packages are authored in place, not installed. "+
			"Create ./spwn/packages/%s/package.yaml for a full package or ./spwn/packages/%s.md for a bare skill",
			pack, pack, pack)
	case refs.KindRegistry:
		return fmt.Errorf("%q targets @%s/%s — remote registries are not yet supported. "+
			"Use @spwn/<name> for built-in packages, or author a local package under ./spwn/packages/",
			raw, ref.Owner, ref.Name)
	}

	if !catalogHas(pack) {
		return fmt.Errorf("unknown builtin %q — run `spwn package ls` to see available packages", pack)
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
		if err := agent.AddPackage(a.Name, pack); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := lockfile.LoadOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Add(pack, lockfile.Entry{
		Version: version,
		Source:  lockfile.SourceBuiltin,
	})
	if err := lockfile.Save(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("installed "+pack))
	fmt.Fprintf(out, "     %d agent%s updated, %s pinned\n",
		mutated, plural(mutated), lockfile.FileName)
	return nil
}

// RunUninstall mirrors RunInstall: removes the ref from every
// agent.yaml and from the lockfile.
func RunUninstall(cmd *cobra.Command, raw string) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	pack, _ := refs.SplitVersion(raw)
	ref := refs.Parse(pack)
	if ref.Kind == refs.KindRegistry {
		return fmt.Errorf("%q is a registry ref; nothing to uninstall", raw)
	}
	if ref.Kind == refs.KindLocal {
		return fmt.Errorf("%q is a bare name — delete ./spwn/packages/%s/ (or %s.md) by hand to remove it", pack, pack, pack)
	}

	mutated := 0
	for _, a := range p.Agents {
		if !a.Exists {
			continue
		}
		if err := agent.RemovePackage(a.Name, pack); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := lockfile.LoadOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Remove(pack)
	if err := lockfile.Save(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("uninstalled "+pack))
	fmt.Fprintf(out, "     %d agent%s updated\n", mutated, plural(mutated))
	return nil
}

// FindProject walks up from cwd looking for spwn.yaml and loads it.
// Exported for sibling CLI packages that need the same "find-or-fail"
// boilerplate.
func FindProject() (*project.Project, error) {
	return findProject()
}

func findProject() (*project.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}
	p, err := project.Find(cwd)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no spwn.yaml found — run `spwn init` first")
	}
	return p, nil
}

// catalogHas checks whether `pack` is a known @spwn/* ref. The
// catalog list is supplied by the parent CLI — this package doesn't
// import spwn.sh/catalog to avoid a cycle.
var catalogLookup func(pack string) bool

// SetCatalogLookup wires the built-in catalog into the install verbs.
// Called from the CLI entrypoint to avoid a direct catalog import.
func SetCatalogLookup(f func(pack string) bool) {
	catalogLookup = f
}

func catalogHas(pack string) bool {
	if catalogLookup == nil {
		return true // permissive when no catalog wired — fallback for tests
	}
	return catalogLookup(pack)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
