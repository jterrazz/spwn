package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type Deps struct {
	Apps Apps
	// value: the catalog's apps: one that lives on other machines is still
	// shown, with where it lives.
	Known []catalog.App
	// value: the kind of the machine these calls run on.
	Kind machine.Kind
}

type Apps interface {
	Watch(ctx context.Context) <-chan []apps.Report
	Forget()
	CheckInstall(ctx context.Context, r apps.InstallRequest) error
	Install(ctx context.Context, r apps.InstallRequest, p call.Progress) ([]apps.Outcome, error)
	CheckUninstall(names []string) error
	Uninstall(ctx context.Context, names []string, word bool) ([]apps.Outcome, error)
	CheckUpgrade(names []string) error
	Upgrade(ctx context.Context, r apps.UpgradeRequest) ([]apps.Outcome, error)
	Move(ctx context.Context, name string, to catalog.Method, p call.Progress) ([]apps.Outcome, error)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(List, d), call.Bind(Show, d), call.Bind(Install, d), call.Bind(Update, d), call.Bind(Remove, d), call.Bind(Move, d)}
}
