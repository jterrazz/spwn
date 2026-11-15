package wire

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/remote/each"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

// value: the apps matrix's reach: the fleet as This Mac lists it, each
// machine's apps through this run's Router.
type appsAllMachines struct{ m machine }

func (f appsAllMachines) Machines(ctx context.Context) (machines.AllMachines, error) {
	return settingsAllMachines(f).Machines(ctx)
}

func (f appsAllMachines) Read(ctx context.Context, refs []coremachine.Ref) map[coremachine.Ref]appscalls.Answer {
	r := f.m.far.router
	names := make([]string, len(refs))
	by := make(map[string]coremachine.Ref, len(refs))
	for i, ref := range refs {
		names[i] = ref.Name + "@" + ref.On
		by[names[i]] = ref
	}
	got := each.All(ctx, names, func(ctx context.Context, name string) ([]apps.Report, error) {
		reports, err := call.Read(ctx, r, by[name], appscalls.List, appscalls.ListIn{}, nil)
		if err == nil && reports == nil {
			reports = []apps.Report{}
		}
		return reports, err
	})
	out := make(map[coremachine.Ref]appscalls.Answer, len(refs))
	for name, a := range got {
		if a.Err != nil {
			out[by[name]] = appscalls.Answer{Why: kit.Screened(a.Err, "apps").Message}
			continue
		}
		out[by[name]] = appscalls.Answer{Reports: a.Value}
	}
	return out
}
