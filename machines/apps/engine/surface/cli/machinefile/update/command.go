package update

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

type Input struct{ Machinefile string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "update",
		Short:   "Update every app the machine's Machinefile carries, the managers first, and write the lock anew",
		Example: "  station machinefile update\n  station machinefile update --json",
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile updated, by its name (default: the one this machine carries)")
		},
		Read:  func(*Input) bool { return false },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			steps := kit.Steps(io)
			out, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Upgrade, machinefilecalls.UpgradeIn{Machinefile: in.Machinefile, Interactive: io.Interactive}, steps)
			steps.Stop()
			if len(out.Done) == 0 && out.Lock.Empty() && err != nil {
				return err
			}
			done := worded(out.Done)
			if shown := io.Page(pagemachinefile.Upgraded(out, done, d.ThisMachine), pagemachinefile.UpgradedContract(machinefilecalls.Upgrade.Schema(), out, done)); shown != nil {
				return shown
			}
			if err != nil {
				return err
			}
			failed, total, noun := 0, len(done), "apps"
			for _, w := range done {
				if w.State == catalog.StateBroken {
					failed++
				}
			}
			for _, r := range out.Ran {
				total, noun = total+1, "steps"
				if r.State == catalog.StateBroken {
					failed++
				}
			}
			if failed > 0 {
				return kit.PartialFailure{Failed: failed, Total: total, Noun: noun, FirstHint: "station machinefile update"}
			}
			return nil
		},
	}, env)
}

func worded(list []machinefile.Done) []pagemachinefile.Worded {
	out := make([]pagemachinefile.Worded, 0, len(list))
	for _, d := range list {
		w := pagemachinefile.Worded{Done: d}
		if d.Reason == machinefile.Interrupted || d.Reason == machinefile.NotRun {
			w.Words = kit.UpgradeWords(d.Reason == machinefile.Interrupted)
		}
		out = append(out, w)
	}
	return out
}
