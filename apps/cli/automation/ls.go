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
			fmt.Fprintln(cmd.OutOrStdout(), "see docs/guides/automations.md to add one under worlds.<name>.automations")
			return nil
		}

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

type listEntry struct {
	World string
	Name  string
	Auto  project.Automation
}

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
