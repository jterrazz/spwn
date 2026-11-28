package machinesstop

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct {
	Name, Confirm string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a workspace's stop asks nothing, a server's asks yes or no
	// (--yes, or --confirm=<name>), the VM's own Mac grading which
	// (confirm.Of), so the question is put once its call has refused. The
	// outcome says how to undo it.
	return kit.Command(kit.Leaf[Input]{
		Use:     "stop <vm>",
		Short:   "Stop a VM; it is kept, with its disk, until started or deleted",
		Example: "  station machines stop linux-1\n  station machines stop infra --yes",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the VM's name, typed, to stop a server without the question")
		},
		Bind: func(in *Input, args []string) error {
			name, err := env.VM(args[0])
			in.Name = name
			return err
		},
		Asks:     true,
		Confirm:  func(in *Input) string { return "Stop " + in.Name + "? What it serves stops until you start it again." },
		Verb:     func(*Input) string { return "stop" },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			stopIt := func(answer string) (machinescalls.StopOut, error) {
				stop := io.Wait("stopping " + in.Name)
				defer stop()
				out, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Stop, machinescalls.StopIn{VM: in.Name, Confirm: answer}, nil)
				return out, kit.OnItsMac(ctx, env, in.Name, err)
			}
			first := in.Confirm
			if first == "" {
				first = io.YesAnswer()
			}
			out, err := stopIt(first)
			if yesAsked(err) {
				err = io.Confirmed(func(answer string) error {
					out, err = stopIt(answer)
					return err
				})
			}
			if err != nil {
				return err
			}
			if out.Already {
				return io.Page(pagemachines.Already("stop", in.Name, false, d.ThisMachine), nil)
			}
			start := "station machines start " + in.Name
			if env.Far() {
				start += " --on " + env.Target().Name
			}
			if out.Kind == machine.Server {
				return io.Page(pagemachines.ServerStopped(in.Name, start, d.ThisMachine), nil)
			}
			p := pagemachines.Change("stop", in.Name, "stopped, kept", d.ThisMachine)
			p.Hints = []page.Hint{*page.Cmd(start)}
			return io.Page(p, nil)
		},
	}, env)
}

func yesAsked(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Code == confirm.YesNeeded
}
