package agentsstart

import (
	"context"

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
	// value: the names every machine holds, which a new agent may not take:
	// read across every machine on a Mac, asked of its Mac's engine inside a
	// workspace.
	Held func(ctx context.Context) (agents.Held, error)
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
				return fail.New(fail.Refused, agentscalls.NoWhere, "station agents start", map[string]string{"both": nowhere(in.New, env)})
			}
			held, err := d.Held(ctx)
			if err != nil {
				return err
			}
			at := env.Target()
			vm := at.Name
			if in.New {
				vm = in.Name
			}
			if ws, taken := held.Takes(in.Name, vm); in.Name != "" && taken {
				return fail.New(fail.Refused, agentscalls.NameTaken, in.Name, map[string]string{"workspace": ws, "nth": ws + agents.Nth + "<n>"})
			}
			if in.New {
				var made string
				if made, err = create(ctx, env, in, o); err != nil {
					return err
				}
				at = machine.Ref{Name: made}
			}
			if in.Name == "" {
				in.Name = agents.NameFor(at.Name, held.Agents)
			}
			a, err := call.Invoke(ctx, env.Calls, at, agentscalls.Start, agentscalls.StartIn{App: agents.App(in.App), Name: in.Name, Folder: in.Folder, Task: in.Task}, nil)
			if in.New {
				if err != nil {
					return &machines.AgentNotStartedError{Name: at.Name, Cause: err}
				}
				return o.Page(pageagents.Started(a, true, d.ThisMachine), a)
			}
			if err != nil {
				return err
			}
			return o.Page(pageagents.Started(a, false, d.ThisMachine), a)
		},
	}, env)
}

// value: how a start names no one place: neither, both, or --new alone
// from inside a workspace.
func nowhere(asked bool, env kit.Env) string {
	switch {
	case !asked:
		return "neither"
	case !env.Far() && env.Kind == machine.Workspace:
		return "inside"
	}
	return "both"
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
