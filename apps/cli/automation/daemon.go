package automation

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/automation"
	"spwn.sh/packages/project"
)

// daemonCmd runs the automation engine for the current project and
// blocks until the user interrupts. The engine reads spwn.yaml,
// registers every automation, fires triggers, and dispatches to the
// architect. Receipts land at <root>/.spwn/runs.jsonl, state at
// <root>/.spwn/automations/state.json.
//
// This is the "make it actually run" path. The eventual home is the
// long-running architect daemon (`spwn architect start`) which would
// host one engine per project; for now this is the simplest one-
// project-at-a-time entry point users can drive from a terminal or
// a launchd/systemd unit.
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the engine for the current project until interrupted",
	Long: `Loads the project's automations from spwn.yaml, registers them
with the engine, and blocks. Triggers fire as configured (cron
expressions evaluated against the host clock; filesystem watches via
fsnotify). Each fire writes a receipt to .spwn/runs.jsonl.

Stop with Ctrl-C — the engine drains in-flight dispatches before
exiting.

Catch-up: on startup, every cron automation that fired before is
checked for missed slots. With catchup: collapse (the default), one
fire is dispatched on resume regardless of how many slots elapsed
during downtime, with the missed count exposed to the prompt
template via {{ .Missed }}. catchup: skip drops missed slots and
resumes at the next scheduled time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := cliproject.Require()
		if err != nil {
			return err
		}

		// Skip the Docker handshake when there's nothing to run.
		// Lets a freshly-scaffolded project run `automation daemon`
		// without first installing Docker.
		if countAutomations(p) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no automations declared — nothing to run")
			return nil
		}

		arc, err := architect.NewFromEnv()
		if err != nil {
			return fmt.Errorf("connect to architect: %w", err)
		}

		// Production engine: fsnotify-backed source for fs triggers,
		// real clock for cron. The factory wires up the rest
		// (dispatcher, receipts, state, command resolver) from the
		// project root.
		fsSource, err := automation.NewFSNotifySource(nil) // nil → stderr default
		if err != nil {
			return fmt.Errorf("fsnotify: %w", err)
		}
		fsWatcher := automation.NewFSWatcher(fsSource, automation.RealClock{})

		eng, err := arc.NewAutomationEngine(architect.AutomationEngineConfig{
			ProjectRoot: p.Root,
			Manifest:    p.Manifest,
			FS:          fsWatcher,
		})
		if err != nil {
			return fmt.Errorf("automation engine: %w", err)
		}

		// SIGINT / SIGTERM → cancel context → engine.Stop drains.
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		fmt.Fprintf(cmd.OutOrStdout(), "automation daemon — project %s\n", p.Manifest.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "registered automations: %d\n", countAutomations(p))
		fmt.Fprintln(cmd.OutOrStdout(), "receipts → "+receiptsPath(p.Root))
		fmt.Fprintln(cmd.OutOrStdout(), "ctrl-c to stop")

		if err := eng.Start(ctx); err != nil {
			return fmt.Errorf("start engine: %w", err)
		}

		<-ctx.Done()
		fmt.Fprintln(cmd.OutOrStdout(), "\nstopping engine…")
		eng.Stop()
		fmt.Fprintln(cmd.OutOrStdout(), "stopped")
		return nil
	},
}

// countAutomations sums the per-world automation maps. Used by
// daemonCmd as a quick "is there anything to run" check before
// reaching for Docker.
func countAutomations(p *project.Project) int {
	if p == nil || p.Manifest == nil {
		return 0
	}
	n := 0
	for _, w := range p.Manifest.Worlds {
		n += len(w.Automations)
	}
	return n
}

// countAutomationsTyped is reserved for future external callers
// (e.g. a status server) that may want to import this helper without
// pulling in the cliproject package. Forwards to countAutomations.
func countAutomationsTyped(p *project.Project) int { return countAutomations(p) }
