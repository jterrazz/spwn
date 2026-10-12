package automation

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/automation"
	"spwn.sh/packages/project"
)

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

		if countAutomations(p) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no automations declared — nothing to run")
			return nil
		}

		lockPath := filepath.Join(p.Root, ".spwn", "automations", "daemon.lock")
		if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
			return fmt.Errorf("create lockfile directory: %w", err)
		}
		lockFile, err := acquireDaemonLock(lockPath)
		if err != nil {
			return err
		}
		defer releaseDaemonLock(lockFile)

		arc, err := architect.NewFromEnv()
		if err != nil {
			return fmt.Errorf("connect to architect: %w", err)
		}

		fsSource, err := automation.NewFSNotifySource(nil)
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
