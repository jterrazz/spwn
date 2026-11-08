package machinesstop

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct {
	Name, Confirm string
	// value: the live agents its call named, which stop first.
	agents string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a workspace's stop asks yes or no (--yes), a server's its
	// name typed (--confirm=<name>), which --yes never answers; the VM's own
	// Mac decides which, so the question is put once its call has refused.
	return kit.Command(kit.Leaf[Input]{
		Use:     "stop <vm>",
		Short:   "Stop a VM; it is kept, with its disk, until started or deleted",
		Example: "  station machines stop linux-1 --yes\n  station machines stop infra --confirm=infra",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "a server's name, typed, to stop it without the question")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Asks: true,
		Confirm: func(in *Input) string {
			question := "Stop " + in.Name + "? It is kept, stopped, until you start it"
			if in.agents != "" {
				question += ". Its agents stop first: " + in.agents + "."
			}
			return question
		},
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
			switch {
			case typedAsked(err) && in.Confirm == "" && !io.Yes && io.Interactive && io.Prompt != nil:
				out, err = typed(in.Name, io, stopIt)
			case yesAsked(err):
				in.agents = agentsNamed(err)
				err = io.Confirmed(func(answer string) error {
					out, err = stopIt(answer)
					return err
				})
			}
			if err != nil {
				return err
			}
			switch {
			case out.Already:
				return io.Page(pagemachines.Already("stop", in.Name, false, d.ThisMachine), nil)
			case out.Kind == machine.Server:
				start := "station machines start " + in.Name
				if env.Far() {
					start += " --on " + env.Target().Name
				}
				return io.Page(pagemachines.ServerStopped(in.Name, start, d.ThisMachine), nil)
			}
			if len(out.Agents) > 0 {
				return io.Page(pagemachines.StoppedWithAgents(in.Name, out.Agents, d.ThisMachine), nil)
			}
			return io.Page(pagemachines.Change("stop", in.Name, "stopped, kept", d.ThisMachine), nil)
		},
	}, env)
}

func typed(name string, io kit.IO, stopIt func(string) (machinescalls.StopOut, error)) (machinescalls.StopOut, error) {
	answers, err := io.Prompt.Form(pagemachines.StopServerForm(name))
	if err != nil {
		return machinescalls.StopOut{}, err
	}
	out, err := stopIt(answers["name"])
	var wrong *fail.Error
	if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
		return out, fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
	}
	return out, err
}

func typedAsked(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Code == confirm.Needed
}

func yesAsked(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Code == confirm.YesNeeded
}

func agentsNamed(err error) string {
	var e *fail.Error
	if errors.As(err, &e) {
		return e.Detail["agents"]
	}
	return ""
}
