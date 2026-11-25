package devicescmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedevices "github.com/jterrazz/studio-station/apps/engine/pages/devices"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "devices",
		Short:   "The other devices this Mac lets reach it and its VMs, and this Mac's own key",
		Example: "  station devices\n  station devices --on mac-mini\n  station devices --json",
		Group:   kit.Machines,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), devicescalls.List, struct{}{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagedevices.Page(out, d.ThisMachine), pagedevices.Document{Schema: devicescalls.List.Schema(), Listing: out})
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
