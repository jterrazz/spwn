package automation

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/project"
)

// lsCmd renders every automation in the current project's spwn.yaml
// as a table. The renderer reads only the manifest + the on-disk
// receipt log to derive last-fired times — no architect / Docker
// roundtrips, so the command stays useful even when the daemon is
// not running.
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List every automation declared in spwn.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := cliproject.Require()
		if err != nil {
			return err
		}

		entries := collectAutomations(p)
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no automations declared in this project")
			fmt.Fprintln(cmd.OutOrStdout(), "see docs/automations.md to add one under worlds.<name>.automations")
			return nil
		}

		// Receipts file is optional — projects that have never run
		// the daemon won't have one yet, the column just shows "—".
		lastFired, _ := readLastFiredFromReceipts(p.Root)

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "WORLD\tNAME\tTRIGGER\tAGENT\tLAST FIRED")
		for _, e := range entries {
			lf := "—"
			if t, ok := lastFired[e.World+"/"+e.Name]; ok {
				lf = humanTimeAgo(t)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.World, e.Name, formatTrigger(e.Auto), e.Auto.Agent, lf)
		}
		return w.Flush()
	},
}

// listEntry pairs a world key with one automation under it. Stable
// sort key used by collectAutomations.
type listEntry struct {
	World string
	Name  string
	Auto  project.Automation
}

// collectAutomations flattens manifest.Worlds[*].Automations into a
// stable-sorted slice for rendering.
func collectAutomations(p *project.Project) []listEntry {
	var out []listEntry
	if p == nil || p.Manifest == nil {
		return nil
	}
	for wname, w := range p.Manifest.Worlds {
		for aname, a := range w.Automations {
			out = append(out, listEntry{World: wname, Name: aname, Auto: a})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].World != out[j].World {
			return out[i].World < out[j].World
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// formatTrigger turns an Automation's trigger into a one-line label.
// Cron exprs render as the expression; fs triggers render as
// "fs:<path>" — both readable in a 80-column table.
func formatTrigger(a project.Automation) string {
	switch {
	case a.On.Cron != "":
		return "cron " + a.On.Cron
	case a.On.FS != nil:
		return "fs " + a.On.FS.Path
	default:
		return "?"
	}
}

// humanTimeAgo renders a duration as "5m ago" / "2h ago" / "3d ago".
// Same idiom git uses; agents reading the table parse it the same way
// users do.
func humanTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
