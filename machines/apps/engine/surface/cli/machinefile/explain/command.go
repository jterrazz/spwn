package explain

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

type Input struct{ Key, Machinefile string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "explain [<key>]",
		Short:   "Each value of a Machinefile, and the origin it comes from; <key> narrows it, --machinefile picks the file",
		Example: "  station machinefile explain\n  station machinefile explain apps.jq\n  station machinefile explain vm --machinefile base-ubuntu",
		Args:    cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile explained (default: the one this machine is bound to, else the one named after it)")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Key = args[0]
			}
			return nil
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Explain, machinefilecalls.ExplainIn{Key: in.Key, Machinefile: in.Machinefile}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachinefile.Explain(out.Name, in.Key, out.Values, d.ThisMachine), pagemachinefile.Contract(out.Name, out.Files, out.Values))
		},
	}, env)
}
