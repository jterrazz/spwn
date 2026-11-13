package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Deps struct {
	Accounts Accounts
	Shares   Shares
	Kind     machine.Kind
}

type Accounts interface {
	// value: withVMs reads each VM's own accounts through the engine too.
	Read(ctx context.Context, withVMs bool) <-chan accounts.Report
}

type Shares interface {
	ShareList() []string
	Shares(ctx context.Context) (map[string][]accounts.Share, error)
	CheckFrom(ctx context.Context, account, vm, from string) error
	CheckShare(ctx context.Context, account, vm string) (string, error)
	Share(ctx context.Context, account, vm string, term time.Duration) (accounts.Shared, error)
	CheckTakeBack(ctx context.Context, account, vm string) (string, error)
	TakeBack(ctx context.Context, account, vm string) error
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(List, d), call.Bind(Share, d), call.Bind(TakeBack, d)}
}
