package agentsstart

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the machine agents.start is asked of: this Mac, which reaches
	// every machine, or inside a workspace its target.
	At func() machine.Ref
}

type Input struct {
	Task, App, Name, Folder string
	// value: an ephemeral workspace made first, on that Mac, with New VM's
	// options, then the agent started on it.
	New                      bool
	Mac, Image, Memory, Disk string
	CPUs                     int
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "start [<task>]",
		Short: "Start an agent on a running workspace (--on), or on a new ephemeral workspace (--new), its task as its first prompt",
		Example: "  station agents start \"Build the landing page\" --on linux-1\n" +
			"  station agents start \"Fix the flaky e2e test\" --new --mac mac-mini\n" +
			"  station agents start --on linux-1 --name docs-bot",
		Args: cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.New, "new", false, "a new ephemeral workspace first, then the agent on it")
			f.StringVar(&in.Name, "name", "", "the agent's name, short and unique; by default its workspace's, then <workspace>"+agents.Nth+"2")
			f.StringVar(&in.App, "app", "", "the agent's app, by its adapter's name; the first station reads by default")
			f.StringVar(&in.Folder, "folder", "", "the folder it works in, inside the workspace")
			f.StringVar(&in.Mac, "mac", "", "with --new: the Mac the ephemeral workspace is made on (default this Mac)")
			f.StringVar(&in.Image, "image", "station-linux", "with --new: the image the ephemeral workspace is made from")
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
			// invariant: SPEC B5.22: the engine's agents.start judges where the
			// agent runs and its name; --on names the workspace, --new the
			// ephemeral made first.
			start := agentscalls.StartIn{Task: in.Task, Name: in.Name, App: agents.App(in.App), Folder: in.Folder}
			if env.Far() {
				start.On = env.Target().Name
			}
			if in.New {
				start.New = &agentscalls.New{Mac: in.Mac, Image: in.Image, CPUs: in.CPUs, Memory: in.Memory, Disk: in.Disk, Confirm: o.YesAnswer()}
			}
			steps := kit.Steps(o)
			a, err := call.Invoke(ctx, env.Calls, d.At(), agentscalls.Start, start, steps)
			steps.Stop()
			if err != nil {
				return err
			}
			return o.Page(pageagents.Started(a, in.New, d.ThisMachine), a)
		},
	}, env)
}
