// Package logs implements `spwn logs` - the system-wide semantic event log.
// It tails ~/.spwn/activity.jsonl and is the top of a small family of
// per-scope event viewers (spwn world logs, spwn agent logs, spwn architect logs).
package logs

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	coreactivity "spwn.sh/packages/activity"

	"github.com/spf13/cobra"
)

// knownEventTypes is the curated set of event types that `--type`
// will accept. Values must match the constants defined in
// packages/activity/event.go. Kept local so the logs CLI can
// validate input early with a helpful "did you mean X?" error
// instead of silently returning "No events yet."
var knownEventTypes = []coreactivity.Type{
	coreactivity.TypeWorldSpawned,
	coreactivity.TypeWorldDestroyed,
	coreactivity.TypeWorldSnapshot,
	coreactivity.TypeWorldStateChange,
	coreactivity.TypeAgentCreated,
	coreactivity.TypeAgentDeleted,
	coreactivity.TypeAgentJoined,
	coreactivity.TypeAgentLeft,
	coreactivity.TypeAgentDreamed,
	coreactivity.TypeAgentSlept,
	coreactivity.TypeAgentForked,
	coreactivity.TypeAgentTalked,
	coreactivity.TypeArchitectStarted,
	coreactivity.TypeArchitectStopped,
	coreactivity.TypeArchitectTalked,
	coreactivity.TypeSessionEnded,
}

func isKnownEventType(t string) bool {
	for _, k := range knownEventTypes {
		if string(k) == t {
			return true
		}
	}
	return false
}

func sortedEventTypes() []string {
	out := make([]string, 0, len(knownEventTypes))
	for _, t := range knownEventTypes {
		out = append(out, string(t))
	}
	sort.Strings(out)
	return out
}

// Cmd is the root `spwn logs` command.
var Cmd = &cobra.Command{
	Use:   "logs",
	Short: "Show the system event log across worlds and agents",
	Long: `Show the spwn event log - spawned worlds, created agents, dream cycles,
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
	// SkipWorldValidation is set by scoped subcommands (e.g.
	// `spwn world logs <id>`) that pass a live world ID rather than
	// a name declared in spwn.yaml, so the validation against the
	// project manifest would reject a perfectly valid argument.
	SkipWorldValidation bool
}

// Run reads the event log with the given filters and renders it to cmd's
// output. Scoped subcommands call this with a preset WorldID/AgentID.
func Run(cmd *cobra.Command, opts RunOpts) error {
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	// Early input validation: unknown --type / --world values used
	// to silently return "No events yet." which made it impossible
	// to distinguish "empty log" from "typo". Reject them up-front
	// with a concrete list of what's allowed.
	if opts.Type != "" && !isKnownEventType(opts.Type) {
		return fmt.Errorf("unknown event type %q\n\n  Known types:\n    %s",
			opts.Type, strings.Join(sortedEventTypes(), "\n    "))
	}
	if opts.WorldID != "" && !opts.SkipWorldValidation {
		if err := validateWorldFilter(opts.WorldID); err != nil {
			return err
		}
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

// validateWorldFilter rejects --world values that don't match any
// world declared in the current project's spwn.yaml. If no project
// is active we accept the value as-is (there's nothing to validate
// against in legacy global mode).
func validateWorldFilter(name string) error {
	proj, err := cliproject.Find()
	if err != nil || proj == nil || proj.Manifest == nil || len(proj.Manifest.Worlds) == 0 {
		return nil
	}
	if _, ok := proj.Manifest.Worlds[name]; ok {
		return nil
	}
	names := make([]string, 0, len(proj.Manifest.Worlds))
	for k := range proj.Manifest.Worlds {
		names = append(names, k)
	}
	sort.Strings(names)
	return fmt.Errorf("unknown world %q — not declared in spwn.yaml\n\n  Known worlds: %s",
		name, strings.Join(names, ", "))
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
