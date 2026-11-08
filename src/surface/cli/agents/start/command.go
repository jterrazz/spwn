package agentsstart

import (
	"context"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: every agent of every machine, whose names a new one may not take.
	Everywhere func(ctx context.Context) ([]agents.Agent, error)
}

type Input struct {
	Task, App, Name, Folder string
	// value: a throwaway workspace made first, on that Mac, with New VM's
	// options, then the agent started on it.
	New                      bool
	Mac, Image, Memory, Disk string
	CPUs                     int
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "start [<task>]",
		Short: "Start an agent on a running workspace (--on), or on a new throwaway (--new), its task as its first prompt",
		Example: "  station agents start \"Build the landing page\" --on linux-1\n" +
			"  station agents start \"Fix the flaky e2e test\" --new --mac mac-mini\n" +
			"  station agents start --on linux-1 --name docs-bot",
		Args: cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.New, "new", false, "a new throwaway workspace first, then the agent on it")
			f.StringVar(&in.Name, "name", "", "the agent's name, short and unique; by default its workspace's, then <workspace>"+agents.Nth+"2")
			f.StringVar(&in.App, "app", "", "the agent's app, by its adapter's name; the first station reads by default")
			f.StringVar(&in.Folder, "folder", "", "the folder it works in, inside the workspace")
			f.StringVar(&in.Mac, "mac", "", "with --new: the Mac the throwaway is made on (default this Mac)")
			f.StringVar(&in.Image, "image", "linux", "with --new: the image the throwaway is made from")
			f.IntVar(&in.CPUs, "cpus", 0, "with --new: virtual CPUs (default 4)")
			f.StringVar(&in.Memory, "memory", "", "with --new: memory with its unit, e.g. 8G (default 4G)")
			f.StringVar(&in.Disk, "disk", "", "with --new: disk size with its unit, e.g. 50G (default 20G)")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) > 0 {
				in.Task = args[0]
			}
			return nil
		},
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			// invariant: SPEC B5.22: exactly one of --on and --new on a Mac;
			// inside a workspace the agent starts on it.
			onMac := env.Kind == machine.Mac && !env.Far()
			if in.New == !onMac {
				return fail.New(fail.Refused, agentscalls.NoWhere, "station agents start", map[string]string{"both": boolWord(in.New)})
			}
			if in.Name != "" && env.Kind == machine.Mac {
				if err := free(ctx, d, in.Name); err != nil {
					return err
				}
			}
			at := env.Target()
			if in.New {
				vm, err := create(ctx, env, in, o)
				if err != nil {
					return err
				}
				at = machine.Ref{Name: vm}
				a, err := call.Invoke(ctx, env.Calls, at, agentscalls.Start, agentscalls.StartIn{App: agents.App(in.App), Name: in.Name, Folder: in.Folder, Task: in.Task}, nil)
				if err != nil {
					return &machines.AgentNotStartedError{Name: vm, Cause: err}
				}
				return o.Page(pageagents.Started(a, true, d.ThisMachine), a)
			}
			a, err := call.Invoke(ctx, env.Calls, at, agentscalls.Start, agentscalls.StartIn{App: agents.App(in.App), Name: in.Name, Folder: in.Folder, Task: in.Task}, nil)
			if err != nil {
				return err
			}
			return o.Page(pageagents.Started(a, false, d.ThisMachine), a)
		},
	}, env)
}

func boolWord(both bool) string {
	if both {
		return "both"
	}
	return "neither"
}

// value: refuses a name another agent of any machine holds, in its key.
func free(ctx context.Context, d Deps, name string) error {
	all, err := d.Everywhere(ctx)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(all, func(a agents.Agent) bool { return agents.Clashes([]string{a.Name}, name) }) {
		return fail.New(fail.Refused, agentscalls.NameTaken, name, nil)
	}
	return nil
}

// value: the throwaway workspace made for the agent, by its name, on the
// Mac asked; its steps said as they run.
func create(ctx context.Context, env kit.Env, in *Input, o kit.IO) (string, error) {
	size, err := machines.ParseResources(in.CPUs, in.Memory, in.Disk)
	if err != nil {
		return "", err
	}
	target := env.Self
	if in.Mac != "" {
		target = machine.Ref{Name: in.Mac}
	}
	steps := kit.Steps(o)
	defer steps.Stop()
	created, err := call.Invoke(ctx, env.Calls, target, machinescalls.CreateWorkspace, machinescalls.CreateIn{
		Request: machines.CreateRequest{Image: in.Image, Name: in.Name, Lifetime: machines.Throwaway, Size: size},
		Confirm: o.YesAnswer(),
	}, steps)
	if err != nil {
		return "", err
	}
	return created.VM.Name, nil
}
