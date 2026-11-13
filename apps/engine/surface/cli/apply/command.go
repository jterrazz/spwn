package apply

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

type Input struct {
	Machinefile string
	Remove      bool
	Phase       string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "apply",
		Short:   "Carry out the plan: install, set and adopt what the machine's Machinefile lists",
		Example: "  station apply\n  station apply --machinefile jterrazz-macbook-pro\n  station apply --remove --yes",
		Group:   kit.StationItself,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile, by its name or its file, which this machine then carries (default: the one it carries)")
			f.BoolVar(&in.Remove, "remove", false, "also remove what station installed here that the Machinefile lists no longer")
			f.StringVar(&in.Phase, "phase", "", "carry out one phase: image (what a VM image bakes), boot (a VM's first start), apply (default)")
		},
		Asks:    true,
		Confirm: func(*Input) string { return "Remove what the Machinefile lists no longer?" },
		Verb:    func(*Input) string { return "remove what the Machinefile lists no longer" },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			return Run(ctx, d, env, *in, io)
		},
	}, env)
}

func Run(ctx context.Context, d Deps, env kit.Env, in Input, io kit.IO) error {
	// value: the plan, then the run: what `station apply` and `station init
	// --machinefile` print.
	p, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Plan, machinefilecalls.PlanIn{Machinefile: in.Machinefile, Phase: catalog.Phase(in.Phase)}, nil)
	if err != nil {
		return err
	}
	if shown := io.Page(pagemachinefile.Plan(p, d.ThisMachine), nil); shown != nil {
		return shown
	}
	var out machinefile.Applied
	steps := kit.Steps(io)
	err = io.Confirmed(func(answer string) (err error) {
		out, err = call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Apply, machinefilecalls.ApplyIn{Machinefile: in.Machinefile, Remove: in.Remove, Phase: catalog.Phase(in.Phase), Interactive: io.Interactive, Confirm: answer}, steps)
		return err
	})
	steps.Stop()
	if len(out.Done) == 0 && len(out.Removed) == 0 && out.Ran == nil {
		return err
	}
	done, removed := worded(out.Done), worded(out.Removed)
	if shown := io.Page(pagemachinefile.Applied(out, done, removed, d.ThisMachine), pagemachinefile.AppliedContract(machinefilecalls.Apply.Schema(), out, done, removed)); shown != nil {
		return shown
	}
	if err != nil {
		return err
	}
	failed, total, noun := 0, len(out.Blocks), "packages"
	for _, b := range out.Blocks {
		if b.State == catalog.StateBroken {
			failed++
		}
	}
	for _, w := range append(done, removed...) {
		total++
		if w.State == catalog.StateBroken {
			failed++
		}
	}
	if out.Ran != nil {
		total, noun = total+1, "steps"
		if out.Ran.State == catalog.StateBroken {
			failed++
		}
	}
	if failed > 0 {
		return kit.PartialFailure{Failed: failed, Total: total, Noun: noun, FirstHint: "station apply"}
	}
	return nil
}

func worded(list []machinefile.Done) []pagemachinefile.Worded {
	out := make([]pagemachinefile.Worded, 0, len(list))
	for _, d := range list {
		w := pagemachinefile.Worded{Done: d}
		if d.Reason != "" {
			w.Words = kit.ApplyWords(d.Reason == machinefile.Interrupted)
		}
		if d.Reason == machinefile.NotRun && d.Detail != "" {
			w.Words[len(w.Words)-1] = d.Detail
		}
		out = append(out, w)
	}
	return out
}
