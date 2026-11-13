package plan

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

type Input struct {
	Machinefile string
	Phase       string
	Diff        bool
}

func Diffs(ctx context.Context, env kit.Env, p machinefile.Plan) string {
	// value: the diff of every file apply would write, from what the file
	// holds (`-`) to what apply writes (`+`): a setting's, `station settings
	// diff` drawn the other way, and each shell block of ~/.zshrc, in the
	// plan's order; a setting whose diff cannot be read says nothing here.
	var out strings.Builder
	shown := map[string]bool{}
	for _, l := range p.Lines {
		if l.Step.Kind == catalog.KindSetting && (l.Action == machinefile.Set || l.Seen.Condition == machinefile.Changed) {
			if d, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Diff, settingscalls.DiffIn{Name: l.Step.Item}, nil); err == nil {
				out.WriteString(settings.ReversedDiff(d.Diff))
			}
		}
		for _, b := range p.Blocks {
			if b.Item == l.Step.Item && !shown[b.ID] {
				shown[b.ID] = true
				out.WriteString(b.Diff)
			}
		}
	}
	for _, b := range p.Blocks {
		if !shown[b.ID] {
			out.WriteString(b.Diff)
		}
	}
	return out.String()
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "plan",
		Short:   "What station apply would do here: what it asks of you, installs, sets and adopts",
		Example: "  station plan\n  station plan --diff\n  station plan --machinefile station-macos",
		Group:   kit.StationItself,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile planned, by its name or its file (default: the one this machine carries)")
			f.StringVar(&in.Phase, "phase", "", "plan one phase: image (what a VM image bakes), boot (a VM's first start), apply (default)")
			f.BoolVar(&in.Diff, "diff", false, "what apply would change in each file it writes, from the file to station's: each shell block of ~/.zshrc and each setting's file")
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Plan, machinefilecalls.PlanIn{Machinefile: in.Machinefile, Phase: catalog.Phase(in.Phase)}, nil)
			if err != nil {
				return err
			}
			if in.Diff && !io.JSON && !io.PageJSON {
				if text := Diffs(ctx, env, p); text != "" {
					return io.Raw([]byte(text))
				}
			}
			return io.Page(pagemachinefile.Plan(p, d.ThisMachine), pagemachinefile.PlanContract("station.machinefile.plan/v1", p))
		},
	}, env)
}
