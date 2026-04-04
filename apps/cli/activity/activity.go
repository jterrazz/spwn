package activity

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"spwn.sh/apps/cli/ui"
	coreactivity "spwn.sh/core/foundation/activity"

	"github.com/spf13/cobra"
)

// Cmd is the root `spwn activity` command.
var Cmd = &cobra.Command{
	Use:   "activity",
	Short: "View recent activity across worlds and agents",
	Long:  "Tails the system-wide activity log at ~/.spwn/activity.jsonl",
	RunE:  runActivity,
}

var (
	limit   int
	filter  string
	world   string
	agent   string
	jsonOut bool
)

func init() {
	Cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of events to show")
	Cmd.Flags().StringVarP(&filter, "type", "t", "", "Filter by event type (e.g. agent.dreamed)")
	Cmd.Flags().StringVarP(&world, "world", "w", "", "Filter by world ID")
	Cmd.Flags().StringVarP(&agent, "agent", "a", "", "Filter by agent name")
	Cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
}

func runActivity(cmd *cobra.Command, args []string) error {
	events, err := coreactivity.Read(coreactivity.ReadOpts{
		Limit:   limit,
		Type:    coreactivity.Type(filter),
		WorldID: world,
		AgentID: agent,
	})
	if err != nil {
		return fmt.Errorf("reading activity log: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"events": events})
	}

	if len(events) == 0 {
		fmt.Fprintln(os.Stdout, ui.Faint("No activity yet."))
		return nil
	}

	// Reverse so newest is at the bottom (terminal-friendly)
	for i := len(events) - 1; i >= 0; i-- {
		printEvent(events[i])
	}
	return nil
}

func printEvent(e coreactivity.Event) {
	ts := e.Timestamp.Local().Format("15:04:05")
	color := ui.Faint
	switch {
	case e.Type == coreactivity.TypeWorldSpawned || e.Type == coreactivity.TypeArchitectStarted:
		color = ui.Green
	case e.Type == coreactivity.TypeWorldDestroyed || e.Type == coreactivity.TypeAgentDeleted || e.Type == coreactivity.TypeArchitectStopped:
		color = ui.Red
	case e.Type == coreactivity.TypeAgentDreamed || e.Type == coreactivity.TypeAgentSlept:
		color = ui.Yellow
	case e.Type == coreactivity.TypeAgentJoined || e.Type == coreactivity.TypeAgentCreated || e.Type == coreactivity.TypeAgentForked:
		color = ui.Cyan
	}

	meta := ""
	if e.CostUSD > 0 || e.DurationMs > 0 {
		parts := []string{}
		if e.DurationMs > 0 {
			parts = append(parts, formatDuration(time.Duration(e.DurationMs)*time.Millisecond))
		}
		if e.CostUSD > 0 {
			parts = append(parts, "$"+strconv.FormatFloat(e.CostUSD, 'f', 3, 64))
		}
		meta = " " + ui.Faint("("+joinComma(parts)+")")
	}

	fmt.Fprintf(os.Stdout, "  %s  %s  %s%s\n",
		ui.Faint(ts),
		color(string(e.Type)),
		e.Phrase,
		meta,
	)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func joinComma(parts []string) string {
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += ", "
		}
		s += p
	}
	return s
}
