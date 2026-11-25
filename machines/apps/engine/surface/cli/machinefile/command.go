package machinefilecmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "machinefile",
		Short:   "The Machinefiles this Mac reads: what each makes, where it is read, and whether its machines match it",
		Example: "  station machinefile\n  station machinefile --json\n  station machinefile apply",
		Group:   kit.OnAMachine,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, machine.Ref{}, machinefilecalls.ListFiles, machinefilecalls.FilesIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachinefile.Files(out.Files, out.Verdict.Sentence, d.ThisMachine), pagemachinefile.FilesContract(machinefilecalls.ListFiles.Schema(), out.Files, out.Verdict.Sentence, string(out.Verdict.Tone)))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
