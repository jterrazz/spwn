// Package plugin implements the `spwn plugin` command group — managing
// the spwn plugin system — the unified installable concept (formerly "package").
//
// Packages are declared in each agent's agent.yaml#plugins list and
// pinned in the project's spwn.lock.yaml. The shape is deliberately
// npm-ish:
//
//   - @spwn/<name> is a catalog plugin compiled into the spwn binary.
//     `spwn plugin install @spwn/unix` adds it to every agent's
//     agent.yaml and records the pin in the lockfile.
//   - <bare-name> is a local plugin authored under
//     spwn/plugins/<name>/ (directory form) or spwn/plugins/<name>.md
//     (bare-markdown skill form). The install verb rejects bare names
//     with a hint — they are not "installed", they are authored.
//   - @<owner>/<name> (owner != spwn) is a future community-registry
//     ref, currently rejected as unsupported.
package plugin

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

// Cmd is the root `spwn plugin` command group.
var Cmd = &cobra.Command{
	Use:     "plugin",
	Aliases: []string{"pkgs", "plugins"},
	Short:   "Manage plugins (e.g. @spwn/unix, @spwn/mempalace)",
	Long: `Plugins are the unified building blocks that agents plug into their worlds.
One schema covers what used to be split between tools, runtime-config providers, and skills.

Install a catalog plugin into the project's agents + lockfile with:
  spwn plugin install @spwn/python

Remove it with:
  spwn plugin uninstall @spwn/python

List what's installed with:
  spwn plugin ls

Local plugins authored under spwn/plugins/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)

	Cmd.SetHelpFunc(pluginHelp)
}

func pluginHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "plugin" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ plugin")+" "+ui.Faint("- reusable plugins for agents"),
		[]ui.HelpGroup{
			{Title: "Manage", Commands: []ui.HelpEntry{
				{Name: "install <ref>", Desc: "Add a plugin to every agent + lockfile"},
				{Name: "uninstall <ref>", Desc: "Remove a plugin from every agent + lockfile"},
				{Name: "ls", Desc: "List installed plugins"},
				{Name: "show <plugin>", Desc: "Inspect a plugin"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn plugin install @spwn/python", Desc: ""},
				{Name: "spwn plugin uninstall @spwn/python", Desc: "Remove it again"},
				{Name: "spwn plugin ls", Desc: "What's pinned in the lockfile"},
			}},
		},
		"spwn plugin [command]",
		"",
	)
}

// lsCmd reads the project lockfile and prints what's pinned.
var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List installed plugins",
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
			fmt.Fprintln(out, "No plugins installed.")
			fmt.Fprintln(out, "Install one with 'spwn plugin install @spwn/<name>'.")
			return nil
		}
		fmt.Fprintln(out, "Installed plugins:")
		for _, r := range refs {
			e := lock.Plugins[r]
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
	Use:   "show <plugin>",
	Short: "Inspect a plugin",
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
		e, ok := lock.Plugins[ref]
		if !ok {
			return fmt.Errorf("%q is not recorded in %s", ref, lockfile.FileName)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n  version: %s\n  source:  %s\n", ref, e.Version, e.Source)
		return nil
	},
}

// installCmd adds a ref to every agent's agent.yaml#plugins and
// pins it in the project lockfile. Bare names are rejected with a
// pointer to the local-plugin authoring flow.
var installCmd = &cobra.Command{
	Use:   "install <ref>",
	Short: "Install a plugin into the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args[0])
	},
}

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <ref>",
	Aliases: []string{"rm", "remove"},
	Short:   "Uninstall a plugin from the project",
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
		return fmt.Errorf("%q is a bare name — local plugins are authored in place, not installed. "+
			"Create ./spwn/plugins/%s/plugin.yaml for a full plugin or ./spwn/plugins/%s.md for a bare skill",
			pack, pack, pack)
	case refs.KindRegistry:
		return fmt.Errorf("%q targets @%s/%s — remote registries are not yet supported. "+
			"Use @spwn/<name> for built-in plugins, or author a local plugin under ./spwn/plugins/",
			raw, ref.Owner, ref.Name)
	}

	if !catalogHas(pack) {
		return fmt.Errorf("unknown builtin %q — run `spwn plugin ls` to see available plugins", pack)
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
		if err := agent.AddPlugin(a.Name, pack); err != nil {
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
		return fmt.Errorf("%q is a bare name — delete ./spwn/plugins/%s/ (or %s.md) by hand to remove it", pack, pack, pack)
	}

	mutated := 0
	for _, a := range p.Agents {
		if !a.Exists {
			continue
		}
		if err := agent.RemovePlugin(a.Name, pack); err != nil {
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
