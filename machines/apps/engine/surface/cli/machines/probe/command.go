package machinesprobe

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names    func() []string
	Overview func(ctx context.Context, io kit.IO) error
}

type Input struct{ Name string }

func Alias(d Deps, env kit.Env) *cobra.Command {
	return kit.Alias("probe", kit.Command(kit.Leaf[Input]{
		Use:   "probe <machine>",
		Short: "One machine's Overview",
		Args:  cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Read:     func(*Input) bool { return true },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if env.On != nil {
				env.On.Machine = in.Name
			}
			return d.Overview(ctx, io)
		},
	}, env), "")
}
