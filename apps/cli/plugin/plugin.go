// Package plugin implements the `spwn plugin` command group - management
// of runtime-targeted tool packs. Plugins are sugar for bundling existing
// primitives with runtime config injection (e.g. mempalace plugging MCP
// settings into Claude Code).
package plugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	plugins "spwn.sh/packages/catalog/plugins"
	"spwn.sh/apps/cli/ui"
	ib "spwn.sh/packages/image"
)

// Cmd is the root `spwn plugin` command group.
var Cmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage runtime-targeted plugin packs (e.g. @spwn/mempalace)",
	Long: `Plugin packs are tool packs that target specific runtimes and inject
configuration into the runtime at spawn time (e.g. MCP servers into
Claude Code's settings.json).

Attach one to an agent with:
  spwn agent add <agent> --plugin <pack>

Plugins coexist with --tool in the agent manifest. Both lists resolve
through the same builder registry, so plugins see the full tool
dependency graph and vice-versa.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(getCmd)

	Cmd.SetHelpFunc(pluginHelp)
}

func pluginHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "plugin" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ plugin")+" "+ui.Faint("- runtime-targeted packs for agents"),
		[]ui.HelpGroup{
			{Title: "Manage", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List built-in plugin packs"},
				{Name: "show <pack>", Desc: "Inspect a plugin pack"},
			}},
			{Title: "Registry", Commands: []ui.HelpEntry{
				{Name: "get <pack>", Desc: "Install a shared plugin " + ui.Faint("[planned]")},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn plugin ls", Desc: "See every built-in plugin"},
				{Name: "spwn agent add neo --plugin @spwn/mempalace", Desc: ""},
			}},
		},
		"spwn plugin [command]",
		"",
	)
}

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List installed plugin packs",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		if len(plugins.All) == 0 {
			fmt.Fprintln(w, "No plugins registered.")
			return nil
		}
		rows := make([][4]string, 0, len(plugins.All))
		for _, p := range plugins.All {
			runtimes := "-"
			if rs := ib.PluginRuntimes(p); len(rs) > 0 {
				runtimes = strings.Join(rs, ",")
			}
			rows = append(rows, [4]string{p.Name(), p.Version(), runtimes, briefDesc(p)})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
		fmt.Fprintln(w, "Built-in plugin packs:")
		for _, r := range rows {
			fmt.Fprintf(w, "  %-22s %-8s %-24s %s\n", r[0], r[1], r[2], r[3])
		}
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <plugin>",
	Short: "Inspect a plugin pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var p ib.Tool
		for _, candidate := range plugins.All {
			if candidate.Name() == name {
				p = candidate
				break
			}
		}
		if p == nil {
			return fmt.Errorf("plugin %q not found (try `spwn plugin ls`)", name)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Name:         %s\n", p.Name())
		fmt.Fprintf(w, "Kind:         %s\n", p.Kind())
		fmt.Fprintf(w, "Version:      %s\n", p.Version())
		fmt.Fprintf(w, "Dependencies: %s\n", joinOrDash(p.Dependencies()))
		fmt.Fprintf(w, "Runtimes:     %s\n", joinOrDash(ib.PluginRuntimes(p)))
		spec := p.Install()
		fmt.Fprintf(w, "Packages:     %d\n", len(spec.Packages))
		fmt.Fprintf(w, "Commands:     %d\n", len(spec.Commands))
		fmt.Fprintf(w, "Verify:       %d cmd(s)\n", len(p.Verify()))
		// Config exposure
		for _, r := range ib.PluginRuntimes(p) {
			has := "no"
			if ib.PluginConfig(p, r) != nil {
				has = "yes"
			}
			fmt.Fprintf(w, "Config[%s]: %s\n", r, has)
		}
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <plugin>",
	Short: "Install a plugin pack from the registry [planned]",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "install %q: the plugin registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "Built-in plugin packs (@spwn/*) are always available - no install needed.")
		return nil
	},
}

func init() {
	ui.MarkExperimental(getCmd)
}

func joinOrDash(s []string) string {
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, ", ")
}

func briefDesc(t ib.Tool) string {
	// Plugins don't ship a description field on Tool; derive a short
	// hint from kind + runtimes.
	runtimes := ib.PluginRuntimes(t)
	if len(runtimes) == 0 {
		return string(t.Kind())
	}
	return fmt.Sprintf("%s → %s", t.Kind(), strings.Join(runtimes, ","))
}
