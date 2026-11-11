package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

type ListIn struct {
	// value: each VM's own accounts too, as the pages draw them.
	VMs bool
}

var List = call.Watch[Deps, ListIn, accounts.Report]{
	Name: "accounts.list", Section: section.Accounts, Where: call.AtTarget, Applies: accounts.Applies,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) (<-chan call.Update[accounts.Report], error) {
		// invariant: a Mac's reading carries the shares it holds and its share
		// list with every update; a workspace's carries none of its own.
		var shared []accounts.Shared
		var list []string
		if d.Kind == machine.Computer && d.Shares != nil {
			held, err := d.Shares.Shares(ctx)
			if err != nil {
				return nil, err
			}
			shared, list = accounts.SharedOf(held), d.Shares.ShareList()
		}
		reads := d.Accounts.Read(ctx, in.VMs)
		out := make(chan accounts.Report)
		go func() {
			defer close(out)
			for r := range reads {
				r.Shares, r.ShareList = shared, list
				select {
				case out <- r:
				case <-ctx.Done():
					return
				}
			}
		}()
		return call.Updates(ctx, out), nil
	},
}
