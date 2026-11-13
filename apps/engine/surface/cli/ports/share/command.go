package portsshare

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ VM, Port string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "share <vm> <port>",
		Short: "Reserve a VM's port now and share it on " + catalog.TailnetProduct + ", its URL printed before anything listens; inside a VM, the port alone",
		Example: "  station ports share linux-1 3000\n" +
			"  API_URL=$(station ports share 3000) pnpm dev     # inside a VM",
		Args: cobra.RangeArgs(1, 2),
		Bind: func(in *Input, args []string) error {
			in.VM, in.Port = "", args[len(args)-1]
			if len(args) == 2 {
				in.VM = args[0]
			}
			return nil
		},
		Read: func(*Input) bool { return false },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			v, err := call.Invoke(ctx, env.Calls, env.Target(), portscalls.Share, portscalls.ShareIn{VM: in.VM, Port: in.Port}, nil)
			if err != nil {
				return err
			}
			return io.PagePiped(pageports.Shared(v, d.ThisMachine), pageports.PortObject(v), v.URLs.Public+"\n")
		},
	}, env)
}
