package machineslog

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/core/call"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names func() []string
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "log <vm>",
		Short:   "Print the lines a VM's runs wrote: its console, then its hypervisor's own",
		Example: "  station machines log infra --on mac-mini",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Read:     func(*Input) bool { return true },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Log, machinescalls.VMIn{VM: in.Name}, nil)
			if err != nil {
				return kit.OnItsMac(ctx, env, in.Name, err)
			}
			if len(out.Lines) == 0 {
				return nil
			}
			return io.Raw([]byte(strings.Join(out.Lines, "\n") + "\n"))
		},
	}, env)
}
