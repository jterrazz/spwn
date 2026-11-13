package machinesrestart

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
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

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "restart <machine>",
		Short:   "Restart a Mac with a FileVault-aware reboot, or a VM with a poweroff and a start",
		Example: "  station machines restart mac-mini --yes",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Asks: true,
		Confirm: func(in *Input) string {
			if !d.IsMac(in.Name) {
				return "Restart " + in.Name + " (poweroff, then start)?"
			}
			if paused := d.Pausing(in.Name); len(paused) > 0 {
				return "Restart " + in.Name + "? " + strings.Join(paused, ", ") + " pauses until it is back"
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
			var report machines.RestartReport
			err := io.Confirmed(func(answer string) error {
				var err error
				report, err = call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Restart, machinescalls.RestartIn{Name: in.Name, Confirm: answer}, nil)
				return kit.OnItsMac(ctx, env, in.Name, err)
			})
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Restarted(report, d.ThisMachine), nil)
		},
	}, env)
}
