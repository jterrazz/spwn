package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type ReadIn struct {
	// value: each VM's own accounts too, as the pages draw them.
	VMs bool
}

var Read = call.Watch[Deps, ReadIn, accounts.Report]{
	Name: "accounts.read", Section: section.Accounts, Where: call.AtTarget, Applies: accounts.Applies,
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (<-chan call.Update[accounts.Report], error) {
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
