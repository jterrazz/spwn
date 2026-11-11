package activitycmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/activity"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pageactivity "github.com/jterrazz/studio-station/src/pages/activity"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Now         func() time.Time
	// value: the short name a Mac is said by (mac-mini).
	Label func(mac string) string
}

type Input struct {
	AllMachines, History bool
	// value: the window the TUI's graphs draw; a screen's flag, never shown.
	Graph time.Duration
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "activity",
		Short:   "What a machine is doing now: CPU, memory, disk, network, thermal, battery, the busiest processes",
		Example: "  station activity\n  station activity --on linux-1\n  station activity --all-machines\n  station activity --json --history",
		Group:   kit.OnAMachine,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.AllMachines, "all-machines", false, "one row per machine")
			f.BoolVar(&in.History, "history", false, "with --json: the 24 h series the engine keeps")
			f.DurationVar(&in.Graph, "graph", 0, "the window the TUI's graphs draw")
			_ = f.MarkHidden("graph")
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.History && !io.JSON {
				return kit.Refuse("--history is for scripts: it needs --json", "station activity --json --history")
			}
			if in.AllMachines && env.Far() {
				return kit.Refuse("--all-machines reads every machine: it takes no --on", "station activity --all-machines")
			}
			if in.AllMachines {
				views, err := every(ctx, d, env)
				if err != nil {
					return err
				}
				return io.Page(pageactivity.AllMachines(views, d.ThisMachine, in.Graph > 0), pageactivity.AllMachinesContract(views, d.Now()))
			}
			v, err := one(ctx, d, env, env.Target(), read{history: in.History, graph: in.Graph})
			if err != nil {
				return err
			}
			return io.Page(pageactivity.Page(v, d.ThisMachine), pageactivity.Contract(v, d.Now()))
		},
	}, env)
}

type read struct {
	history bool
	graph   time.Duration
}

func mac(ctx context.Context, env kit.Env, ref machine.Ref, label string, how read) (pageactivity.View, error) {
	v := pageactivity.View{Name: ref.Name, Label: label, Kind: string(machine.Computer), ThisMac: ref == env.Self, Graph: how.graph}
	if v.ThisMac {
		v.Label = "This Mac"
	}
	r, err := call.Read(ctx, env.Calls, ref, activitycalls.Read, activitycalls.ReadIn{History: how.history, Graph: how.graph}, nil)
	v.Report = r
	return v, err
}

func one(ctx context.Context, d Deps, env kit.Env, target machine.Ref, how read) (pageactivity.View, error) {
	if target == env.Self {
		return mac(ctx, env, env.Self, env.Self.Name, how)
	}
	held, ok := kit.Holder(ctx, env, target.Name)
	if !ok {
		return mac(ctx, env, target, target.Name, how)
	}
	holder := machine.Ref{Name: held.Mac}
	if held.MacIsThisMac {
		holder = env.Self
	}
	if !held.Reachable {
		return pageactivity.View{}, fail.New(fail.Unreachable, call.VMMacUnreachable, held.VM, map[string]string{"mac": held.Mac, "label": d.Label(held.Mac)})
	}
	return vm(ctx, d, env, holder, held.VM, held.Kind, held.Running, how)
}

func vm(ctx context.Context, d Deps, env kit.Env, holder machine.Ref, name string, kind machine.Kind, running bool, how read) (pageactivity.View, error) {
	// invariant: a VM's history is its Mac's engine's, and so is the word
	// that its guest does not answer (the relay into it is that engine's);
	// one it holds no last 10 s of is read inside the VM, twice 1 s apart,
	// its series still that engine's.
	v := pageactivity.View{Name: name, Label: name, Kind: string(kind), Mac: holder.Name, On: d.Label(holder.Name), Stopped: !running, Graph: how.graph}
	if holder == env.Self {
		v.On = "This Mac"
	}
	if !running {
		return v, nil
	}
	in := activitycalls.ReadIn{Of: name, Machine: machine.Ref{Name: name, On: holder.Name}, Kind: kind, History: how.history, Graph: how.graph}
	r, err := call.Read(ctx, env.Calls, holder, activitycalls.Read, in, nil)
	if err == nil && r.Source != activity.NotKept {
		v.Report = r
		return v, nil
	}
	kept := r
	in.Of, in.History, in.Graph = "", false, 0
	r, err = inside(ctx, env, in)
	if err != nil {
		if mac, macErr := call.Read(ctx, env.Calls, holder, activitycalls.Read, activitycalls.ReadIn{Of: name, Machine: in.Machine, Kind: kind, Run: true}, nil); macErr == nil && mac.Last.Run != nil {
			r, err = mac, nil
		}
	}
	if kept.History != nil {
		r.Since, r.History = kept.Since, kept.History
	}
	if kept.Graph != nil {
		r.Graph = kept.Graph
	}
	r.Findings = activity.Persisting(r.Findings, kept.Findings)
	v.Report = r
	return v, err
}

// invariant: a read inside a VM whose guest stopped answering, before its
// Mac's engine noticed, is given up after activity.Inside, never waited on
// until the connection times out.
func inside(ctx context.Context, env kit.Env, in activitycalls.ReadIn) (activity.Report, error) {
	ctx, cancel := context.WithTimeout(ctx, activity.Inside)
	defer cancel()
	return call.Read(ctx, env.Calls, in.Machine, activitycalls.Read, in, nil)
}

func every(ctx context.Context, d Deps, env kit.Env) ([]pageactivity.View, error) {
	// invariant: This Mac then each of its VMs, then every other Mac and
	// its VMs; a Mac that does not answer is its own row and the others are
	// filled.
	list, err := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil, err
	}
	var views []pageactivity.View
	for _, h := range list.All.Macs {
		ref := machine.Ref{Name: h.Name}
		if h.ThisMac {
			ref = env.Self
		}
		if !h.Reachable && !h.ThisMac {
			views = append(views, pageactivity.View{Name: h.Name, Label: d.Label(h.Name), Kind: string(machine.Computer), Silent: "does not answer", SilentType: string(fail.Unreachable)})
			continue
		}
		v, err := mac(ctx, env, ref, d.Label(h.Name), read{})
		views = append(views, silenced(v, err))
		for _, m := range h.VMs {
			v, err := vm(ctx, d, env, ref, m.Name, m.Lifetime.Kind(), m.State == machines.Running, read{})
			views = append(views, silenced(v, err))
		}
	}
	return views, ctx.Err()
}

func silenced(v pageactivity.View, err error) pageactivity.View {
	if err == nil {
		return v
	}
	worded := kit.Translate(err, "")
	v.Silent, v.SilentType, v.SilentCode = worded.Message, string(worded.Type), string(worded.Code)
	if v.SilentType == "" {
		v.SilentType = string(fail.Failed)
	}
	return v
}
