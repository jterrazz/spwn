package calls

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	settingRestartAfterPowerCut = "restart-after-power-cut"
	settingNeverSleeps          = "never-sleeps"
	settingAutoLogin            = "autologin"
)

// invariant: the readiness reads the settings and the Mac's own sleep first,
// and the Mac's other facts only when the Mac needs a readiness at all.
func (r *Reader[Item]) readiness(ctx context.Context) []overview.Check {
	states := map[string]catalog.State{}
	if items, err := r.d.Settings.List(ctx); err == nil {
		for _, it := range items {
			name, state := r.d.Settings.Placed(it)
			states[name] = state
		}
	}
	on := func(name string) machines.ReadinessSetting {
		state, ok := states[name]
		if !ok {
			state = catalog.StateUnknown
		}
		return machines.ReadinessSetting{Name: name, State: state, Root: r.needsRoot(name), Recorded: r.d.Settings.Recorded(ctx, name)}
	}
	f := machines.ReadinessFacts{
		Mac:                   r.label(),
		RestartsAfterPowerCut: on(settingRestartAfterPowerCut), NeverSleeps: on(settingNeverSleeps), LogsIn: on(settingAutoLogin),
		EngineKnown: r.d.ThisMac.Agent != nil,
	}
	if sleep, ok := r.fact(ctx, catalog.FactSystemSleep); ok && settings.NeverSleeps(sleep) {
		f.MacNeverSleeps = true
	}
	if machines.Readiness(f) == nil && r.d.Servers != nil {
		f.MacsServer = r.d.Servers.Hosted(ctx)
	}
	if machines.Readiness(f) == nil {
		return []overview.Check{}
	}
	f.FileVault, f.AuthRestart = r.factState(ctx, catalog.FactFileVault), r.factState(ctx, catalog.FactAuthRestart)
	f.EngineAtLogin = f.EngineKnown && r.d.ThisMac.Agent()
	items := machines.Readiness(f)
	out := make([]overview.Check, 0, len(items))
	for _, it := range items {
		out = append(out, overview.Check{Name: it.Name, State: it.State, Value: it.Value, Detail: it.Fix})
	}
	return out
}

func (r *Reader[Item]) needsRoot(setting string) bool {
	it, _ := r.d.Catalog.Setting(setting)
	return it.NeedsRoot
}

func (r *Reader[Item]) factState(ctx context.Context, f catalog.Fact) catalog.State {
	if got, ok := r.fact(ctx, f); ok {
		return got.State
	}
	return catalog.StateUnknown
}

func (r *Reader[Item]) fact(ctx context.Context, f catalog.Fact) (settings.Fact, bool) {
	ctx, cancel := context.WithTimeout(ctx, defaultBudget)
	defer cancel()
	got, err := r.d.Facts.Read(ctx, f)
	return got, err == nil && ctx.Err() == nil
}

func (r *Reader[Item]) label() string {
	registered, _ := r.d.Name.Get()
	name := machines.ThisMacName(registered, r.d.ThisMac.Hostname)
	if short, ok := strings.CutPrefix(name, r.d.ThisMac.User+"-"); ok && short != "" && r.d.ThisMac.User != "" {
		return short
	}
	return name
}
