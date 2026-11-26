package automation

import (
	"fmt"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/automation"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
)

var daemonYes bool

var askTerminal = cliproject.AskTerminal

func init() {
	daemonCmd.Flags().BoolVar(&daemonYes, "yes", false, "Allow, for this run only, the watch folders outside the project that spwn.yaml names, printing each; never kept")
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the engine for the current project until interrupted",
	Long: `Loads the project's automations from spwn.yaml, registers them
with the engine, and blocks. Triggers fire as configured (cron
expressions evaluated against the host clock; filesystem watches via
fsnotify). Each fire writes a receipt to .spwn/runs.jsonl.

Stop with Ctrl-C: the engine drains in-flight dispatches before
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
			fmt.Fprintln(cmd.OutOrStdout(), "no automations declared, nothing to run")
			return nil
		}

		lockPath := filepath.Join(p.Root, ".spwn", "automations", "daemon.lock")
		tree := platform.Within(p.Root)
		if err := tree.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
			tree.Close()
			return fmt.Errorf("create lockfile directory: %w", err)
		}
		lockCheck, err := tree.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
		tree.Close()
		if err != nil {
			return fmt.Errorf("open lock file %s: %w", lockPath, err)
		}
		lockCheck.Close()
		lockFile, err := acquireDaemonLock(lockPath)
		if err != nil {
			return err
		}
		defer releaseDaemonLock(lockFile)

		reaches, declared, err := watchReaches(p)
		if err != nil {
			return err
		}
		allowed, err := cliproject.Consent(p.Root, reaches, daemonYes, askTerminal(), cmd.InOrStdin(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("%w\n%s", err, daemonRefusalHint)
		}

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

		fmt.Fprintf(cmd.OutOrStdout(), "automation daemon for project %s\n", ui.Clean(p.Manifest.Name))
		fmt.Fprintf(cmd.OutOrStdout(), "registered automations: %d\n", countAutomations(p))
		fmt.Fprintln(cmd.OutOrStdout(), "receipts: "+ui.Clean(receiptsPath(p.Root)))
		fmt.Fprintln(cmd.OutOrStdout(), "ctrl-c to stop")

		if err := cliproject.Recheck(p.Root, declared); err != nil {
			return err
		}
		if err := eng.Start(ctx); err != nil {
			return fmt.Errorf("start engine: %w", err)
		}
		if err := cliproject.Recheck(p.Root, declared); err != nil {
			eng.Stop()
			return err
		}
		if err := allowed.Keep(reaches); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "the yes was not kept in %s: %s\n", ui.Clean(cliproject.GrantsPath()), ui.CleanError(err))
		}

		<-ctx.Done()
		fmt.Fprintln(cmd.OutOrStdout(), "\nstopping engine...")
		eng.Stop()
		fmt.Fprintln(cmd.OutOrStdout(), "stopped")
		return nil
	},
}

const daemonRefusalHint = "Run `spwn automation daemon` in a terminal to be asked once (a yes is kept), or `spwn automation daemon --yes` to allow these paths for this run only"

func watchReaches(p *project.Project) ([]cliproject.Reach, []cliproject.Declared, error) {
	var reaches []cliproject.Reach
	var declared []cliproject.Declared
	for _, wname := range slices.Sorted(maps.Keys(p.Manifest.Worlds)) {
		w := p.Manifest.Worlds[wname]
		for _, aname := range slices.Sorted(maps.Keys(w.Automations)) {
			a := w.Automations[aname]
			if a.On.FS == nil || a.On.FS.Path == "" {
				continue
			}
			for what, text := range map[string]string{"world name": wname, "automation name": aname} {
				if err := cliproject.CheckText(what, text); err != nil {
					return nil, nil, err
				}
			}
			resolved, leaves, err := cliproject.ResolveHostPath(p.Root, "watch path", a.On.FS.Path)
			if err != nil {
				return nil, nil, fmt.Errorf("automation %q of world %q: %w", aname, wname, err)
			}
			declared = append(declared, cliproject.Declared{Text: a.On.FS.Path, Resolved: resolved})
			a.On.FS.Path = resolved
			if leaves {
				reaches = append(reaches, cliproject.Reach{Path: resolved, Access: "names and times recorded",
					Grants: fmt.Sprintf("watched by automation %q of world %q: its files' names and times are recorded in the receipts and passed to the project's agent", aname, wname), World: wname})
			}
		}
	}
	return reaches, declared, nil
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
