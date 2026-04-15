// Package tool implements the `spwn tool` command group — managing
// reusable tool packs (e.g. @spwn/unix, @spwn/python).
//
// Tools are declared in each agent's agent.yaml#tools list and pinned
// in the project's spwn.lock.yaml. The shape is deliberately npm-ish:
//
//   - @spwn/<name> is a catalog pack compiled into the spwn binary.
//     `spwn tool install @spwn/unix` adds it to every agent's agent.yaml
//     and records the pin in the lockfile.
//   - <bare-name> is a local pack authored under spwn/tools/<name>/.
//     The install verb rejects bare names with a hint — they are not
//     "installed", they are authored.
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

// Cmd is the root `spwn tool` command group.
var Cmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage reusable tool packs (e.g. @spwn/unix, @spwn/python)",
	Long: `Tool packs are composable building blocks that agents plug into their worlds.

Install a catalog pack into the project's agents + lockfile with:
  spwn tool install @spwn/python

Remove it with:
  spwn tool uninstall @spwn/python

List what's installed with:
  spwn tool ls

Local tool packs authored under spwn/tools/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(uninstallCmd)

	Cmd.SetHelpFunc(toolHelp)
}

func toolHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "tool" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ tool")+" "+ui.Faint("- reusable tool packs for agents"),
		[]ui.HelpGroup{
			{Title: "Manage", Commands: []ui.HelpEntry{
				{Name: "install <ref>", Desc: "Add a pack to every agent + lockfile"},
				{Name: "uninstall <ref>", Desc: "Remove a pack from every agent + lockfile"},
				{Name: "ls", Desc: "List installed tool packs"},
				{Name: "show <pack>", Desc: "Inspect a tool pack"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn tool install @spwn/python", Desc: ""},
				{Name: "spwn tool uninstall @spwn/git", Desc: ""},
				{Name: "spwn tool ls", Desc: "What's pinned in the lockfile"},
			}},
		},
		"spwn tool [command]",
		"",
	)
}

// lsCmd reads the project lockfile and prints what's pinned.
var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List installed tool packs",
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
		refs := lock.RefsIn(lockfile.KindTool)
		if len(refs) == 0 {
			fmt.Fprintln(out, "No tool packs installed.")
			fmt.Fprintln(out, "Install one with 'spwn tool install @spwn/<name>'.")
			return nil
		}
		fmt.Fprintln(out, "Installed tool packs:")
		for _, r := range refs {
			e := lock.Tools[r]
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
	Use:   "show <tool-pack>",
	Short: "Inspect a tool pack",
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
		e, ok := lock.Tools[ref]
		if !ok {
			return fmt.Errorf("%q is not recorded in %s", ref, lockfile.FileName)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n  version: %s\n  source:  %s\n", ref, e.Version, e.Source)
		return nil
	},
}

// installCmd adds a ref to every agent's agent.yaml#tools and pins it
// in the project lockfile. Bare names are rejected with a pointer to
// the local-pack authoring flow.
var installCmd = &cobra.Command{
	Use:   "install <ref>",
	Short: "Install a tool pack into the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args[0], lockfile.KindTool, installTool)
	},
}

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <ref>",
	Aliases: []string{"rm", "remove"},
	Short:   "Uninstall a tool pack from the project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUninstall(cmd, args[0], lockfile.KindTool, uninstallTool)
	},
}

func installTool(agentName, ref string) error  { return agent.AddTool(agentName, ref) }
func uninstallTool(agentName, ref string) error { return agent.RemoveTool(agentName, ref) }

// RunInstall is shared between tool + plugin + skill install verbs.
// Parses the ref, rejects bare/registry refs with verb-specific hints,
// mutates every agent.yaml in the project, and updates the lockfile.
func RunInstall(cmd *cobra.Command, raw string, kind lockfile.Kind, mutate func(agentName, ref string) error) error {
	p, err := findProject()
	if err != nil {
		return err
	}

	pack, version := refs.SplitVersion(raw)
	ref := refs.Parse(pack)
	switch ref.Kind {
	case refs.KindLocal:
		return fmt.Errorf("%q is a bare name — local %s are authored in place, not installed. "+
			"Create the directory under ./spwn/%s/%s/ and reference it from agent.yaml",
			pack, kindPlural(kind), kindDir(kind), pack)
	case refs.KindRegistry:
		return fmt.Errorf("%q targets @%s/%s — remote registries are not yet supported. "+
			"Use @spwn/<name> for built-in packs, or author a local pack under ./spwn/%s/",
			raw, ref.Owner, ref.Name, kindDir(kind))
	}

	// @spwn/<name> — verify it's in the catalog before touching anything.
	if kind != lockfile.KindSkill {
		if !catalogHas(pack, kind) {
			return fmt.Errorf("unknown builtin %q — run `spwn tool ls` to see available packs", pack)
		}
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
		if err := mutate(a.Name, pack); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := lockfile.LoadOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Add(kind, pack, lockfile.Entry{
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

func RunUninstall(cmd *cobra.Command, raw string, kind lockfile.Kind, mutate func(agentName, ref string) error) error {
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
		return fmt.Errorf("%q is a bare name — delete ./spwn/%s/%s/ by hand if you want to remove it", pack, kindDir(kind), pack)
	}

	mutated := 0
	for _, a := range p.Agents {
		if !a.Exists {
			continue
		}
		if err := mutate(a.Name, pack); err != nil {
			return fmt.Errorf("update %s: %w", a.Name, err)
		}
		mutated++
	}

	lock, err := lockfile.LoadOrEmpty(p.Root)
	if err != nil {
		return err
	}
	lock.Remove(kind, pack)
	if err := lockfile.Save(p.Root, lock); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  %s  %s\n", ui.Green("\u2713"), ui.Strong("uninstalled "+pack))
	fmt.Fprintf(out, "     %d agent%s updated\n", mutated, plural(mutated))
	return nil
}

// FindProject walks up from cwd looking for spwn.yaml and loads it.
// Exported so sibling CLI packages (skill, plugin) can reuse the
// "find-or-fail" boilerplate without duplicating the error shape.
func FindProject() (*project.Project, error) {
	return findProject()
}

// findProject walks up from cwd looking for spwn.yaml and loads it.
// Used by every install/uninstall verb.
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
// catalog list is supplied by the parent CLI — tool.go doesn't import
// spwn.sh/catalog to avoid a cycle. The lookup function is set at
// package init by apps/cli/cmd/spwn.
var catalogLookup func(pack string, kind lockfile.Kind) bool

// SetCatalogLookup wires the built-in catalog into the install verbs.
// Called from the CLI entrypoint to avoid a direct catalog import.
func SetCatalogLookup(f func(pack string, kind lockfile.Kind) bool) {
	catalogLookup = f
}

func catalogHas(pack string, kind lockfile.Kind) bool {
	if catalogLookup == nil {
		return true // permissive when no catalog wired — fallback
	}
	return catalogLookup(pack, kind)
}

func kindPlural(k lockfile.Kind) string {
	switch k {
	case lockfile.KindTool:
		return "tools"
	case lockfile.KindPlugin:
		return "plugins"
	case lockfile.KindSkill:
		return "skills"
	}
	return ""
}

func kindDir(k lockfile.Kind) string {
	switch k {
	case lockfile.KindTool, lockfile.KindPlugin:
		return "tools"
	case lockfile.KindSkill:
		return "skills"
	}
	return "tools"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
