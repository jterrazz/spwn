package fake

import (
	"context"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Applied struct {
	// invariant: what one Mac of the world keeps of the settings station wrote.
	w   *world.World
	mac string
}

var (
	_ contract.AppliedStore = (*Applied)(nil)
	_ store.Migrator        = (*Applied)(nil)
)

func AppliedOf(w *world.World, mac string) *Applied { return &Applied{w: w, mac: mac} }

func NewApplied() *Applied {
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac")
	return AppliedOf(w, "this-mac")
}

func (a *Applied) kept() map[string]settings.Applied {
	st := a.w.Mac(a.mac).Store()
	if st.Applied == nil {
		st.Applied = map[string]settings.Applied{}
	}
	return st.Applied
}

func (a *Applied) Get(_ context.Context, setting string) (settings.Applied, error) {
	a.w.Lock()
	defer a.w.Unlock()
	got, ok := a.kept()[setting]
	if !ok {
		return settings.Applied{}, store.ErrNotFound
	}
	return got, nil
}

func (a *Applied) Put(_ context.Context, applied settings.Applied) error {
	a.w.Lock()
	defer a.w.Unlock()
	a.kept()[applied.Setting] = applied
	return nil
}

func (a *Applied) List(context.Context) ([]settings.Applied, error) {
	a.w.Lock()
	defer a.w.Unlock()
	kept := a.kept()
	out := make([]settings.Applied, 0, len(kept))
	for _, name := range slices.Sorted(maps.Keys(kept)) {
		out = append(out, kept[name])
	}
	return out, nil
}

func (a *Applied) Delete(_ context.Context, setting string) error {
	a.w.Lock()
	defer a.w.Unlock()
	delete(a.kept(), setting)
	return nil
}

func (a *Applied) Migrate(context.Context) error { return nil }
