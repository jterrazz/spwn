package approvalscmd

import (
	"context"

	"github.com/spf13/cobra"

	approvalscalls "github.com/jterrazz/studio-station/apps/engine/approvals/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageapprovals "github.com/jterrazz/studio-station/apps/engine/pages/approvals"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[struct{}]{
		Use:     "approvals",
		Short:   "What macOS lets Spwn do on this Mac, as its engine read it",
		Example: "  station approvals\n  station approvals --on mac-mini\n  station approvals --json",
		Group:   kit.Machine,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), approvalscalls.List, struct{}{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageapprovals.Page(out, d.ThisMachine), pageapprovals.Document{Schema: approvalscalls.List.Schema(), Listing: out})
		},
	}, env)
}
