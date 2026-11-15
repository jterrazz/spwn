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
	// invariant: a Mac's restart asks yes or no (--yes), a server's its name
	// typed (--confirm=<name>), which --yes never answers, a workspace's
	// nothing; the VM's own Mac decides which, so the question is put once
	// its call has refused.
	return kit.Command(kit.Leaf[Input]{
		Use:     "restart <machine>",
		Short:   "Restart a Mac with a FileVault-aware reboot, or a VM with a poweroff and a start",
		Example: "  station machines restart mac-mini --yes\n  station machines restart infra --confirm=infra",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "a server's name, typed, to restart it without the question")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Asks: true,
		Confirm: func(in *Input) string {
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
			restartIt := func(answer string) (machines.RestartReport, error) {
				report, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Restart, machinescalls.RestartIn{Name: in.Name, Confirm: answer}, nil)
				return report, kit.OnItsMac(ctx, env, in.Name, err)
			}
			first := in.Confirm
			if first == "" {
				first = io.YesAnswer()
			}
			report, err := restartIt(first)
			if typedAsked(err) && in.Confirm == "" && !io.Yes && io.Interactive && io.Prompt != nil {
				report, err = typed(in.Name, io, restartIt)
			}
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Restarted(report, d.ThisMachine), nil)
		},
	}, env)
}

func typed(name string, io kit.IO, restartIt func(string) (machines.RestartReport, error)) (machines.RestartReport, error) {
	answers, err := io.Prompt.Form(pagemachines.RestartServerForm(name))
	if err != nil {
		return machines.RestartReport{}, err
	}
	report, err := restartIt(answers["name"])
	var wrong *fail.Error
	if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
		return report, fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
	}
	return report, err
}

func typedAsked(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Code == confirm.Needed
}
