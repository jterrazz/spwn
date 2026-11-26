package machinesrestart

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names func() []string
	// value: whether a name is another Mac, restarted from here; any other
	// name is a VM's.
	IsMac func(name string) bool
	// value: the VMs that run on a Mac now, which its restart pauses.
	Pausing     func(mac string) []string
	ThisMachine page.Context
}

type Input struct{ Name, Confirm string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a Mac's restart and an up server's ask yes or no (--yes, or
	// --confirm=<name>), naming what pauses; a down server's and a
	// workspace's ask nothing. The VM's own Mac grades which (confirm.Of),
	// so the question is put once its call has refused.
	return kit.Command(kit.Leaf[Input]{
		Use:     "restart <machine>",
		Short:   "Restart a Mac with a FileVault-aware reboot, or a VM with a poweroff and a start",
		Example: "  station machines restart mac-mini --yes\n  station machines restart infra --yes",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the machine's name, typed, to restart it without the question")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Asks: true,
		Confirm: func(in *Input) string {
			if !d.IsMac(in.Name) {
				return "Restart " + in.Name + "? What it serves stops while it restarts."
			}
			if paused := d.Pausing(in.Name); len(paused) > 0 {
				return "Restart " + in.Name + "? " + and(paused) + " " + pause(len(paused)) + " until it is back."
			}
			return "Restart " + in.Name + "?"
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if d.IsMac(in.Name) {
				var back machines.Back
				err := io.Confirmed(func(answer string) error {
					var err error
					back, err = call.Invoke(ctx, env.Calls, machine.Ref{}, machinescalls.RestartMac, machinescalls.RestartMacIn{Name: in.Name, Confirm: answer}, nil)
					return err
				})
				if err != nil {
					return err
				}
				return io.Page(pagemachines.Back(back, d.ThisMachine), nil)
			}
			restartIt := func(answer string) (machines.RestartReport, error) {
				report, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Restart, machinescalls.RestartIn{Name: in.Name, Confirm: answer}, nil)
				return report, kit.OnItsMac(ctx, env, in.Name, err)
			}
			first := in.Confirm
			if first == "" {
				first = io.YesAnswer()
			}
			report, err := restartIt(first)
			if yesAsked(err) {
				err = io.Confirmed(func(answer string) error {
					report, err = restartIt(answer)
					return err
				})
			}
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Restarted(report, d.ThisMachine), nil)
		},
	}, env)
}

// value: names said as a sentence: "a", "a and b", "a, b and c".
func and(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func pause(n int) string {
	if n == 1 {
		return "pauses"
	}
	return "pause"
}

func yesAsked(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Code == confirm.YesNeeded
}
