package portscmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageports "github.com/jterrazz/studio-station/src/pages/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the catalog this run reads: the words of a setting that declared a port.
	Catalog catalog.Catalog
}

type Input struct{ AllMachines, All bool }

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:   "ports",
		Short: "What listens on your machines, who reaches it, and what your VMs share on " + catalog.TailnetProduct,
		Example: "  station ports\n  station ports --all-machines\n  station ports --all\n  station ports --json\n" +
			"  station ports share linux-1 3000\n  station ports stop-sharing linux-1 3000",
		Group: kit.OnAMachine,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.AllMachines, "all-machines", false, "every Mac's ports")
			f.BoolVar(&in.All, "all", false, "the ports that listen on their machine only, too")
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			l, err := call.Invoke(ctx, env.Calls, env.Target(), portscalls.List, portscalls.ListIn{AllMachines: in.AllMachines}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageports.Listed(l, d.Catalog, d.ThisMachine, in.All), pageports.ListContract(l))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
