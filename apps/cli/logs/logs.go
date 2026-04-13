// Package logs implements `spwn logs` — the system-wide semantic event log.
// It tails ~/.spwn/activity.jsonl and is the top of a small family of
// per-scope event viewers (spwn world logs, spwn agent logs, spwn architect logs).
package logs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"spwn.sh/apps/cli/ui"
	coreactivity "spwn.sh/packages/foundation/activity"

	"github.com/spf13/cobra"
)

// Cmd is the root `spwn logs` command.
var Cmd = &cobra.Command{
	Use:   "logs",
	Short: "Show the system event log across worlds and agents",
	Long: `Show the spwn event log — spawned worlds, created agents, dream cycles,
snapshots, messages, and every other discrete thing that happened.

Scope it with --world or --agent, or use the per-entity shortcuts:
  spwn world logs <id>        events for one world
  spwn agent logs <name>      events for one agent
  spwn architect logs         architect daemon events`,
	RunE: runLogs,
}

var (
	limit  int
	filter string
	world  string
	agent  string
)

func init() {
	Cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of events to show")
	Cmd.Flags().StringVarP(&filter, "type", "t", "", "Filter by event type (e.g. agent.dreamed)")
	Cmd.Flags().StringVarP(&world, "world", "w", "", "Filter by world ID")
	Cmd.Flags().StringVarP(&agent, "agent", "a", "", "Filter by agent name")
}

func runLogs(cmd *cobra.Command, args []string) error {
	return Run(cmd, RunOpts{
		Limit:   limit,
		Type:    filter,
		WorldID: world,
		AgentID: agent,
	})
}

// RunOpts is the parameter set for Run, shared with scoped subcommands in
// other packages (world/agent/architect) that want to reuse the rendering.
type RunOpts struct {
	Limit   int
	Type    string
	WorldID string
	AgentID string
	Actor   string
}

// Run reads the event log with the given filters and renders it to cmd's
// output. Scoped subcommands call this with a preset WorldID/AgentID.
func Run(cmd *cobra.Command, opts RunOpts) error {
	if opts.Limit == 0 {
		opts.Limit = 20
	}
	events, err := coreactivity.Read(coreactivity.ReadOpts{
		Limit:   opts.Limit,
		Type:    coreactivity.Type(opts.Type),
		WorldID: opts.WorldID,
		AgentID: opts.AgentID,
		Actor:   opts.Actor,
	})
	if err != nil {
		return fmt.Errorf("reading event log: %w", err)
	}

	if len(events) == 0 {
		fmt.Fprintln(os.Stdout, ui.Faint("No events yet."))
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
