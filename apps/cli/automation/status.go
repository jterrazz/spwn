package automation

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Engine state and last-fired snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := cliproject.Require()
		if err != nil {
			return err
		}

		entries := collectAutomations(p)
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no automations declared in this project")
			return nil
		}

		rows, _ := readReceipts(p.Root)
		stats := summariseReceipts(rows)

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "WORLD/NAME\tFIRES\tOK\tFAIL\tLAST\tLAST RESULT")
		for _, e := range entries {
			key := e.World + "/" + e.Name
			s := stats[key]
			last := "—"
			result := "—"
			if !s.lastFired.IsZero() {
				last = humanTimeAgo(s.lastFired)
				result = "ok"
				if !s.lastOK {
					result = "FAIL"
				}
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\n",
				key, s.fires, s.okCount, s.failCount, last, result)
		}
		return w.Flush()
	},
}

type stats struct {
	fires     int
	okCount   int
	failCount int
	lastFired time.Time
	lastOK    bool
}

func summariseReceipts(rows []receiptRow) map[string]stats {
	out := map[string]stats{}
	for _, r := range rows {
		key := r.World + "/" + r.Automation
		s := out[key]
		s.fires++
		if r.OK {
			s.okCount++
		} else {
			s.failCount++
		}
		if r.Fired.After(s.lastFired) {
			s.lastFired = r.Fired
			s.lastOK = r.OK
		}
		out[key] = s
	}
	return out
}
